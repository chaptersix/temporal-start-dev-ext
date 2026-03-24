//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/magefile/mage/sh"
)

// Build compiles the temporal-ts_net binary
func Build() error {
	fmt.Println("Building temporal-ts_net...")
	return sh.Run("go", "build", "-o", "./bin/temporal-ts_net", "./cmd/temporal-ts_net")
}

// Test runs all tests with race detection and randomized order
func Test() error {
	fmt.Println("Running tests with race detector and shuffle...")
	return sh.RunV("go", "test", "-race", "-shuffle=on", "./...")
}

// Fmt formats all Go source files
func Fmt() error {
	fmt.Println("Formatting code...")
	return sh.Run("go", "fmt", "./...")
}

// Clean removes build artifacts
func Clean() error {
	fmt.Println("Cleaning build artifacts...")
	return sh.Rm("./bin")
}

// Install builds and installs the binary to $GOPATH/bin
func Install() error {
	fmt.Println("Installing temporal-ts_net...")
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		cmd := exec.Command("go", "env", "GOPATH")
		out, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to determine GOPATH: %w", err)
		}
		gopath = string(out[:len(out)-1]) // trim newline
	}
	return sh.Run("go", "build", "-o", gopath+"/bin/temporal-ts_net", "./cmd/temporal-ts_net")
}
