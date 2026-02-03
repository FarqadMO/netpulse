// Package main is the entry point for netpulse.
package main

import (
	"os"
)

func main() {
	if err := Execute(); err != nil {
		os.Exit(1)
	}
}
