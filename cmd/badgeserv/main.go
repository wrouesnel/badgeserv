package main

import (
	"os"

	"github.com/wrouesnel/badgeserv/pkg/entrypoint"
)

func main() {
	exitCode := entrypoint.Entrypoint(os.Stdout, os.Stderr)
	os.Exit(exitCode)
}
