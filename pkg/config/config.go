package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Cache    CacheConfig
	Logger   LoggerConfig
	Security SecurityConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host            string
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	EnableCORS      bool
	TrustedProxies  []string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Type     string // "json", "postgres", "mongodb"
	DataPath string // For JSON storage
	DSN      string // For SQL databases
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	Enabled bool
	Type    string // "memory", "redis"
	TTL     time.Duration
	RedisConfig
}

// RedisConfig holds Redis-specific configuration
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// LoggerConfig holds logging configuration
type LoggerConfig struct {
	Level       string // "debug", "info", "warn", "error"
	Format      string // "json", "console"
	OutputPaths []string
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	RateLimitEnabled     bool
	RateLimitRequests    int
	RateLimitWindow      time.Duration
	APIKeyEnabled        bool
	JWTSecret           string
	BCryptCost          int
	AllowedOrigins      []string
	MaxRequestBodySize  int64
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			Port:            getEnv("SERVER_PORT", "8080"),
			ReadTimeout:     getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getDurationEnv("SERVER_SHUTDOWN_TIMEOUT", 10*time.Second),
			EnableCORS:      getBoolEnv("SERVER_ENABLE_CORS", true),
		},
		Database: DatabaseConfig{
			Type:     getEnv("DATABASE_TYPE", "json"),
			DataPath: getEnv("DATABASE_DATA_PATH", "./data/products"),
			DSN:      getEnv("DATABASE_DSN", ""),
		},
		Cache: CacheConfig{
			Enabled: getBoolEnv("CACHE_ENABLED", false),
			Type:    getEnv("CACHE_TYPE", "memory"),
			TTL:     getDurationEnv("CACHE_TTL", 5*time.Minute),
			RedisConfig: RedisConfig{
				Host:     getEnv("REDIS_HOST", "localhost"),
				Port:     getEnv("REDIS_PORT", "6379"),
				Password: getEnv("REDIS_PASSWORD", ""),
				DB:       getIntEnv("REDIS_DB", 0),
			},
		},
		Logger: LoggerConfig{
			Level:       getEnv("LOG_LEVEL", "info"),
			Format:      getEnv("LOG_FORMAT", "json"),
			OutputPaths: []string{"stdout"},
		},
		Security: SecurityConfig{
			RateLimitEnabled:    getBoolEnv("RATE_LIMIT_ENABLED", true),
			RateLimitRequests:   getIntEnv("RATE_LIMIT_REQUESTS", 100),
			RateLimitWindow:     getDurationEnv("RATE_LIMIT_WINDOW", 1*time.Minute),
			APIKeyEnabled:       getBoolEnv("API_KEY_ENABLED", false),
			JWTSecret:          getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			BCryptCost:         getIntEnv("BCRYPT_COST", 10),
			AllowedOrigins:     []string{"*"},
			MaxRequestBodySize: getInt64Env("MAX_REQUEST_BODY_SIZE", 10*1024*1024), // 10MB
		},
	}
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
