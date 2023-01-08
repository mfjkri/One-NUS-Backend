package utils

import (
	"regexp"
	"unicode"

	"golang.org/x/exp/utf8string"
)

func ContainsNumbers(s string) bool {
	return regexp.MustCompile(`\d`).MatchString(s)
}

func ContainsWhitespaces(s string) bool {
	return regexp.MustCompile(`\s`).MatchString(s)
}

func ContainsWhitespacesOrNumbers(s string) bool {
	return ContainsNumbers(s) || ContainsWhitespaces(s)
}

func ContainsLettersOnly(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func ContainsValidCharactersOnly(s string) bool {
	return utf8string.NewString(s).IsASCII()
}

func TrimString(s string, maxLen int) string {
	if len(s) >= maxLen {
		return s[:maxLen]
	}

	return s
}
