package main

import (
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/gdamore/tcell"
	"github.com/snsinfu/torque-qtop/torque"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
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

	top := NewTop(conn)
	app := NewApp(top, scr)

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for range sig {
			app.Quit()
		}
	}()

	return app.Start()
}

type Top struct {
	conn torque.Conn
	sum  *Summary
}

func NewTop(conn torque.Conn) *Top {
	return &Top{
		conn: conn,
	}
}

func (top *Top) Update() error {
	nodes, err := torque.QueryNodes(top.conn)
	if err != nil {
		return err
	}

	jobs, err := torque.QueryJobs(top.conn)
	if err != nil {
		return err
	}

	sum := summarize(nodes, jobs)
	top.sum = &sum
	return nil
}

func (top *Top) Current() *Summary {
	return top.sum
}

type App struct {
	top      *Top
	scr      tcell.Screen
	quit     chan bool
	update   chan bool
	interval time.Duration
}

func NewApp(top *Top, scr tcell.Screen) *App {
	return &App{
		top:      top,
		scr:      scr,
		quit:     make(chan bool),
		update:   make(chan bool),
		interval: 5*time.Second,
	}
}

func (app *App) Start() error {
	go app.dispatch()

	app.top.Update()
	app.scr.Clear()

	tick := time.Tick(app.interval)

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
	scr := app.scr
	sum := app.top.Current()

	w, _ := scr.Size()

	now := time.Now().Format(time.Stamp)
	printStr(scr, w-len(now)-2, 0, now, tcell.StyleDefault)

	stat := fmt.Sprintf(
		"%d running, %d waiting / %d free",
		sum.Cluster.RunningJobs,
		sum.Cluster.WaitingJobs,
		sum.Cluster.FreeSlots,
	)
	printStr(scr, 2, 0, stat, tcell.StyleDefault)

	nodeRows := len(sum.Nodes)
	nodeCols := 0
	for _, node := range sum.Nodes {
		if len(node.Name) > nodeCols {
			nodeCols = len(node.Name)
		}
	}

	for i, node := range sum.Nodes {
		name := fmt.Sprintf("%*s", -nodeCols, node.Name)
		util := fmt.Sprintf("[%2d/%2d]", node.UsedSlots, node.AvailSlots)
		meter := strings.Repeat("|", node.UsedSlots)

		if !node.Active {
			util = "[--/--]"
		}

		x := 2
		y := 2 + i

		x += printStr(scr, x, y, name, tcell.StyleDefault.Foreground(tcell.ColorTeal))
		x += 1
		x += printStr(scr, x, y, util, tcell.StyleDefault.Foreground(tcell.ColorGray))
		x += 1
		x += printStr(scr, x, y, meter, tcell.StyleDefault.Foreground(tcell.ColorGreen))
	}

	if true {
		x := 0
		y := 3 + nodeRows
		style := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorGreen)

		x += printStr(scr, x, y, "  ", style)
		x += printStr(scr, x, y, fmt.Sprintf("%-10s", "USER"), style)
		x += printStr(scr, x, y, " ", style)
		x += printStr(scr, x, y, fmt.Sprintf("%-15s", "JOB"), style)
		x += printStr(scr, x, y, " ", style)
		x += printStr(scr, x, y, "S", style)
		x += printStr(scr, x, y, " ", style)
		x += printStr(scr, x, y, "NJOB", style)
		x += printStr(scr, x, y, " ", style)
		x += printStr(scr, x, y, "NCPU", style)
		x += printStr(scr, x, y, " ", style)
		x += printStr(scr, x, y, fmt.Sprintf("%6s", "CPU%"), style)
		x += printStr(scr, x, y, " ", style)
		x += printStr(scr, x, y, "TIME", style)

		if x < w {
			x += printStr(scr, x, y, strings.Repeat(" ", w - x), style)
		}
	}

	me, _ := user.Current()

	for i, job := range sum.Jobs {
		user := job.Owner[:strings.Index(job.Owner, "@")]

		timeRange := formatClock(job.MinWalltime)
		if job.MinWalltime != job.MaxWalltime {
			timeRange += " - "
			timeRange += formatClock(job.MaxWalltime)
		}

		x := 0
		y := 4 + nodeRows + i
		style := tcell.StyleDefault

		styleUser := style
		styleState := style

		if user != me.Username {
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

		x += printStr(scr, x, y, "  ", style)
		x += printStr(scr, x, y, fmt.Sprintf("%-10s", user), styleUser)
		x += printStr(scr, x, y, " ", style)
		x += printStr(scr, x, y, fmt.Sprintf("%-15s", job.Name), style)
		x += printStr(scr, x, y, " ", style)
		x += printStr(scr, x, y, job.State, styleState)
		x += printStr(scr, x, y, " ", style)
		x += printStr(scr, x, y, fmt.Sprintf("%4d", job.Count), style)
		x += printStr(scr, x, y, " ", style)
		x += printStr(scr, x, y, fmt.Sprintf("%4d", job.Occupancy), style)
		x += printStr(scr, x, y, " ", style)
		x += printStr(scr, x, y, fmt.Sprintf("%6.1f", job.CPUUsage*100), style)
		x += printStr(scr, x, y, " ", style)
		x += printStr(scr, x, y, timeRange, style)

		if x < w {
			x += printStr(scr, x, y, strings.Repeat(" ", w - x), style)
		}
	}

	scr.Show()
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

	return fmt.Sprintf("%3d:%02d:%02d", hour, min % 60, sec % 60)
}

