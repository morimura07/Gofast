package config

import (
	"net/http"
	"os"
	"strings"
	"time"
)

func IsRunningTest() bool {
	for _, arg := range os.Args {
		if strings.HasSuffix(arg, ".test") {
			return true
		}
	}

	return false
}

func MustSetEnv(active bool, key string) string {
	value := os.Getenv(key)
	if active && value == "" {
		if IsRunningTest() {
			return "test"
		}
		panic("Missing environment variable: " + key)
	}

	return os.Getenv(key)
}

type Config struct {
	// General
	LogLevel    string
	ServiceName string
	Port        string
	Domain      string
	CoreURL     string
	ClientURL   string
	AlloyURL    string
	CronToken   string

	// Constants
	ContextTimeout             time.Duration
	ReadTimeout                time.Duration
	WriteTimeout               time.Duration
	IdleTimeout                time.Duration
	AccessTokenExp             time.Duration
	RefreshTokenExp            time.Duration
	MaxFileSize                int64
	CookieSameSite             http.SameSite
	SubscriptionSafePeriodDays int

	// Postgres
	PostgresHost     string
	PostgresPort     string
	PostgresDB       string
	PostgresUser     string
	PostgresPassword string

	// Auth
	GoogleClientID        string
	GoogleClientSecret    string
	GithubClientID        string
	GithubClientSecret    string
	FacebookClientID      string
	FacebookClientSecret  string
	MicrosoftClientID     string
	MicrosoftClientSecret string
	TwilioServiceSID      string
	DevUserID             string
	OAuthRedirectURI      string

	// GF_CONFIG_STRUCT_INSERT

}

func LoadConfig() *Config {
	const (
		ContextTimeout             = 5 * time.Second
		ReadTimeout                = 5 * time.Second
		WriteTimeout               = 10 * time.Second
		IdleTimeout                = 120 * time.Second
		AccessTokenExp             = 15 * time.Minute    // 15 minutes
		RefreshTokenExp            = 30 * 24 * time.Hour // 30 days
		MaxFileSize                = 10 << 20            // 10 MB
		SubscriptionSafePeriodDays = 3                   // 3 days
		UploadRateLimit            = 200 * time.Millisecond
	)

	// Determine SameSite based on environment
	// PR environments need SameSite=None for cross-site cookies (workers.dev frontend)
	// Staging/Production use SameSite=Lax (same domain)
	cookieSameSite := http.SameSiteLaxMode
	if os.Getenv("ENV") == "pr" {
		cookieSameSite = http.SameSiteNoneMode
	}

	cfg := &Config{
		LogLevel:                   MustSetEnv(true, "LOG_LEVEL"),
		ServiceName:                MustSetEnv(true, "SERVICE_NAME"),
		Port:                       MustSetEnv(true, "PORT"),
		Domain:                     MustSetEnv(true, "DOMAIN"),
		CoreURL:                    MustSetEnv(true, "CORE_URL"),
		ClientURL:                  MustSetEnv(true, "CLIENT_URL"),
		AlloyURL:                   MustSetEnv(false, "ALLOY_URL"),
		CronToken:                  MustSetEnv(true, "CRON_TOKEN"),
		ContextTimeout:             ContextTimeout,
		ReadTimeout:                ReadTimeout,
		WriteTimeout:               WriteTimeout,
		IdleTimeout:                IdleTimeout,
		AccessTokenExp:             AccessTokenExp,
		RefreshTokenExp:            RefreshTokenExp,
		MaxFileSize:                MaxFileSize,
		CookieSameSite:             cookieSameSite,
		SubscriptionSafePeriodDays: SubscriptionSafePeriodDays,
		PostgresHost:               MustSetEnv(true, "POSTGRES_HOST"),
		PostgresPort:               MustSetEnv(true, "POSTGRES_PORT"),
		PostgresDB:                 MustSetEnv(true, "POSTGRES_DB"),
		PostgresUser:               MustSetEnv(true, "POSTGRES_USER"),
		PostgresPassword:           MustSetEnv(true, "POSTGRES_PASSWORD"),
		GoogleClientID:             MustSetEnv(false, "GOOGLE_CLIENT_ID"),
		GoogleClientSecret:         MustSetEnv(false, "GOOGLE_CLIENT_SECRET"),
		GithubClientID:             MustSetEnv(false, "GITHUB_CLIENT_ID"),
		GithubClientSecret:         MustSetEnv(false, "GITHUB_CLIENT_SECRET"),
		FacebookClientID:           MustSetEnv(false, "FACEBOOK_CLIENT_ID"),
		FacebookClientSecret:       MustSetEnv(false, "FACEBOOK_CLIENT_SECRET"),
		MicrosoftClientID:          MustSetEnv(false, "MICROSOFT_CLIENT_ID"),
		MicrosoftClientSecret:      MustSetEnv(false, "MICROSOFT_CLIENT_SECRET"),
		TwilioServiceSID:           MustSetEnv(false, "TWILIO_SERVICE_SID"),
		DevUserID:                  MustSetEnv(false, "DEV_USER_ID"),
		OAuthRedirectURI:           MustSetEnv(true, "OAUTH_REDIRECT_URI"),
		// GF_CONFIG_INIT_INSERT
	}

	return cfg
}
