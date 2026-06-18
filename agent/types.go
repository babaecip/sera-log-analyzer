package main

// Agent-specific types (minimal, only what the agent needs)

type AgentConfig struct {
	MasterURL    string `json:"master_url"`
	MasterKey    string `json:"master_key"`
	AgentName    string `json:"agent_name"`
	AgentID      string `json:"agent_id"`
	PollInterval int    `json:"poll_interval"`
	ScanRoots    string `json:"scan_roots"`
	Extensions   string `json:"extensions"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type Command struct {
	ID      string `json:"id"`
	AgentID string `json:"agent_id"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Payload string `json:"payload"`
	Result  string `json:"result"`
}

type PollResponse struct {
	Commands []Command `json:"commands"`
}
