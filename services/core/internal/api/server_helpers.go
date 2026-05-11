package api

import (
	"database/sql"
	"strings"

	"forge/projectforge/services/core/internal/permissions"
)

type requestAuditMeta struct {
	CorrelationID string
	TraceID       string
	WorkspaceID   string
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func profileIDOrEmpty(p *permissions.Profile) string {
	if p == nil {
		return ""
	}
	return p.ID
}

func listSourcePaths(db *sql.DB) []string {
	rows, err := db.Query(`SELECT path FROM sources ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return out
		}
		out = append(out, p)
	}
	return out
}
