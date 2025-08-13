package main

import (
	"os"

	"github.com/fr12k/feller/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
