package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

var (
	db        *sql.DB
	config    MasterConfig
	store     = NewStore()
	agentConn = make(map[string]chan Command) // agent_id -> command channel
)

func main() {
	loadConfig()
	initDB()

	// Setup admin user
	adminUser := getEnv("ADMIN_USER", "admin")
	adminPass := getEnv("ADMIN_PASS", "sera-admin-2024")
	setupAdmin(adminUser, adminPass)

	log.Printf("[MASTER] Starting on port %s", config.Port)
	log.Printf("[MASTER] Max storage: %d MB", config.MaxStorageMB)
	log.Printf("[MASTER] AI Provider: %s, Model: %s", config.AI.Provider, config.AI.Model)
	log.Printf("[MASTER] Telegram enabled: %v", config.Telegram.Enabled)

	mux := http.NewServeMux()

	// === Agent API ===
	mux.HandleFunc("/api/agent/register", authMiddleware(handleAgentRegister))
	mux.HandleFunc("/api/agent/heartbeat", authMiddleware(handleAgentHeartbeat))
	mux.HandleFunc("/api/agent/poll", authMiddleware(handleAgentPoll))
	mux.HandleFunc("/api/agent/chunk", authMiddleware(handleAgentChunk))
	mux.HandleFunc("/api/agent/report", authMiddleware(handleAgentReport))
	mux.HandleFunc("/api/agent/scan-result", authMiddleware(handleAgentScanResult))

	// === Login / Auth API ===
	mux.HandleFunc("/api/login", handleLogin)
	mux.HandleFunc("/api/logout", handleLogout)
	mux.HandleFunc("/api/captcha", handleCaptcha)
	mux.HandleFunc("/api/session", handleCheckSession)

	// === Admin API (session-protected) ===
	mux.HandleFunc("/api/agents", sessionMiddleware(handleGetAgents))
	mux.HandleFunc("/api/agents/", sessionMiddleware(handleAgentActions))
	mux.HandleFunc("/api/agents/bulk-delete", sessionMiddleware(handleBulkDeleteAgents))
	mux.HandleFunc("/api/command", sessionMiddleware(handleSendCommand))
	mux.HandleFunc("/api/files", sessionMiddleware(handleGetFiles))
	mux.HandleFunc("/api/files/", sessionMiddleware(handleFileActions))
	mux.HandleFunc("/api/files/select", sessionMiddleware(handleSelectFiles))
	mux.HandleFunc("/api/files/stop-monitoring", sessionMiddleware(handleStopMonitoring))
	mux.HandleFunc("/api/files/bulk-delete", sessionMiddleware(handleBulkDeleteFiles))
	mux.HandleFunc("/api/reports", sessionMiddleware(handleGetReports))
	mux.HandleFunc("/api/ai-logs", sessionMiddleware(handleGetAILogs))
	mux.HandleFunc("/api/storage", sessionMiddleware(handleGetStorage))
	mux.HandleFunc("/api/config/ai", sessionMiddleware(handleAIConfig))
	mux.HandleFunc("/api/config/telegram", sessionMiddleware(handleTelegramConfig))

	// === Dashboard (session-protected) ===
	mux.HandleFunc("/", sessionMiddleware(handleDashboard))
	mux.HandleFunc("/login", handleLoginPage)
	mux.HandleFunc("/health", handleHealth)

	log.Printf("[MASTER] Listening on :%s", config.Port)
	if err := http.ListenAndServe(":"+config.Port, mux); err != nil {
		log.Fatalf("[MASTER] Failed to start: %v", err)
	}
}

