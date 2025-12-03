// Package main provides the entry point for the nvs application.
package main

import (
	"os"

	"github.com/y3owk1n/nvs/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
