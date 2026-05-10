package main

import (
	"os"

	"github.com/gurgeous/gshoot/internal/sub"
)

func main() {
	os.Exit(sub.Main(os.Args[1:], os.Stdout, os.Stderr))
}
