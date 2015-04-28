// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcs_test

import (
	"errors"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/mock_gcs"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
)

func TestFastStatBucket(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const ttl = time.Second

type fastStatBucketTest struct {
	cache   mock_gcs.MockStatCache
	clock   timeutil.SimulatedClock
	wrapped mock_gcs.MockBucket

	bucket gcs.Bucket
}

func (t *fastStatBucketTest) SetUp(ti *TestInfo) {
	// Set up a fixed, non-zero time.
	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))

	// Set up dependencies.
	t.cache = mock_gcs.NewMockStatCache(ti.MockController, "cache")
	t.wrapped = mock_gcs.NewMockBucket(ti.MockController, "wrapped")

	t.bucket = gcs.NewFastStatBucket(
		ttl,
		t.cache,
		&t.clock,
		t.wrapped)
}

////////////////////////////////////////////////////////////////////////
// CreateObject
////////////////////////////////////////////////////////////////////////

type CreateObjectTest struct {
	fastStatBucketTest
}

func init() { RegisterTestSuite(&CreateObjectTest{}) }

func (t *CreateObjectTest) CallsEraseAndWrapped() {
	const name = "taco"

	// Erase
	ExpectCall(t.cache, "Erase")(name)

	// Wrapped
	var wrappedReq *gcs.CreateObjectRequest
	ExpectCall(t.wrapped, "CreateObject")(Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &wrappedReq), Return(nil, errors.New(""))))

	// Call
	req := &gcs.CreateObjectRequest{
		Name: name,
	}

	_, _ = t.bucket.CreateObject(nil, req)

	AssertNe(nil, wrappedReq)
	ExpectEq(req, wrappedReq)
}

func (t *CreateObjectTest) WrappedFails() {
	const name = ""
	var err error

	// Erase
	ExpectCall(t.cache, "Erase")(Any())

	// Wrapped
	ExpectCall(t.wrapped, "CreateObject")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, err = t.bucket.CreateObject(nil, &gcs.CreateObjectRequest{})

	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *CreateObjectTest) WrappedSucceeds() {
	const name = "taco"
	var err error

	// Erase
	ExpectCall(t.cache, "Erase")(Any())

	// Wrapped
	obj := &gcs.Object{
		Name:       name,
		Generation: 1234,
	}

	ExpectCall(t.wrapped, "CreateObject")(Any(), Any()).
		WillOnce(Return(obj, nil))

	// Insert
	ExpectCall(t.cache, "Insert")(obj, t.clock.Now().Add(ttl))

	// Call
	o, err := t.bucket.CreateObject(nil, &gcs.CreateObjectRequest{})

	AssertEq(nil, err)
	ExpectEq(obj, o)
}