func loadConfig() {
	config = MasterConfig{
		Port:         getEnv("PORT", "8080"),
		APIKey:       getEnv("API_KEY", "sera-default-key"),
		MaxStorageMB: getEnvInt("MAX_STORAGE_MB", 500),
		AI: AIConfig{
			Provider:    getEnv("AI_PROVIDER", "ollama"),
			BaseURL:     getEnv("AI_BASE_URL", "http://ollama:11434"),
			Model:       getEnv("AI_MODEL", "qwen2.5:0.5b"),
			APIKey:      getEnv("AI_API_KEY", ""),
			MaxTokens:   getEnvInt("AI_MAX_TOKENS", 512),
			Temperature: 0.3,
			ChunkSize:   getEnvInt("AI_CHUNK_SIZE", 3),
			Timeout:     getEnvInt("AI_TIMEOUT", 120),
		},
		Telegram: TelegramConfig{
			BotToken: getEnv("TG_BOT_TOKEN", ""),
			ChatID:   getEnv("TG_CHAT_ID", ""),
			Enabled:  getEnv("TG_ENABLED", "false") == "true",
		},
	}
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./data/master.db?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("[MASTER] Failed to open DB: %v", err)
	}

	// Ensure data directory exists
	os.MkdirAll("./data", 0755)

	// Create tables
	schema := `
	CREATE TABLE IF NOT EXISTS agents (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		ip TEXT NOT NULL,
		status TEXT DEFAULT 'offline',
		last_heartbeat DATETIME,
		registered_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS commands (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL,
		type TEXT NOT NULL,
		status TEXT DEFAULT 'pending',
		payload TEXT DEFAULT '{}',
		result TEXT DEFAULT '{}',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS files (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL,
		path TEXT NOT NULL,
		size INTEGER DEFAULT 0,
		mod_time TEXT DEFAULT '',
		selected INTEGER DEFAULT 0,
		status TEXT DEFAULT 'pending',
		chunk_size INTEGER DEFAULT 3,
		offset INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS log_chunks (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL,
		file_path TEXT NOT NULL,
		chunk_num INTEGER DEFAULT 0,
		lines TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS ai_reports (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL,
		file_path TEXT NOT NULL,
		chunk_num INTEGER DEFAULT 0,
		needs_action INTEGER DEFAULT 0,
		summary TEXT DEFAULT '',
		severity TEXT DEFAULT 'info',
		details TEXT DEFAULT '',
		sent_to_tg INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS ai_request_logs (
		id TEXT PRIMARY KEY,
		provider TEXT DEFAULT '',
		model TEXT DEFAULT '',
		url TEXT DEFAULT '',
		request TEXT DEFAULT '',
		response TEXT DEFAULT '',
		duration_ms INTEGER DEFAULT 0,
		success INTEGER DEFAULT 1,
		error TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err = db.Exec(schema)
	if err != nil {
		log.Fatalf("[MASTER] Failed to create tables: %v", err)
	}

	// Periodic cleanup of old chunks (keep for max 1 hour)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			cleanupOldChunks()
			// Also cleanup old AI logs (keep last 500)
			db.Exec("DELETE FROM ai_request_logs WHERE id NOT IN (SELECT id FROM ai_request_logs ORDER BY created_at DESC LIMIT 500)")
		}
	}()

	log.Println("[MASTER] Database initialized")
}

func cleanupOldChunks() {
	_, err := db.Exec("DELETE FROM log_chunks WHERE created_at < datetime('now', '-1 hour')")
	if err != nil {
		log.Printf("[MASTER] Cleanup error: %v", err)
	}
}

// ============================================================
// Auth Middleware
// ============================================================

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Agent endpoints use X-API-Key
		key := r.Header.Get("X-API-Key")
		if key == "" {
			// Try query param for dashboard
			key = r.URL.Query().Get("key")
		}
		if key != config.APIKey {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(APIResponse{Error: "unauthorized"})
			return
		}
		next(w, r)
	}
}

// ============================================================
// Agent Handlers
// ============================================================

func handleAgentRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
		IP   string `json:"ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	agentID := uuid.New().String()
	agent := &Agent{
		ID:            agentID,
		Name:          req.Name,
		IP:            req.IP,
		Status:        AgentOnline,
		LastHeartbeat: time.Now(),
		RegisteredAt:  time.Now(),
	}

	_, err := db.Exec(
		"INSERT OR REPLACE INTO agents (id, name, ip, status, last_heartbeat, registered_at) VALUES (?, ?, ?, ?, ?, ?)",
		agent.ID, agent.Name, agent.IP, agent.Status, agent.LastHeartbeat, agent.RegisteredAt,
	)
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}

	log.Printf("[MASTER] Agent registered: %s (%s) from %s", agent.Name, agent.ID, agent.IP)
	jsonResponse(w, APIResponse{Success: true, Data: agent})
}

func handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(
		"UPDATE agents SET last_heartbeat = ?, status = ? WHERE id = ?",
		time.Now(), req.Status, req.AgentID,
	)
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}

	// Check storage before allowing more processing
	storage := getStorageInfo()
	jsonResponse(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"ok":      true,
			"storage": storage,
		},
	})
}

func handleAgentPoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Check storage - don't send commands if full
	storage := getStorageInfo()
	if storage.IsFull {
		jsonResponse(w, APIResponse{
			Success: true,
			Data:    PollResponse{Commands: []Command{}},
		})
		return
	}

	// Get pending commands for this agent
	rows, err := db.Query(
		"SELECT id, agent_id, type, status, payload, result, created_at, completed_at FROM commands WHERE agent_id = ? AND status = 'pending' ORDER BY created_at ASC LIMIT 10",
		req.AgentID,
	)
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var commands []Command
	for rows.Next() {
		var cmd Command
		var completedAt sql.NullTime
		err := rows.Scan(&cmd.ID, &cmd.AgentID, &cmd.Type, &cmd.Status, &cmd.Payload, &cmd.Result, &cmd.CreatedAt, &completedAt)
		if err != nil {
			continue
		}
		if completedAt.Valid {
			cmd.CompletedAt = &completedAt.Time
		}
		commands = append(commands, cmd)
	}

	// Mark as accepted
	for _, cmd := range commands {
		db.Exec("UPDATE commands SET status = 'accepted' WHERE id = ?", cmd.ID)
	}

	if commands == nil {
		commands = []Command{}
	}

	jsonResponse(w, APIResponse{Success: true, Data: PollResponse{Commands: commands}})
}

func handleAgentChunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check storage first
	storage := getStorageInfo()
	if storage.IsFull {
		jsonError(w, "storage full, cannot accept more logs", http.StatusServiceUnavailable)
		return
	}

	var req SendChunkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	chunkID := uuid.New().String()
	_, err := db.Exec(
		"INSERT INTO log_chunks (id, agent_id, file_path, chunk_num, lines, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		chunkID, req.AgentID, req.FilePath, req.ChunkNum, req.Lines, time.Now(),
	)
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}

	log.Printf("[MASTER] Received chunk %d for %s from agent %s (%d bytes)",
		req.ChunkNum, req.FilePath, req.AgentID, len(req.Lines))

	// Process the chunk with AI asynchronously
	go processChunkWithAI(chunkID, req)

	jsonResponse(w, APIResponse{Success: true, Data: map[string]string{"chunk_id": chunkID}})
}

func handleAgentScanResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID   string     `json:"agent_id"`
		Result    ScanResult `json:"result"`
		CommandID string     `json:"command_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Store found files in DB (deduplicate by agent_id + path)
	for _, f := range req.Result.Files {
		var existingID string
		err := db.QueryRow("SELECT id FROM files WHERE agent_id = ? AND path = ?", req.AgentID, f.Path).Scan(&existingID)
		if err == nil {
			db.Exec("UPDATE files SET size = ?, mod_time = ? WHERE id = ?", f.Size, f.ModTime, existingID)
		} else {
			fileID := uuid.New().String()
			db.Exec(
				"INSERT INTO files (id, agent_id, path, size, mod_time, selected, status, chunk_size, offset) VALUES (?, ?, ?, ?, ?, 0, 'pending', ?, 0)",
				fileID, req.AgentID, f.Path, f.Size, f.ModTime, config.AI.ChunkSize,
			)
		}
	}

	// Mark command completed
	if req.CommandID != "" {
		now := time.Now()
		db.Exec("UPDATE commands SET status = 'completed', completed_at = ?, result = ? WHERE id = ?",
			now, marshalJSON(req.Result), req.CommandID)
	}

	// Reset agent status to online after scan completes
	db.Exec("UPDATE agents SET status = 'online' WHERE id = ?", req.AgentID)

	log.Printf("[MASTER] Scan result from %s: %d files found", req.AgentID, req.Result.Total)
	jsonResponse(w, APIResponse{Success: true})
}

func handleFileActions(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/files/"):]
	if id == "" {
		jsonError(w, "file id required", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodDelete {
		db.Exec("DELETE FROM files WHERE id = ?", id)
		log.Printf("[MASTER] File deleted: %s", id)
		jsonResponse(w, APIResponse{Success: true})
		return
	}

	jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
}

func handleAgentReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Agent can send direct reports (e.g., errors)
	var report AIReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	report.ID = uuid.New().String()
	report.CreatedAt = time.Now()

	db.Exec(
		"INSERT INTO ai_reports (id, agent_id, file_path, chunk_num, needs_action, summary, severity, details, sent_to_tg, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		report.ID, report.AgentID, report.FilePath, report.ChunkNum, boolToInt(report.NeedsAction),
		report.Summary, report.Severity, report.Details, 0, report.CreatedAt,
	)

	if report.NeedsAction && config.Telegram.Enabled {
		go sendTelegramReport(report)
	}

	jsonResponse(w, APIResponse{Success: true})
}

// ============================================================
// Admin API Handlers
// ============================================================

func handleGetAgents(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, ip, status, last_heartbeat, registered_at FROM agents ORDER BY registered_at DESC")
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		var a Agent
		rows.Scan(&a.ID, &a.Name, &a.IP, &a.Status, &a.LastHeartbeat, &a.RegisteredAt)
		agents = append(agents, a)
	}
	if agents == nil {
		agents = []Agent{}
	}

	// Mark agents offline if no heartbeat in 60s, or reset scanning if stuck > 5min
	for i := range agents {
		hbAge := time.Since(agents[i].LastHeartbeat)
		if hbAge > 60*time.Second {
			agents[i].Status = AgentOffline
			db.Exec("UPDATE agents SET status = 'offline' WHERE id = ?", agents[i].ID)
		} else if agents[i].Status == AgentScanning && hbAge > 5*time.Minute {
			agents[i].Status = AgentOnline
			db.Exec("UPDATE agents SET status = 'online' WHERE id = ?", agents[i].ID)
		}
	}

	jsonResponse(w, APIResponse{Success: true, Data: agents})
}

func handleAgentActions(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/agents/"):]
	if id == "" {
		jsonError(w, "agent id required", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodGet {
		var a Agent
		err := db.QueryRow("SELECT id, name, ip, status, last_heartbeat, registered_at FROM agents WHERE id = ?", id).
			Scan(&a.ID, &a.Name, &a.IP, &a.Status, &a.LastHeartbeat, &a.RegisteredAt)
		if err != nil {
			jsonError(w, "agent not found", http.StatusNotFound)
			return
		}
		jsonResponse(w, APIResponse{Success: true, Data: a})
		return
	}

	if r.Method == http.MethodDelete {
		// Delete agent and its related data
		db.Exec("DELETE FROM agents WHERE id = ?", id)
		db.Exec("DELETE FROM files WHERE agent_id = ?", id)
		db.Exec("DELETE FROM commands WHERE agent_id = ?", id)
		db.Exec("DELETE FROM log_chunks WHERE agent_id = ?", id)
		db.Exec("DELETE FROM ai_reports WHERE agent_id = ?", id)
		log.Printf("[MASTER] Agent deleted: %s", id)
		jsonResponse(w, APIResponse{Success: true})
		return
	}

	jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
}

func handleSendCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID string      `json:"agent_id"`
		Type    CommandType `json:"type"`
		Payload string      `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	cmdID := uuid.New().String()
	_, err := db.Exec(
		"INSERT INTO commands (id, agent_id, type, status, payload, created_at) VALUES (?, ?, ?, 'pending', ?, ?)",
		cmdID, req.AgentID, req.Type, req.Payload, time.Now(),
	)
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}

	log.Printf("[MASTER] Command %s sent to agent %s: %s", cmdID, req.AgentID, req.Type)
	jsonResponse(w, APIResponse{Success: true, Data: map[string]string{"command_id": cmdID}})
}

func handleBulkDeleteAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	for _, id := range req.IDs {
		db.Exec("DELETE FROM agents WHERE id = ?", id)
		db.Exec("DELETE FROM files WHERE agent_id = ?", id)
		db.Exec("DELETE FROM commands WHERE agent_id = ?", id)
		db.Exec("DELETE FROM log_chunks WHERE agent_id = ?", id)
		db.Exec("DELETE FROM ai_reports WHERE agent_id = ?", id)
	}
	log.Printf("[MASTER] Bulk deleted %d agents", len(req.IDs))
	jsonResponse(w, APIResponse{Success: true})
}

