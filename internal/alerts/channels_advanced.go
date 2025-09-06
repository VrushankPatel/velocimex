package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"velocimex/internal/logger"
)

// SlackChannel sends alerts to Slack
type SlackChannel struct {
	webhookURL string
	channel    string
	username   string
	iconEmoji  string
	enabled    bool
	logger     logger.Logger
}

// NewSlackChannel creates a new Slack channel
func NewSlackChannel(webhookURL, channel, username string) *SlackChannel {
	return &SlackChannel{
		webhookURL: webhookURL,
		channel:    channel,
		username:   username,
		iconEmoji:  ":warning:",
		enabled:    true,
	}
}

// Send sends an alert to Slack
func (sc *SlackChannel) Send(alert *Alert) error {
	if !sc.enabled {
		return nil
	}

	// Create Slack message
	message := sc.formatMessage(alert)

	// Send to Slack
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(sc.webhookURL, "application/json", bytes.NewBuffer(message))
	if err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack API returned status %d", resp.StatusCode)
	}

	return nil
}

// GetName returns the channel name
func (sc *SlackChannel) GetName() string {
	return "slack"
}

// IsEnabled returns whether the channel is enabled
func (sc *SlackChannel) IsEnabled() bool {
	return sc.enabled
}

// SetEnabled enables or disables the channel
func (sc *SlackChannel) SetEnabled(enabled bool) {
	sc.enabled = enabled
}

// formatMessage formats the alert for Slack
func (sc *SlackChannel) formatMessage(alert *Alert) []byte {
	// Determine color based on severity
	color := "good"
	switch alert.Severity {
	case AlertSeverityCritical:
		color = "danger"
	case AlertSeverityHigh:
		color = "warning"
	case AlertSeverityMedium:
		color = "warning"
	case AlertSeverityLow:
		color = "good"
	}

	// Create Slack attachment
	attachment := map[string]interface{}{
		"color":     color,
		"title":     alert.Title,
		"text":      alert.Message,
		"timestamp": alert.CreatedAt.Unix(),
		"fields": []map[string]interface{}{
			{
				"title": "Severity",
				"value": alert.Severity.String(),
				"short": true,
			},
			{
				"title": "Type",
				"value": alert.Type,
				"short": true,
			},
		},
	}

	// Add metadata fields
	if alert.Metadata != nil {
		for key, value := range alert.Metadata {
			if key != "title" && key != "text" {
				attachment["fields"] = append(attachment["fields"].([]map[string]interface{}), map[string]interface{}{
					"title": strings.Title(key),
					"value": fmt.Sprintf("%v", value),
					"short": true,
				})
			}
		}
	}

	// Create payload
	payload := map[string]interface{}{
		"channel":     sc.channel,
		"username":    sc.username,
		"icon_emoji":  sc.iconEmoji,
		"attachments": []map[string]interface{}{attachment},
	}

	jsonData, _ := json.Marshal(payload)
	return jsonData
}

// EmailChannel sends alerts via email
type EmailChannel struct {
	smtpHost     string
	smtpPort     int
	username     string
	password     string
	fromEmail    string
	toEmails     []string
	subject      string
	enabled      bool
	logger       logger.Logger
}

// NewEmailChannel creates a new email channel
func NewEmailChannel(smtpHost string, smtpPort int, username, password, fromEmail string, toEmails []string) *EmailChannel {
	return &EmailChannel{
		smtpHost:  smtpHost,
		smtpPort:  smtpPort,
		username:  username,
		password:  password,
		fromEmail: fromEmail,
		toEmails:  toEmails,
		subject:   "Velocimex Alert",
		enabled:   true,
	}
}

