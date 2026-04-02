package shared

// Entry represents a key-value pair with a logical version number.
// The version increments on every write, so readers can compare
// responses from multiple nodes and pick the freshest one.
type Entry struct {
	Value   string `json:"value"`
	Version int    `json:"version"`
}

// --- Request / Response types for HTTP communication ---

// SetRequest is the body the client sends to set a key.
type SetRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// GetResponse is what the client receives on a successful get.
type GetResponse struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Version int    `json:"version"`
}

// InternalSetRequest is what the Leader (or Coordinator) sends
// to other nodes when replicating a write.
// It includes the version so every replica stores the same version number.
type InternalSetRequest struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Version int    `json:"version"`
}

// InternalGetResponse is what a node returns when the Leader
// queries it during a multi-node read (R > 1).
type InternalGetResponse struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Version int    `json:"version"`
	Found   bool   `json:"found"`
}
