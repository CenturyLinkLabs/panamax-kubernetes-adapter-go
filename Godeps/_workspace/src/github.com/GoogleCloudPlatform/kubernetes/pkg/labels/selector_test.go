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

package labels

import (
	"reflect"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
)

func TestSelectorParse(t *testing.T) {
	testGoodStrings := []string{
		"x=a,y=b,z=c",
		"",
		"x!=a,y=b",
	}
	testBadStrings := []string{
		"x=a||y=b",
		"x==a==b",
	}
	for _, test := range testGoodStrings {
		lq, err := ParseSelector(test)
		if err != nil {
			t.Errorf("%v: error %v (%#v)\n", test, err, err)
		}
		if test != lq.String() {
			t.Errorf("%v restring gave: %v\n", test, lq.String())
		}
		lq, err = Parse(test)
		if err != nil {
			t.Errorf("%v: error %v (%#v)\n", test, err, err)
		}
		if test != lq.String() {
			t.Errorf("%v restring gave: %v\n", test, lq.String())
		}
	}
	for _, test := range testBadStrings {
		_, err := ParseSelector(test)
		if err == nil {
			t.Errorf("%v: did not get expected error\n", test)
		}
		_, err = Parse(test)
		if err == nil {
			t.Errorf("%v: did not get expected error\n", test)
		}
	}
}

func TestDeterministicParse(t *testing.T) {
	s1, err := ParseSelector("x=a,a=x")
	s2, err2 := ParseSelector("a=x,x=a")
	if err != nil || err2 != nil {
		t.Errorf("Unexpected parse error")
	}
	if s1.String() != s2.String() {
		t.Errorf("Non-deterministic parse")
	}
	s1, err = Parse("x=a,a=x")
	s2, err2 = Parse("a=x,x=a")
	if err != nil || err2 != nil {
		t.Errorf("Unexpected parse error")
	}
	if s1.String() != s2.String() {
		t.Errorf("Non-deterministic parse")
	}
}

func expectMatch(t *testing.T, selector string, ls Set) {
	lq, err := ParseSelector(selector)
	if err != nil {
		t.Errorf("Unable to parse %v as a selector\n", selector)
		return
	}
	if !lq.Matches(ls) {
		t.Errorf("Wanted %s to match '%s', but it did not.\n", selector, ls)
	}
	lq, err = Parse(selector)
	if err != nil {
		t.Errorf("Unable to parse %v as a selector\n", selector)
		return
	}
	if !lq.Matches(ls) {
		t.Errorf("Wanted %s to match '%s', but it did not.\n", selector, ls)
	}
}

func expectNoMatch(t *testing.T, selector string, ls Set) {
	lq, err := ParseSelector(selector)
	if err != nil {
		t.Errorf("Unable to parse %v as a selector\n", selector)
		return
	}
	if lq.Matches(ls) {
		t.Errorf("Wanted '%s' to not match '%s', but it did.", selector, ls)
	}
	lq, err = Parse(selector)
	if err != nil {
		t.Errorf("Unable to parse %v as a selector\n", selector)
		return
	}
	if lq.Matches(ls) {
		t.Errorf("Wanted '%s' to not match '%s', but it did.", selector, ls)
	}
}

func TestEverything(t *testing.T) {
	if !Everything().Matches(Set{"x": "y"}) {
		t.Errorf("Nil selector didn't match")
	}
	if !Everything().Empty() {
		t.Errorf("Everything was not empty")
	}
}

func TestSelectorMatches(t *testing.T) {
	expectMatch(t, "", Set{"x": "y"})
	expectMatch(t, "x=y", Set{"x": "y"})
	expectMatch(t, "x=y,z=w", Set{"x": "y", "z": "w"})
	expectMatch(t, "x!=y,z!=w", Set{"x": "z", "z": "a"})
	expectMatch(t, "notin=in", Set{"notin": "in"}) // in and notin in exactMatch
	expectNoMatch(t, "x=y", Set{"x": "z"})
	expectNoMatch(t, "x=y,z=w", Set{"x": "w", "z": "w"})
	expectNoMatch(t, "x!=y,z!=w", Set{"x": "z", "z": "w"})

	labelset := Set{
		"foo": "bar",
		"baz": "blah",
	}
	expectMatch(t, "foo=bar", labelset)
	expectMatch(t, "baz=blah", labelset)
	expectMatch(t, "foo=bar,baz=blah", labelset)
	expectNoMatch(t, "foo=blah", labelset)
	expectNoMatch(t, "baz=bar", labelset)
	expectNoMatch(t, "foo=bar,foobar=bar,baz=blah", labelset)
}

