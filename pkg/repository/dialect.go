package repository

import (
	"strconv"
	"strings"
)

// Rebind converts SQLite-style ? placeholders to PostgreSQL $n placeholders.
func RebindPostgres(query string) string {
	var b strings.Builder
	n := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			n++
		} else {
			b.WriteByte(query[i])
		}
	}
	return b.String()
}
