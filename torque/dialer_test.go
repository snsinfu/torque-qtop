package torque

import (
	"fmt"
	"net"
	"os"
	"testing"
)

func Test_Dialer_GetActiveServer_MakesQuery(t *testing.T) {
	const (
		authSock = "test-torque-Dialer-GetActiveServer.socket"
		testHost = "torque.example.com"
		testPort = 12345
	)

	// Mock auth server
	ln, err := net.Listen("unix", authSock)
	if err != nil {
		t.Fatalf("Listen failed: %s", err)
	}
	defer os.Remove(authSock)
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			t.Fatalf("Accept failed: %s", err)
		}
		defer conn.Close()

		response := fmt.Sprintf("0|%d|%s|%d|", len(testHost), testHost, testPort)

		if _, err := conn.Write([]byte(response)); err != nil {
			t.Fatalf("Write failed: %s", err)
		}
	}()

	// Test dialer
	expected := fmt.Sprintf("%s:%d", testHost, testPort)

	dialer := Dialer{AuthAddr: authSock}
	actual, err := dialer.GetActiveServer()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if actual != expected {
		t.Errorf("unexpected result: got %q, want %q", actual, expected)
	}
}

func Test_Dialer_Dial_MakesAuthRequest(t *testing.T) {
	const (
		authSock   = "test-torque-Dialer-Dial.socket"
		serverAddr = "localhost:12345"
	)

	// Mock auth server
	auth, err := net.Listen("unix", authSock)
	if err != nil {
		t.Fatalf("Listen failed: %s", err)
	}
	defer os.Remove(authSock)
	defer auth.Close()

	go func() {
		conn, err := auth.Accept()
		if err != nil {
			t.Fatalf("Accept failed: %s", err)
		}
		defer conn.Close()

		buf := make([]byte, 256)
		if _, err := conn.Read(buf); err != nil {
			t.Fatalf("Read failed: %s", err)
		}

		response := "0|"

		if _, err := conn.Write([]byte(response)); err != nil {
			t.Fatalf("Write failed: %s", err)
		}
	}()

	// Mock PBS server
	pbs, err := net.Listen("tcp", serverAddr)
	if err != nil {
		t.Fatalf("Listen failed: %s", err)
	}
	defer pbs.Close()

	go func() {
		conn, err := pbs.Accept()
		if err != nil {
			t.Fatalf("Accept failed: %s", err)
		}
		conn.Close()
	}()

	// Test dialer
	dialer := Dialer{AuthAddr: authSock}

	conn, err := dialer.Dial(serverAddr)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	defer conn.Close()
}