func TestOneTermEqualSelector(t *testing.T) {
	if !OneTermEqualSelector("x", "y").Matches(Set{"x": "y"}) {
		t.Errorf("No match when match expected.")
	}
	if OneTermEqualSelector("x", "y").Matches(Set{"x": "z"}) {
		t.Errorf("Match when none expected.")
	}
}

func TestOneTermEqualSelectorParse(t *testing.T) {
	if !OneTermEqualSelectorParse("x", "y").Matches(Set{"x": "y"}) {
		t.Errorf("No match when match expected.")
	}
	if OneTermEqualSelectorParse("x", "y").Matches(Set{"x": "z"}) {
		t.Errorf("Match when none expected.")
	}
}

func expectMatchDirect(t *testing.T, selector, ls Set) {
	if !SelectorFromSet(selector).Matches(ls) {
		t.Errorf("Wanted %s to match '%s', but it did not.\n", selector, ls)
	}
	s, e := SelectorFromSetParse(selector)
	if e == nil && !s.Matches(ls) {
		t.Errorf("Wanted '%s' to match '%s', but it did not.\n", selector, ls)
	}
}

func expectNoMatchDirect(t *testing.T, selector, ls Set) {
	if SelectorFromSet(selector).Matches(ls) {
		t.Errorf("Wanted '%s' to not match '%s', but it did.", selector, ls)
	}
	s, e := SelectorFromSetParse(selector)
	if e == nil && s.Matches(ls) {
		t.Errorf("Wanted '%s' to not match '%s', but it did.", selector, ls)
	}
}

func TestSetMatches(t *testing.T) {
	labelset := Set{
		"foo": "bar",
		"baz": "blah",
	}
	expectMatchDirect(t, Set{}, labelset)
	expectMatchDirect(t, Set{"foo": "bar"}, labelset)
	expectMatchDirect(t, Set{"baz": "blah"}, labelset)
	expectMatchDirect(t, Set{"foo": "bar", "baz": "blah"}, labelset)
	expectNoMatchDirect(t, Set{"foo": "=blah"}, labelset)
	expectNoMatchDirect(t, Set{"baz": "=bar"}, labelset)
	expectNoMatchDirect(t, Set{"foo": "=bar", "foobar": "bar", "baz": "blah"}, labelset)
}

func TestNilMapIsValid(t *testing.T) {
	selector := Set(nil).AsSelector()
	if selector == nil {
		t.Errorf("Selector for nil set should be Everything")
	}
	if !selector.Empty() {
		t.Errorf("Selector for nil set should be Empty")
	}
}

func TestSetIsEmpty(t *testing.T) {
	if !(Set{}).AsSelector().Empty() {
		t.Errorf("Empty set should be empty")
	}
	if !(andTerm(nil)).Empty() {
		t.Errorf("Nil andTerm should be empty")
	}
	if (&hasTerm{}).Empty() {
		t.Errorf("hasTerm should not be empty")
	}
	if (&notHasTerm{}).Empty() {
		t.Errorf("notHasTerm should not be empty")
	}
	if !(andTerm{andTerm{}}).Empty() {
		t.Errorf("Nested andTerm should be empty")
	}
	if (andTerm{&hasTerm{"a", "b"}}).Empty() {
		t.Errorf("Nested andTerm should not be empty")
	}
}

