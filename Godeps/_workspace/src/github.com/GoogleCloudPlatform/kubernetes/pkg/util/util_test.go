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
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
)

func TestUntil(t *testing.T) {
	ch := make(chan struct{})
	close(ch)
	Until(func() {
		t.Fatal("should not have been invoked")
	}, 0, ch)

	ch = make(chan struct{})
	called := make(chan struct{})
	go func() {
		Until(func() {
			called <- struct{}{}
		}, 0, ch)
		close(called)
	}()
	<-called
	close(ch)
	<-called
}

func TestHandleCrash(t *testing.T) {
	count := 0
	expect := 10
	for i := 0; i < expect; i = i + 1 {
		defer HandleCrash()
		if i%2 == 0 {
			panic("Test Panic")
		}
		count = count + 1
	}
	if count != expect {
		t.Errorf("Expected %d iterations, found %d", expect, count)
	}
}

func TestCustomHandleCrash(t *testing.T) {
	old := PanicHandlers
	defer func() { PanicHandlers = old }()
	var result interface{}
	PanicHandlers = []func(interface{}){
		func(r interface{}) {
			result = r
		},
	}
	func() {
		defer HandleCrash()
		panic("test")
	}()
	if result != "test" {
		t.Errorf("did not receive custom handler")
	}
}

func TestCustomHandleError(t *testing.T) {
	old := ErrorHandlers
	defer func() { ErrorHandlers = old }()
	var result error
	ErrorHandlers = []func(error){
		func(err error) {
			result = err
		},
	}
	err := fmt.Errorf("test")
	HandleError(err)
	if result != err {
		t.Errorf("did not receive custom handler")
	}
}

func TestNewIntOrStringFromInt(t *testing.T) {
	i := NewIntOrStringFromInt(93)
	if i.Kind != IntstrInt || i.IntVal != 93 {
		t.Errorf("Expected IntVal=93, got %+v", i)
	}
}

func TestNewIntOrStringFromString(t *testing.T) {
	i := NewIntOrStringFromString("76")
	if i.Kind != IntstrString || i.StrVal != "76" {
		t.Errorf("Expected StrVal=\"76\", got %+v", i)
	}
}

type IntOrStringHolder struct {
	IOrS IntOrString `json:"val"`
}

func TestIntOrStringUnmarshalYAML(t *testing.T) {
	cases := []struct {
		input  string
		result IntOrString
	}{
		{"val: 123\n", IntOrString{Kind: IntstrInt, IntVal: 123}},
		{"val: \"123\"\n", IntOrString{Kind: IntstrString, StrVal: "123"}},
	}

	for _, c := range cases {
		var result IntOrStringHolder
		if err := yaml.Unmarshal([]byte(c.input), &result); err != nil {
			t.Errorf("Failed to unmarshal input '%v': %v", c.input, err)
		}
		if result.IOrS != c.result {
			t.Errorf("Failed to unmarshal input '%v': expected: %+v, got %+v", c.input, c.result, result)
		}
	}
}

func TestIntOrStringMarshalYAML(t *testing.T) {
	cases := []struct {
		input  IntOrString
		result string
	}{
		{IntOrString{Kind: IntstrInt, IntVal: 123}, "val: 123\n"},
		{IntOrString{Kind: IntstrString, StrVal: "123"}, "val: \"123\"\n"},
	}

	for _, c := range cases {
		input := IntOrStringHolder{c.input}
		result, err := yaml.Marshal(&input)
		if err != nil {
			t.Errorf("Failed to marshal input '%v': %v", input, err)
		}
		if string(result) != c.result {
			t.Errorf("Failed to marshal input '%v': expected: %+v, got %q", input, c.result, string(result))
		}
	}
}

func TestIntOrStringUnmarshalJSON(t *testing.T) {
	cases := []struct {
		input  string
		result IntOrString
	}{
		{"{\"val\": 123}", IntOrString{Kind: IntstrInt, IntVal: 123}},
		{"{\"val\": \"123\"}", IntOrString{Kind: IntstrString, StrVal: "123"}},
	}

	for _, c := range cases {
		var result IntOrStringHolder
		if err := json.Unmarshal([]byte(c.input), &result); err != nil {
			t.Errorf("Failed to unmarshal input '%v': %v", c.input, err)
		}
		if result.IOrS != c.result {
			t.Errorf("Failed to unmarshal input '%v': expected %+v, got %+v", c.input, c.result, result)
		}
	}
}

func TestIntOrStringMarshalJSON(t *testing.T) {
	cases := []struct {
		input  IntOrString
		result string
	}{
		{IntOrString{Kind: IntstrInt, IntVal: 123}, "{\"val\":123}"},
		{IntOrString{Kind: IntstrString, StrVal: "123"}, "{\"val\":\"123\"}"},
	}

	for _, c := range cases {
		input := IntOrStringHolder{c.input}
		result, err := json.Marshal(&input)
		if err != nil {
			t.Errorf("Failed to marshal input '%v': %v", input, err)
		}
		if string(result) != c.result {
			t.Errorf("Failed to marshal input '%v': expected: %+v, got %q", input, c.result, string(result))
		}
	}
}