type Summary struct {
	Cluster ClusterSummary
	Nodes   []NodeSummary
	Jobs    []JobSummary
}

type ClusterSummary struct {
	RunningJobs int
	WaitingJobs int
	UsedSlots   int
	FreeSlots   int
}

type NodeSummary struct {
	Name       string
	Active     bool
	AvailSlots int
	UsedSlots  int
}

type JobSummary struct {
	Name          string
	Owner         string
	State         string
	Count         int
	Occupancy     int
	HostOccupancy map[string]int
	MinWalltime   int
	MaxWalltime   int
	CPUUsage      float64
}

func summarize(nodes []torque.Node, jobs []torque.Job) Summary {
	return Summary{
		Cluster: summarizeCluster(nodes, jobs),
		Nodes:   summarizeNodes(nodes, jobs),
		Jobs:    summarizeJobs(jobs),
	}
}

func summarizeCluster(nodes []torque.Node, jobs []torque.Job) ClusterSummary {
	var sum ClusterSummary

	availSlots := 0

	for _, node := range nodes {
		if node.State != "down" {
			availSlots += node.SlotCount
		}
	}

	for _, job := range jobs {
		switch job.State {
		case "C":
			continue

		case "R":
			sum.RunningJobs++

		default:
			sum.WaitingJobs++
		}

		sum.UsedSlots += len(job.ExecSlots)
	}

	sum.FreeSlots = availSlots - sum.UsedSlots

	return sum
}

func summarizeNodes(nodes []torque.Node, jobs []torque.Job) []NodeSummary {
	var sums []NodeSummary

	index := map[string]int{}

	for i, node := range nodes {
		sums = append(sums, NodeSummary{
			Name:       node.Name,
			Active:     node.State != "down",
			AvailSlots: node.SlotCount,
		})
		index[node.Name] = i
	}

	for _, job := range jobs {
		if job.State == "C" {
			continue
		}

		for _, slot := range job.ExecSlots {
			sums[index[slot.Node]].UsedSlots++
		}
	}

	return sums
}

func summarizeJobs(jobs []torque.Job) []JobSummary {
	type groupKey struct {
		Name  string
		Owner string
		State string
	}
	sumsMap := map[groupKey]*JobSummary{}

	for _, job := range jobs {
		key := groupKey{
			Name:  basename(job.Name),
			Owner: job.Owner,
			State: job.State,
		}

		sum, ok := sumsMap[key]
		if !ok {
			sum = &JobSummary{
				State:         key.State,
				Name:          key.Name,
				Owner:         key.Owner,
				HostOccupancy: map[string]int{},
			}
			sumsMap[key] = sum
		}

		sum.Count++

		if job.State != "R" {
			continue
		}

		sum.Occupancy += len(job.ExecSlots)
		for _, slot := range job.ExecSlots {
			sum.HostOccupancy[slot.Node]++
		}

		if sum.MinWalltime == 0 || job.Walltime < sum.MinWalltime {
			sum.MinWalltime = job.Walltime
		}

		if sum.MaxWalltime == 0 || job.Walltime > sum.MaxWalltime {
			sum.MaxWalltime = job.Walltime
		}

		if job.Walltime > 0 {
			cpuUsage := float64(job.CPUTime) / float64(job.Walltime)
			sum.CPUUsage += (cpuUsage - sum.CPUUsage) / float64(sum.Count)
		}
	}

	var sums []JobSummary

	for _, sum := range sumsMap {
		sums = append(sums, *sum)
	}

	sort.Slice(sums, func(i, j int) bool {
		return compareJobs(sums[i], sums[j]) < 0
	})

	return sums
}

func basename(s string) string {
	i := strings.LastIndex(s, "-")
	if i == -1 {
		return s
	}
	return s[:i]
}

func compareJobs(a, b JobSummary) int {
	if r := strings.Compare(a.Owner, b.Owner); r != 0 {
		return r
	}

	if r := strings.Compare(a.Name, b.Name); r != 0 {
		return r
	}

	if r := strings.Compare(a.State, b.State); r != 0 {
		return r
	}

	return 0
}
