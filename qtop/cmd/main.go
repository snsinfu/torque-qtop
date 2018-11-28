package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/gdamore/tcell"

	"github.com/snsinfu/torque-qtop/qtop"
	"github.com/snsinfu/torque-qtop/torque"
)

const usage = `
Monitor PBS jobs

Usage:
  qtop [-h] [-t <interval>]

Options:
  -t <interval>  Specify update interval in seconds [default: 5]
  -h, --help     Print this help message and exit
`

const minInterval = 1

type config struct {
	Interval float64 `docopt:"-t"`
}

func (c *config) validate() error {
	if c.Interval < minInterval {
		return errors.New("update interval is too short")
	}
	return nil
}

func main() {
	opts, err := docopt.ParseDoc(usage)
	if err != nil {
		panic(err)
	}

	var c config

	if err := opts.Bind(&c); err != nil {
		fmt.Fprintln(os.Stderr, "option error:", err)
		os.Exit(64)
	}

	if err := c.validate(); err != nil {
		fmt.Fprintln(os.Stderr, "option error:", err)
		os.Exit(64)
	}

	if err := run(c); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(c config) error {
	conn, err := torque.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	scr, err := tcell.NewScreen()
	if err != nil {
		return err
	}

	if err := scr.Init(); err != nil {
		return err
	}
	defer scr.Fini()

	top := qtop.NewTop(conn)
	app := qtop.NewApp(top, scr, qtop.Config{
		Interval: time.Duration(c.Interval * float64(time.Second)),
	})

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for range sig {
			app.Quit()
		}
	}()

	return app.Start()
}
