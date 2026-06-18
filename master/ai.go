package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ============================================================
// AI Integration (Ollama / OpenAI-compatible)
// ============================================================

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"`
}

type ChatResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Done  bool   `json:"done"`
	Error string `json:"error,omitempty"`
}

type OpenAIRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

const systemPrompt = `You are a log analyzer AI. Analyze the following log lines and determine if they indicate a problem that needs human attention.

Respond with ONLY a JSON object (no markdown, no code blocks) in this exact format:
{
  "needs_action": true/false,
  "severity": "info" | "warning" | "critical",
  "summary": "one line summary",
  "details": "detailed explanation if needs_action is true, otherwise empty string"
}

Rules:
- "needs_action": true if the logs show errors, exceptions, crashes, security issues, performance degradation, or any problem requiring human intervention
- "needs_action": false for normal operational logs (info messages, routine operations, health checks)
- severity: "critical" for crashes, data loss, security breaches; "warning" for errors, retries, timeouts; "info" for everything else
- Keep summary under 100 characters
- Keep details under 500 characters`

func processChunkWithAI(chunkID string, req SendChunkRequest) {
	// Mark chunk as processing
	log.Printf("[AI] Processing chunk %d for %s", req.ChunkNum, req.FilePath)

	// Skip empty chunks
	if req.Lines == "" {
		log.Printf("[AI] Empty chunk, skipping")
		return
	}

	// Call AI
	aiResp, err := callAI(req.Lines)
	if err != nil {
		log.Printf("[AI] Error calling AI: %v", err)
		// Store error report
		report := AIReport{
			ID:          uuid.New().String(),
			AgentID:     req.AgentID,
			FilePath:    req.FilePath,
			ChunkNum:    req.ChunkNum,
			NeedsAction: false,
			Summary:     fmt.Sprintf("AI processing error: %v", err),
			Severity:    SeverityInfo,
			Details:     err.Error(),
			CreatedAt:   time.Now(),
		}
		db.Exec(
			"INSERT INTO ai_reports (id, agent_id, file_path, chunk_num, needs_action, summary, severity, details, sent_to_tg, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			report.ID, report.AgentID, report.FilePath, report.ChunkNum, boolToInt(report.NeedsAction),
			report.Summary, report.Severity, report.Details, 0, report.CreatedAt,
		)
		return
	}

	// Parse AI response
	var parsed struct {
		NeedsAction bool   `json:"needs_action"`
		Severity    string `json:"severity"`
		Summary     string `json:"summary"`
		Details     string `json:"details"`
	}

	cleanResp := cleanJSONResponse(aiResp)
	if err := json.Unmarshal([]byte(cleanResp), &parsed); err != nil {
		log.Printf("[AI] Failed to parse response: %v\nRaw: %s", err, aiResp)
		parsed = struct {
			NeedsAction bool   `json:"needs_action"`
			Severity    string `json:"severity"`
			Summary     string `json:"summary"`
			Details     string `json:"details"`
		}{
			NeedsAction: false,
			Severity:    "info",
			Summary:     "Failed to parse AI response",
			Details:     aiResp,
		}
	}

	// Store report
	report := AIReport{
		ID:          uuid.New().String(),
		AgentID:     req.AgentID,
		FilePath:    req.FilePath,
		ChunkNum:    req.ChunkNum,
		NeedsAction: parsed.NeedsAction,
		Summary:     parsed.Summary,
		Severity:    Severity(parsed.Severity),
		Details:     parsed.Details,
		CreatedAt:   time.Now(),
	}

	_, err = db.Exec(
		"INSERT INTO ai_reports (id, agent_id, file_path, chunk_num, needs_action, summary, severity, details, sent_to_tg, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		report.ID, report.AgentID, report.FilePath, report.ChunkNum, boolToInt(report.NeedsAction),
		report.Summary, report.Severity, report.Details, 0, report.CreatedAt,
	)
	if err != nil {
		log.Printf("[AI] Failed to store report: %v", err)
		return
	}

	log.Printf("[AI] Report stored: needs_action=%v, severity=%s, summary=%s",
		report.NeedsAction, report.Severity, report.Summary)

	// Send to Telegram if action needed
	if report.NeedsAction && config.Telegram.Enabled {
		go sendTelegramReport(report)
	}

	// Update file offset - send next chunk command
	updateFileOffset(req.AgentID, req.FilePath, req.ChunkNum)
}

func callAI(logLines string) (string, error) {
	userMsg := fmt.Sprintf("Analyze these log lines:\n\n%s", logLines)

	if config.AI.Provider == "ollama" || config.AI.Provider == "openai-compatible" {
		return callOllama(userMsg)
	} else if config.AI.Provider == "openai" {
		return callOpenAI(userMsg)
	}

	return "", fmt.Errorf("unsupported AI provider: %s", config.AI.Provider)
}

