// Package migrations embeds the canonical SQL migrations.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
