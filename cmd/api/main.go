package main

import (
	"fmt"
	"os"

	"educore/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "application startup error: %v\n", err)
		os.Exit(1)
	}
}
