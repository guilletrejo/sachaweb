// Package migrations embeds SQL migration files into the Go binary.
//
// WHAT IS go:embed?
// Go 1.16 introduced the embed package, which lets you include files
// directly inside the compiled binary. When you deploy your app,
// the migration SQL is baked into the binary — no need to ship
// separate .sql files alongside it.
//
// The //go:embed directive below tells the Go compiler: "at build time,
// read all .sql files in this directory and store them in the FS variable."
// At runtime, you can read them like a normal filesystem.
package migrations

import "embed"

// FS contains all .sql files in this directory, embedded at compile time.
// Other packages import this and read the SQL files from it.
//
//go:embed *.sql
var FS embed.FS
