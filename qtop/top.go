package qtop

import (
	"regexp"
	"sort"
	"strings"

	"github.com/snsinfu/torque-qtop/torque"
)

var jobSuffixPattern = regexp.MustCompile(`-\d+$`)

type Top struct {
	conn torque.Conn
	sum  *Summary
}

func NewTop(conn torque.Conn) *Top {
	return &Top{conn: conn}
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

	sum := Summarize(nodes, jobs)
	top.sum = &sum
	return nil
}

func (top *Top) Current() *Summary {
	return top.sum
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
	Owners     []NodeOwnerSummary
}

type NodeOwnerSummary struct {
	Owner     string
	Occupancy int
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
	IDs           []string
}

func Summarize(nodes []torque.Node, jobs []torque.Job) Summary {
	return Summary{
		Cluster: SummarizeCluster(nodes, jobs),
		Nodes:   SummarizeNodes(nodes, jobs),
		Jobs:    SummarizeJobs(jobs),
	}
}

func SummarizeCluster(nodes []torque.Node, jobs []torque.Job) ClusterSummary {
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
			continue
		}

		sum.UsedSlots += len(job.ExecSlots)
	}

	sum.FreeSlots = availSlots - sum.UsedSlots

	return sum
}

func SummarizeNodes(nodes []torque.Node, jobs []torque.Job) []NodeSummary {
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
		if job.State != "R" {
			continue
		}

		for _, slot := range job.ExecSlots {
			sums[index[slot.Node]].UsedSlots++
		}
	}

	// FIXME: inefficient
	jobSum := SummarizeJobs(jobs)

	hostOwners := map[string]map[string]int{}
	for _, node := range nodes {
		hostOwners[node.Name] = map[string]int{}
	}

	for _, job := range jobSum {
		for host, occ := range job.HostOccupancy {
			hostOwners[host][job.Owner] += occ
		}
	}

	for host, owners := range hostOwners {
		ownersSum := []NodeOwnerSummary{}

		for name, occ := range owners {
			ownersSum = append(ownersSum, NodeOwnerSummary{
				Owner:     name,
				Occupancy: occ,
			})
		}

		sort.Slice(ownersSum, func(i, j int) bool {
			return ownersSum[i].Occupancy > ownersSum[j].Occupancy
		})

		sums[index[host]].Owners = ownersSum
	}

	return sums
}

func SummarizeJobs(jobs []torque.Job) []JobSummary {
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
		sum.IDs = append(sum.IDs, job.ID)

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
	return jobSuffixPattern.ReplaceAllString(s, "")
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
