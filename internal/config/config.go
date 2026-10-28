package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr      string
	ChromePath      string
	BrowserEndpoint string
	Headless        bool
	DefaultTimeout  time.Duration
	UserAgent       string
}

func Load() *Config {
	cfg := &Config{
		ListenAddr:      getEnv("LISTEN_ADDR", ":8080"),
		ChromePath:      getEnv("CHROME_PATH", ""),
		BrowserEndpoint: getEnv("BROWSER_WS_ENDPOINT", ""),
		Headless:        getEnvBool("HEADLESS", true),
		DefaultTimeout:  getEnvDuration("DEFAULT_TIMEOUT", 30*time.Second),
		UserAgent:       getEnv("USER_AGENT", "MonitorEngine-Go/2.0"),
	}
	return cfg
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
