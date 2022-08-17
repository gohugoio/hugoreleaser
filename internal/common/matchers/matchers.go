package matchers

type Matcher interface {
	Match(string) bool
}

func And(matchers ...Matcher) Matcher {
	return and{matchers}
}

type and struct {
	matchers []Matcher
}

func (m and) Match(s string) bool {
	for _, matcher := range m.matchers {
		if !matcher.Match(s) {
			return false
		}
	}
	return true
}
