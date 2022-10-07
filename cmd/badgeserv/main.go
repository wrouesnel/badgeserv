package main

import (
	"github.com/wrouesnel/badgeserv/pkg/entrypoint"
	"os"
)

func main() {
	exit_code := entrypoint.Entrypoint(os.Stdout, os.Stderr)
	os.Exit(exit_code)
}
