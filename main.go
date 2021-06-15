package main

import (
	"fmt"
	"os"

	"github.com/1984weed/backlog-git-pr-diff-checker/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}
