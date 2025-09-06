package security

import (
	"crypto/tls"
	"net/http"
	"time"
)

// SecurityLevel represents the security level
type SecurityLevel string

const (
	SecurityLevelLow    SecurityLevel = "low"
	SecurityLevelMedium SecurityLevel = "medium"
	SecurityLevelHigh   SecurityLevel = "high"
	SecurityLevelCritical SecurityLevel = "critical"
)

// AuthMethod represents authentication methods
type AuthMethod string

const (
	AuthMethodAPIKey    AuthMethod = "api_key"
	AuthMethodJWT       AuthMethod = "jwt"
	AuthMethodOAuth2    AuthMethod = "oauth2"
	AuthMethodBasic     AuthMethod = "basic"
	AuthMethodCertificate AuthMethod = "certificate"
)

// Permission represents a permission
type Permission string

const (
	PermissionReadMarketData    Permission = "read_market_data"
	PermissionWriteOrders       Permission = "write_orders"
	PermissionReadOrders        Permission = "read_orders"
	PermissionReadPositions     Permission = "read_positions"
	PermissionWritePositions    Permission = "write_positions"
	PermissionReadStrategies    Permission = "read_strategies"
	PermissionWriteStrategies   Permission = "write_strategies"
	PermissionReadRisk          Permission = "read_risk"
	PermissionWriteRisk         Permission = "write_risk"
	PermissionReadBacktesting   Permission = "read_backtesting"
	PermissionWriteBacktesting  Permission = "write_backtesting"
	PermissionReadPlugins       Permission = "read_plugins"
	PermissionWritePlugins      Permission = "write_plugins"
	PermissionAdmin             Permission = "admin"
)

// Role represents a user role
type Role string

const (
	RoleViewer     Role = "viewer"
	RoleTrader     Role = "trader"
	RoleStrategist Role = "strategist"
	RoleRiskManager Role = "risk_manager"
	RoleAdmin      Role = "admin"
)

