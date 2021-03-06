package raft

// The request sent to a server to start from the snapshot.
type SnapshotRequest struct {
	LeaderName string `json:"leaderName"`
	LastIndex  uint64 `json:"lastTerm"`
	LastTerm   uint64 `json:"lastIndex"`
	State      []byte `json:"state"`
}

// The response returned from a server appending entries to the log.
type SnapshotResponse struct {
	Term        uint64 `json:"term"`
	Success     bool   `json:"success"`
	CommitIndex uint64 `json:"commitIndex"`
}

//------------------------------------------------------------------------------
//
// Constructors
//
//------------------------------------------------------------------------------

// Creates a new Snapshot request.
func NewSnapshotRequest(leaderName string, snapshot *Snapshot) *SnapshotRequest {
	return &SnapshotRequest{
		LeaderName: leaderName,
		LastIndex:  snapshot.lastIndex,
		LastTerm:   snapshot.lastTerm,
		State:      snapshot.state,
	}
}

// Creates a new Snapshot response.
func NewSnapshotResponse(term uint64, success bool, commitIndex uint64) *SnapshotResponse {
	return &SnapshotResponse{
		Term:        term,
		Success:     success,
		CommitIndex: commitIndex,
	}
}
