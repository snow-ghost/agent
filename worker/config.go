package worker

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds configuration for the worker
type Config struct {
	WorkerType       string
	WorkerPort       string
	LLMMode          string
	PolicyAllowTools []string
	SandboxMemMB     int
	TaskTimeout      time.Duration
	HypothesesDir    string
	ArtifactsDir     string
	LogLevel         string

	// LLM Router configuration
	LLMRouterURL string
	DefaultModel string
	ModelTag     string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	config := &Config{
		WorkerType:       getEnv("WORKER_TYPE", "heavy"),
		WorkerPort:       getEnv("WORKER_PORT", "8081"),
		LLMMode:          getEnv("LLM_MODE", "mock"),
		PolicyAllowTools: parseCommaSeparated(getEnv("POLICY_ALLOW_TOOLS", "example.com,api.example.com")),
		SandboxMemMB:     getEnvInt("SANDBOX_MEM_MB", 4),
		TaskTimeout:      getEnvDuration("TASK_TIMEOUT", "30s"),
		HypothesesDir:    getEnv("HYPOTHESES_DIR", "./hypotheses"),
		ArtifactsDir:     getEnv("ARTIFACTS_DIR", "./artifacts"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),

		// LLM Router configuration
		LLMRouterURL: getEnv("LLM_ROUTER_URL", "http://llmrouter:8090"),
		DefaultModel: getEnv("DEFAULT_MODEL", "openai:gpt-4o-mini"),
		ModelTag:     getEnv("MODEL_TAG", "general"),
	}

	return config
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an integer environment variable with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvDuration gets a duration environment variable with a default value
func getEnvDuration(key, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	duration, _ := time.ParseDuration(defaultValue)
	return duration
}

// parseCommaSeparated parses a comma-separated string into a slice
func parseCommaSeparated(value string) []string {
	if value == "" {
		return []string{}
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// LogConfig logs the configuration in a structured format
func (c *Config) LogConfig(logger interface{}) {
	// Use type assertion to check if logger has Info method
	if slogLogger, ok := logger.(interface {
		Info(msg string, args ...any)
	}); ok {
		slogLogger.Info("worker configuration loaded",
			"worker_type", c.WorkerType,
			"worker_port", c.WorkerPort,
			"llm_mode", c.LLMMode,
			"policy_allow_tools", c.PolicyAllowTools,
			"sandbox_mem_mb", c.SandboxMemMB,
			"task_timeout", c.TaskTimeout.String(),
			"hypotheses_dir", c.HypothesesDir,
			"artifacts_dir", c.ArtifactsDir,
			"log_level", c.LogLevel,
			"llm_router_url", c.LLMRouterURL,
			"default_model", c.DefaultModel,
			"model_tag", c.ModelTag,
		)
	}
}
