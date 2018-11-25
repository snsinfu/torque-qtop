package main

import (
	"encoding/json"
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

	fmt.Println("Nodes:")

	nodes, err := torque.QueryNodes(conn)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		fmt.Printf("%s [%d]: %s\n", node.Name, node.NP, node.State)
		attrs, _ := json.Marshal(node.Attrs)
		fmt.Println(string(attrs))
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("Jobs:")

	jobs, err := torque.QueryJobs(conn)
	if err != nil {
		return err
	}

	for _, job := range jobs {
		fmt.Printf("%s [%s]: %s %q\n", job.Name, job.State, job.Owner, job.Name)
		attrs, _ := json.Marshal(job.Attrs)
		fmt.Println(string(attrs))
		fmt.Println()
	}

	return nil
}