func callOllama(userMsg string) (string, error) {
	reqBody := ChatRequest{
		Model: config.AI.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMsg},
		},
		MaxTokens:   config.AI.MaxTokens,
		Temperature: config.AI.Temperature,
		Stream:      false,
	}

	body, _ := json.Marshal(reqBody)
	url := config.AI.BaseURL + "/api/chat"

	timeoutSec := config.AI.Timeout
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}

	start := time.Now()
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	duration := time.Since(start).Milliseconds()

	if err != nil {
		logAILog("ollama", config.AI.Model, url, string(body), "", duration, false, err.Error())
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		logAILog("ollama", config.AI.Model, url, string(body), string(respBody), duration, false, err.Error())
		return "", fmt.Errorf("ollama response parse failed: %w", err)
	}

	if chatResp.Error != "" {
		logAILog("ollama", config.AI.Model, url, string(body), string(respBody), duration, false, chatResp.Error)
		return "", fmt.Errorf("ollama error: %s", chatResp.Error)
	}

	logAILog("ollama", config.AI.Model, url, string(body), string(respBody), duration, true, "")
	return chatResp.Message.Content, nil
}

func callOpenAI(userMsg string) (string, error) {
	reqBody := OpenAIRequest{
		Model: config.AI.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMsg},
		},
		MaxTokens:   config.AI.MaxTokens,
		Temperature: config.AI.Temperature,
		Stream:      false,
	}

	body, _ := json.Marshal(reqBody)
	url := config.AI.BaseURL + "/v1/chat/completions"

	req, _ := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if config.AI.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+config.AI.APIKey)
	}

	timeoutSec := config.AI.Timeout
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}

	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		logAILog("openai", config.AI.Model, url, string(body), "", duration, false, err.Error())
		return "", fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var chatResp OpenAIResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		logAILog("openai", config.AI.Model, url, string(body), string(respBody), duration, false, err.Error())
		return "", fmt.Errorf("openai response parse failed: %w", err)
	}

	if chatResp.Error != nil {
		logAILog("openai", config.AI.Model, url, string(body), string(respBody), duration, false, chatResp.Error.Message)
		return "", fmt.Errorf("openai error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		logAILog("openai", config.AI.Model, url, string(body), string(respBody), duration, false, "no choices returned")
		return "", fmt.Errorf("openai returned no choices")
	}

	logAILog("openai", config.AI.Model, url, string(body), string(respBody), duration, true, "")
	return chatResp.Choices[0].Message.Content, nil
}

func cleanJSONResponse(resp string) string {
	// Remove markdown code blocks if present
	start := 0
	end := len(resp)

	for i := 0; i < len(resp)-3; i++ {
		if resp[i] == '{' {
			start = i
			break
		}
	}

	for i := len(resp) - 1; i >= 0; i-- {
		if resp[i] == '}' {
			end = i + 1
			break
		}
	}

	if start < end {
		return resp[start:end]
	}
	return resp
}

func updateFileOffset(agentID, filePath string, chunkNum int) {
	// Update the file offset in DB for next chunk reading
	var chunkSize int
	err := db.QueryRow("SELECT chunk_size FROM files WHERE agent_id = ? AND path = ?", agentID, filePath).Scan(&chunkSize)
	if err != nil || chunkSize <= 0 {
		chunkSize = config.AI.ChunkSize
	}

	// The next offset = current chunk_num * chunk_size (in lines)
	// We'll let the agent calculate this, just send process_chunk command
	newOffset := int64((chunkNum + 1) * chunkSize)

	// Check if we've processed too many chunks (safety limit)
	if chunkNum > 10000 {
		db.Exec("UPDATE files SET status = 'done', offset = 0 WHERE agent_id = ? AND path = ?", agentID, filePath)
		log.Printf("[AI] File %s reached max chunks, marking done", filePath)
		return
	}

	// Send next chunk command
	payload := marshalJSON(ProcessChunkPayload{
		FilePath:  filePath,
		ChunkSize: chunkSize,
		Offset:    newOffset,
	})

	cmdID := uuid.New().String()
	db.Exec(
		"INSERT INTO commands (id, agent_id, type, status, payload, created_at) VALUES (?, ?, ?, 'pending', ?, ?)",
		cmdID, agentID, CmdProcessChunk, payload, time.Now(),
	)

	db.Exec("UPDATE files SET offset = ?, status = 'processing' WHERE agent_id = ? AND path = ?", newOffset, agentID, filePath)
}

func logAILog(provider, model, url, request, response string, durationMs int64, success bool, errMsg string) {
	id := uuid.New().String()
	successInt := 0
	if success {
		successInt = 1
	}
	// Truncate very long request/response to avoid bloating DB (keep first 2000 chars each)
	if len(request) > 2000 {
		request = request[:2000] + "...[truncated]"
	}
	if len(response) > 2000 {
		response = response[:2000] + "...[truncated]"
	}
	_, err := db.Exec(
		"INSERT INTO ai_request_logs (id, provider, model, url, request, response, duration_ms, success, error, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		id, provider, model, url, request, response, durationMs, successInt, errMsg, time.Now(),
	)
	if err != nil {
		log.Printf("[AI] Failed to log AI request: %v", err)
	}
}