// Send sends an alert via email
func (ec *EmailChannel) Send(alert *Alert) error {
	if !ec.enabled {
		return nil
	}

	// Create email message
	subject := fmt.Sprintf("%s - %s", ec.subject, alert.Title)
	body := ec.formatMessage(alert)

	// Set up authentication
	auth := smtp.PlainAuth("", ec.username, ec.password, ec.smtpHost)

	// Create message
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", strings.Join(ec.toEmails, ","), subject, body))

	// Send email
	addr := fmt.Sprintf("%s:%d", ec.smtpHost, ec.smtpPort)
	err := smtp.SendMail(addr, auth, ec.fromEmail, ec.toEmails, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// GetName returns the channel name
func (ec *EmailChannel) GetName() string {
	return "email"
}

// IsEnabled returns whether the channel is enabled
func (ec *EmailChannel) IsEnabled() bool {
	return ec.enabled
}

// SetEnabled enables or disables the channel
func (ec *EmailChannel) SetEnabled(enabled bool) {
	ec.enabled = enabled
}

// formatMessage formats the alert for email
func (ec *EmailChannel) formatMessage(alert *Alert) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Alert: %s\n", alert.Title))
	sb.WriteString(fmt.Sprintf("Severity: %s\n", alert.Severity.String()))
	sb.WriteString(fmt.Sprintf("Type: %s\n", alert.Type))
	sb.WriteString(fmt.Sprintf("Time: %s\n", alert.CreatedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Message: %s\n\n", alert.Message))

	if alert.Metadata != nil && len(alert.Metadata) > 0 {
		sb.WriteString("Additional Information:\n")
		for key, value := range alert.Metadata {
			sb.WriteString(fmt.Sprintf("  %s: %v\n", strings.Title(key), value))
		}
	}

	return sb.String()
}

// WebhookChannel sends alerts to webhooks
type WebhookChannel struct {
	url         string
	method      string
	headers     map[string]string
	timeout     time.Duration
	enabled     bool
	logger      logger.Logger
}

// NewWebhookChannel creates a new webhook channel
func NewWebhookChannel(url, method string, headers map[string]string) *WebhookChannel {
	return &WebhookChannel{
		url:     url,
		method:  method,
		headers: headers,
		timeout: 10 * time.Second,
		enabled: true,
	}
}

// Send sends an alert to a webhook
func (wc *WebhookChannel) Send(alert *Alert) error {
	if !wc.enabled {
		return nil
	}

	// Create request body
	body, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	// Create request
	req, err := http.NewRequest(wc.method, wc.url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range wc.headers {
		req.Header.Set(key, value)
	}

	// Send request
	client := &http.Client{Timeout: wc.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// GetName returns the channel name
func (wc *WebhookChannel) GetName() string {
	return "webhook"
}

// IsEnabled returns whether the channel is enabled
func (wc *WebhookChannel) IsEnabled() bool {
	return wc.enabled
}

// SetEnabled enables or disables the channel
func (wc *WebhookChannel) SetEnabled(enabled bool) {
	wc.enabled = enabled
}

// TeamsChannel sends alerts to Microsoft Teams
type TeamsChannel struct {
	webhookURL string
	enabled    bool
	logger     logger.Logger
}

// NewTeamsChannel creates a new Teams channel
func NewTeamsChannel(webhookURL string) *TeamsChannel {
	return &TeamsChannel{
		webhookURL: webhookURL,
		enabled:    true,
	}
}

// Send sends an alert to Teams
func (tc *TeamsChannel) Send(alert *Alert) error {
	if !tc.enabled {
		return nil
	}

	// Create Teams message
	message := tc.formatMessage(alert)

	// Send to Teams
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(tc.webhookURL, "application/json", bytes.NewBuffer(message))
	if err != nil {
		return fmt.Errorf("failed to send Teams message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Teams API returned status %d", resp.StatusCode)
	}

	return nil
}

// GetName returns the channel name
func (tc *TeamsChannel) GetName() string {
	return "teams"
}

// IsEnabled returns whether the channel is enabled
func (tc *TeamsChannel) IsEnabled() bool {
	return tc.enabled
}

// SetEnabled enables or disables the channel
func (tc *TeamsChannel) SetEnabled(enabled bool) {
	tc.enabled = enabled
}

// formatMessage formats the alert for Teams
func (tc *TeamsChannel) formatMessage(alert *Alert) []byte {
	// Determine color based on severity
	color := "00ff00" // Green
	switch alert.Severity {
	case AlertSeverityCritical:
		color = "ff0000" // Red
	case AlertSeverityHigh:
		color = "ff8800" // Orange
	case AlertSeverityMedium:
		color = "ffaa00" // Yellow
	case AlertSeverityLow:
		color = "00ff00" // Green
	}

	// Create Teams card
	card := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "http://schema.org/extensions",
		"summary":    alert.Title,
		"themeColor": color,
		"sections": []map[string]interface{}{
			{
				"activityTitle":    alert.Title,
				"activitySubtitle": fmt.Sprintf("Severity: %s | Type: %s", alert.Severity.String(), alert.Type),
				"text":             alert.Message,
				"facts": []map[string]interface{}{
					{
						"name":  "Time",
						"value": alert.CreatedAt.Format(time.RFC3339),
					},
					{
						"name":  "Severity",
						"value": alert.Severity.String(),
					},
					{
						"name":  "Type",
						"value": alert.Type,
					},
				},
			},
		},
	}

	// Add metadata facts
	if alert.Metadata != nil {
		facts := card["sections"].([]map[string]interface{})[0]["facts"].([]map[string]interface{})
		for key, value := range alert.Metadata {
			if key != "title" && key != "text" {
				facts = append(facts, map[string]interface{}{
					"name":  strings.Title(key),
					"value": fmt.Sprintf("%v", value),
				})
			}
		}
		card["sections"].([]map[string]interface{})[0]["facts"] = facts
	}

	jsonData, _ := json.Marshal(card)
	return jsonData
}

// DiscordChannel sends alerts to Discord
type DiscordChannel struct {
	webhookURL string
	username   string
	avatarURL  string
	enabled    bool
	logger     logger.Logger
}

// NewDiscordChannel creates a new Discord channel
func NewDiscordChannel(webhookURL, username string) *DiscordChannel {
	return &DiscordChannel{
		webhookURL: webhookURL,
		username:   username,
		enabled:    true,
	}
}

// Send sends an alert to Discord
func (dc *DiscordChannel) Send(alert *Alert) error {
	if !dc.enabled {
		return nil
	}

	// Create Discord message
	message := dc.formatMessage(alert)

	// Send to Discord
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(dc.webhookURL, "application/json", bytes.NewBuffer(message))
	if err != nil {
		return fmt.Errorf("failed to send Discord message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Discord API returned status %d", resp.StatusCode)
	}

	return nil
}

// GetName returns the channel name
func (dc *DiscordChannel) GetName() string {
	return "discord"
}

// IsEnabled returns whether the channel is enabled
func (dc *DiscordChannel) IsEnabled() bool {
	return dc.enabled
}

// SetEnabled enables or disables the channel
func (dc *DiscordChannel) SetEnabled(enabled bool) {
	dc.enabled = enabled
}

// formatMessage formats the alert for Discord
func (dc *DiscordChannel) formatMessage(alert *Alert) []byte {
	// Determine color based on severity
	color := 0x00ff00 // Green
	switch alert.Severity {
	case AlertSeverityCritical:
		color = 0xff0000 // Red
	case AlertSeverityHigh:
		color = 0xff8800 // Orange
	case AlertSeverityMedium:
		color = 0xffaa00 // Yellow
	case AlertSeverityLow:
		color = 0x00ff00 // Green
	}

	// Create Discord embed
	embed := map[string]interface{}{
		"title":       alert.Title,
		"description": alert.Message,
		"color":       color,
		"timestamp":   alert.CreatedAt.Format(time.RFC3339),
		"fields": []map[string]interface{}{
			{
				"name":   "Severity",
				"value":  alert.Severity.String(),
				"inline": true,
			},
			{
				"name":   "Type",
				"value":  alert.Type,
				"inline": true,
			},
		},
	}

	// Add metadata fields
	if alert.Metadata != nil {
		fields := embed["fields"].([]map[string]interface{})
		for key, value := range alert.Metadata {
			if key != "title" && key != "description" {
				fields = append(fields, map[string]interface{}{
					"name":   strings.Title(key),
					"value":  fmt.Sprintf("%v", value),
					"inline": true,
				})
			}
		}
		embed["fields"] = fields
	}

	// Create payload
	payload := map[string]interface{}{
		"username": dc.username,
		"embeds":   []map[string]interface{}{embed},
	}

	if dc.avatarURL != "" {
		payload["avatar_url"] = dc.avatarURL
	}

	jsonData, _ := json.Marshal(payload)
	return jsonData
}

// SMSChannel sends alerts via SMS (using Twilio or similar)
type SMSChannel struct {
	apiURL      string
	apiKey      string
	fromNumber  string
	toNumbers   []string
	enabled     bool
	logger      logger.Logger
}

// NewSMSChannel creates a new SMS channel
func NewSMSChannel(apiURL, apiKey, fromNumber string, toNumbers []string) *SMSChannel {
	return &SMSChannel{
		apiURL:     apiURL,
		apiKey:     apiKey,
		fromNumber: fromNumber,
		toNumbers:  toNumbers,
		enabled:    true,
	}
}

// Send sends an alert via SMS
func (sc *SMSChannel) Send(alert *Alert) error {
	if !sc.enabled {
		return nil
	}

	// Create SMS message
	message := sc.formatMessage(alert)

	// Send SMS to each number
	for _, toNumber := range sc.toNumbers {
		if err := sc.sendSMS(toNumber, message); err != nil {
			sc.logger.Error("alerts", fmt.Sprintf("Failed to send SMS to %s", toNumber), map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	return nil
}

// GetName returns the channel name
func (sc *SMSChannel) GetName() string {
	return "sms"
}

// IsEnabled returns whether the channel is enabled
func (sc *SMSChannel) IsEnabled() bool {
	return sc.enabled
}

// SetEnabled enables or disables the channel
func (sc *SMSChannel) SetEnabled(enabled bool) {
	sc.enabled = enabled
}

// formatMessage formats the alert for SMS
func (sc *SMSChannel) formatMessage(alert *Alert) string {
	// SMS messages should be short and concise
	return fmt.Sprintf("ALERT: %s - %s", alert.Severity.String(), alert.Message)
}

// sendSMS sends an SMS message
func (sc *SMSChannel) sendSMS(toNumber, message string) error {
	// Create request payload
	payload := map[string]interface{}{
		"to":   toNumber,
		"from": sc.fromNumber,
		"body": message,
	}

	jsonData, _ := json.Marshal(payload)

	// Create request
	req, err := http.NewRequest("POST", sc.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create SMS request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+sc.apiKey)

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("SMS API returned status %d", resp.StatusCode)
	}

	return nil
}
