package main

import (
	"os"

	"github.com/gurgeous/gshoot/internal/sub"
)

func main() {
	os.Exit(sub.Run(os.Args[1:], os.Stdout, os.Stderr))
}
