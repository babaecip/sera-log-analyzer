package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// ============================================================
// Telegram Integration
// ============================================================

type TGMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

func sendTelegramReport(report AIReport) {
	if !config.Telegram.Enabled || config.Telegram.BotToken == "" || config.Telegram.ChatID == "" {
		return
	}

	severity := "ℹ️ INFO"
	switch report.Severity {
	case SeverityWarning:
		severity = "⚠️ WARNING"
	case SeverityCritical:
		severity = "🔴 CRITICAL"
	}

	text := fmt.Sprintf(
		"*%s*\n\n"+
			"*File:* `%s`\n"+
			"*Chunk:* #%d\n"+
			"*Agent:* %s\n\n"+
			"*Summary:* %s\n\n"+
			"*Details:*\n%s",
		severity,
		report.FilePath,
		report.ChunkNum,
		report.AgentID,
		report.Summary,
		report.Details,
	)

	sendTGMessage(text)

	// Mark as sent
	db.Exec("UPDATE ai_reports SET sent_to_tg = 1 WHERE id = ?", report.ID)
	log.Printf("[TG] Report sent for %s chunk #%d", report.FilePath, report.ChunkNum)
}

func sendTGMessage(text string) error {
	msg := TGMessage{
		ChatID:    config.Telegram.ChatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	body, _ := json.Marshal(msg)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.Telegram.BotToken)

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[TG] Failed to send message: %v", err)
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		log.Printf("[TG] API error %d: %s", resp.StatusCode, string(respBody))
		return fmt.Errorf("telegram API error %d", resp.StatusCode)
	}

	return nil
}

func sendTGMessageRaw(text string) error {
	return sendTGMessage(text)
}

// Send a general notification (not just reports)
func sendTGNotification(title, message string) {
	text := fmt.Sprintf("*%s*\n\n%s", title, message)
	sendTGMessage(text)
}

// Startup notification
func notifyStartup() {
	if !config.Telegram.Enabled {
		return
	}
	text := fmt.Sprintf(
		"🟢 *Sera Log Analyzer Master Started*\n\n"+
			"Port: %s\n"+
			"AI: %s (%s)\n"+
			"Max Storage: %d MB\n"+
			"Time: %s",
		config.Port,
		config.AI.Provider,
		config.AI.Model,
		config.MaxStorageMB,
		time.Now().Format("2006-01-02 15:04:05"),
	)
	sendTGMessage(text)
}
