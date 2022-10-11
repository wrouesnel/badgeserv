package main

import (
	"github.com/wrouesnel/badgeserv/pkg/entrypoint"
	"os"
)

func main() {
	exitCode := entrypoint.Entrypoint(os.Stdout, os.Stderr)
	os.Exit(exitCode)
}