func TestIntOrStringMarshalJSONUnmarshalYAML(t *testing.T) {
	cases := []struct {
		input IntOrString
	}{
		{IntOrString{Kind: IntstrInt, IntVal: 123}},
		{IntOrString{Kind: IntstrString, StrVal: "123"}},
	}

	for _, c := range cases {
		input := IntOrStringHolder{c.input}
		jsonMarshalled, err := json.Marshal(&input)
		if err != nil {
			t.Errorf("1: Failed to marshal input: '%v': %v", input, err)
		}

		var result IntOrStringHolder
		err = yaml.Unmarshal(jsonMarshalled, &result)
		if err != nil {
			t.Errorf("2: Failed to unmarshal '%+v': %v", string(jsonMarshalled), err)
		}

		if !reflect.DeepEqual(input, result) {
			t.Errorf("3: Failed to marshal input '%+v': got %+v", input, result)
		}
	}
}

func TestStringDiff(t *testing.T) {
	diff := StringDiff("aaabb", "aaacc")
	expect := "aaa\n\nA: bb\n\nB: cc\n\n"
	if diff != expect {
		t.Errorf("diff returned %v", diff)
	}
}

func TestCompileRegex(t *testing.T) {
	uncompiledRegexes := []string{"endsWithMe$", "^startingWithMe"}
	regexes, err := CompileRegexps(uncompiledRegexes)

	if err != nil {
		t.Errorf("Failed to compile legal regexes: '%v': %v", uncompiledRegexes, err)
	}
	if len(regexes) != len(uncompiledRegexes) {
		t.Errorf("Wrong number of regexes returned: '%v': %v", uncompiledRegexes, regexes)
	}

	if !regexes[0].MatchString("Something that endsWithMe") {
		t.Errorf("Wrong regex returned: '%v': %v", uncompiledRegexes[0], regexes[0])
	}
	if regexes[0].MatchString("Something that doesn't endsWithMe.") {
		t.Errorf("Wrong regex returned: '%v': %v", uncompiledRegexes[0], regexes[0])
	}
	if !regexes[1].MatchString("startingWithMe is very important") {
		t.Errorf("Wrong regex returned: '%v': %v", uncompiledRegexes[1], regexes[1])
	}
	if regexes[1].MatchString("not startingWithMe should fail") {
		t.Errorf("Wrong regex returned: '%v': %v", uncompiledRegexes[1], regexes[1])
	}
}

func TestAllPtrFieldsNil(t *testing.T) {
	testCases := []struct {
		obj      interface{}
		expected bool
	}{
		{struct{}{}, true},
		{struct{ Foo int }{12345}, true},
		{&struct{ Foo int }{12345}, true},
		{struct{ Foo *int }{nil}, true},
		{&struct{ Foo *int }{nil}, true},
		{struct {
			Foo int
			Bar *int
		}{12345, nil}, true},
		{&struct {
			Foo int
			Bar *int
		}{12345, nil}, true},
		{struct {
			Foo *int
			Bar *int
		}{nil, nil}, true},
		{&struct {
			Foo *int
			Bar *int
		}{nil, nil}, true},
		{struct{ Foo *int }{new(int)}, false},
		{&struct{ Foo *int }{new(int)}, false},
		{struct {
			Foo *int
			Bar *int
		}{nil, new(int)}, false},
		{&struct {
			Foo *int
			Bar *int
		}{nil, new(int)}, false},
		{(*struct{})(nil), true},
	}
	for i, tc := range testCases {
		if AllPtrFieldsNil(tc.obj) != tc.expected {
			t.Errorf("case[%d]: expected %t, got %t", i, tc.expected, !tc.expected)
		}
	}
}

func TestSplitQualifiedName(t *testing.T) {
	testCases := []struct {
		input  string
		output []string
	}{
		{"kubernetes.io/blah", []string{"kubernetes.io", "blah"}},
		{"blah", []string{"", "blah"}},
		{"kubernetes.io/blah/blah", []string{"kubernetes.io", "blah"}},
	}
	for i, tc := range testCases {
		namespace, name := SplitQualifiedName(tc.input)
		if namespace != tc.output[0] || name != tc.output[1] {
			t.Errorf("case[%d]: expected (%q, %q), got (%q, %q)", i, tc.output[0], tc.output[1], namespace, name)
		}
	}
}

func TestJoinQualifiedName(t *testing.T) {
	testCases := []struct {
		input  []string
		output string
	}{
		{[]string{"kubernetes.io", "blah"}, "kubernetes.io/blah"},
		{[]string{"blah", ""}, "blah"},
		{[]string{"kubernetes.io", "blah"}, "kubernetes.io/blah"},
	}
	for i, tc := range testCases {
		res := JoinQualifiedName(tc.input[0], tc.input[1])
		if res != tc.output {
			t.Errorf("case[%d]: expected %q, got %q", i, tc.output, res)
		}
	}
}