func TestRequiresExactMatch(t *testing.T) {
	testCases := map[string]struct {
		S     Selector
		Label string
		Value string
		Found bool
	}{
		"empty set":                 {Set{}.AsSelector(), "test", "", false},
		"nil andTerm":               {andTerm(nil), "test", "", false},
		"empty hasTerm":             {&hasTerm{}, "test", "", false},
		"skipped hasTerm":           {&hasTerm{"a", "b"}, "test", "", false},
		"valid hasTerm":             {&hasTerm{"test", "b"}, "test", "b", true},
		"valid hasTerm no value":    {&hasTerm{"test", ""}, "test", "", true},
		"valid notHasTerm":          {&notHasTerm{"test", "b"}, "test", "", false},
		"valid notHasTerm no value": {&notHasTerm{"test", ""}, "test", "", false},
		"nested andTerm":            {andTerm{andTerm{}}, "test", "", false},
		"nested andTerm matches":    {andTerm{&hasTerm{"test", "b"}}, "test", "b", true},
		"andTerm with non-match":    {andTerm{&hasTerm{}, &hasTerm{"test", "b"}}, "test", "b", true},
	}
	for k, v := range testCases {
		value, found := v.S.RequiresExactMatch(v.Label)
		if value != v.Value {
			t.Errorf("%s: expected value %s, got %s", k, v.Value, value)
		}
		if found != v.Found {
			t.Errorf("%s: expected found %t, got %t", k, v.Found, found)
		}
	}
}

func TestRequiresExactMatchParse(t *testing.T) {
	testCases := map[string]struct {
		S     Selector
		Label string
		Value string
		Found bool
	}{
		"empty set":     {Set{}.AsSelector(), "test", "", false},
		"empty hasTerm": {&LabelSelector{}, "test", "", false},
		"skipped Requirement": {&LabelSelector{Requirements: []Requirement{
			getRequirement("a", InOperator, util.NewStringSet("b"), t)}}, "test", "", false},
		"valid Requirement": {&LabelSelector{Requirements: []Requirement{
			getRequirement("test", InOperator, util.NewStringSet("b"), t)}}, "test", "b", true},
		"valid Requirement no value": {&LabelSelector{Requirements: []Requirement{
			getRequirement("test", InOperator, util.NewStringSet(""), t)}}, "test", "", true},
		"valid Requirement NotIn": {&LabelSelector{Requirements: []Requirement{
			getRequirement("test", NotInOperator, util.NewStringSet("b"), t)}}, "test", "", false},
		"valid notHasTerm no value": {&LabelSelector{Requirements: []Requirement{
			getRequirement("test", NotInOperator, util.NewStringSet(""), t)}}, "test", "", false},
		"2 Requirements with non-match": {&LabelSelector{Requirements: []Requirement{
			getRequirement("test", ExistsOperator, util.NewStringSet("b"), t),
			getRequirement("test", InOperator, util.NewStringSet("b"), t)}}, "test", "b", true},
	}
	for k, v := range testCases {
		value, found := v.S.RequiresExactMatch(v.Label)
		if value != v.Value {
			t.Errorf("%s: expected value %s, got %s", k, v.Value, value)
		}
		if found != v.Found {
			t.Errorf("%s: expected found %t, got %t", k, v.Found, found)
		}
	}
}

func TestLexer(t *testing.T) {
	testcases := []struct {
		s string
		t Token
	}{
		{"", EOS},
		{",", COMMA},
		{"notin", NOTIN},
		{"in", IN},
		{"=", EQUAL},
		{"==", EEQUAL},
		{"!=", NEQUAL},
		{"(", OPAR},
		{")", CPAR},
		{"||", IDENTIFIER},
		{"!", ERROR},
	}
	for _, v := range testcases {
		l := &Lexer{s: v.s, pos: 0}
		token, lit := l.Lex()
		if token != v.t {
			t.Errorf("Got %d it should be %d for '%s'", token, v.t, v.s)
		}
		if v.t != ERROR && lit != v.s {
			t.Errorf("Got '%s' it should be '%s'", lit, v.s)
		}
	}
}

func min(l, r int) (m int) {
	m = r
	if l < r {
		m = l
	}
	return m
}

