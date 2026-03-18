package main

import (
	"os"

	"github.com/chaptersix/temporal-start-dev-ext/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
