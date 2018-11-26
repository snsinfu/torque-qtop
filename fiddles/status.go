package main

import (
	"fmt"
	"os"
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

	fmt.Println("Nodes:")

	nodes, err := torque.QueryNodes(conn)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		fmt.Printf("%s [%d]: %s\n", node.Name, node.SlotCount, node.State)
	}

	fmt.Println()
	fmt.Println("Jobs:")

	jobs, err := torque.QueryJobs(conn)
	if err != nil {
		return err
	}

	for _, job := range jobs {
		shortID := job.ID[:strings.Index(job.ID, ".")]
		shortOwner := job.Owner[:strings.Index(job.Owner, "@")]

		fmt.Printf("%s [%s]: %s %q\n", shortID, job.State, shortOwner, job.Name)
		fmt.Printf("  slots    = %d\n", len(job.ExecSlots))
		fmt.Printf("  walltime = %d\n", job.Walltime)
		fmt.Printf("  cputime  = %d\n", job.CPUTime)
		fmt.Println()
	}

	return nil
}