func handleStopMonitoring(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Reset all monitoring/processing files back to pending
	db.Exec("UPDATE files SET status = 'pending', selected = 0, offset = 0 WHERE status IN ('monitoring', 'processing')")

	// Send stop_all command to all agents with pending commands
	rows, err := db.Query("SELECT DISTINCT agent_id FROM files WHERE status IN ('monitoring', 'processing') OR agent_id IN (SELECT DISTINCT agent_id FROM commands WHERE type = 'start_monitor')")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var agentID string
			rows.Scan(&agentID)
			cmdID := uuid.New().String()
			db.Exec(
				"INSERT INTO commands (id, agent_id, type, status, payload, created_at) VALUES (?, ?, ?, 'pending', '{}', ?)",
				cmdID, agentID, CmdStopAll, time.Now(),
			)
		}
	}

	log.Printf("[MASTER] Stop monitoring requested for all agents")
	jsonResponse(w, APIResponse{Success: true})
}

func handleBulkDeleteFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	for _, id := range req.IDs {
		db.Exec("DELETE FROM files WHERE id = ?", id)
	}
	log.Printf("[MASTER] Bulk deleted %d files", len(req.IDs))
	jsonResponse(w, APIResponse{Success: true})
}

func handleGetAILogs(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		db.Exec("DELETE FROM ai_request_logs")
		log.Println("[MASTER] AI request logs cleared")
		jsonResponse(w, APIResponse{Success: true})
		return
	}

	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "50"
	}

	rows, err := db.Query(
		"SELECT id, provider, model, url, request, response, duration_ms, success, error, created_at FROM ai_request_logs ORDER BY created_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []AIRequestLog
	for rows.Next() {
		var l AIRequestLog
		var successInt int
		rows.Scan(&l.ID, &l.Provider, &l.Model, &l.URL, &l.Request, &l.Response, &l.DurationMs, &successInt, &l.Error, &l.CreatedAt)
		l.Success = successInt == 1
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []AIRequestLog{}
	}

	jsonResponse(w, APIResponse{Success: true, Data: logs})
}

