package main

import (
	"fmt"
	"os"

	"github.com/lukas/focus/internal/app"
)

func main() {
	application := app.New(os.Stdout)
	if err := application.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "focus:", err)
		os.Exit(1)
	}
}
