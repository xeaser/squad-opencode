package main

import (
	"os"

	"github.com/xeaser/squad-opencode/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:]))
}
