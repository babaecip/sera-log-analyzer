package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	config         AgentConfig
	agentID        string
	monitoredFiles = make(map[string]*MonitoredFile)
	mu             sync.RWMutex
)

type MonitoredFile struct {
	Path      string `json:"path"`
	ChunkSize int    `json:"chunk_size"`
	Offset    int64  `json:"offset"`
	ChunkNum  int    `json:"chunk_num"`
	Active    bool   `json:"active"`
}

func main() {
	loadConfig()
	agentID = config.AgentID
	if agentID == "" {
		agentID = uuid.New().String()
		log.Printf("[AGENT] Generated agent ID: %s", agentID)
	}

	log.Printf("[AGENT] Name: %s", config.AgentName)
	log.Printf("[AGENT] Master URL: %s", config.MasterURL)
	log.Printf("[AGENT] Poll interval: %ds", config.PollInterval)
	log.Printf("[AGENT] Scan roots: %s", config.ScanRoots)
	log.Printf("[AGENT] Extensions: %s", config.Extensions)

	// Register with master
	registerWithMaster()

	// Start heartbeat
	go heartbeatLoop()

	// Start polling for commands
	go pollLoop()

	// Start file watcher for monitored files
	go fileWatcherLoop()

	// Keep alive
	select {}
}

func loadConfig() {
	config = AgentConfig{
		MasterURL:    getEnv("MASTER_URL", "http://master:8080"),
		MasterKey:    getEnv("MASTER_KEY", "sera-default-key"),
		AgentName:    getEnv("AGENT_NAME", "agent-1"),
		AgentID:      getEnv("AGENT_ID", ""),
		PollInterval: getEnvInt("POLL_INTERVAL", 3),
		ScanRoots:    getEnv("SCAN_ROOTS", "/var/log,/tmp"),
		Extensions:   getEnv("EXTENSIONS", ".log"),
	}
}

// ============================================================
// Master Communication
// ============================================================

func apiRequest(method, path string, body interface{}) (map[string]interface{}, error) {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, config.MasterURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", config.MasterKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return result, nil
}

func registerWithMaster() {
	log.Println("[AGENT] Registering with master...")
	req := map[string]string{
		"name": config.AgentName,
		"ip":   getLocalIP(),
	}

	for i := 0; i < 10; i++ {
		resp, err := apiRequest("POST", "/api/agent/register", req)
		if err != nil {
			log.Printf("[AGENT] Registration attempt %d failed: %v", i+1, err)
			time.Sleep(5 * time.Second)
			continue
		}

		if success, ok := resp["success"].(bool); ok && success {
			if data, ok := resp["data"].(map[string]interface{}); ok {
				if id, ok := data["id"].(string); ok {
					agentID = id
					log.Printf("[AGENT] Registered successfully! ID: %s", agentID)
					return
				}
			}
		}
		log.Printf("[AGENT] Registration response: %v", resp)
		time.Sleep(5 * time.Second)
	}
	log.Fatal("[AGENT] Failed to register with master after 10 attempts")
}

func heartbeatLoop() {
	ticker := time.NewTicker(15 * time.Second)
	for range ticker.C {
		status := "online"
		mu.RLock()
		if len(monitoredFiles) > 0 {
			status = "scanning"
		}
		mu.RUnlock()

		apiRequest("POST", "/api/agent/heartbeat", map[string]interface{}{
			"agent_id": agentID,
			"status":   status,
		})
	}
}

func pollLoop() {
	ticker := time.NewTicker(time.Duration(config.PollInterval) * time.Second)
	for range ticker.C {
		pollCommands()
	}
}

func pollCommands() {
	resp, err := apiRequest("POST", "/api/agent/poll", map[string]string{"agent_id": agentID})
	if err != nil {
		log.Printf("[AGENT] Poll error: %v", err)
		return
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		return
	}

	cmdsRaw, ok := data["commands"].([]interface{})
	if !ok || len(cmdsRaw) == 0 {
		return
	}

	for _, cmdRaw := range cmdsRaw {
		cmd, ok := cmdRaw.(map[string]interface{})
		if !ok {
			continue
		}
		handleCommand(cmd)
	}
}