// SecurityConfig represents security configuration
type SecurityConfig struct {
	// TLS Configuration
	TLS TLSConfig `yaml:"tls"`
	
	// Authentication Configuration
	Auth AuthConfig `yaml:"auth"`
	
	// Rate Limiting Configuration
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	
	// Input Validation Configuration
	Validation ValidationConfig `yaml:"validation"`
	
	// Audit Configuration
	Audit AuditConfig `yaml:"audit"`
	
	// Security Headers Configuration
	Headers HeadersConfig `yaml:"headers"`
	
	// CORS Configuration
	CORS CORSConfig `yaml:"cors"`
	
	// Session Configuration
	Session SessionConfig `yaml:"session"`
	
	// Encryption Configuration
	Encryption EncryptionConfig `yaml:"encryption"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled     bool   `yaml:"enabled"`
	CertFile    string `yaml:"cert_file"`
	KeyFile     string `yaml:"key_file"`
	MinVersion  string `yaml:"min_version"`
	MaxVersion  string `yaml:"max_version"`
	CipherSuites []string `yaml:"cipher_suites"`
	InsecureSkipVerify bool `yaml:"insecure_skip_verify"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Enabled       bool          `yaml:"enabled"`
	Method        AuthMethod    `yaml:"method"`
	JWTSecret     string        `yaml:"jwt_secret"`
	JWTExpiry     time.Duration `yaml:"jwt_expiry"`
	APIKeyHeader  string        `yaml:"api_key_header"`
	SessionTimeout time.Duration `yaml:"session_timeout"`
	MaxSessions   int           `yaml:"max_sessions"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Enabled     bool          `yaml:"enabled"`
	RequestsPerMinute int     `yaml:"requests_per_minute"`
	BurstSize   int           `yaml:"burst_size"`
	WindowSize  time.Duration `yaml:"window_size"`
	BlockDuration time.Duration `yaml:"block_duration"`
}

// ValidationConfig represents input validation configuration
type ValidationConfig struct {
	Enabled           bool     `yaml:"enabled"`
	MaxRequestSize    int64    `yaml:"max_request_size"`
	AllowedOrigins    []string `yaml:"allowed_origins"`
	BlockedIPs        []string `yaml:"blocked_ips"`
	AllowedIPs        []string `yaml:"allowed_ips"`
	SanitizeInput     bool     `yaml:"sanitize_input"`
	ValidateJSON      bool     `yaml:"validate_json"`
	MaxArraySize      int      `yaml:"max_array_size"`
	MaxStringLength   int      `yaml:"max_string_length"`
}

// AuditConfig represents audit logging configuration
type AuditConfig struct {
	Enabled       bool     `yaml:"enabled"`
	LogLevel      string   `yaml:"log_level"`
	LogFile       string   `yaml:"log_file"`
	MaxLogSize    int64    `yaml:"max_log_size"`
	MaxLogFiles   int      `yaml:"max_log_files"`
	Events        []string `yaml:"events"`
	RetentionDays int      `yaml:"retention_days"`
}

// HeadersConfig represents security headers configuration
type HeadersConfig struct {
	Enabled           bool   `yaml:"enabled"`
	HSTS              bool   `yaml:"hsts"`
	HSTSMaxAge        int    `yaml:"hsts_max_age"`
	ContentTypeOptions bool  `yaml:"content_type_options"`
	XSSProtection     bool   `yaml:"xss_protection"`
	FrameOptions      string `yaml:"frame_options"`
	ReferrerPolicy    string `yaml:"referrer_policy"`
	ContentSecurityPolicy string `yaml:"content_security_policy"`
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	Enabled           bool     `yaml:"enabled"`
	AllowedOrigins    []string `yaml:"allowed_origins"`
	AllowedMethods    []string `yaml:"allowed_methods"`
	AllowedHeaders    []string `yaml:"allowed_headers"`
	ExposedHeaders    []string `yaml:"exposed_headers"`
	AllowCredentials  bool     `yaml:"allow_credentials"`
	MaxAge            int      `yaml:"max_age"`
}

// SessionConfig represents session configuration
type SessionConfig struct {
	Enabled       bool          `yaml:"enabled"`
	Store         string        `yaml:"store"`
	Secret        string        `yaml:"secret"`
	MaxAge        time.Duration `yaml:"max_age"`
	Secure        bool          `yaml:"secure"`
	HttpOnly      bool          `yaml:"http_only"`
	SameSite      string        `yaml:"same_site"`
}

// EncryptionConfig represents encryption configuration
type EncryptionConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Algorithm     string `yaml:"algorithm"`
	KeySize       int    `yaml:"key_size"`
	IVSize        int    `yaml:"iv_size"`
	KeyDerivation string `yaml:"key_derivation"`
	SaltSize      int    `yaml:"salt_size"`
}

// User represents a user
type User struct {
	ID          string       `json:"id"`
	Username    string       `json:"username"`
	Email       string       `json:"email"`
	Role        Role         `json:"role"`
	Permissions []Permission `json:"permissions"`
	CreatedAt   time.Time    `json:"created_at"`
	LastLogin   time.Time    `json:"last_login"`
	IsActive    bool         `json:"is_active"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	IsActive  bool      `json:"is_active"`
}

