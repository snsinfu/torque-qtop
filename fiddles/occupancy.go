package main

import (
	"fmt"
	"os"
	"strconv"
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

	type procUsage struct {
		used int
		max  int
	}

	usage := map[string]procUsage{}
	nodeNames := []string{}

	for _, node := range nodes {
		usage[node.Name] = procUsage{
			max: node.NP,
		}
		nodeNames = append(nodeNames, node.Name)
	}

	jobs, err := torque.QueryJobs(conn)
	if err != nil {
		return err
	}

	for _, job := range jobs {
		execHost, ok := job.Attrs["exec_host"]
		if !ok {
			continue
		}

		// exec_host = host_cpu *( "+" host_cpu )
		// host_cpu  = host "/" cpus
		// cpus      = cpu_range *( "," cpu_range )
		// cpu_range = cpu ?( "-" cpu )
		// cpu       = int

		for _, hostCPUs := range strings.Split(execHost, "+") {
			hostAndCPUs := strings.Split(hostCPUs, "/")
			host := hostAndCPUs[0]
			cpus := hostAndCPUs[1]

			hostUse := 0

			for _, cpuRange := range strings.Split(cpus, ",") {
				firstAndLast := strings.Split(cpuRange, "-")
				switch len(firstAndLast) {
				case 1:
					hostUse++
				case 2:
					first, _ := strconv.Atoi(firstAndLast[0])
					last, _ := strconv.Atoi(firstAndLast[1])
					hostUse += last - first + 1
				}
			}

			usage[host] = procUsage{
				max:  usage[host].max,
				used: usage[host].used + hostUse,
			}
		}
	}

	for _, node := range nodeNames {
		fmt.Printf("%s: %2d/%2d\n", node, usage[node].used, usage[node].max)
	}

	return nil
}
