package matchers

import (
	"errors"
	"strings"
	"sync"

	"github.com/gobwas/glob"
)

type globCacheMap struct {
	sync.RWMutex
	globs map[string]Matcher
}

var globCache = globCacheMap{
	globs: make(map[string]Matcher),
}

// Glob returns a matcher that matches if all given glob patterns matches the given string.
// A pattern can be negated with a leading !.
func Glob(patterns ...string) (Matcher, error) {
	if len(patterns) == 0 {
		return nil, errors.New("empty patterns")
	}
	if len(patterns) == 1 {
		return globOne(patterns[0])
	}

	matchers := make([]Matcher, len(patterns))
	for i, p := range patterns {
		g, err := globOne(p)
		if err != nil {
			return nil, err
		}
		matchers[i] = g
	}

	return And(matchers...), nil
}

func globOne(pattern string) (Matcher, error) {
	if pattern == "" {
		return nil, errors.New("empty pattern")
	}
	globCache.RLock()
	g, ok := globCache.globs[pattern]
	globCache.RUnlock()
	if ok {
		return g, nil
	}

	var negate bool
	if pattern[0] == '!' {
		negate = true
		pattern = strings.TrimPrefix(pattern, "!")
	}

	g, err := glob.Compile(pattern)
	if err != nil {
		return nil, err
	}

	if negate {
		g = Not(g)
	}

	globCache.Lock()
	globCache.globs[pattern] = g
	globCache.Unlock()

	return g, nil
}
