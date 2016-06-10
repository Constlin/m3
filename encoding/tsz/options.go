package tsz

import (
	"code.uber.internal/infra/memtsdb"
)

var (
	// default encoding options
	defaultOptions = newOptions()
)

// Options represents different options for encoding time as well as markers.
type Options interface {
	// TimeEncodingSchemes sets the time encoding schemes for different time units.
	TimeEncodingSchemes(value TimeEncodingSchemes) Options

	// GetTimeEncodingSchemes returns the time encoding schemes for different time units.
	GetTimeEncodingSchemes() TimeEncodingSchemes

	// MarkerEncodingScheme sets the marker encoding scheme.
	MarkerEncodingScheme(value MarkerEncodingScheme) Options

	// GetMarkerEncodingScheme returns the marker encoding scheme.
	GetMarkerEncodingScheme() MarkerEncodingScheme

	// Pool sets the encoder pool.
	Pool(value memtsdb.EncoderPool) Options

	// GetPool returns the encoder pool.
	GetPool() memtsdb.EncoderPool

	// IteratorPool sets the iterator pool.
	IteratorPool(value memtsdb.IteratorPool) Options

	// GetIteratorPool returns the iterator pool.
	GetIteratorPool() memtsdb.IteratorPool

	// BytesPool sets the bytes pool.
	BytesPool(value memtsdb.BytesPool) Options

	// GetBytesPool returns the bytes pool.
	GetBytesPool() memtsdb.BytesPool

	// SegmentReaderPool sets the segment reader pool.
	SegmentReaderPool(value memtsdb.SegmentReaderPool) Options

	// GetSegmentReaderPool returns the segment reader pool.
	GetSegmentReaderPool() memtsdb.SegmentReaderPool
}

type options struct {
	timeEncodingSchemes  TimeEncodingSchemes
	markerEncodingScheme MarkerEncodingScheme
	pool                 memtsdb.EncoderPool
	iteratorPool         memtsdb.IteratorPool
	bytesPool            memtsdb.BytesPool
	segmentReaderPool    memtsdb.SegmentReaderPool
}

func newOptions() Options {
	return &options{
		timeEncodingSchemes:  defaultTimeEncodingSchemes,
		markerEncodingScheme: defaultMarkerEncodingScheme,
	}
}

// NewOptions creates a new options.
func NewOptions() Options {
	return defaultOptions
}

func (o *options) TimeEncodingSchemes(value TimeEncodingSchemes) Options {
	opts := *o
	opts.timeEncodingSchemes = value
	return &opts
}

func (o *options) GetTimeEncodingSchemes() TimeEncodingSchemes {
	return o.timeEncodingSchemes
}

func (o *options) MarkerEncodingScheme(value MarkerEncodingScheme) Options {
	opts := *o
	opts.markerEncodingScheme = value
	return &opts
}

func (o *options) GetMarkerEncodingScheme() MarkerEncodingScheme {
	return o.markerEncodingScheme
}

func (o *options) Pool(value memtsdb.EncoderPool) Options {
	opts := *o
	opts.pool = value
	return &opts
}

func (o *options) GetPool() memtsdb.EncoderPool {
	return o.pool
}

func (o *options) IteratorPool(value memtsdb.IteratorPool) Options {
	opts := *o
	opts.iteratorPool = value
	return &opts
}

func (o *options) GetIteratorPool() memtsdb.IteratorPool {
	return o.iteratorPool
}

func (o *options) BytesPool(value memtsdb.BytesPool) Options {
	opts := *o
	opts.bytesPool = value
	return &opts
}

func (o *options) GetBytesPool() memtsdb.BytesPool {
	return o.bytesPool
}

func (o *options) SegmentReaderPool(value memtsdb.SegmentReaderPool) Options {
	opts := *o
	opts.segmentReaderPool = value
	return &opts
}

func (o *options) GetSegmentReaderPool() memtsdb.SegmentReaderPool {
	return o.segmentReaderPool
}
