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
	"fmt"
	"math"
)

type Scale int32

// Idx gives the bucket index v belongs into
func (scale Scale) Idx(v float64) int {
	// from: https://opentelemetry.io/docs/specs/otel/metrics/data-model/#all-scales-use-the-logarithm-function

	// Special case for power-of-two values.
	if frac, exp := math.Frexp(v); frac == 0.5 {
		return ((exp - 1) << scale) - 1
	}

	scaleFactor := math.Ldexp(math.Log2E, int(scale))
	// Note: math.Floor(value) equals math.Ceil(value)-1 when value
	// is not a power of two, which is checked above.
	return int(math.Floor(math.Log(v) * scaleFactor))
}

// Bounds returns the half-open interval (min,max] of the bucket at index.
// This means a value min < v <= max belongs to this bucket.
//
// NOTE: this is different from Go slice intervals, which are [a,b)
func (scale Scale) Bounds(index int) (min, max float64) {
	// from: https://opentelemetry.io/docs/specs/otel/metrics/data-model/#all-scales-use-the-logarithm-function
	lower := func(index int) float64 {
		inverseFactor := math.Ldexp(math.Ln2, int(-scale))
		return math.Exp(float64(index) * inverseFactor)
	}

	return lower(index), lower(index + 1)
}

// Downscale collapses the buckets of bs until scale 'to' is reached
func Downscale(bs Buckets, from, to Scale) {
	switch {
	case from == to:
		return
	case from < to:
		// because even distribution within the buckets cannot be assumed, it is
		// not possible to correctly upscale (split) buckets.
		// any attempt to do so would yield erronous data.
		panic(fmt.Sprintf("cannot upscale without introducing error (%d -> %d)", from, to))
	}

	collapseN := int(from-to) * 2
	offset := int(bs.Offset()) % collapseN
	if offset < 0 {
		offset = -offset
	}
	bs.BucketCounts().Collapse(collapseN, offset)
	bs.SetOffset((bs.Offset() - int32(offset)) / 2)
}
