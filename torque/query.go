package torque

import (
	"fmt"
	"strconv"
	"strings"
)

// See torque: src/include/pbs_batchreqtype_db.h
const (
	pbsBatchProtType       = 2
	pbsBatchProtVer        = 2
	pbsBatchStatusJob      = 19
	pbsBatchStatusNode     = 58
	batchReplyChoiceStatus = 6
)

// A Node contains information of a compute node.
type Node struct {
	Name      string
	State     string
	SlotCount int
}

// QueryNodes returns the state of the compute nodes in the cluster.
func QueryNodes(c Conn) ([]Node, error) {
	entities, err := queryEntity(c, pbsBatchStatusNode)
	if err != nil {
		return nil, err
	}

	nodes := []Node{}

	for _, ent := range entities {
		np, err := strconv.Atoi(ent.attrs["np"])
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, Node{
			Name:      ent.name,
			State:     ent.attrs["state"],
			SlotCount: np,
		})
	}

	return nodes, err
}

// A Job contains information of a batch job.
type Job struct {
	ID        string
	Name      string
	Owner     string
	State     string
	ExecSlots []Slot
	Walltime  int
	CPUTime   int
}

// A Slot identifies a single execution slot in the job scheduler.
type Slot struct {
	Node  string
	Index int
}

// QueryJobs returns the state of the batch jobs in the cluster.
func QueryJobs(c Conn) ([]Job, error) {
	entities, err := queryEntity(c, pbsBatchStatusJob)
	if err != nil {
		return nil, err
	}

	jobs := []Job{}

	for _, ent := range entities {
		job := Job{
			ID:    ent.name,
			Name:  ent.attrs["Job_Name"],
			Owner: ent.attrs["Job_Owner"],
			State: ent.attrs["job_state"],
		}

		if execHost, ok := ent.attrs["exec_host"]; ok {
			job.ExecSlots, err = parseExecHost(execHost)
			if err != nil {
				return nil, err
			}
		}

		if walltime, ok := ent.attrs["resources_used.walltime"]; ok {
			job.Walltime, err = parseClock(walltime)
			if err != nil {
				return nil, err
			}
		}

		if cputime, ok := ent.attrs["resources_used.cput"]; ok {
			job.CPUTime, err = parseClock(cputime)
			if err != nil {
				return nil, err
			}
		}

		jobs = append(jobs, job)
	}

	return jobs, err
}

// parseExecHost parses s as an exec_host attribute.
//
// exec_host  = host_slots *( "+" host_slots )
// host_slots = host "/" slots
// host       = string
// slots      = slot_range *( "," slot_range )
// slot_range = slot [ "-" slot ]
// slot       = int
//
func parseExecHost(s string) ([]Slot, error) {
	r := []Slot{}

	for _, hostSlots := range strings.Split(s, "+") {
		host, slots := splitOnce(hostSlots, "/")

		for _, srs := range strings.Split(slots, ",") {
			first, last := splitOnce(srs, "-")

			if last == "" {
				last = first
			}

			i, err := strconv.Atoi(first)
			if err != nil {
				return nil, err
			}

			j, err := strconv.Atoi(last)
			if err != nil {
				return nil, err
			}

			for ; i <= j; i++ {
				r = append(r, Slot{Node: host, Index: i})
			}
		}
	}

	return r, nil
}

// parseClock parses a time string of the form [[hh:]mm:]ss and returns the time
// represented by the string in seconds.
func parseClock(s string) (int, error) {
	clock := 0

	for s != "" {
		component, rest := splitOnce(s, ":")

		n, err := strconv.Atoi(component)
		if err != nil {
			return 0, err
		}

		clock *= 60
		clock += int(n)

		s = rest
	}

	return clock, nil
}

// splitOnce splits s at the first occurrence of sep.
func splitOnce(s, sep string) (string, string) {
	n := strings.Index(s, sep)
	if n == -1 {
		return s, ""
	}
	return s[:n], s[n+len(sep):]
}