func TestLexerSequence(t *testing.T) {
	testcases := []struct {
		s string
		t []Token
	}{
		{"key in ( value )", []Token{IDENTIFIER, IN, OPAR, IDENTIFIER, CPAR}},
		{"key notin ( value )", []Token{IDENTIFIER, NOTIN, OPAR, IDENTIFIER, CPAR}},
		{"key in ( value1, value2 )", []Token{IDENTIFIER, IN, OPAR, IDENTIFIER, COMMA, IDENTIFIER, CPAR}},
		{"key", []Token{IDENTIFIER}},
		{"()", []Token{OPAR, CPAR}},
		{"x in (),y", []Token{IDENTIFIER, IN, OPAR, CPAR, COMMA, IDENTIFIER}},
		{"== != (), = notin", []Token{EEQUAL, NEQUAL, OPAR, CPAR, COMMA, EQUAL, NOTIN}},
	}
	for _, v := range testcases {
		var literals []string
		var tokens []Token
		l := &Lexer{s: v.s, pos: 0}
		for {
			token, lit := l.Lex()
			if token == EOS {
				break
			}
			tokens = append(tokens, token)
			literals = append(literals, lit)
		}
		if len(tokens) != len(v.t) {
			t.Errorf("Bad number of tokens for '%s %d, %d", v.s, len(tokens), len(v.t))
		}
		for i := 0; i < min(len(tokens), len(v.t)); i++ {
			if tokens[i] != v.t[i] {
				t.Errorf("Test '%s': Mismatching in token type found '%s' it should be '%s'", v.s, tokens[i], v.t[i])
			}
		}
	}
}
func TestParserLookahead(t *testing.T) {
	testcases := []struct {
		s string
		t []Token
	}{
		{"key in ( value )", []Token{IDENTIFIER, IN, OPAR, IDENTIFIER, CPAR, EOS}},
		{"key notin ( value )", []Token{IDENTIFIER, NOTIN, OPAR, IDENTIFIER, CPAR, EOS}},
		{"key in ( value1, value2 )", []Token{IDENTIFIER, IN, OPAR, IDENTIFIER, COMMA, IDENTIFIER, CPAR, EOS}},
		{"key", []Token{IDENTIFIER, EOS}},
		{"()", []Token{OPAR, CPAR, EOS}},
		{"", []Token{EOS}},
		{"x in (),y", []Token{IDENTIFIER, IN, OPAR, CPAR, COMMA, IDENTIFIER, EOS}},
		{"== != (), = notin", []Token{EEQUAL, NEQUAL, OPAR, CPAR, COMMA, EQUAL, NOTIN, EOS}},
	}
	for _, v := range testcases {
		p := &Parser{l: &Lexer{s: v.s, pos: 0}, position: 0}
		p.scan()
		if len(p.scannedItems) != len(v.t) {
			t.Errorf("Expected %d items found %d", len(v.t), len(p.scannedItems))
		}
		for {
			token, lit := p.lookahead(KeyAndOperator)

			token2, lit2 := p.consume(KeyAndOperator)
			if token == EOS {
				break
			}
			if token != token2 || lit != lit2 {
				t.Errorf("Bad values")
			}
		}
	}
}

func TestRequirementConstructor(t *testing.T) {
	requirementConstructorTests := []struct {
		Key     string
		Op      Operator
		Vals    util.StringSet
		Success bool
	}{
		{"x", InOperator, nil, false},
		{"x", NotInOperator, util.NewStringSet(), false},
		{"x", InOperator, util.NewStringSet("foo"), true},
		{"x", NotInOperator, util.NewStringSet("foo"), true},
		{"x", ExistsOperator, nil, true},
		{"1foo", InOperator, util.NewStringSet("bar"), true},
		{"1234", InOperator, util.NewStringSet("bar"), true},
		{strings.Repeat("a", 64), ExistsOperator, nil, false}, //breaks DNS rule that len(key) <= 63
	}
	for _, rc := range requirementConstructorTests {
		if _, err := NewRequirement(rc.Key, rc.Op, rc.Vals); err == nil && !rc.Success {
			t.Errorf("expected error with key:%#v op:%v vals:%v, got no error", rc.Key, rc.Op, rc.Vals)
		} else if err != nil && rc.Success {
			t.Errorf("expected no error with key:%#v op:%v vals:%v, got:%v", rc.Key, rc.Op, rc.Vals, err)
		}
	}
}