func handleCommand(cmd map[string]interface{}) {
	cmdID, _ := cmd["id"].(string)
	cmdType, _ := cmd["type"].(string)
	payloadStr, _ := cmd["payload"].(string)

	log.Printf("[AGENT] Received command: %s (id: %s)", cmdType, cmdID)

	switch cmdType {
	case "scan_files":
		handleScanFiles(cmdID, payloadStr)
	case "start_monitor":
		handleStartMonitor(cmdID, payloadStr)
	case "stop_monitor":
		handleStopMonitor(cmdID, payloadStr)
	case "process_chunk":
		handleProcessChunk(cmdID, payloadStr)
	case "stop_all":
		handleStopAll(cmdID)
	default:
		log.Printf("[AGENT] Unknown command type: %s", cmdType)
	}
}

// ============================================================
// Command Handlers
// ============================================================

func handleScanFiles(cmdID, payloadStr string) {
	var payload struct {
		Extensions []string `json:"extensions"`
		RootPaths  []string `json:"root_paths"`
		MaxDepth   int      `json:"max_depth"`
	}
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		log.Printf("[AGENT] Failed to parse scan payload: %v", err)
		return
	}

	if payload.Extensions == nil {
		payload.Extensions = strings.Split(config.Extensions, ",")
	}
	if payload.RootPaths == nil {
		payload.RootPaths = strings.Split(config.ScanRoots, ",")
	}
	if payload.MaxDepth == 0 {
		payload.MaxDepth = 5
	}

	log.Printf("[AGENT] Scanning for files with extensions: %v in paths: %v", payload.Extensions, payload.RootPaths)

	files := scanFiles(payload.RootPaths, payload.Extensions, payload.MaxDepth)
	log.Printf("[AGENT] Found %d files", len(files))

	// Send result back to master
	apiRequest("POST", "/api/agent/scan-result", map[string]interface{}{
		"agent_id":   agentID,
		"command_id": cmdID,
		"result": map[string]interface{}{
			"files":      files,
			"total":      len(files),
			"root_paths": payload.RootPaths,
		},
	})
}

func handleStartMonitor(cmdID, payloadStr string) {
	var payload struct {
		Files     []string `json:"files"`
		ChunkSize int      `json:"chunk_size"`
	}
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		log.Printf("[AGENT] Failed to parse monitor payload: %v", err)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	for _, fpath := range payload.Files {
		monitoredFiles[fpath] = &MonitoredFile{
			Path:      fpath,
			ChunkSize: payload.ChunkSize,
			Offset:    0,
			ChunkNum:  0,
			Active:    true,
		}
		log.Printf("[AGENT] Monitoring file: %s (chunk_size: %d)", fpath, payload.ChunkSize)
	}

	// Start processing monitored files
	go processMonitoredFiles()
}

func handleStopMonitor(cmdID, payloadStr string) {
	var payload struct {
		Files []string `json:"files"`
	}
	json.Unmarshal([]byte(payloadStr), &payload)

	mu.Lock()
	defer mu.Unlock()

	for _, fpath := range payload.Files {
		if mf, ok := monitoredFiles[fpath]; ok {
			mf.Active = false
			delete(monitoredFiles, fpath)
			log.Printf("[AGENT] Stopped monitoring: %s", fpath)
		}
	}
}

func handleProcessChunk(cmdID, payloadStr string) {
	var payload struct {
		FilePath  string `json:"file_path"`
		ChunkSize int    `json:"chunk_size"`
		Offset    int64  `json:"offset"`
	}
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		log.Printf("[AGENT] Failed to parse chunk payload: %v", err)
		return
	}

	// Read chunk and send to master
	lines, err := readFileChunk(payload.FilePath, payload.Offset, payload.ChunkSize)
	if err != nil {
		log.Printf("[AGENT] Error reading file %s: %v", payload.FilePath, err)
		return
	}

	if lines == "" {
		log.Printf("[AGENT] End of file: %s", payload.FilePath)
		// Mark file as done
		mu.Lock()
		if mf, ok := monitoredFiles[payload.FilePath]; ok {
			mf.Active = false
			delete(monitoredFiles, payload.FilePath)
		}
		mu.Unlock()
		return
	}

	chunkNum := int(payload.Offset / int64(payload.ChunkSize))

	log.Printf("[AGENT] Sending chunk %d for %s (%d bytes)", chunkNum, payload.FilePath, len(lines))

	apiRequest("POST", "/api/agent/chunk", map[string]interface{}{
		"agent_id":  agentID,
		"file_path": payload.FilePath,
		"chunk_num": chunkNum,
		"lines":     lines,
	})
}

