package alerts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ConsoleChannel sends alerts to console output
type ConsoleChannel struct {
	name string
}

func NewConsoleChannel(name string) *ConsoleChannel {
	return &ConsoleChannel{
		name: name,
	}
}

func (c *ConsoleChannel) Send(alert *Alert) error {
	fmt.Printf("🚨 ALERT [%s] %s: %s\n", alert.Severity, alert.Title, alert.Message)
	fmt.Printf("   Time: %s\n", alert.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("   Type: %s\n", alert.Type)
	if alert.Data != nil {
		dataJSON, _ := json.MarshalIndent(alert.Data, "   ", "   ")
		fmt.Printf("   Data: %s\n", string(dataJSON))
	}
	fmt.Println()
	return nil
}

func (c *ConsoleChannel) Name() string {
	return c.name
}

func (c *ConsoleChannel) Type() string {
	return "console"
}

// FileChannel writes alerts to a log file
type FileChannel struct {
	name     string
	filename string
	file     *os.File
	mutex    sync.Mutex
}

func NewFileChannel(name, filename string) (*FileChannel, error) {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	
	return &FileChannel{
		name:     name,
		filename: filename,
		file:     file,
	}, nil
}

func (f *FileChannel) Send(alert *Alert) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	
	alertJSON, err := json.Marshal(map[string]interface{}{
		"id":           alert.ID,
		"rule_id":      alert.RuleID,
		"type":         alert.Type,
		"severity":     alert.Severity,
		"title":        alert.Title,
		"message":      alert.Message,
		"data":         alert.Data,
		"timestamp":    alert.Timestamp,
		"acknowledged": alert.Acknowledged,
		"resolved":     alert.Resolved,
		"resolved_at":  alert.ResolvedAt,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}
	
	_, err = fmt.Fprintf(f.file, "%s\n", string(alertJSON))
	if err != nil {
		return fmt.Errorf("failed to write alert: %w", err)
	}
	
	return nil
}

func (f *FileChannel) Name() string {
	return f.name
}

func (f *FileChannel) Type() string {
	return "file"
}

func (f *FileChannel) Close() error {
	if f.file != nil {
		return f.file.Close()
	}
	return nil
}

// WebSocketChannel sends alerts via WebSocket connections
type WebSocketChannel struct {
	name        string
	connections map[*websocket.Conn]bool
	mutex       sync.RWMutex
}

func NewWebSocketChannel(name string) *WebSocketChannel {
	return &WebSocketChannel{
		name:        name,
		connections: make(map[*websocket.Conn]bool),
	}
}

func (w *WebSocketChannel) Send(alert *Alert) error {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	
	alertJSON, err := json.Marshal(map[string]interface{}{
		"type":     "alert",
		"alert":    alert,
		"timestamp": time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}
	
	var failedConnections []*websocket.Conn
	
	for conn := range w.connections {
		err := conn.WriteJSON(map[string]interface{}{
			"type":  "alert",
			"data":  alert,
			"timestamp": time.Now(),
		})
		if err != nil {
			failedConnections = append(failedConnections, conn)
		}
	}
	
	// Remove failed connections
	if len(failedConnections) > 0 {
		w.mutex.RUnlock()
		w.mutex.Lock()
		for _, conn := range failedConnections {
			delete(w.connections, conn)
			conn.Close()
		}
		w.mutex.Unlock()
		w.mutex.RLock()
	}
	
	return nil
}

func (w *WebSocketChannel) Name() string {
	return w.name
}

func (w *WebSocketChannel) Type() string {
	return "websocket"
}

func (w *WebSocketChannel) AddConnection(conn *websocket.Conn) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	w.connections[conn] = true
}

func (w *WebSocketChannel) RemoveConnection(conn *websocket.Conn) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	if _, exists := w.connections[conn]; exists {
		delete(w.connections, conn)
		conn.Close()
	}
}

func (w *WebSocketChannel) GetConnectionCount() int {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	
	return len(w.connections)
}

// EmailChannel sends alerts via email (placeholder implementation)
type EmailChannel struct {
	name     string
	smtpHost string
	smtpPort int
	username string
	password string
	from     string
	to       []string
}