// An entity holds status of either a job, node or queue.
type entity struct {
	name  string
	attrs map[string]string
}

// queryEntity sends a status request of some entity to the server and returns
// the response as an array of entity objects.
func queryEntity(conn Conn, fun int) ([]entity, error) {

	// Request (See torque: src/lib/Libifl/PBSD_status2.c)
	//
	// request   = type version fun user id attr_list ext
	// type      = int
	// version   = int
	// fun       = int
	// user      = string
	// id        = string
	// attr_list = count *( ... )
	// ext       = "0" / "1" ...

	conn.WriteInt(pbsBatchProtType)
	conn.WriteInt(pbsBatchProtVer)
	conn.WriteInt(int64(fun))
	conn.WriteString(conn.User())
	conn.WriteString("")
	conn.WriteInt(0)
	conn.WriteInt(0)

	if err := conn.Flush(); err != nil {
		return nil, err
	}

	// Response (See torque: src/lib/Libifl/enc_reply.c)
	//
	// response = header job_list
	// job_list = count *( type name attr_list )
	// type     = int
	// name     = string

	choice, err := readResponseHeader(conn)
	if err != nil {
		return nil, err
	}

	if choice != batchReplyChoiceStatus {
		return nil, fmt.Errorf("unrecognized choice=%d", choice)
	}

	count, err := conn.ReadInt()
	if err != nil {
		return nil, err
	}

	entities := []entity{}

	for i := 0; i < int(count); i++ {
		_, err := conn.ReadInt() // entity type
		if err != nil {
			return nil, err
		}

		name, err := conn.ReadString()
		if err != nil {
			return nil, err
		}

		attrs, err := readAttrList(conn)
		if err != nil {
			return nil, err
		}

		entities = append(entities, entity{
			name:  name,
			attrs: attrs,
		})
	}

	return entities, nil
}

// readResponseHeader reads and validates response header from r and returns
// payload type (called "choice").
func readResponseHeader(conn Conn) (int, error) {

	// header   = type version errc aux_errc choice
	// type     = int
	// version  = int
	// errc     = int
	// aux_errc = int
	// choice   = int

	resType, err := conn.ReadInt()
	if err != nil {
		return 0, err
	}

	resVer, err := conn.ReadInt()
	if err != nil {
		return 0, err
	}

	if resType != pbsBatchProtType || resVer != pbsBatchProtVer {
		return 0, fmt.Errorf("unrecognized protocol: type=%d ver=%d", resType, resVer)
	}

	resCode, err := conn.ReadInt()
	if err != nil {
		return 0, err
	}

	resAux, err := conn.ReadInt()
	if err != nil {
		return 0, err
	}

	if resCode != 0 {
		return 0, fmt.Errorf("code=%d aux=%d", resCode, resAux)
	}

	choice, err := conn.ReadInt()
	if err != nil {
		return 0, err
	}

	return int(choice), nil
}

// readAttrList reads an attribute list from r and returns the attributes as a
// map. Resource subkeys, if any, are concatenated to main keys with delimiter
// ".".
func readAttrList(conn Conn) (map[string]string, error) {

	// attr_list = count *( key subkey value op )
	// count     = int
	// key       = string
	// subkey    = "0" / "1" string
	// value     = string
	// op        = int

	count, err := conn.ReadInt()
	if err != nil {
		return nil, err
	}

	attrs := map[string]string{}

	for i := 0; i < int(count); i++ {
		if _, err := conn.ReadInt(); err != nil {
			return nil, err
		}

		name, err := conn.ReadString()
		if err != nil {
			return nil, err
		}

		hasRes, err := conn.ReadInt()
		if err != nil {
			return nil, err
		}

		if hasRes != 0 {
			res, err := conn.ReadString()
			if err != nil {
				return nil, err
			}
			name += "." + res
		}

		value, err := conn.ReadString()
		if err != nil {
			return nil, err
		}

		if _, err := conn.ReadInt(); err != nil {
			return nil, err
		}

		attrs[name] = value
	}

	return attrs, nil
}
