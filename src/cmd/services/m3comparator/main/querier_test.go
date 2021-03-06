// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"testing"
	"time"

	"github.com/m3db/m3/src/dbnode/encoding"
	"github.com/m3db/m3/src/query/models"
	"github.com/m3db/m3/src/query/storage"
	xtest "github.com/m3db/m3/src/x/test"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testSeriesLoadHandler struct {
	iters encoding.SeriesIterators
}

func (h *testSeriesLoadHandler) getSeriesIterators(name string) (encoding.SeriesIterators, error) {
	return h.iters, nil
}

var _ seriesLoadHandler = (*testSeriesLoadHandler)(nil)

type tagMap map[string]string

var (
	iteratorOpts = iteratorOptions{
		encoderPool:   encoderPool,
		iteratorPools: iterPools,
		tagOptions:    tagOptions,
		iOpts:         iOpts,
	}
	metricNameTag = string(iteratorOpts.tagOptions.MetricName())
)

const (
	blockSize         = time.Hour * 12
	defaultResolution = time.Second * 30
)

func TestFetchCompressedReturnsPreloadedData(t *testing.T) {
	ctrl := xtest.NewController(t)
	defer ctrl.Finish()

	predefinedSeriesCount := 100
	iters := encoding.NewMockSeriesIterators(ctrl)
	iters.EXPECT().Len().Return(predefinedSeriesCount).MinTimes(1)
	iters.EXPECT().Close()

	seriesLoader := &testSeriesLoadHandler{iters}

	querier, _ := newQuerier(iteratorOpts, seriesLoader, blockSize, defaultResolution)

	query := matcherQuery(t, metricNameTag, "preloaded")

	result, cleanup, err := querier.FetchCompressed(nil, query, nil)
	assert.NoError(t, err)
	defer cleanup()

	assert.Equal(t, predefinedSeriesCount, result.SeriesIterators.Len())
}

func TestFetchCompressedGeneratesRandomData(t *testing.T) {
	tests := []struct {
		name       string
		givenQuery *storage.FetchQuery
		wantSeries []tagMap
	}{
		{
			name:       "random data for known metrics",
			givenQuery: matcherQuery(t, metricNameTag, "quail"),
			wantSeries: []tagMap{
				{
					metricNameTag: "quail",
					"foobar":      "qux",
					"name":        "quail",
				},
			},
		},
		{
			name:       "a hardcoded list of metrics",
			givenQuery: matcherQuery(t, metricNameTag, "unknown"),
			wantSeries: []tagMap{
				{
					metricNameTag: "foo",
					"foobar":      "qux",
					"name":        "foo",
				},
				{
					metricNameTag: "bar",
					"foobar":      "qux",
					"name":        "bar",
				},
				{
					metricNameTag: "quail",
					"foobar":      "qux",
					"name":        "quail",
				},
			},
		},
		{
			name:       "a given number of single series metrics",
			givenQuery: matcherQuery(t, "gen", "2"),
			wantSeries: []tagMap{
				{
					metricNameTag: "foo_0",
					"foobar":      "qux",
					"name":        "foo_0",
				},
				{
					metricNameTag: "foo_1",
					"foobar":      "qux",
					"name":        "foo_1",
				},
			},
		},
		{
			name:       "single metrics with a given number of series",
			givenQuery: matcherQuery(t, metricNameTag, "multi_4"),
			wantSeries: []tagMap{
				{
					metricNameTag: "multi_4",
					"const":       "x",
					"id":          "0",
					"parity":      "0",
				},
				{
					metricNameTag: "multi_4",
					"const":       "x",
					"id":          "1",
					"parity":      "1",
				},
				{
					metricNameTag: "multi_4",
					"const":       "x",
					"id":          "2",
					"parity":      "0",
				},
				{
					metricNameTag: "multi_4",
					"const":       "x",
					"id":          "3",
					"parity":      "1",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := xtest.NewController(t)
			defer ctrl.Finish()

			querier, _ := newQuerier(iteratorOpts, emptySeriesLoader(ctrl), blockSize, defaultResolution)

			result, cleanup, err := querier.FetchCompressed(nil, tt.givenQuery, nil)
			assert.NoError(t, err)
			defer cleanup()

			assert.Equal(t, len(tt.wantSeries), result.SeriesIterators.Len())
			for i, expectedTags := range tt.wantSeries {
				iter := result.SeriesIterators.Iters()[i]
				assert.Equal(t, expectedTags, extractTags(iter))
				assert.True(t, iter.Next(), "Must have some datapoints generated.")
			}
		})
	}
}

func matcherQuery(t *testing.T, matcherName, matcherValue string) *storage.FetchQuery {
	matcher, err := models.NewMatcher(models.MatchEqual, []byte(matcherName), []byte(matcherValue))
	assert.NoError(t, err)

	now := time.Now()

	return &storage.FetchQuery{
		TagMatchers: []models.Matcher{matcher},
		Start:       now.Add(-time.Hour),
		End:         now,
	}
}

func emptySeriesLoader(ctrl *gomock.Controller) seriesLoadHandler {
	iters := encoding.NewMockSeriesIterators(ctrl)
	iters.EXPECT().Len().Return(0).AnyTimes()

	return &testSeriesLoadHandler{iters}
}

func extractTags(seriesIter encoding.SeriesIterator) tagMap {
	tagsIter := seriesIter.Tags().Duplicate()
	defer tagsIter.Close()

	tags := make(tagMap)
	for tagsIter.Next() {
		tag := tagsIter.Current()
		tags[tag.Name.String()] = tag.Value.String()
	}

	return tags
}
