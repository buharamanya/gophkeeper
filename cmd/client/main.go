package main

import (
	"fmt"
	"os"

	"github.com/buharamanya/gophkeeper/client/cli"
)

var (
	version   = "1.0.0"
	buildDate = "2024-12-01"
)

func main() {
	if err := cli.Execute(version, buildDate); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
