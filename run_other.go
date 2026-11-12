//go:build !windows

package main

import (
	"fmt"
	"os"
)

func run(options) int {
	showError("DeadKeys only runs on Windows.")
	return 1
}

func showError(msg string) { fmt.Fprintln(os.Stderr, "deadkeys:", msg) }

func showInfo(msg string) { fmt.Println(msg) }
