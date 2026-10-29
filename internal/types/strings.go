package types

import "strings"

// IsBlank mirrors org.apache.commons.lang3.StringUtils.isBlank: true when the
// string is empty or contains only whitespace.
func IsBlank(s string) bool { return strings.TrimSpace(s) == "" }

// IsNotBlank is the negation of IsBlank.
func IsNotBlank(s string) bool { return !IsBlank(s) }

// JavaSplit mirrors java.lang.String.split(sep) with the default limit of 0,
// which discards trailing empty strings. The legacy code relies on this for
// comma-separated expressions such as tagScope ("1,2") and MJ marketExpr
// ("100,20"), so reproducing it keeps parsing identical.
func JavaSplit(s, sep string) []string {
	parts := strings.Split(s, sep)
	for len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}
