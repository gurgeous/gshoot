package main

import (
	"os"

	"github.com/gurgeous/gshoot/internal/smoke"
)

func main() {
	os.Exit(smoke.Run(os.Args[1:], os.Stdout, os.Stderr))
}