func TestToString(t *testing.T) {
	var req Requirement
	toStringTests := []struct {
		In    *LabelSelector
		Out   string
		Valid bool
	}{
		{&LabelSelector{Requirements: []Requirement{
			getRequirement("x", InOperator, util.NewStringSet("abc", "def"), t),
			getRequirement("y", NotInOperator, util.NewStringSet("jkl"), t),
			getRequirement("z", ExistsOperator, nil, t),
		}}, "x in (abc,def),y notin (jkl),z", true},
		{&LabelSelector{Requirements: []Requirement{
			getRequirement("x", InOperator, util.NewStringSet("abc", "def"), t),
			req,
		}}, "x in (abc,def),", false},
		{&LabelSelector{Requirements: []Requirement{
			getRequirement("x", NotInOperator, util.NewStringSet("abc"), t),
			getRequirement("y", InOperator, util.NewStringSet("jkl", "mno"), t),
			getRequirement("z", NotInOperator, util.NewStringSet(""), t),
		}}, "x notin (abc),y in (jkl,mno),z notin ()", true},
		{&LabelSelector{Requirements: []Requirement{
			getRequirement("x", EqualsOperator, util.NewStringSet("abc"), t),
			getRequirement("y", DoubleEqualsOperator, util.NewStringSet("jkl"), t),
			getRequirement("z", NotEqualsOperator, util.NewStringSet("a"), t),
		}}, "x=abc,y==jkl,z!=a", true},
	}
	for _, ts := range toStringTests {
		if out := ts.In.String(); out == "" && ts.Valid {
			t.Errorf("%+v.String() => '%v' expected no error", ts.In)
		} else if out != ts.Out {
			t.Errorf("%+v.String() => '%v' want '%v'", ts.In, out, ts.Out)
		}
	}
}

func TestRequirementLabelSelectorMatching(t *testing.T) {
	var req Requirement
	labelSelectorMatchingTests := []struct {
		Set   Set
		Sel   *LabelSelector
		Match bool
	}{
		{Set{"x": "foo", "y": "baz"}, &LabelSelector{Requirements: []Requirement{
			req,
		}}, false},
		{Set{"x": "foo", "y": "baz"}, &LabelSelector{Requirements: []Requirement{
			getRequirement("x", InOperator, util.NewStringSet("foo"), t),
			getRequirement("y", NotInOperator, util.NewStringSet("alpha"), t),
		}}, true},
		{Set{"x": "foo", "y": "baz"}, &LabelSelector{Requirements: []Requirement{
			getRequirement("x", InOperator, util.NewStringSet("foo"), t),
			getRequirement("y", InOperator, util.NewStringSet("alpha"), t),
		}}, false},
		{Set{"y": ""}, &LabelSelector{Requirements: []Requirement{
			getRequirement("x", NotInOperator, util.NewStringSet(""), t),
			getRequirement("y", ExistsOperator, nil, t),
		}}, true},
		{Set{"y": "baz"}, &LabelSelector{Requirements: []Requirement{
			getRequirement("x", InOperator, util.NewStringSet(""), t),
		}}, false},
	}
	for _, lsm := range labelSelectorMatchingTests {
		if match := lsm.Sel.Matches(lsm.Set); match != lsm.Match {
			t.Errorf("%+v.Matches(%#v) => %v, want %v", lsm.Sel, lsm.Set, match, lsm.Match)
		}
	}
}

