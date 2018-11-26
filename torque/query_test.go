package torque

import (
	"reflect"
	"testing"
)

type mockConn struct {
	response []interface{}
}

func (c *mockConn) User() string {
	return ""
}

func (c *mockConn) ReadInt() (int64, error) {
	n := int64(c.response[0].(int))
	c.response = c.response[1:]
	return n, nil
}

func (c *mockConn) ReadString() (string, error) {
	s := c.response[0].(string)
	c.response = c.response[1:]
	return s, nil
}

func (c *mockConn) WriteInt(n int64) error {
	return nil
}

func (c *mockConn) WriteString(s string) error {
	return nil
}

func (c *mockConn) Flush() error {
	return nil
}

func (c *mockConn) Close() error {
	return nil
}

func Test_QueryNodes_ParsesServerResponse(t *testing.T) {
	conn := &mockConn{[]interface{}{
		2, 2, 0, 0, 6, 2,

		-1, "foo", 2,
		-1, "state", 0, "free", 0,
		-1, "np", 0, "10", 0,

		-1, "bar", 2,
		-1, "state", 0, "down", 0,
		-1, "np", 0, "20", 0,
	}}

	expected := []Node{
		{
			Name:      "foo",
			State:     "free",
			SlotCount: 10,
		},
		{
			Name:      "bar",
			State:     "down",
			SlotCount: 20,
		},
	}

	actual, err := QueryNodes(conn)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(actual) != len(expected) {
		t.Fatalf("unexpecgted node count: got %d, want %d", len(actual), len(expected))
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("unexpected result: got %v, want %v", actual, expected)
	}
}

func Test_QueryJobs_ParsesServerResponse(t *testing.T) {
	conn := &mockConn{[]interface{}{
		2, 2, 0, 0, 6, 2,

		-1, "101", 6,
		-1, "Job_Name", 0, "foo", 0,
		-1, "Job_Owner", 0, "alice@example.com", 0,
		-1, "job_state", 0, "R", 0,
		-1, "exec_host", 0, "node01/1,5-6+node02/3", 0,
		-1, "resources_used", 1, "walltime", "12:34:56", 0,
		-1, "resources_used", 1, "cput", "7:08:09", 0,

		-1, "102", 3,
		-1, "Job_Name", 0, "bar", 0,
		-1, "Job_Owner", 0, "bob@example.com", 0,
		-1, "job_state", 0, "Q", 0,
	}}

	expected := []Job{
		{
			ID:       "101",
			Name:     "foo",
			Owner:    "alice@example.com",
			State:    "R",
			Walltime: (12*60+34)*60 + 56,
			CPUTime:  (7*60+8)*60 + 9,
			ExecSlots: []Slot{
				{"node01", 1},
				{"node01", 5},
				{"node01", 6},
				{"node02", 3},
			},
		},
		{
			ID:    "102",
			Name:  "bar",
			Owner: "bob@example.com",
			State: "Q",
		},
	}

	actual, err := QueryJobs(conn)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(actual) != len(expected) {
		t.Fatalf("unexpecgted job count: got %d, want %d", len(actual), len(expected))
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("unexpected result: got %v, want %v", actual, expected)
	}
}