func handleStopAll(cmdID string) {
	mu.Lock()
	defer mu.Unlock()

	monitoredFiles = make(map[string]*MonitoredFile)
	log.Println("[AGENT] Stopped all monitoring")
}

// ============================================================
// File Scanner
// ============================================================

func scanFiles(rootPaths, extensions []string, maxDepth int) []map[string]interface{} {
	var files []map[string]interface{}

	for _, root := range rootPaths {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			// Get depth
			rel, _ := filepath.Rel(root, path)
			depth := strings.Count(rel, string(filepath.Separator))
			if depth > maxDepth {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if d.IsDir() {
				return nil
			}

			// Check extension
			for _, ext := range extensions {
				ext = strings.TrimSpace(ext)
				if ext == "" {
					continue
				}
				if strings.HasSuffix(path, ext) || strings.HasSuffix(path, ext+".") {
					info, err := d.Info()
					if err != nil {
						continue
					}
					files = append(files, map[string]interface{}{
						"path":     path,
						"size":     info.Size(),
						"mod_time": info.ModTime().Format(time.RFC3339),
					})
					break
				}
			}
			return nil
		})
	}

	return files
}

// ============================================================
// File Chunk Reader
// ============================================================

func readFileChunk(filePath string, offset int64, chunkSize int) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("cannot open file: %w", err)
	}
	defer f.Close()

	// Seek to offset
	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return "", fmt.Errorf("seek error: %w", err)
		}
	}

	// Read lines
	var lines []string
	buf := make([]byte, 0, 4096)
	oneByte := make([]byte, 1)
	lineCount := 0

	for lineCount < chunkSize {
		n, err := f.Read(oneByte)
		if n == 0 || err != nil {
			break
		}

		if oneByte[0] == '\n' {
			lines = append(lines, string(buf)+string(oneByte))
			buf = buf[:0]
			lineCount++
		} else {
			buf = append(buf, oneByte[0])
		}
	}

	// If we have remaining bytes in buffer (last line without newline)
	if len(buf) > 0 {
		lines = append(lines, string(buf))
	}

	if len(lines) == 0 {
		return "", nil
	}

	// Get current position for next offset
	pos, _ := f.Seek(0, io.SeekCurrent)

	result := strings.Join(lines, "")
	_ = pos // offset tracked by caller

	return result, nil
}

// ============================================================
// File Watcher (periodic check for new content)
// ============================================================

func fileWatcherLoop() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		processMonitoredFiles()
	}
}

func processMonitoredFiles() {
	mu.RLock()
	var activeFiles []MonitoredFile
	for _, mf := range monitoredFiles {
		if mf.Active {
			activeFiles = append(activeFiles, *mf)
		}
	}
	mu.RUnlock()

	for _, mf := range activeFiles {
		// Check if file has new content
		info, err := os.Stat(mf.Path)
		if err != nil {
			continue
		}

		// Simple check: if file size > offset, there's new content
		if info.Size() > mf.Offset {
			lines, err := readFileChunk(mf.Path, mf.Offset, mf.ChunkSize)
			if err != nil {
				log.Printf("[AGENT] Error reading %s: %v", mf.Path, err)
				continue
			}

			if lines != "" {
				chunkNum := mf.ChunkNum + 1
				log.Printf("[AGENT] Sending chunk %d for %s", chunkNum, mf.Path)

				apiRequest("POST", "/api/agent/chunk", map[string]interface{}{
					"agent_id":  agentID,
					"file_path": mf.Path,
					"chunk_num": chunkNum,
					"lines":     lines,
				})

				// Update offset
				mu.Lock()
				if currentMF, ok := monitoredFiles[mf.Path]; ok {
					currentMF.Offset = info.Size()
					currentMF.ChunkNum = chunkNum
				}
				mu.Unlock()
			}
		}
	}
}

// ============================================================
// Helpers
// ============================================================

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

func getLocalIP() string {
	// Try to get the container's IP
	addrs, err := os.Hostname()
	if err == nil {
		return addrs
	}
	return "unknown"
}
