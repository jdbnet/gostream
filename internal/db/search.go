package db

import (
	"strings"
	"unicode/utf8"
)

const (
	searchFullText = "fulltext"
	searchLike     = "like"
)

func hasShortToken(query string) bool {
	for _, token := range strings.Fields(query) {
		if utf8.RuneCountInString(token) < 3 {
			return true
		}
	}
	return false
}

func textSearchMode(driver, query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}
	if driver == "mysql" && hasShortToken(query) {
		return searchLike
	}
	return searchFullText
}

func shouldFallbackToLike(driver, query, mode string, total int, queryErr error) bool {
	if driver != "mysql" || strings.TrimSpace(query) == "" || mode != searchFullText {
		return false
	}
	if queryErr != nil {
		return true
	}
	return total == 0
}

func likePattern(query string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + replacer.Replace(query) + "%"
}

func ftsMatchQuery(query string) string {
	parts := strings.Fields(query)
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.ReplaceAll(part, `"`, "")
		if part == "" {
			continue
		}
		quoted = append(quoted, `"`+part+`"`)
	}
	return strings.Join(quoted, " ")
}
