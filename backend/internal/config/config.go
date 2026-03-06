package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	Port          string
	OpenAIKey     string
	OpenAIBaseURL string
	OpenAIModel   string
	UploadDir     string
	ThumbnailDir  string
	DBPath        string

	GitHubSyncEnabled bool
	GitHubToken       string
	GitHubOwner       string
	GitHubRepo        string
	GitHubBranch      string
	GitHubBaseDir     string
}

func Load() *Config {
	loadDotEnv(".env")
	loadDotEnv("backend/.env")

	cfg := &Config{
		Port:          getEnv("PORT", "8080"),
		OpenAIKey:     getEnv("OPENAI_API_KEY", ""),
		OpenAIBaseURL: getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OpenAIModel:   getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		UploadDir:     getEnv("UPLOAD_DIR", "./uploads"),
		ThumbnailDir:  getEnv("THUMBNAIL_DIR", "./thumbnails"),
		DBPath:        getEnv("DB_PATH", "./skill_hub.db"),

		GitHubSyncEnabled: getEnvBool("GITHUB_SYNC_ENABLED", false),
		GitHubToken:       getEnv("GITHUB_TOKEN", ""),
		GitHubOwner:       getEnv("GITHUB_OWNER", ""),
		GitHubRepo:        getEnv("GITHUB_REPO", ""),
		GitHubBranch:      getEnv("GITHUB_BRANCH", "main"),
		GitHubBaseDir:     getEnv("GITHUB_BASE_DIR", "skills"),
	}
	return cfg
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}

		value = trimDotEnvValue(value)
		if os.Getenv(key) != "" {
			continue
		}
		_ = os.Setenv(key, value)
	}
}

func trimDotEnvValue(value string) string {
	if len(value) < 2 {
		return value
	}

	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		return strings.Trim(value, "\"")
	}

	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return strings.Trim(value, "'")
	}

	if idx := strings.Index(value, " #"); idx >= 0 {
		return strings.TrimSpace(value[:idx])
	}
	return value
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}

	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
