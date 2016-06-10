package tsz

import (
	xtime "code.uber.internal/infra/memtsdb/x/time"
)

const (
	// special markers
	defaultEndOfStreamMarker Marker = iota
	defaultAnnotationMarker
	defaultTimeUnitMarker

	// marker encoding information
	defaultMarkerOpcode        = 0x100
	defaultNumMarkerOpcodeBits = 9
	defaultNumMarkerValueBits  = 2
)

var (
	// default time encoding schemes
	defaultZeroBucket             = newTimeBucket(0x0, 1, 0)
	defaultNumValueBitsForBuckets = []int{7, 9, 12}
	// TODO(xichen): set more reasonable defaults once we have more knowledge
	// of the use cases for time units other than seconds.
	defaultTimeEncodingSchemes = map[xtime.Unit]TimeEncodingScheme{
		xtime.Second:      newTimeEncodingScheme(defaultNumValueBitsForBuckets, 32),
		xtime.Millisecond: newTimeEncodingScheme(defaultNumValueBitsForBuckets, 32),
		xtime.Microsecond: newTimeEncodingScheme(defaultNumValueBitsForBuckets, 64),
		xtime.Nanosecond:  newTimeEncodingScheme(defaultNumValueBitsForBuckets, 64),
	}

	// default marker encoding scheme
	defaultMarkerEncodingScheme = newMarkerEncodingScheme(
		defaultMarkerOpcode,
		defaultNumMarkerOpcodeBits,
		defaultNumMarkerValueBits,
		defaultEndOfStreamMarker,
		defaultAnnotationMarker,
		defaultTimeUnitMarker,
	)
)

// TimeBucket represents a bucket for encoding time values.
type TimeBucket interface {

	// Opcode is the opcode prefix used to encode all time values in this range.
	Opcode() uint64

	// NumOpcodeBits is the number of bits used to write the opcode.
	NumOpcodeBits() int

	// Min is the minimum time value accepted in this range.
	Min() int64

	// Max is the maximum time value accepted in this range.
	Max() int64

	// NumValueBits is the number of bits used to write the time value.
	NumValueBits() int
}

type timeBucket struct {
	min           int64
	max           int64
	opcode        uint64
	numOpcodeBits int
	numValueBits  int
}

// newTimeBucket creates a new time bucket.
func newTimeBucket(opcode uint64, numOpcodeBits, numValueBits int) TimeBucket {
	return &timeBucket{
		opcode:        opcode,
		numOpcodeBits: numOpcodeBits,
		numValueBits:  numValueBits,
		min:           -(1 << uint(numValueBits-1)),
		max:           (1 << uint(numValueBits-1)) - 1,
	}
}

func (tb *timeBucket) Opcode() uint64     { return tb.opcode }
func (tb *timeBucket) NumOpcodeBits() int { return tb.numOpcodeBits }
func (tb *timeBucket) Min() int64         { return tb.min }
func (tb *timeBucket) Max() int64         { return tb.max }
func (tb *timeBucket) NumValueBits() int  { return tb.numValueBits }

// TimeEncodingScheme captures information related to time encoding.
type TimeEncodingScheme interface {

	// ZeroBucket is time bucket for encoding zero time values.
	ZeroBucket() TimeBucket

	// Buckets are the ordered time buckets used to encode non-zero, non-default time values.
	Buckets() []TimeBucket

	// DefaultBucket is the time bucket for catching all other time values not included in the regular buckets.
	DefaultBucket() TimeBucket
}

type timeEncodingScheme struct {
	zeroBucket    TimeBucket
	buckets       []TimeBucket
	defaultBucket TimeBucket
}

