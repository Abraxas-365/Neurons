// Package dbx holds small SQL helpers shared by the gamification repositories.
package dbx

import "strings"

// Prefixed qualifies a comma-separated column list with a table alias so the
// same list can be reused in JOIN queries without ambiguous references.
func Prefixed(alias, columns string) string {
	parts := strings.Split(columns, ",")
	for i, p := range parts {
		parts[i] = alias + "." + strings.TrimSpace(p)
	}
	return strings.Join(parts, ", ")
}
