// Package redact removes secret-shaped and card-shaped substrings before
// storage, logs, or model prompts. It does not attempt to detect all PII.
package redact

import "regexp"

var (
	cardLike = regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)
	email    = regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`)
	secret   = regexp.MustCompile(`(?i)\b(?:sk|api|key|token|secret)[-_][A-Za-z0-9_-]{8,}\b`)
)

// Text redacts card-like numbers, email addresses, and token-shaped strings.
func Text(s string) string {
	s = secret.ReplaceAllString(s, "[REDACTED]")
	s = email.ReplaceAllString(s, "[REDACTED_EMAIL]")
	s = cardLike.ReplaceAllStringFunc(s, func(m string) string {
		digits := 0
		for _, r := range m {
			if r >= '0' && r <= '9' {
				digits++
			}
		}
		if digits >= 13 && digits <= 19 {
			return "[REDACTED_PAN]"
		}
		return m
	})
	return s
}