// newTimeEncodingScheme creates a new time encoding scheme.
// NB(xichen): numValueBitsForBbuckets should be ordered by value in ascending order (smallest value first).
func newTimeEncodingScheme(numValueBitsForBuckets []int, numValueBitsForDefault int) TimeEncodingScheme {
	numBuckets := len(numValueBitsForBuckets)
	buckets := make([]TimeBucket, 0, numBuckets)
	numOpcodeBits := 1
	opcode := uint64(0)
	i := 0
	for i < numBuckets {
		opcode = uint64(1<<uint(i+1)) | opcode
		buckets = append(buckets, newTimeBucket(opcode, numOpcodeBits+1, numValueBitsForBuckets[i]))
		i++
		numOpcodeBits++
	}
	defaultBucket := newTimeBucket(opcode|0x1, numOpcodeBits, numValueBitsForDefault)

	return &timeEncodingScheme{
		zeroBucket:    defaultZeroBucket,
		buckets:       buckets,
		defaultBucket: defaultBucket,
	}
}

func (tes *timeEncodingScheme) ZeroBucket() TimeBucket    { return tes.zeroBucket }
func (tes *timeEncodingScheme) Buckets() []TimeBucket     { return tes.buckets }
func (tes *timeEncodingScheme) DefaultBucket() TimeBucket { return tes.defaultBucket }

// TimeEncodingSchemes defines the time encoding schemes for different time units.
type TimeEncodingSchemes map[xtime.Unit]TimeEncodingScheme

// Marker represents the markers.
type Marker byte

// MarkerEncodingScheme captures the information related to marker encoding.
type MarkerEncodingScheme interface {

	// Opcode returns the marker opcode.
	Opcode() uint64

	// NumOpcodeBits returns the number of bits used for the opcode.
	NumOpcodeBits() int

	// NumValueBits returns the number of bits used for the marker value.
	NumValueBits() int

	// EndOfStream returns the end of stream marker.
	EndOfStream() Marker

	// Annotation returns the annotation marker.
	Annotation() Marker

	// TimeUnit returns the time unit marker.
	TimeUnit() Marker

	// Tail will return the tail portion of a stream including the relevant bits
	// in the last byte along with the end of stream marker.
	Tail(streamLastByte byte, streamCurrentPosition int) []byte
}

type markerEncodingScheme struct {
	opcode        uint64
	numOpcodeBits int
	numValueBits  int
	endOfStream   Marker
	annotation    Marker
	timeUnit      Marker
	tails         [256][8][]byte
}

func newMarkerEncodingScheme(
	opcode uint64,
	numOpcodeBits int,
	numValueBits int,
	endOfStream Marker,
	annotation Marker,
	timeUnit Marker,
) MarkerEncodingScheme {
	scheme := &markerEncodingScheme{
		opcode:        opcode,
		numOpcodeBits: numOpcodeBits,
		numValueBits:  numValueBits,
		endOfStream:   endOfStream,
		annotation:    annotation,
		timeUnit:      timeUnit,
	}
	// NB(r): we precompute all possible tail streams dependent on last byte
	// so we never have to pool or allocate tails for each stream when we
	// want to take a snapshot of the current stream returned by the `Stream` method.
	for i := range scheme.tails {
		for j := range scheme.tails[i] {
			pos := j + 1
			tmp := newOStream(nil, false)
			tmp.WriteBits(uint64(i)>>uint(8-pos), pos)
			writeSpecialMarker(tmp, scheme, endOfStream)
			tail, _ := tmp.rawbytes()
			scheme.tails[i][j] = tail
		}
	}
	return scheme
}

func (mes *markerEncodingScheme) Opcode() uint64              { return mes.opcode }
func (mes *markerEncodingScheme) NumOpcodeBits() int          { return mes.numOpcodeBits }
func (mes *markerEncodingScheme) NumValueBits() int           { return mes.numValueBits }
func (mes *markerEncodingScheme) EndOfStream() Marker         { return mes.endOfStream }
func (mes *markerEncodingScheme) Annotation() Marker          { return mes.annotation }
func (mes *markerEncodingScheme) TimeUnit() Marker            { return mes.timeUnit }
func (mes *markerEncodingScheme) Tail(b byte, pos int) []byte { return mes.tails[int(b)][pos-1] }
