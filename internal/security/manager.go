package security

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"
)

// Manager implements the SecurityManager interface
type Manager struct {
	config        SecurityConfig
	users         map[string]*User
	sessions      map[string]*Session
	apiKeys       map[string]*APIKey
	securityEvents []*SecurityEvent
	rateLimiters  map[string]*rate.Limiter
	metrics       *SecurityMetrics
	mu            sync.RWMutex
	running       bool
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewManager creates a new security manager
func NewManager(config SecurityConfig) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		config:        config,
		users:         make(map[string]*User),
		sessions:      make(map[string]*Session),
		apiKeys:       make(map[string]*APIKey),
		securityEvents: make([]*SecurityEvent, 0),
		rateLimiters:  make(map[string]*rate.Limiter),
		metrics:       &SecurityMetrics{},
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Authenticate authenticates a user
func (sm *Manager) Authenticate(token string) (*User, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	// Check if it's an API key
	if apiKey, exists := sm.apiKeys[token]; exists {
		if !apiKey.IsActive || time.Now().After(apiKey.ExpiresAt) {
			return nil, fmt.Errorf("invalid API key")
		}
		
		// Update last used
		apiKey.LastUsed = time.Now()
		
		// Get user
		user, exists := sm.users[apiKey.UserID]
		if !exists {
			return nil, fmt.Errorf("user not found")
		}
		
		return user, nil
	}
	
	// Check if it's a JWT token
	if sm.config.Auth.Method == AuthMethodJWT {
		return sm.validateJWT(token)
	}
	
	// Check if it's a session token
	if session, exists := sm.sessions[token]; exists {
		if !session.IsActive || time.Now().After(session.ExpiresAt) {
			return nil, fmt.Errorf("invalid session")
		}
		
		user, exists := sm.users[session.UserID]
		if !exists {
			return nil, fmt.Errorf("user not found")
		}
		
		return user, nil
	}
	
	return nil, fmt.Errorf("invalid token")
}

// Authorize checks if a user has permission
func (sm *Manager) Authorize(user *User, permission Permission) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	// Admin has all permissions
	if user.Role == RoleAdmin {
		return true
	}
	
	// Check user permissions
	for _, p := range user.Permissions {
		if p == permission {
			return true
		}
	}
	
	// Check role-based permissions
	return sm.hasRolePermission(user.Role, permission)
}

// CreateSession creates a new user session
func (sm *Manager) CreateSession(userID string, ipAddress, userAgent string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Generate session token
	token, err := sm.generateToken()
	if err != nil {
		return nil, err
	}
	
	session := &Session{
		ID:        sm.generateID(),
		UserID:    userID,
		Token:     token,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sm.config.Auth.SessionTimeout),
		IPAddress: ipAddress,
		UserAgent: userAgent,
		IsActive:  true,
	}
	
	sm.sessions[token] = session
	sm.metrics.ActiveSessions++
	
	return session, nil
}

// ValidateSession validates a session
func (sm *Manager) ValidateSession(sessionID string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	
	if !session.IsActive || time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}
	
	return session, nil
}

// RevokeSession revokes a session
func (sm *Manager) RevokeSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}
	
	session.IsActive = false
	sm.metrics.ActiveSessions--
	
	return nil
}

// CreateAPIKey creates a new API key
func (sm *Manager) CreateAPIKey(userID, name string, permissions []Permission) (*APIKey, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Generate API key
	key, err := sm.generateAPIKey()
	if err != nil {
		return nil, err
	}
	
	// Generate secret
	secret, err := sm.generateSecret()
	if err != nil {
		return nil, err
	}
	
	apiKey := &APIKey{
		ID:          sm.generateID(),
		Name:        name,
		Key:         key,
		Secret:      secret,
		UserID:      userID,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(365 * 24 * time.Hour), // 1 year
		IsActive:    true,
		Metadata:    make(map[string]interface{}),
	}
	
	sm.apiKeys[key] = apiKey
	sm.metrics.ActiveAPIKeys++
	
	return apiKey, nil
}

