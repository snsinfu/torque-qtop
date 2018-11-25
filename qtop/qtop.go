package qtop

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nsf/termbox-go"
	"github.com/snsinfu/torque-qtop/torque"
)

const (
	updateInterval = 3 * time.Second
)

func Run() error {
	conn, err := torque.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := termbox.Init(); err != nil {
		return err
	}
	defer termbox.Close()

	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT)
	signal.Notify(signals, syscall.SIGTERM)

	events := make(chan termbox.Event)
	go func() {
		for {
			events <- termbox.PollEvent()
		}
	}()

	ticks := time.Tick(updateInterval)

mainLoop:
	for {
		jobs, err := torque.QueryJobs(conn)
		if err != nil {
			return err
		}

	render:
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

		for i := 0; i < len(jobs); i++ {
			x := 1 + i
			y := 1
			termbox.SetCell(x, y, '|', termbox.ColorGreen, termbox.ColorDefault)
		}

		width, _ := termbox.Size()
		for i := 0; i < width; i++ {
			termbox.SetCell(i, 10, ' ', termbox.ColorDefault, termbox.ColorGreen)
		}

		if err := termbox.Flush(); err != nil {
			return err
		}

		select {
		case <-signals:
			break mainLoop

		case ev := <-events:
			switch ev.Type {
			case termbox.EventKey:
				switch ev.Ch {
				case 'Q', 'q':
					break mainLoop

				case 0:
					switch ev.Key {
					case termbox.KeyCtrlC:
						break mainLoop
					}
				}

			case termbox.EventResize:
				goto render

			case termbox.EventError:
				return ev.Err
			}
			continue

		case <-ticks:
			//
		}
	}

	return nil
}