func NewEmailChannel(name, smtpHost string, smtpPort int, username, password, from string, to []string) *EmailChannel {
	return &EmailChannel{
		name:     name,
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		username: username,
		password: password,
		from:     from,
		to:       to,
	}
}

func (e *EmailChannel) Send(alert *Alert) error {
	// Placeholder implementation
	// In production, integrate with SMTP library like gomail
	fmt.Printf("📧 EMAIL ALERT to %v: [%s] %s - %s\n", e.to, alert.Severity, alert.Title, alert.Message)
	return nil
}

func (e *EmailChannel) Name() string {
	return e.name
}

func (e *EmailChannel) Type() string {
	return "email"
}

// SlackChannel sends alerts to Slack (placeholder implementation)
type SlackChannel struct {
	name    string
	webhook string
	channel string
}

func NewSlackChannel(name, webhook, channel string) *SlackChannel {
	return &SlackChannel{
		name:    name,
		webhook: webhook,
		channel: channel,
	}
}

func (s *SlackChannel) Send(alert *Alert) error {
	// Placeholder implementation
	// In production, integrate with Slack webhook API
	fmt.Printf("💬 SLACK ALERT to #%s: [%s] %s - %s\n", s.channel, alert.Severity, alert.Title, alert.Message)
	return nil
}

func (s *SlackChannel) Name() string {
	return s.name
}

func (s *SlackChannel) Type() string {
	return "slack"
}

// WebhookChannel sends alerts to HTTP webhooks
type WebhookChannel struct {
	name   string
	url    string
	method string
	headers map[string]string
}

func NewWebhookChannel(name, url, method string, headers map[string]string) *WebhookChannel {
	return &WebhookChannel{
		name:    name,
		url:     url,
		method:  method,
		headers: headers,
	}
}

func (w *WebhookChannel) Send(alert *Alert) error {
	// Placeholder implementation
	// In production, use http client to send webhook
	fmt.Printf("🔗 WEBHOOK ALERT to %s: [%s] %s - %s\n", w.url, alert.Severity, alert.Title, alert.Message)
	return nil
}

func (w *WebhookChannel) Name() string {
	return w.name
}

func (w *WebhookChannel) Type() string {
	return "webhook"
}

// ChannelFactory creates alert channels based on configuration
type ChannelFactory struct{}

func NewChannelFactory() *ChannelFactory {
	return &ChannelFactory{}
}

func (f *ChannelFactory) CreateChannel(config map[string]interface{}) (AlertChannel, error) {
	channelType, ok := config["type"].(string)
	if !ok {
		return nil, fmt.Errorf("channel type is required")
	}
	
	name, ok := config["name"].(string)
	if !ok {
		return nil, fmt.Errorf("channel name is required")
	}
	
	switch channelType {
	case "console":
		return NewConsoleChannel(name), nil
	
	case "file":
		filename, ok := config["filename"].(string)
		if !ok {
			return nil, fmt.Errorf("filename is required for file channel")
		}
		return NewFileChannel(name, filename)
	
	case "websocket":
		return NewWebSocketChannel(name), nil
	
	case "email":
		smtpHost, _ := config["smtp_host"].(string)
		smtpPort, _ := config["smtp_port"].(float64)
		username, _ := config["username"].(string)
		password, _ := config["password"].(string)
		from, _ := config["from"].(string)
		
		var to []string
		if toConfig, ok := config["to"].([]interface{}); ok {
			for _, addr := range toConfig {
				if strAddr, ok := addr.(string); ok {
					to = append(to, strAddr)
				}
			}
		}
		
		return NewEmailChannel(name, smtpHost, int(smtpPort), username, password, from, to), nil
	
	case "slack":
		webhook, _ := config["webhook"].(string)
		channel, _ := config["channel"].(string)
		return NewSlackChannel(name, webhook, channel), nil
	
	case "webhook":
		url, _ := config["url"].(string)
		method, _ := config["method"].(string)
		
		var headers map[string]string
		if h, ok := config["headers"].(map[string]interface{}); ok {
			headers = make(map[string]string)
			for k, v := range h {
				headers[k] = fmt.Sprintf("%v", v)
			}
		}
		
		return NewWebhookChannel(name, url, method, headers), nil
	
	default:
		return nil, fmt.Errorf("unsupported channel type: %s", channelType)
	}
}