// ValidateAPIKey validates an API key
func (sm *Manager) ValidateAPIKey(key string) (*APIKey, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	apiKey, exists := sm.apiKeys[key]
	if !exists {
		return nil, fmt.Errorf("invalid API key")
	}
	
	if !apiKey.IsActive || time.Now().After(apiKey.ExpiresAt) {
		return nil, fmt.Errorf("API key expired")
	}
	
	// Update last used
	apiKey.LastUsed = time.Now()
	
	return apiKey, nil
}

// RevokeAPIKey revokes an API key
func (sm *Manager) RevokeAPIKey(keyID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	for _, apiKey := range sm.apiKeys {
		if apiKey.ID == keyID {
			apiKey.IsActive = false
			sm.metrics.ActiveAPIKeys--
			return nil
		}
	}
	
	return fmt.Errorf("API key not found")
}

// ListAPIKeys lists API keys for a user
func (sm *Manager) ListAPIKeys(userID string) ([]*APIKey, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	var keys []*APIKey
	for _, apiKey := range sm.apiKeys {
		if apiKey.UserID == userID {
			keys = append(keys, apiKey)
		}
	}
	
	return keys, nil
}

// CreateUser creates a new user
func (sm *Manager) CreateUser(username, email string, role Role) (*User, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Check if user already exists
	for _, user := range sm.users {
		if user.Username == username || user.Email == email {
			return nil, fmt.Errorf("user already exists")
		}
	}
	
	user := &User{
		ID:          sm.generateID(),
		Username:    username,
		Email:       email,
		Role:        role,
		Permissions: sm.getRolePermissions(role),
		CreatedAt:   time.Now(),
		IsActive:    true,
		Metadata:    make(map[string]interface{}),
	}
	
	sm.users[user.ID] = user
	
	return user, nil
}

// GetUser gets a user by ID
func (sm *Manager) GetUser(userID string) (*User, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	user, exists := sm.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	
	return user, nil
}

// UpdateUser updates a user
func (sm *Manager) UpdateUser(userID string, updates map[string]interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	user, exists := sm.users[userID]
	if !exists {
		return fmt.Errorf("user not found")
	}
	
	// Update fields
	for key, value := range updates {
		switch key {
		case "username":
			if v, ok := value.(string); ok {
				user.Username = v
			}
		case "email":
			if v, ok := value.(string); ok {
				user.Email = v
			}
		case "role":
			if v, ok := value.(Role); ok {
				user.Role = v
				user.Permissions = sm.getRolePermissions(v)
			}
		case "is_active":
			if v, ok := value.(bool); ok {
				user.IsActive = v
			}
		}
	}
	
	return nil
}

// DeleteUser deletes a user
func (sm *Manager) DeleteUser(userID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Revoke all sessions
	for _, session := range sm.sessions {
		if session.UserID == userID {
			session.IsActive = false
		}
	}
	
	// Revoke all API keys
	for _, apiKey := range sm.apiKeys {
		if apiKey.UserID == userID {
			apiKey.IsActive = false
		}
	}
	
	delete(sm.users, userID)
	
	return nil
}

// ListUsers lists all users
func (sm *Manager) ListUsers() ([]*User, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	var users []*User
	for _, user := range sm.users {
		users = append(users, user)
	}
	
	return users, nil
}

// LogSecurityEvent logs a security event
func (sm *Manager) LogSecurityEvent(event *SecurityEvent) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	event.ID = sm.generateID()
	event.Timestamp = time.Now()
	
	sm.securityEvents = append(sm.securityEvents, event)
	sm.metrics.SecurityEvents++
	sm.metrics.LastSecurityEvent = event.Timestamp
	
	// Log to file if configured
	if sm.config.Audit.Enabled {
		log.Printf("Security Event: %s - %s", event.Type, event.Message)
	}
	
	return nil
}

