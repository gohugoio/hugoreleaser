// Copyright 2022 The Hugoreleaser Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package matchers

type Matcher interface {
	Match(string) bool
}

func And(matchers ...Matcher) Matcher {
	return and(matchers)
}

type and []Matcher

func (m and) Match(s string) bool {
	for _, matcher := range m {
		if !matcher.Match(s) {
			return false
		}
	}
	return true
}

type or []Matcher

// Or returns a matcher that matches if any of the given matchers match.
func Or(matchers ...Matcher) Matcher {
	return or(matchers)
}

func (m or) Match(s string) bool {
	for _, matcher := range m {
		if matcher.Match(s) {
			return true
		}
	}
	return false
}

type not struct {
	m Matcher
}

// Not returns a matcher that matches if the given matcher does not match.
func Not(matcher Matcher) Matcher {
	return not{m: matcher}
}

func (m not) Match(s string) bool {
	return !m.m.Match(s)
}

type MatcherFunc func(string) bool

func (m MatcherFunc) Match(s string) bool {
	return m(s)
}

// MatchEverything returns a matcher that matches everything.
var MatchEverything Matcher = MatcherFunc(func(s string) bool {
	return true
})
