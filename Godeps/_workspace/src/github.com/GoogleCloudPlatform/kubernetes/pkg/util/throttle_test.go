/*
Copyright 2014 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"testing"
	"time"
)

func TestBasicThrottle(t *testing.T) {
	ticker := make(chan time.Time, 1)
	r := newTokenBucketRateLimiterFromTicker(ticker, 3)
	for i := 0; i < 3; i++ {
		if !r.CanAccept() {
			t.Error("unexpected false accept")
		}
	}
	if r.CanAccept() {
		t.Error("unexpected true accept")
	}
}

func TestIncrementThrottle(t *testing.T) {
	ticker := make(chan time.Time, 1)
	r := newTokenBucketRateLimiterFromTicker(ticker, 1)
	if !r.CanAccept() {
		t.Error("unexpected false accept")
	}
	if r.CanAccept() {
		t.Error("unexpected true accept")
	}
	ticker <- time.Now()
	r.step()

	if !r.CanAccept() {
		t.Error("unexpected false accept")
	}
}

func TestOverBurst(t *testing.T) {
	ticker := make(chan time.Time, 1)
	r := newTokenBucketRateLimiterFromTicker(ticker, 3)

	for i := 0; i < 4; i++ {
		ticker <- time.Now()
		r.step()
	}
}