// GetSecurityEvents gets security events
func (sm *Manager) GetSecurityEvents(filter SecurityEventFilter) ([]*SecurityEvent, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	var events []*SecurityEvent
	for _, event := range sm.securityEvents {
		if sm.matchesFilter(event, filter) {
			events = append(events, event)
		}
	}
	
	// Apply limit and offset
	if filter.Limit > 0 {
		start := filter.Offset
		end := start + filter.Limit
		if end > len(events) {
			end = len(events)
		}
		events = events[start:end]
	}
	
	return events, nil
}

// ResolveSecurityEvent resolves a security event
func (sm *Manager) ResolveSecurityEvent(eventID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	for _, event := range sm.securityEvents {
		if event.ID == eventID {
			event.Resolved = true
			event.ResolvedAt = time.Now()
			return nil
		}
	}
	
	return fmt.Errorf("security event not found")
}

// CheckRateLimit checks rate limit for an IP address
func (sm *Manager) CheckRateLimit(ipAddress string) (bool, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if !sm.config.RateLimit.Enabled {
		return true, nil
	}
	
	limiter, exists := sm.rateLimiters[ipAddress]
	if !exists {
		limiter = rate.NewLimiter(
			rate.Limit(sm.config.RateLimit.RequestsPerMinute),
			sm.config.RateLimit.BurstSize,
		)
		sm.rateLimiters[ipAddress] = limiter
	}
	
	allowed := limiter.Allow()
	if !allowed {
		sm.metrics.RateLimitHits++
	}
	
	return allowed, nil
}

// RecordRequest records a request for rate limiting
func (sm *Manager) RecordRequest(ipAddress string) error {
	sm.metrics.TotalRequests++
	return nil
}

// ValidateInput validates input data
func (sm *Manager) ValidateInput(data interface{}) error {
	if !sm.config.Validation.Enabled {
		return nil
	}
	
	// Implement input validation logic
	// This is a simplified implementation
	return nil
}

// SanitizeInput sanitizes input data
func (sm *Manager) SanitizeInput(data string) string {
	if !sm.config.Validation.SanitizeInput {
		return data
	}
	
	// Implement input sanitization logic
	// This is a simplified implementation
	return data
}

// Encrypt encrypts data
func (sm *Manager) Encrypt(data []byte) ([]byte, error) {
	if !sm.config.Encryption.Enabled {
		return data, nil
	}
	
	// Implement encryption logic
	// This is a simplified implementation
	return data, nil
}

// Decrypt decrypts data
func (sm *Manager) Decrypt(data []byte) ([]byte, error) {
	if !sm.config.Encryption.Enabled {
		return data, nil
	}
	
	// Implement decryption logic
	// This is a simplified implementation
	return data, nil
}

// GetTLSConfig returns TLS configuration
func (sm *Manager) GetTLSConfig() *tls.Config {
	if !sm.config.TLS.Enabled {
		return nil
	}
	
	config := &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
	}
	
	return config
}

// GetSecurityMetrics returns security metrics
func (sm *Manager) GetSecurityMetrics() *SecurityMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	// Return a copy
	metrics := *sm.metrics
	return &metrics
}

// Start starts the security manager
func (sm *Manager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if sm.running {
		return fmt.Errorf("security manager already running")
	}
	
	sm.running = true
	
	// Start cleanup routine
	go sm.cleanupRoutine()
	
	log.Println("Security manager started")
	return nil
}

// Stop stops the security manager
func (sm *Manager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if !sm.running {
		return nil
	}
	
	sm.running = false
	sm.cancel()
	
	log.Println("Security manager stopped")
	return nil
}

// IsRunning returns whether the manager is running
func (sm *Manager) IsRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.running
}

// Private methods

func (sm *Manager) generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (sm *Manager) generateToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (sm *Manager) generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return "vk_" + base64.URLEncoding.EncodeToString(bytes), nil
}