// APIKey represents an API key
type APIKey struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Key         string       `json:"key"`
	Secret      string       `json:"secret"`
	UserID      string       `json:"user_id"`
	Permissions []Permission `json:"permissions"`
	CreatedAt   time.Time    `json:"created_at"`
	ExpiresAt   time.Time    `json:"expires_at"`
	LastUsed    time.Time    `json:"last_used"`
	IsActive    bool         `json:"is_active"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SecurityEvent represents a security event
type SecurityEvent struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Level       SecurityLevel          `json:"level"`
	UserID      string                 `json:"user_id,omitempty"`
	IPAddress   string                 `json:"ip_address"`
	UserAgent   string                 `json:"user_agent"`
	Endpoint    string                 `json:"endpoint"`
	Method      string                 `json:"method"`
	Status      int                    `json:"status"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details"`
	Timestamp   time.Time              `json:"timestamp"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  time.Time              `json:"resolved_at,omitempty"`
}

// SecurityMetrics represents security metrics
type SecurityMetrics struct {
	TotalRequests     int64 `json:"total_requests"`
	BlockedRequests   int64 `json:"blocked_requests"`
	FailedAuth        int64 `json:"failed_auth"`
	RateLimitHits     int64 `json:"rate_limit_hits"`
	SecurityEvents    int64 `json:"security_events"`
	ActiveSessions    int64 `json:"active_sessions"`
	ActiveAPIKeys     int64 `json:"active_api_keys"`
	LastSecurityEvent time.Time `json:"last_security_event"`
}

// SecurityManager defines the interface for security management
type SecurityManager interface {
	// Authentication
	Authenticate(token string) (*User, error)
	Authorize(user *User, permission Permission) bool
	CreateSession(userID string, ipAddress, userAgent string) (*Session, error)
	ValidateSession(sessionID string) (*Session, error)
	RevokeSession(sessionID string) error
	
	// API Key Management
	CreateAPIKey(userID, name string, permissions []Permission) (*APIKey, error)
	ValidateAPIKey(key string) (*APIKey, error)
	RevokeAPIKey(keyID string) error
	ListAPIKeys(userID string) ([]*APIKey, error)
	
	// User Management
	CreateUser(username, email string, role Role) (*User, error)
	GetUser(userID string) (*User, error)
	UpdateUser(userID string, updates map[string]interface{}) error
	DeleteUser(userID string) error
	ListUsers() ([]*User, error)
	
	// Security Events
	LogSecurityEvent(event *SecurityEvent) error
	GetSecurityEvents(filter SecurityEventFilter) ([]*SecurityEvent, error)
	ResolveSecurityEvent(eventID string) error
	
	// Rate Limiting
	CheckRateLimit(ipAddress string) (bool, error)
	RecordRequest(ipAddress string) error
	
	// Input Validation
	ValidateInput(data interface{}) error
	SanitizeInput(data string) string
	
	// Encryption
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
	
	// TLS
	GetTLSConfig() *tls.Config
	
	// Metrics
	GetSecurityMetrics() *SecurityMetrics
	
	// Control
	Start() error
	Stop() error
	IsRunning() bool
}

// SecurityEventFilter represents a filter for security events
type SecurityEventFilter struct {
	Type      string        `json:"type,omitempty"`
	Level     SecurityLevel `json:"level,omitempty"`
	UserID    string        `json:"user_id,omitempty"`
	IPAddress string        `json:"ip_address,omitempty"`
	StartTime time.Time     `json:"start_time,omitempty"`
	EndTime   time.Time     `json:"end_time,omitempty"`
	Resolved  *bool         `json:"resolved,omitempty"`
	Limit     int           `json:"limit,omitempty"`
	Offset    int           `json:"offset,omitempty"`
}

// SecurityMiddleware defines the interface for security middleware
type SecurityMiddleware interface {
	AuthMiddleware() func(http.Handler) http.Handler
	RateLimitMiddleware() func(http.Handler) http.Handler
	ValidationMiddleware() func(http.Handler) http.Handler
	AuditMiddleware() func(http.Handler) http.Handler
	CORSMiddleware() func(http.Handler) http.Handler
	SecurityHeadersMiddleware() func(http.Handler) http.Handler
}

// PasswordHasher defines the interface for password hashing
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) bool
}

// TokenGenerator defines the interface for token generation
type TokenGenerator interface {
	GenerateToken(userID string, permissions []Permission) (string, error)
	ValidateToken(token string) (*User, error)
	RefreshToken(token string) (string, error)
	RevokeToken(token string) error
}

// EncryptionService defines the interface for encryption services
type EncryptionService interface {
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
	GenerateKey() ([]byte, error)
	DeriveKey(password string, salt []byte) ([]byte, error)
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogEvent(event *SecurityEvent) error
	GetEvents(filter SecurityEventFilter) ([]*SecurityEvent, error)
	ArchiveEvents(before time.Time) error
	Cleanup() error
}
