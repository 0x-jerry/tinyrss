package feeds

import "strings"

// ftsQuery escapes a user query for an FTS5 MATCH string. Each whitespace
// token is quoted (quotes doubled) and ANDed, so special FTS syntax is treated
// literally. Empty input returns "" meaning "no search filter".
func ftsQuery(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	fields := strings.Fields(q)
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		parts = append(parts, `"`+strings.ReplaceAll(f, `"`, `""`)+`"`)
	}
	return strings.Join(parts, " ")
}
