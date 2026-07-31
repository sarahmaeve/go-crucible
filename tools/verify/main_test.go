package main

import (
	"regexp"
	"testing"
)

func TestCommonSpoilerPatternsCatchPrescriptiveComments(t *testing.T) {
	comments := []string{
		"Callers should prefer errors.Is over string comparison.",
		"Implementations should wrap this sentinel with %w.",
		"The goroutine is blocked trying to send on a full channel.",
		"File descriptors accumulate until the function returns.",
	}

	for _, comment := range comments {
		if !matchesAny(commonSpoilerPatterns, comment) {
			t.Errorf("common spoiler patterns did not match %q", comment)
		}
	}
}

func TestCommonSpoilerPatternsAllowContractComments(t *testing.T) {
	comments := []string{
		"Implementations report an error matching ErrDuplicate for an existing key.",
		"Read must honor context cancellation.",
		"The handler rejects oversized bodies with status 413.",
		"The batch must release its resources before returning.",
	}

	for _, comment := range comments {
		if matchesAny(commonSpoilerPatterns, comment) {
			t.Errorf("common spoiler patterns rejected contract comment %q", comment)
		}
	}
}

func matchesAny(patterns []*regexp.Regexp, text string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}
