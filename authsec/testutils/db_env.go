package testutils

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type serviceConfig struct {
	environment map[string]string
	ports       []string
}

func parseDockerCompose(data []byte) (map[string]*serviceConfig, error) {
	services := make(map[string]*serviceConfig)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	var currentService string
	var inEnvironment bool
	var inPorts bool

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Detect service headers (two-space indent, name, colon)
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.HasSuffix(trimmed, ":") {
			currentService = strings.TrimSuffix(trimmed, ":")
			services[currentService] = &serviceConfig{environment: make(map[string]string)}
			inEnvironment = false
			inPorts = false
			continue
		}

		if currentService == "" {
			continue
		}

		if strings.HasPrefix(line, "    environment:") {
			inEnvironment = true
			inPorts = false
			continue
		}

		if strings.HasPrefix(line, "    ports:") {
			inPorts = true
			inEnvironment = false
			continue
		}

		// Reset flags when indentation decreases
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "      ") && !strings.HasPrefix(line, "    ") {
			inEnvironment = false
			inPorts = false
		}

		if inEnvironment && strings.HasPrefix(line, "      ") {
			parts := strings.SplitN(strings.TrimSpace(trimmed), ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				value = strings.Trim(value, `"`)
				services[currentService].environment[key] = value
			}
			continue
		}

		if inPorts && strings.HasPrefix(line, "      -") {
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			value = strings.Trim(value, `"`)
			services[currentService].ports = append(services[currentService].ports, value)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return services, nil
}

func findComposePath() (string, error) {
	dirs := []string{".", "..", "../.."}
	for _, dir := range dirs {
		path := filepath.Clean(filepath.Join(dir, "docker-compose.yml"))
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", errors.New("docker-compose.yml not found")
}

func setDBEnvFromDockerCompose() error {
	composePath, err := findComposePath()
	if err != nil {
		// Fallback: if DB env vars are already set, use them directly
		if os.Getenv("DB_HOST") != "" && os.Getenv("DB_USER") != "" {
			return nil
		}
		// Set sensible defaults for local testing
		defaults := map[string]string{
			"DB_HOST":     "localhost",
			"DB_PORT":     "5432",
			"DB_NAME":     "authsec",
			"DB_USER":     "postgres",
			"DB_PASSWORD": "postgres",
			"DB_SCHEMA":   "public",
		}
		for k, v := range defaults {
			if os.Getenv(k) == "" {
				os.Setenv(k, v)
			}
		}
		return nil
	}

	data, err := os.ReadFile(composePath)
	if err != nil {
		return fmt.Errorf("read docker-compose.yml: %w", err)
	}

	services, err := parseDockerCompose(data)
	if err != nil {
		return fmt.Errorf("parse docker-compose.yml: %w", err)
	}

	service, ok := services["authsec"]
	if !ok {
		return errors.New("authsec entry not found in docker-compose.yml")
	}

	setEnv := func(key, value string) {
		if value != "" {
			_ = os.Setenv(key, value)
		}
	}

	dbHost := service.environment["DB_HOST"]
	dbPort := service.environment["DB_PORT"]
	dbName := service.environment["DB_NAME"]
	dbUser := service.environment["DB_USER"]
	dbPassword := service.environment["DB_PASSWORD"]
	dbSchema := service.environment["DB_SCHEMA"]

	if dbHost == "" || dbHost == "postgres" {
		dbHost = "localhost"
	}
	if dbPort == "" {
		if pgSvc, ok := services["postgres"]; ok && len(pgSvc.ports) > 0 {
			parts := strings.Split(pgSvc.ports[0], ":")
			if len(parts) > 0 {
				dbPort = parts[0]
			}
		}
	}

	setEnv("DB_HOST", dbHost)
	setEnv("DB_PORT", dbPort)
	setEnv("DB_NAME", dbName)
	setEnv("DB_USER", dbUser)
	setEnv("DB_PASSWORD", dbPassword)
	setEnv("DB_SCHEMA", dbSchema)

	return nil
}

// SetDBEnvFromDockerCompose configures database environment variables using docker-compose.yml.
func SetDBEnvFromDockerCompose(t testing.TB) {
	t.Helper()
	if err := setDBEnvFromDockerCompose(); err != nil {
		t.Fatalf("failed to set DB env from docker compose: %v", err)
	}
}

// MustSetDBEnvFromDockerCompose configures the environment and returns an error for non-test callers.
func MustSetDBEnvFromDockerCompose() error {
	return setDBEnvFromDockerCompose()
}