func TestSetSelectorParser(t *testing.T) {
	setSelectorParserTests := []struct {
		In    string
		Out   Selector
		Match bool
		Valid bool
	}{
		{"", &LabelSelector{Requirements: nil}, true, true},
		{"\rx", &LabelSelector{Requirements: []Requirement{
			getRequirement("x", ExistsOperator, nil, t),
		}}, true, true},
		{"this-is-a-dns.domain.com/key-with-dash", &LabelSelector{Requirements: []Requirement{
			getRequirement("this-is-a-dns.domain.com/key-with-dash", ExistsOperator, nil, t),
		}}, true, true},
		{"this-is-another-dns.domain.com/key-with-dash in (so,what)", &LabelSelector{Requirements: []Requirement{
			getRequirement("this-is-another-dns.domain.com/key-with-dash", InOperator, util.NewStringSet("so", "what"), t),
		}}, true, true},
		{"0.1.2.domain/99 notin (10.10.100.1, tick.tack.clock)", &LabelSelector{Requirements: []Requirement{
			getRequirement("0.1.2.domain/99", NotInOperator, util.NewStringSet("10.10.100.1", "tick.tack.clock"), t),
		}}, true, true},
		{"foo  in	 (abc)", &LabelSelector{Requirements: []Requirement{
			getRequirement("foo", InOperator, util.NewStringSet("abc"), t),
		}}, true, true},
		{"x notin\n (abc)", &LabelSelector{Requirements: []Requirement{
			getRequirement("x", NotInOperator, util.NewStringSet("abc"), t),
		}}, true, true},
		{"x  notin	\t	(abc,def)", &LabelSelector{Requirements: []Requirement{
			getRequirement("x", NotInOperator, util.NewStringSet("abc", "def"), t),
		}}, true, true},
		{"x in (abc,def)", &LabelSelector{Requirements: []Requirement{
			getRequirement("x", InOperator, util.NewStringSet("abc", "def"), t),
		}}, true, true},
		{"x in (abc,)", &LabelSelector{Requirements: []Requirement{
			getRequirement("x", InOperator, util.NewStringSet("abc", ""), t),
		}}, true, true},
		{"x in ()", &LabelSelector{Requirements: []Requirement{
			getRequirement("x", InOperator, util.NewStringSet(""), t),
		}}, true, true},
		{"x notin (abc,,def),bar,z in (),w", &LabelSelector{Requirements: []Requirement{
			getRequirement("bar", ExistsOperator, nil, t),
			getRequirement("w", ExistsOperator, nil, t),
			getRequirement("x", NotInOperator, util.NewStringSet("abc", "", "def"), t),
			getRequirement("z", InOperator, util.NewStringSet(""), t),
		}}, true, true},
		{"x,y in (a)", &LabelSelector{Requirements: []Requirement{
			getRequirement("y", InOperator, util.NewStringSet("a"), t),
			getRequirement("x", ExistsOperator, nil, t),
		}}, false, true},
		{"x=a", &LabelSelector{Requirements: []Requirement{
			getRequirement("x", EqualsOperator, util.NewStringSet("a"), t),
		}}, true, true},
		{"x=a,y!=b", &LabelSelector{Requirements: []Requirement{
			getRequirement("x", EqualsOperator, util.NewStringSet("a"), t),
			getRequirement("y", NotEqualsOperator, util.NewStringSet("b"), t),
		}}, true, true},
		{"x=a,y!=b,z in (h,i,j)", &LabelSelector{Requirements: []Requirement{
			getRequirement("x", EqualsOperator, util.NewStringSet("a"), t),
			getRequirement("y", NotEqualsOperator, util.NewStringSet("b"), t),
			getRequirement("z", InOperator, util.NewStringSet("h", "i", "j"), t),
		}}, true, true},
		{"x=a||y=b", &LabelSelector{Requirements: []Requirement{}}, false, false},
		{"x,,y", nil, true, false},
		{",x,y", nil, true, false},
		{"x nott in (y)", nil, true, false},
		{"x notin ( )", &LabelSelector{Requirements: []Requirement{
			getRequirement("x", NotInOperator, util.NewStringSet(""), t),
		}}, true, true},
		{"x notin (, a)", &LabelSelector{Requirements: []Requirement{

			getRequirement("x", NotInOperator, util.NewStringSet("", "a"), t),
		}}, true, true},
		{"a in (xyz),", nil, true, false},
		{"a in (xyz)b notin ()", nil, true, false},
		{"a ", &LabelSelector{Requirements: []Requirement{
			getRequirement("a", ExistsOperator, nil, t),
		}}, true, true},
		{"a in (x,y,notin, z,in)", &LabelSelector{Requirements: []Requirement{
			getRequirement("a", InOperator, util.NewStringSet("in", "notin", "x", "y", "z"), t),
		}}, true, true}, // operator 'in' inside list of identifiers
		{"a in (xyz abc)", nil, false, false}, // no comma
		{"a notin(", nil, true, false},        // bad formed
		{"a (", nil, false, false},            // cpar
		{"(", nil, false, false},              // opar
	}

	for _, ssp := range setSelectorParserTests {
		if sel, err := Parse(ssp.In); err != nil && ssp.Valid {
			t.Errorf("Parse(%s) => %v expected no error", ssp.In, err)
		} else if err == nil && !ssp.Valid {
			t.Errorf("Parse(%s) => %+v expected error", ssp.In, sel)
		} else if ssp.Match && !reflect.DeepEqual(sel, ssp.Out) {
			t.Errorf("Parse(%s) => parse output %+v doesn't match %+v, expected match", ssp.In, sel, ssp.Out)
		}
	}
}

func getRequirement(key string, op Operator, vals util.StringSet, t *testing.T) Requirement {
	req, err := NewRequirement(key, op, vals)
	if err != nil {
		t.Errorf("NewRequirement(%v, %v, %v) resulted in error:%v", key, op, vals, err)
	}
	return *req
}
