package main

import (
	"os"

	"github.com/paoloanzn/flare-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