func handleGetFiles(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent_id")
	status := r.URL.Query().Get("status")

	query := "SELECT id, agent_id, path, size, mod_time, selected, status, chunk_size, offset FROM files WHERE 1=1"
	args := []interface{}{}

	if agentID != "" {
		query += " AND agent_id = ?"
		args = append(args, agentID)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY path"

	rows, err := db.Query(query, args...)
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type FileWithProgress struct {
		FileEntry
		ProcessedChunks int `json:"processed_chunks"`
		TotalReports    int `json:"total_reports"`
		ActionReports   int `json:"action_reports"`
	}

	var files []FileWithProgress
	for rows.Next() {
		var f FileEntry
		var sel int
		rows.Scan(&f.ID, &f.AgentID, &f.Path, &f.Size, &f.ModTime, &sel, &f.Status, &f.ChunkSize, &f.Offset)
		f.Selected = sel == 1

		fwp := FileWithProgress{FileEntry: f}
		db.QueryRow("SELECT COUNT(*) FROM ai_reports WHERE agent_id = ? AND file_path = ?", f.AgentID, f.Path).Scan(&fwp.TotalReports)
		db.QueryRow("SELECT COUNT(*) FROM ai_reports WHERE agent_id = ? AND file_path = ? AND needs_action = 1", f.AgentID, f.Path).Scan(&fwp.ActionReports)
		if f.ChunkSize > 0 {
			fwp.ProcessedChunks = int(f.Offset) / f.ChunkSize
		}

		files = append(files, fwp)
	}
	if files == nil {
		files = []FileWithProgress{}
	}

	jsonResponse(w, APIResponse{Success: true, Data: files})
}

func handleSelectFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FileIDs   []string `json:"file_ids"`
		SelectAll bool     `json:"select_all"`
		AgentID   string   `json:"agent_id"`
		ChunkSize int      `json:"chunk_size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	chunkSize := req.ChunkSize
	if chunkSize <= 0 {
		chunkSize = config.AI.ChunkSize
	}

	if req.SelectAll {
		db.Exec("UPDATE files SET selected = 1, chunk_size = ?, status = 'monitoring' WHERE agent_id = ?", chunkSize, req.AgentID)
	} else {
		for _, fid := range req.FileIDs {
			db.Exec("UPDATE files SET selected = 1, chunk_size = ?, status = 'monitoring' WHERE id = ?", chunkSize, fid)
		}
	}

	// Send start_monitor command to agent
	// Get all selected file paths
	rows, _ := db.Query("SELECT path FROM files WHERE selected = 1 AND agent_id = ? AND status = 'monitoring'", req.AgentID)
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		rows.Scan(&p)
		paths = append(paths, p)
	}

	if len(paths) > 0 {
		payload := marshalJSON(MonitorPayload{Files: paths, ChunkSize: chunkSize})
		cmdID := uuid.New().String()
		db.Exec(
			"INSERT INTO commands (id, agent_id, type, status, payload, created_at) VALUES (?, ?, ?, 'pending', ?, ?)",
			cmdID, req.AgentID, CmdStartMonitor, payload, time.Now(),
		)
		log.Printf("[MASTER] Start monitor command sent for %d files", len(paths))
	}

	jsonResponse(w, APIResponse{Success: true})
}

func handleGetReports(w http.ResponseWriter, r *http.Request) {
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "100"
	}

	rows, err := db.Query(
		"SELECT id, agent_id, file_path, chunk_num, needs_action, summary, severity, details, sent_to_tg, created_at FROM ai_reports ORDER BY created_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var reports []AIReport
	for rows.Next() {
		var rpt AIReport
		var needsAction, sentToTG int
		rows.Scan(&rpt.ID, &rpt.AgentID, &rpt.FilePath, &rpt.ChunkNum, &needsAction,
			&rpt.Summary, &rpt.Severity, &rpt.Details, &sentToTG, &rpt.CreatedAt)
		rpt.NeedsAction = needsAction == 1
		rpt.SentToTG = sentToTG == 1
		reports = append(reports, rpt)
	}
	if reports == nil {
		reports = []AIReport{}
	}

	jsonResponse(w, APIResponse{Success: true, Data: reports})
}

func handleGetStorage(w http.ResponseWriter, r *http.Request) {
	storage := getStorageInfo()
	jsonResponse(w, APIResponse{Success: true, Data: storage})
}

func handleAIConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		jsonResponse(w, APIResponse{Success: true, Data: config.AI})
		return
	}
	if r.Method == http.MethodPut {
		var ai AIConfig
		if err := json.NewDecoder(r.Body).Decode(&ai); err != nil {
			jsonError(w, "invalid request", http.StatusBadRequest)
			return
		}
		config.AI = ai
		jsonResponse(w, APIResponse{Success: true, Data: config.AI})
		return
	}
	jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
}

func handleTelegramConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		jsonResponse(w, APIResponse{Success: true, Data: config.Telegram})
		return
	}
	if r.Method == http.MethodPut {
		var tg TelegramConfig
		if err := json.NewDecoder(r.Body).Decode(&tg); err != nil {
			jsonError(w, "invalid request", http.StatusBadRequest)
			return
		}
		config.Telegram = tg
		jsonResponse(w, APIResponse{Success: true, Data: config.Telegram})
		return
	}
	jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
}

// ============================================================
// Health & Dashboard
// ============================================================

func handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, APIResponse{Success: true, Data: map[string]string{"status": "ok"}})
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, dashboardHTML)
}

func handleLoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, loginPageHTML)
}

// ============================================================
// Helpers
// ============================================================

func getStorageInfo() StorageInfo {
	var sizeMB float64
	db.QueryRow("SELECT COALESCE(SUM(length(lines)), 0) / (1024.0 * 1024.0) FROM log_chunks").Scan(&sizeMB)

	// Also count report sizes
	var reportMB float64
	db.QueryRow("SELECT COALESCE(SUM(length(details) + length(summary)), 0) / (1024.0 * 1024.0) FROM ai_reports").Scan(&reportMB)

	totalMB := sizeMB + reportMB
	maxMB := float64(config.MaxStorageMB)
	pct := (totalMB / maxMB) * 100

	return StorageInfo{
		UsedMB:     totalMB,
		MaxMB:      config.MaxStorageMB,
		Percentage: pct,
		IsFull:     totalMB >= maxMB,
	}
}

func jsonResponse(w http.ResponseWriter, resp APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(APIResponse{Error: msg})
}

func marshalJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}