func (sm *Manager) generateSecret() (string, error) {
	bytes := make([]byte, 64)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (sm *Manager) validateJWT(tokenString string) (*User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(sm.config.Auth.JWTSecret), nil
	})
	
	if err != nil {
		return nil, err
	}
	
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["user_id"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid token claims")
		}
		
		return sm.GetUser(userID)
	}
	
	return nil, fmt.Errorf("invalid token")
}

func (sm *Manager) hasRolePermission(role Role, permission Permission) bool {
	rolePermissions := sm.getRolePermissions(role)
	for _, p := range rolePermissions {
		if p == permission {
			return true
		}
	}
	return false
}

func (sm *Manager) getRolePermissions(role Role) []Permission {
	switch role {
	case RoleAdmin:
		return []Permission{
			PermissionAdmin,
			PermissionReadMarketData,
			PermissionWriteOrders,
			PermissionReadOrders,
			PermissionReadPositions,
			PermissionWritePositions,
			PermissionReadStrategies,
			PermissionWriteStrategies,
			PermissionReadRisk,
			PermissionWriteRisk,
			PermissionReadBacktesting,
			PermissionWriteBacktesting,
			PermissionReadPlugins,
			PermissionWritePlugins,
		}
	case RoleTrader:
		return []Permission{
			PermissionReadMarketData,
			PermissionWriteOrders,
			PermissionReadOrders,
			PermissionReadPositions,
			PermissionWritePositions,
		}
	case RoleStrategist:
		return []Permission{
			PermissionReadMarketData,
			PermissionReadStrategies,
			PermissionWriteStrategies,
			PermissionReadBacktesting,
			PermissionWriteBacktesting,
		}
	case RoleRiskManager:
		return []Permission{
			PermissionReadMarketData,
			PermissionReadOrders,
			PermissionReadPositions,
			PermissionReadRisk,
			PermissionWriteRisk,
		}
	case RoleViewer:
		return []Permission{
			PermissionReadMarketData,
			PermissionReadOrders,
			PermissionReadPositions,
			PermissionReadStrategies,
			PermissionReadRisk,
			PermissionReadBacktesting,
		}
	default:
		return []Permission{}
	}
}

func (sm *Manager) matchesFilter(event *SecurityEvent, filter SecurityEventFilter) bool {
	if filter.Type != "" && event.Type != filter.Type {
		return false
	}
	if filter.Level != "" && event.Level != filter.Level {
		return false
	}
	if filter.UserID != "" && event.UserID != filter.UserID {
		return false
	}
	if filter.IPAddress != "" && event.IPAddress != filter.IPAddress {
		return false
	}
	if !filter.StartTime.IsZero() && event.Timestamp.Before(filter.StartTime) {
		return false
	}
	if !filter.EndTime.IsZero() && event.Timestamp.After(filter.EndTime) {
		return false
	}
	if filter.Resolved != nil && event.Resolved != *filter.Resolved {
		return false
	}
	return true
}

func (sm *Manager) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			sm.cleanupExpiredSessions()
			sm.cleanupExpiredAPIKeys()
			sm.cleanupOldSecurityEvents()
		case <-sm.ctx.Done():
			return
		}
	}
}

func (sm *Manager) cleanupExpiredSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	now := time.Now()
	for token, session := range sm.sessions {
		if now.After(session.ExpiresAt) {
			delete(sm.sessions, token)
			sm.metrics.ActiveSessions--
		}
	}
}

func (sm *Manager) cleanupExpiredAPIKeys() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	now := time.Now()
	for key, apiKey := range sm.apiKeys {
		if now.After(apiKey.ExpiresAt) {
			delete(sm.apiKeys, key)
			sm.metrics.ActiveAPIKeys--
		}
	}
}

func (sm *Manager) cleanupOldSecurityEvents() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	retentionDays := sm.config.Audit.RetentionDays
	if retentionDays <= 0 {
		retentionDays = 30 // Default retention
	}
	
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	var events []*SecurityEvent
	
	for _, event := range sm.securityEvents {
		if event.Timestamp.After(cutoff) {
			events = append(events, event)
		}
	}
	
	sm.securityEvents = events
}
