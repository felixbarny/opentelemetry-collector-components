// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// This is a copy of the internal module from opentelemetry-collector-contrib:
// https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/deltatocumulativeprocessor/internal/data

package expo // import "github.com/elastic/opentelemetry-collector-components/processor/lsmintervalprocessor/internal/data/expo"

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
)

const (
	// a zero offset is compatible with any other positive offsets
	//negative offsets are only relevant when the histogram tracks floats
	defaultOffset = 0
	// aligned with the max buckets configuration option in lsm interval processor
	// the merge logic is compatible, but not optimized for histograms with more buckets
	defaultCapacity = 160
)

// Merge combines the counts of buckets a and b into a.
// Both buckets MUST be of same scale.
//
// The code has been modified from the upstream code to optimize allocations
// performed while merging the histograms. This also makes the Merge operation
// mutating w.r.t. the brel bucket.
func Merge(arel, brel Buckets) {
	if brel.BucketCounts().Len() == 0 {
		return
	}
	if arel.BucketCounts().Len() == 0 {
		brel.CopyTo(arel)
		return
	}
	if arel.BucketCounts().TryIncrementFrom(brel.BucketCounts(), int(brel.Offset()-arel.Offset())) {
		// b fits into a
		return
	}
	if brel.BucketCounts().TryIncrementFrom(arel.BucketCounts(), int(arel.Offset()-brel.Offset())) {
		// a fits into b
		brel.BucketCounts().MoveTo(arel.BucketCounts())
		arel.SetOffset(brel.Offset())
		return
	}
	// creates a new bucket in a way so that consecutive merges will almost always fit into it and therefore reduces allocations
	// this relies on the fact that the 'arel' histogram bucket will be reused as the merge target
	aupper := int(arel.Offset()) + arel.BucketCounts().Len()
	bupper := int(brel.Offset()) + brel.BucketCounts().Len()
	capacity := max(aupper, bupper, defaultCapacity)
	offset := min(arel.Offset(), brel.Offset(), defaultOffset)
	counts := pcommon.NewUInt64Slice()
	counts.EnsureCapacity(capacity)
	counts.TryIncrementFrom(arel.BucketCounts(), int(arel.Offset()-offset))
	counts.TryIncrementFrom(brel.BucketCounts(), int(brel.Offset()-offset))
	counts.MoveTo(arel.BucketCounts())
	arel.SetOffset(offset)
}
