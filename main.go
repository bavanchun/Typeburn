package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"

	"github.com/bavanchun/Typeburn/internal/cli"
)

func main() {
	root := cli.NewRoot()
	if err := fang.Execute(context.Background(), root, fang.WithoutVersion()); err != nil {
		os.Exit(cli.ExitCode(err))
	}
}
