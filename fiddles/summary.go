package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

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

	nodes, err := torque.QueryNodes(conn)
	if err != nil {
		return err
	}

	jobs, err := torque.QueryJobs(conn)
	if err != nil {
		return err
	}

	clusterSumm := summarizeCluster(nodes, jobs)
	nodeSumms := summarizeNodes(nodes, jobs)
	jobSumms := summarizeJobs(jobs)

	fmt.Printf(
		"%d running %d waiting / %d used %d free\n",
		clusterSumm.RunningJobs,
		clusterSumm.WaitingJobs,
		clusterSumm.UsedSlots,
		clusterSumm.FreeSlots,
	)

	for _, summ := range nodeSumms {
		fmt.Printf(
			"%s [%2d/%2d] %s\n",
			summ.Name,
			summ.UsedSlots,
			summ.AvailSlots,
			strings.Repeat("|", summ.UsedSlots),
		)
	}

	fmt.Printf(
		"%-10s %-16s %1s %4s %4s %6s %s\n",
		"USER",
		"JOB",
		"S",
		"NJOB",
		"NCPU",
		"CPU%",
		"TIME",
	)

	for _, summ := range jobSumms {
		shortOwner := summ.Owner[:strings.Index(summ.Owner, "@")]

		fmt.Printf(
			"%-10s %-16s %1s %4d %4d %6.1f %s\n",
			shortOwner,
			summ.Name,
			summ.State,
			summ.Count,
			summ.Occupancy,
			summ.CPUUsage*100,
			fmt.Sprintf("%d", summ.MinWalltime),
		)
	}

	return nil
}

type nodeSummary struct {
	Name       string
	Active     bool
	AvailSlots int
	UsedSlots  int
}

type jobGroupSummary struct {
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

type clusterSummary struct {
	RunningJobs int
	WaitingJobs int
	UsedSlots   int
	FreeSlots   int
}

func summarizeCluster(nodes []torque.Node, jobs []torque.Job) clusterSummary {
	summ := clusterSummary{}

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
			summ.RunningJobs++

		default:
			summ.WaitingJobs++
		}

		summ.UsedSlots += len(job.ExecSlots)
	}

	summ.FreeSlots = availSlots - summ.UsedSlots

	return summ
}

func summarizeNodes(nodes []torque.Node, jobs []torque.Job) []nodeSummary {
	summary := []nodeSummary{}
	index := map[string]int{}

	for i, node := range nodes {
		summary = append(summary, nodeSummary{
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
			summary[index[slot.Node]].UsedSlots++
		}
	}

	return summary
}

func summarizeJobs(jobs []torque.Job) []jobGroupSummary {
	type groupKey struct {
		Name  string
		Owner string
		State string
	}
	summs := map[groupKey]*jobGroupSummary{}

	for _, job := range jobs {
		key := groupKey{
			Name:  job.Name,
			Owner: job.Owner,
			State: job.State,
		}

		summ, ok := summs[key]
		if !ok {
			summ = &jobGroupSummary{
				State:         key.State,
				Name:          key.Name,
				Owner:         key.Owner,
				HostOccupancy: map[string]int{},
			}
			summs[key] = summ
		}

		summ.Count++

		if job.State != "R" {
			continue
		}

		summ.Occupancy += len(job.ExecSlots)
		for _, slot := range job.ExecSlots {
			summ.HostOccupancy[slot.Node]++
		}

		if summ.MinWalltime == 0 || job.Walltime < summ.MinWalltime {
			summ.MinWalltime = job.Walltime
		}

		if summ.MaxWalltime == 0 || job.Walltime > summ.MaxWalltime {
			summ.MaxWalltime = job.Walltime
		}

		if job.Walltime > 0 {
			cpuUsage := float64(job.CPUTime) / float64(job.Walltime)
			summ.CPUUsage += (cpuUsage - summ.CPUUsage) / float64(summ.Count)
		}
	}

	r := []jobGroupSummary{}

	for _, summ := range summs {
		r = append(r, *summ)
	}

	sort.Slice(r, func(i, j int) bool {
		switch strings.Compare(r[i].Owner, r[j].Owner) {
		case -1:
			return true
		case 1:
			return false
		}

		switch strings.Compare(r[i].Name, r[j].Name) {
		case -1:
			return true
		case 1:
			return false
		}

		switch strings.Compare(r[i].State, r[j].State) {
		case -1:
			return true
		case 1:
			return false
		}

		return false
	})

	return r
}
