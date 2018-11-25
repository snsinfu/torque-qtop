package main

import (
	"fmt"
	"os"

	"github.com/docopt/docopt-go"
	"github.com/snsinfu/torque-qtop/qtop"
)

const usage = `
Monitor PBS jobs

Usage:
  qtop
`

func main() {
	opts, err := docopt.ParseDoc(usage)
	if err != nil {
		panic(err)
	}

	_ = opts

	if err := qtop.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
