package raft

import (
	"bytes"
	"sync"
	"testing"
	"time"
)

// test take and send snapshot
func TestTakeAndSendSnapshot(t *testing.T) {
	var mutex sync.Mutex
	names := []string{"1", "2", "3"}
	servers := map[string]*Server{}
	transporter := &testTransporter{}
	transporter.sendVoteRequestFunc = func(server *Server, peer *Peer, req *RequestVoteRequest) (*RequestVoteResponse, error) {
		mutex.Lock()
		s := servers[peer.name]
		mutex.Unlock()
		resp, err := s.RequestVote(req)
		return resp, err
	}
	transporter.sendAppendEntriesRequestFunc = func(server *Server, peer *Peer, req *AppendEntriesRequest) (*AppendEntriesResponse, error) {
		mutex.Lock()
		s := servers[peer.name]
		mutex.Unlock()
		resp, err := s.AppendEntries(req)
		return resp, err
	}

	transporter.sendSnapshotRequestFunc = func(server *Server, peer *Peer, req *SnapshotRequest) (*SnapshotResponse, error) {
		mutex.Lock()
		s := servers[peer.name]
		mutex.Unlock()
		resp, err := s.SnapshotRecovery(req)
		return resp, err
	}

	stateMachine := &testStateMachine{}

	stateMachine.saveFunc = func() ([]byte, error) {
		return []byte{0x8}, nil
	}

	stateMachine.recoveryFunc = func(state []byte) error {
		return nil
	}

	var leader *Server
	for _, name := range names {
		server := newTestServer(name, transporter)
		server.stateMachine = stateMachine
		server.SetElectionTimeout(testElectionTimeout)
		server.SetHeartbeatTimeout(testHeartbeatTimeout)
		if err := server.Start(); err != nil {
			t.Fatalf("Unable to start server[%s]: %v", name, err)
		}
		defer server.Stop()
		if name == "1" {
			leader = server
			if err := server.Initialize(); err != nil {
				t.Fatalf("Unable to initialize server[%s]: %v", name, err)
			}
		}
		if err := leader.Do(&joinCommand{Name: name}); err != nil {
			t.Fatalf("Unable to join server[%s]: %v", name, err)
		}

		mutex.Lock()
		servers[name] = server
		mutex.Unlock()
	}
	time.Sleep(100 * time.Millisecond)

	// Commit single entry.
	if err := leader.Do(&TestCommand1{"foo", 10}); err != nil {
		t.Fatal(err)
	}

	// Take a snapshot.
	leader.takeSnapshot()

	if count := len(leader.log.entries); count != 0 {
		t.Fatalf("Invalid logLen [Len=%v]", count)
	}
	if leader.log.startIndex != 4 || leader.log.startTerm != 1 {
		t.Fatalf("Invalid log info [StartIndex=%v, StartTERM=%v]", leader.log.startIndex, leader.log.startTerm)
	}

	// Send snapshot to a new node
	newServer := newTestServer("4", transporter)
	newServer.stateMachine = stateMachine
	if err := newServer.Start(); err != nil {
		t.Fatalf("Unable to start server[4]: %v", err)
	}
	defer newServer.Stop()
	if err := leader.Do(&joinCommand{Name: "4"}); err != nil {
		t.Fatalf("Unable to join server[4]: %v", err)
	}
	mutex.Lock()
	servers["4"] = newServer
	mutex.Unlock()

	// wait for heartbeat :P
	time.Sleep(100 * time.Millisecond)

	if leader.log.startIndex != 4 || leader.log.startTerm != 1 {
		t.Fatalf("Invalid log info [StartIndex=%v, StartTERM=%v]", leader.log.startIndex, leader.log.startTerm)
	}
	time.Sleep(100 * time.Millisecond)
}

func TestStartFormSnapshot(t *testing.T) {
	server := newTestServer("1", &testTransporter{})

	stateMachine := &testStateMachine{}
	stateMachine.saveFunc = func() ([]byte, error) {
		return []byte{0x60, 0x61, 0x62, 0x63, 0x64, 0x65}, nil
	}

	stateMachine.recoveryFunc = func(state []byte) error {
		expect := []byte{0x60, 0x61, 0x62, 0x63, 0x64, 0x65}
		if !(bytes.Equal(state, expect)) {
			t.Fatalf("Invalid State [Expcet=%v, Actual=%v]", expect, state)
		}
		return nil
	}
	server.stateMachine = stateMachine
	oldPath := server.path
	server.Start()

	server.Initialize()

	// commit single entry.
	err := server.Do(&TestCommand1{"foo", 10})

	if err != nil {
		t.Fatal(err)
	}

	server.takeSnapshot()

	logLen := len(server.log.entries)

	if logLen != 0 {
		t.Fatalf("Invalid logLen [Len=%v]", logLen)
	}

	if server.log.startIndex != 1 || server.log.startTerm != 1 {
		t.Fatalf("Invalid log info [StartIndex=%v, StartTERM=%v]",
			server.log.startIndex, server.log.startTerm)
	}

	server.Stop()

	server = newTestServer("1", &testTransporter{})
	server.stateMachine = stateMachine
	// reset the oldPath
	server.path = oldPath

	server.Start()

	logLen = len(server.log.entries)

	if logLen != 0 {
		t.Fatalf("Invalid logLen [Len=%v]", logLen)
	}

	if index, term := server.log.CommitInfo(); !(index == 0 && term == 0) {
		t.Fatalf("Invalid commit info [IDX=%v, TERM=%v]", index, term)
	}

	if server.log.startIndex != 0 || server.log.startTerm != 0 {
		t.Fatalf("Invalid log info [StartIndex=%v, StartTERM=%v]",
			server.log.startIndex, server.log.startTerm)
	}

	server.LoadSnapshot()
	if server.log.startIndex != 1 || server.log.startTerm != 1 {
		t.Fatalf("Invalid log info [StartIndex=%v, StartTERM=%v]",
			server.log.startIndex, server.log.startTerm)
	}

}
