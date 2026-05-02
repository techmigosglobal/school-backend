package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                       string
	AppMode                    string
	DatabaseURL                string
	DatabaseDSN                string
	DBHost                     string
	DBPort                     string
	DBUser                     string
	DBPassword                 string
	DBName                     string
	JWTSecret                  string
	Environment                string
	AllowedOrigins             []string
	RedisURL                   string
	RedisPassword              string
	RedisDB                    int
	CacheTTLSeconds            int
	RateLimitWindowSeconds     int
	RateLimitMaxLogin          int
	RateLimitMaxAPI            int
	DisablePublicRegistration  bool
	BootstrapPrincipalEmail    string
	BootstrapPrincipalPassword string
	MigrateOnStart             bool
	UsePostgresOnly            bool
	RequireHTTPSAPIBaseURL     bool
}

func Load() *Config {
	env := getEnv("ENVIRONMENT", "development")
	isProd := env == "production"
	origins := parseCSV(getEnv("ALLOWED_ORIGINS", ""))
	if !isProd && len(origins) == 0 {
		origins = []string{"http://localhost:3000", "http://localhost:8080", "http://127.0.0.1:3000", "http://127.0.0.1:8080"}
	}
	dbHost := getEnv("DB_HOST", "")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "")
	dbPass := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "")
	dbURL := strings.TrimSpace(getEnv("DATABASE_URL", ""))

	if dbURL == "" && dbHost != "" && dbUser != "" && dbPass != "" && dbName != "" {
		dbURL = "postgres://" + dbUser + ":" + dbPass + "@" + dbHost + ":" + dbPort + "/" + dbName + "?sslmode=disable"
	}

	return &Config{
		Port:                       getEnv("PORT", "8080"),
		AppMode:                    getEnv("APP_MODE", "api"),
		DatabaseURL:                dbURL,
		DatabaseDSN:                getEnv("DATABASE_DSN", "school.db"),
		DBHost:                     dbHost,
		DBPort:                     dbPort,
		DBUser:                     dbUser,
		DBPassword:                 dbPass,
		DBName:                     dbName,
		JWTSecret:                  getEnv("JWT_SECRET", "dev-insecure-secret-change-me"),
		Environment:                env,
		AllowedOrigins:             origins,
		RedisURL:                   strings.TrimSpace(getEnv("REDIS_URL", "")),
		RedisPassword:              getEnv("REDIS_PASSWORD", ""),
		RedisDB:                    getEnvAsInt("REDIS_DB", 0),
		CacheTTLSeconds:            getEnvAsInt("CACHE_TTL_SECONDS", 120),
		RateLimitWindowSeconds:     getEnvAsInt("RATE_LIMIT_WINDOW_SECONDS", 60),
		RateLimitMaxLogin:          getEnvAsInt("RATE_LIMIT_MAX_LOGIN", 5),
		RateLimitMaxAPI:            getEnvAsInt("RATE_LIMIT_MAX_API", 120),
		DisablePublicRegistration:  getEnvAsBool("DISABLE_PUBLIC_REGISTRATION", isProd),
		BootstrapPrincipalEmail:    strings.TrimSpace(getEnv("BOOTSTRAP_PRINCIPAL_EMAIL", "principal@schooldesk.local")),
		BootstrapPrincipalPassword: getEnv("BOOTSTRAP_PRINCIPAL_PASSWORD", "PR1234"),
		MigrateOnStart:             getEnvAsBool("MIGRATE_ON_START", true),
		UsePostgresOnly:            getEnvAsBool("USE_POSTGRES_ONLY", isProd),
		RequireHTTPSAPIBaseURL:     getEnvAsBool("REQUIRE_HTTPS_API_BASE_URL", isProd),
	}
}

func (c *Config) Validate() error {
	if c.Environment != "production" {
		return nil
	}

	if strings.TrimSpace(c.JWTSecret) == "" {
		return errors.New("missing JWT_SECRET in production")
	}
	if len(c.JWTSecret) < 32 {
		return errors.New("JWT_SECRET must be at least 32 characters in production")
	}
	if c.DatabaseURL == "" {
		if c.DBHost == "" || c.DBUser == "" || c.DBPassword == "" || c.DBName == "" {
			return errors.New("missing DATABASE_URL, or DB_HOST/DB_USER/DB_PASSWORD/DB_NAME in production")
		}
	}
	if c.RedisURL == "" {
		return errors.New("missing REDIS_URL in production")
	}
	if strings.TrimSpace(c.RedisPassword) == "" {
		return errors.New("missing REDIS_PASSWORD in production")
	}
	if len(c.AllowedOrigins) == 0 {
		return errors.New("missing ALLOWED_ORIGINS in production")
	}
	if strings.TrimSpace(c.BootstrapPrincipalEmail) == "" {
		return errors.New("missing BOOTSTRAP_PRINCIPAL_EMAIL in production")
	}
	if len(c.BootstrapPrincipalPassword) < 12 {
		return errors.New("BOOTSTRAP_PRINCIPAL_PASSWORD must be at least 12 characters in production")
	}
	return nil
}

func parseCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		clean := strings.TrimSpace(p)
		if clean != "" {
			out = append(out, clean)
		}
	}
	return out
}

func getEnvAsInt(key string, defaultValue int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getEnvAsBool(key string, defaultValue bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return defaultValue
	}
	return raw == "1" || raw == "true" || raw == "yes" || raw == "on"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
