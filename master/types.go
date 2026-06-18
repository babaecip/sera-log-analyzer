package main

import (
	"sync"
	"time"
)

// ============================================================
// Enums
// ============================================================

type AgentStatus string

const (
	AgentOnline   AgentStatus = "online"
	AgentOffline  AgentStatus = "offline"
	AgentScanning AgentStatus = "scanning"
)

type CommandType string

const (
	CmdScanFiles    CommandType = "scan_files"
	CmdStartMonitor CommandType = "start_monitor"
	CmdStopMonitor  CommandType = "stop_monitor"
	CmdProcessChunk CommandType = "process_chunk"
	CmdStopAll      CommandType = "stop_all"
)

type CommandStatus string

const (
	CmdPending    CommandStatus = "pending"
	CmdAccepted   CommandStatus = "accepted"
	CmdProcessing CommandStatus = "processing"
	CmdCompleted  CommandStatus = "completed"
	CmdFailed     CommandStatus = "failed"
)

type FileStatus string

const (
	FilePending    FileStatus = "pending"
	FileMonitoring FileStatus = "monitoring"
	FileProcessing FileStatus = "processing"
	FileDone       FileStatus = "done"
	FileError      FileStatus = "error"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// ============================================================
// Models
// ============================================================

type Agent struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	IP            string      `json:"ip"`
	Status        AgentStatus `json:"status"`
	LastHeartbeat time.Time   `json:"last_heartbeat"`
	RegisteredAt  time.Time   `json:"registered_at"`
}

type Command struct {
	ID          string        `json:"id"`
	AgentID     string        `json:"agent_id"`
	Type        CommandType   `json:"type"`
	Status      CommandStatus `json:"status"`
	Payload     string        `json:"payload"`
	Result      string        `json:"result"`
	CreatedAt   time.Time     `json:"created_at"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
}

type FileEntry struct {
	ID        string     `json:"id"`
	AgentID   string     `json:"agent_id"`
	Path      string     `json:"path"`
	Size      int64      `json:"size"`
	ModTime   string     `json:"mod_time"`
	Selected  bool       `json:"selected"`
	Status    FileStatus `json:"status"`
	ChunkSize int        `json:"chunk_size"`
	Offset    int64      `json:"offset"`
}

type AIReport struct {
	ID          string    `json:"id"`
	AgentID     string    `json:"agent_id"`
	FilePath    string    `json:"file_path"`
	ChunkNum    int       `json:"chunk_num"`
	NeedsAction bool      `json:"needs_action"`
	Summary     string    `json:"summary"`
	Severity    Severity  `json:"severity"`
	Details     string    `json:"details"`
	SentToTG    bool      `json:"sent_to_tg"`
	CreatedAt   time.Time `json:"created_at"`
}

// ============================================================
// Payloads
// ============================================================

type ScanPayload struct {
	Extensions []string `json:"extensions"`
	RootPaths  []string `json:"root_paths"`
	MaxDepth   int      `json:"max_depth"`
}

type ScanResult struct {
	Files     []FileEntry `json:"files"`
	Total     int         `json:"total"`
	RootPaths []string    `json:"root_paths"`
}

type ProcessChunkPayload struct {
	FilePath  string `json:"file_path"`
	ChunkSize int    `json:"chunk_size"`
	Offset    int64  `json:"offset"`
}

type MonitorPayload struct {
	Files     []string `json:"files"`
	ChunkSize int      `json:"chunk_size"`
}

// ============================================================
// Configs
// ============================================================

type AIConfig struct {
	Provider    string  `json:"provider"`
	BaseURL     string  `json:"base_url"`
	Model       string  `json:"model"`
	APIKey      string  `json:"api_key,omitempty"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	ChunkSize   int     `json:"chunk_size"`
}

type TelegramConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
	Enabled  bool   `json:"enabled"`
}

type MasterConfig struct {
	Port         string         `json:"port"`
	APIKey       string         `json:"api_key"`
	MaxStorageMB int            `json:"max_storage_mb"`
	AI           AIConfig       `json:"ai"`
	Telegram     TelegramConfig `json:"telegram"`
}

// ============================================================
// API Request / Response
// ============================================================

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type PollRequest struct {
	AgentID string `json:"agent_id"`
}

type PollResponse struct {
	Commands []Command `json:"commands"`
}

type HeartbeatRequest struct {
	AgentID string      `json:"agent_id"`
	Status  AgentStatus `json:"status"`
}

type SendChunkRequest struct {
	AgentID  string `json:"agent_id"`
	FilePath string `json:"file_path"`
	ChunkNum int    `json:"chunk_num"`
	Lines    string `json:"lines"`
}

type StorageInfo struct {
	UsedMB     float64 `json:"used_mb"`
	MaxMB      int     `json:"max_mb"`
	Percentage float64 `json:"percentage"`
	IsFull     bool    `json:"is_full"`
}

type AIRequestLog struct {
	ID         string    `json:"id"`
	Provider   string    `json:"provider"`
	Model      string    `json:"model"`
	URL        string    `json:"url"`
	Request    string    `json:"request"`
	Response   string    `json:"response"`
	DurationMs int64     `json:"duration_ms"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Store struct {
	mu       sync.RWMutex
	Agents   map[string]*Agent
	Commands map[string]*Command
}

func NewStore() *Store {
	return &Store{
		Agents:   make(map[string]*Agent),
		Commands: make(map[string]*Command),
	}
}
