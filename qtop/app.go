package qtop

import (
	"fmt"
	"os/user"
	"strings"
	"time"

	"github.com/gdamore/tcell"
)

const (
	xMargin = 2
	yMargin = 1
)

type Config struct {
	Interval time.Duration
}

type App struct {
	top    *Top
	scr    tcell.Screen
	quit   chan bool
	update chan bool
	config Config
}

func NewApp(top *Top, scr tcell.Screen, config Config) *App {
	return &App{
		top:    top,
		scr:    scr,
		quit:   make(chan bool),
		update: make(chan bool),
		config: config,
	}
}

func (app *App) Start() error {
	go app.dispatch()

	app.top.Update()
	app.scr.Clear()

	tick := time.Tick(app.config.Interval)

loop:
	for {
		select {
		case <-app.quit:
			break loop

		case <-app.update:
			app.scr.Clear()
			app.draw()

		case <-tick:
			if err := app.top.Update(); err != nil {
				return err
			}
			app.scr.Clear()
			app.draw()
		}
	}

	return nil
}

func (app *App) Quit() {
	app.quit <- true
}

func (app *App) dispatch() {
	for {
		ev := app.scr.PollEvent()
		if ev == nil {
			break
		}

		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyCtrlC:
				app.quit <- true

			case tcell.KeyCtrlL:
				app.scr.Sync()

			case tcell.KeyRune:
				switch ev.Rune() {
				case 'q', 'Q':
					app.quit <- true
				}
			}

		case *tcell.EventResize:
			app.update <- true
		}
	}
}

func (app *App) draw() {
	sum := app.top.Current()

	y := 0
	y = app.drawCluster(y, sum.Cluster) + yMargin
	y = app.drawNodes(y, sum.Nodes) + yMargin
	y = app.drawJobs(y, sum.Jobs)

	app.scr.Show()
}

func (app *App) drawCluster(y int, cluster ClusterSummary) int {
	scr := app.scr
	w, _ := scr.Size()

	now := time.Now().Format(time.Stamp)
	printStr(scr, w-len(now)-xMargin, 0, now, tcell.StyleDefault)

	stat := fmt.Sprintf(
		"%d running, %d waiting / %d free",
		cluster.RunningJobs,
		cluster.WaitingJobs,
		cluster.FreeSlots,
	)
	printStr(scr, 2, 0, stat, tcell.StyleDefault)

	return y + 1
}

func (app *App) drawNodes(y int, nodes []NodeSummary) int {
	scr := app.scr

	nodeCols := 0
	for _, node := range nodes {
		if len(node.Name) > nodeCols {
			nodeCols = len(node.Name)
		}
	}

	for _, node := range nodes {
		name := fmt.Sprintf("%*s", -nodeCols, node.Name)
		util := fmt.Sprintf("[%2d/%2d]", node.UsedSlots, node.AvailSlots)
		meter := strings.Repeat("|", node.UsedSlots)

		if !node.Active {
			util = "[--/--]"
		}

		x := xMargin
		x += printStr(scr, x, y, name, tcell.StyleDefault.Foreground(tcell.ColorTeal))
		x += 1
		x += printStr(scr, x, y, util, tcell.StyleDefault.Foreground(tcell.ColorGray))
		x += 1
		x += printStr(scr, x, y, meter, tcell.StyleDefault.Foreground(tcell.ColorGreen))

		y++
	}

	return y
}

func (app *App) drawJobs(y int, jobs []JobSummary) int {
	y = app.drawJobHeader(y)

	for _, job := range jobs {
		y = app.drawJob(y, job)
	}

	return y
}

func (app *App) drawJobHeader(y int) int {
	style := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGreen)

	scr := app.scr
	x := 0
	x += printStr(scr, x, y, "  ", style)
	x += printStr(scr, x, y, fmt.Sprintf("%-10s", "USER"), style)
	x += printStr(scr, x, y, " ", style)
	x += printStr(scr, x, y, fmt.Sprintf("%-20s", "JOB"), style)
	x += printStr(scr, x, y, " ", style)
	x += printStr(scr, x, y, "S", style)
	x += printStr(scr, x, y, " ", style)
	x += printStr(scr, x, y, "NJOB", style)
	x += printStr(scr, x, y, " ", style)
	x += printStr(scr, x, y, "NCPU", style)
	x += printStr(scr, x, y, " ", style)
	x += printStr(scr, x, y, fmt.Sprintf("%6s", "CPU%"), style)
	x += printStr(scr, x, y, " ", style)
	x += printStr(scr, x, y, fmt.Sprintf("%9s", "MAX TIME"), style)

	w, _ := scr.Size()
	if x < w {
		x += printStr(scr, x, y, strings.Repeat(" ", w-x), style)
	}

	return y + 1
}

func (app *App) drawJob(y int, job JobSummary) int {
	owner := job.Owner[:strings.Index(job.Owner, "@")]
	maxTime := formatClock(job.MaxWalltime)

	style := tcell.StyleDefault
	styleUser := style
	styleState := style

	me, _ := user.Current()
	if owner != me.Username {
		styleUser = styleUser.Foreground(tcell.ColorGray)
	}

	switch job.State {
	case "R":
		styleState = styleState.Foreground(tcell.ColorGreen)
	case "C", "E":
		styleState = styleState.Foreground(tcell.ColorOlive)
	case "H", "Q", "T", "W":
		styleState = styleState.Foreground(tcell.ColorTeal)
	}

	scr := app.scr
	x := xMargin
	x += printStr(scr, x, y, fmt.Sprintf("%-10s", owner), styleUser)
	x += printStr(scr, x, y, " ", style)
	x += printStr(scr, x, y, fmt.Sprintf("%-20s", job.Name), style)
	x += printStr(scr, x, y, " ", style)
	x += printStr(scr, x, y, job.State, styleState)
	x += printStr(scr, x, y, " ", style)
	x += printStr(scr, x, y, fmt.Sprintf("%4d", job.Count), style)
	x += printStr(scr, x, y, " ", style)
	x += printStr(scr, x, y, fmt.Sprintf("%4d", job.Occupancy), style)
	x += printStr(scr, x, y, " ", style)
	x += printStr(scr, x, y, fmt.Sprintf("%6.1f", job.CPUUsage*100), style)
	x += printStr(scr, x, y, " ", style)
	x += printStr(scr, x, y, maxTime, style)

	w, _ := scr.Size()
	if x < w {
		x += printStr(scr, x, y, strings.Repeat(" ", w-x), style)
	}

	return y + 1
}

func printStr(scr tcell.Screen, x, y int, s string, style tcell.Style) int {
	for i, c := range s {
		scr.SetContent(x+i, y, c, nil, style)
	}
	return len(s)
}

func formatClock(n int) string {
	sec := n
	min := sec / 60
	hour := min / 60

	return fmt.Sprintf("%3d:%02d:%02d", hour, min%60, sec%60)
}
