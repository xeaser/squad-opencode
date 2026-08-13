package main

import (
	"os"

	"github.com/squad-opencode/squad-opencode/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:]))
}
