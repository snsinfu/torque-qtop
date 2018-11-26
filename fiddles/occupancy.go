package main

import (
	"fmt"
	"os"

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

	type procUsage struct {
		used int
		max  int
	}

	usage := map[string]procUsage{}
	nodeNames := []string{}

	for _, node := range nodes {
		usage[node.Name] = procUsage{
			max: node.SlotCount,
		}
		nodeNames = append(nodeNames, node.Name)
	}

	jobs, err := torque.QueryJobs(conn)
	if err != nil {
		return err
	}

	for _, job := range jobs {
		for _, slot := range job.ExecSlots {
			usage[slot.Node] = procUsage{
				max:  usage[slot.Node].max,
				used: usage[slot.Node].used + 1,
			}
		}
	}

	for _, node := range nodeNames {
		fmt.Printf("%s: %2d/%2d\n", node, usage[node].used, usage[node].max)
	}

	return nil
}
