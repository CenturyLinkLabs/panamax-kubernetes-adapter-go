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

package runtime_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/runtime"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
)

type EmbeddedTest struct {
	runtime.TypeMeta
	ID          string
	Object      runtime.EmbeddedObject
	EmptyObject runtime.EmbeddedObject
}

type EmbeddedTestExternal struct {
	runtime.TypeMeta `json:",inline"`
	ID               string               `json:"id,omitempty"`
	Object           runtime.RawExtension `json:"object,omitempty"`
	EmptyObject      runtime.RawExtension `json:"emptyObject,omitempty"`
}

type ObjectTest struct {
	runtime.TypeMeta

	ID    string
	Items []runtime.Object
}

type ObjectTestExternal struct {
	runtime.TypeMeta `yaml:",inline" json:",inline"`

	ID    string                 `json:"id,omitempty"`
	Items []runtime.RawExtension `json:"items,omitempty"`
}

func (*ObjectTest) IsAnAPIObject()           {}
func (*ObjectTestExternal) IsAnAPIObject()   {}
func (*EmbeddedTest) IsAnAPIObject()         {}
func (*EmbeddedTestExternal) IsAnAPIObject() {}

func TestDecodeEmptyRawExtensionAsObject(t *testing.T) {
	s := runtime.NewScheme()
	s.AddKnownTypes("", &ObjectTest{})
	s.AddKnownTypeWithName("v1test", "ObjectTest", &ObjectTestExternal{})

	_, err := s.Decode([]byte(`{"kind":"ObjectTest","apiVersion":"v1test","items":[{}]}`))
	if err == nil {
		t.Fatalf("unexpected non-error")
	}
}

func TestArrayOfRuntimeObject(t *testing.T) {
	s := runtime.NewScheme()
	s.AddKnownTypes("", &EmbeddedTest{})
	s.AddKnownTypeWithName("v1test", "EmbeddedTest", &EmbeddedTestExternal{})
	s.AddKnownTypes("", &ObjectTest{})
	s.AddKnownTypeWithName("v1test", "ObjectTest", &ObjectTestExternal{})

	internal := &ObjectTest{
		Items: []runtime.Object{
			&EmbeddedTest{ID: "foo"},
			&EmbeddedTest{ID: "bar"},
			// TODO: until YAML is removed, this JSON must be in ascending key order to ensure consistent roundtrip serialization
			&runtime.Unknown{RawJSON: []byte(`{"apiVersion":"unknown","foo":"bar","kind":"OtherTest"}`)},
			&ObjectTest{
				Items: []runtime.Object{
					&EmbeddedTest{ID: "baz"},
				},
			},
		},
	}
	wire, err := s.EncodeToVersion(internal, "v1test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Logf("Wire format is:\n%s\n", string(wire))

	obj := &ObjectTestExternal{}
	if err := json.Unmarshal(wire, obj); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Logf("exact wire is: %#v", string(obj.Items[0].RawJSON))

	decoded, err := s.Decode(wire)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	internal.Items[2].(*runtime.Unknown).Kind = "OtherTest"
	internal.Items[2].(*runtime.Unknown).APIVersion = "unknown"
	if e, a := internal, decoded; !reflect.DeepEqual(e, a) {
		t.Log(string(decoded.(*ObjectTest).Items[2].(*runtime.Unknown).RawJSON))
		t.Errorf("mismatched decoded: %s", util.ObjectDiff(e, a))
	}
}

func TestEmbeddedObject(t *testing.T) {
	s := runtime.NewScheme()
	s.AddKnownTypes("", &EmbeddedTest{})
	s.AddKnownTypeWithName("v1test", "EmbeddedTest", &EmbeddedTestExternal{})

	outer := &EmbeddedTest{
		ID: "outer",
		Object: runtime.EmbeddedObject{
			&EmbeddedTest{
				ID: "inner",
			},
		},
	}

	wire, err := s.EncodeToVersion(outer, "v1test")
	if err != nil {
		t.Fatalf("Unexpected encode error '%v'", err)
	}

	t.Logf("Wire format is:\n%v\n", string(wire))

	decoded, err := s.Decode(wire)
	if err != nil {
		t.Fatalf("Unexpected decode error %v", err)
	}

	if e, a := outer, decoded; !reflect.DeepEqual(e, a) {
		t.Errorf("Expected: %#v but got %#v", e, a)
	}

	// test JSON decoding of the external object, which should preserve
	// raw bytes
	var externalViaJSON EmbeddedTestExternal
	err = json.Unmarshal(wire, &externalViaJSON)
	if err != nil {
		t.Fatalf("Unexpected decode error %v", err)
	}
	if externalViaJSON.Kind == "" || externalViaJSON.APIVersion == "" || externalViaJSON.ID != "outer" {
		t.Errorf("Expected objects to have type info set, got %#v", externalViaJSON)
	}
	if !reflect.DeepEqual(externalViaJSON.EmptyObject.RawJSON, []byte("null")) || len(externalViaJSON.Object.RawJSON) == 0 {
		t.Errorf("Expected deserialization of nested objects into bytes, got %#v", externalViaJSON)
	}

	// test JSON decoding, too, since Decode uses yaml unmarshalling.
	// Generic Unmarshalling of JSON cannot load the nested objects because there is
	// no default schema set.  Consumers wishing to get direct JSON decoding must use
	// the external representation
	var decodedViaJSON EmbeddedTest
	err = json.Unmarshal(wire, &decodedViaJSON)
	if err != nil {
		t.Fatalf("Unexpected decode error %v", err)
	}
	if a := decodedViaJSON; a.Object.Object != nil || a.EmptyObject.Object != nil {
		t.Errorf("Expected embedded objects to be nil: %#v", a)
	}
}