// ============================================================
// AI Queue - Sequential Processing (1 at a time)
// ============================================================

func enqueueAIChunk(agentID, filePath string, chunkNum int, lines string) {
	id := uuid.New().String()
	// Get next position in queue
	var maxPos int
	db.QueryRow("SELECT COALESCE(MAX(position), 0) FROM ai_queue WHERE status IN ('pending','processing')").Scan(&maxPos)

	_, err := db.Exec(
		"INSERT INTO ai_queue (id, agent_id, file_path, chunk_num, lines, status, position, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 'pending', ?, ?, ?)",
		id, agentID, filePath, chunkNum, lines, maxPos+1, time.Now(), time.Now(),
	)
	if err != nil {
		log.Printf("[AI] Failed to enqueue chunk: %v", err)
		return
	}
	log.Printf("[AI] Queued chunk %d for %s (position: %d)", chunkNum, filePath, maxPos+1)
}

// aiQueueWorker runs as a single goroutine, processing queue items one by one
func aiQueueWorker() {
	log.Printf("[AI] Queue worker started (sequential mode)")
	for {
		// Find next pending item
		var item AIQueueItem
		err := db.QueryRow(
			"SELECT id, agent_id, file_path, chunk_num, lines, position FROM ai_queue WHERE status = 'pending' ORDER BY position ASC, created_at ASC LIMIT 1",
		).Scan(&item.ID, &item.AgentID, &item.FilePath, &item.ChunkNum, &item.Lines, &item.Position)
		if err != nil {
			// No pending items, wait and retry
			time.Sleep(2 * time.Second)
			continue
		}

		// Mark as processing
		db.Exec("UPDATE ai_queue SET status = 'processing', updated_at = ? WHERE id = ?", time.Now(), item.ID)
		log.Printf("[AI] Processing queue #%d: chunk %d for %s", item.Position, item.ChunkNum, item.FilePath)

		// Process with AI
		if item.Lines == "" {
			db.Exec("UPDATE ai_queue SET status = 'done', result = 'empty chunk', updated_at = ? WHERE id = ?", time.Now(), item.ID)
			continue
		}

		aiResp, err := callAI(item.Lines)
		if err != nil {
			log.Printf("[AI] Queue #%d error: %v", item.Position, err)
			db.Exec("UPDATE ai_queue SET status = 'failed', result = ?, updated_at = ? WHERE id = ?", err.Error(), time.Now(), item.ID)
			// Still store error report
			report := AIReport{
				ID: uuid.New().String(), AgentID: item.AgentID, FilePath: item.FilePath,
				ChunkNum: item.ChunkNum, NeedsAction: false,
				Summary: fmt.Sprintf("AI error: %v", err), Severity: SeverityInfo,
				Details: err.Error(), CreatedAt: time.Now(),
			}
			db.Exec("INSERT INTO ai_reports (id, agent_id, file_path, chunk_num, needs_action, summary, severity, details, sent_to_tg, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
				report.ID, report.AgentID, report.FilePath, report.ChunkNum, boolToInt(report.NeedsAction),
				report.Summary, report.Severity, report.Details, 0, report.CreatedAt)
			continue
		}

		// Parse AI response
		var parsed struct {
			NeedsAction bool   `json:"needs_action"`
			Severity    string `json:"severity"`
			Summary     string `json:"summary"`
			Details     string `json:"details"`
		}
		cleanResp := cleanJSONResponse(aiResp)
		if err := json.Unmarshal([]byte(cleanResp), &parsed); err != nil {
			parsed = struct {
				NeedsAction bool   `json:"needs_action"`
				Severity    string `json:"severity"`
				Summary     string `json:"summary"`
				Details     string `json:"details"`
			}{false, "info", "Failed to parse AI response", aiResp}
		}

		// Store report
		report := AIReport{
			ID: uuid.New().String(), AgentID: item.AgentID, FilePath: item.FilePath,
			ChunkNum: item.ChunkNum, NeedsAction: parsed.NeedsAction,
			Summary: parsed.Summary, Severity: Severity(parsed.Severity),
			Details: parsed.Details, CreatedAt: time.Now(),
		}
		db.Exec("INSERT INTO ai_reports (id, agent_id, file_path, chunk_num, needs_action, summary, severity, details, sent_to_tg, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			report.ID, report.AgentID, report.FilePath, report.ChunkNum, boolToInt(report.NeedsAction),
			report.Summary, report.Severity, report.Details, 0, report.CreatedAt)

		log.Printf("[AI] Queue #%d done: needs_action=%v, severity=%s", item.Position, report.NeedsAction, report.Severity)

		if report.NeedsAction && config.Telegram.Enabled {
			go sendTelegramReport(report)
		}

		// Mark queue item done
		db.Exec("UPDATE ai_queue SET status = 'done', result = ?, updated_at = ? WHERE id = ?",
			report.Summary, time.Now(), item.ID)

		// Update file offset
		updateFileOffset(item.AgentID, item.FilePath, item.ChunkNum)
	}
}
