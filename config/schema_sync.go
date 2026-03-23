package config

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/authsec-ai/authsec/internal/schemaaudit"
)

const (
	schemaDir             = "schema"
	runtimeMasterSchema   = "runtime_master_schema.sql"
	runtimeAuditReport    = "runtime_audit_report.md"
	generatedTemplateFile = "generated_tenant_template.sql"
)

var (
	masterSchemaCache string
)

var dropPrefixes = []string{
	"SET ",
	"SELECT pg_catalog.set_config",
	"COMMENT ON EXTENSION",
	"REVOKE ",
	"GRANT ",
}

// MasterSchemaSQL returns the cached master schema SQL captured at startup.
func MasterSchemaSQL() string {
	return masterSchemaCache
}

// RuntimeMasterSchemaPath exposes the path to the sanitized runtime schema dump.
func RuntimeMasterSchemaPath() string {
	return filepath.Join(schemaDir, runtimeMasterSchema)
}

func generatedTemplatePath() string {
	return filepath.Join(schemaDir, generatedTemplateFile)
}

func sanitizeDump(contents string) string {
	lines := strings.Split(contents, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			filtered = append(filtered, line)
			continue
		}

		if shouldDropLine(trimmed) {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n")
}

func shouldDropLine(trimmed string) bool {
	for _, prefix := range dropPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}

	if strings.HasPrefix(trimmed, "ALTER TABLE ") && strings.Contains(trimmed, " OWNER TO ") {
		return true
	}
	if strings.HasPrefix(trimmed, "ALTER SEQUENCE ") && strings.Contains(trimmed, " OWNER TO ") {
		return true
	}
	if strings.HasPrefix(trimmed, "ALTER SCHEMA ") && strings.Contains(trimmed, " OWNER TO ") {
		return true
	}
	if strings.HasPrefix(trimmed, "ALTER VIEW ") && strings.Contains(trimmed, " OWNER TO ") {
		return true
	}
	if strings.HasPrefix(trimmed, "ALTER FUNCTION ") && strings.Contains(trimmed, " OWNER TO ") {
		return true
	}
	if strings.HasPrefix(trimmed, "ALTER DATABASE ") && strings.Contains(trimmed, " OWNER TO ") {
		return true
	}
	return false
}

// runTenantTemplateSync captures the master schema and compares it with the tenant
// template. Unless DISABLE_AUTO_TENANT_TEMPLATE_UPDATE=1 it overwrites the template file.
func runTenantTemplateSync(cfg *Config) {
	if cfg == nil {
		return
	}

	if strings.EqualFold(os.Getenv("DISABLE_TENANT_SCHEMA_SYNC"), "1") {
		log.Println("Tenant schema sync disabled via DISABLE_TENANT_SCHEMA_SYNC")
		return
	}

	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		log.Printf("tenant schema sync: unable to create schema directory: %v", err)
		return
	}

	masterDumpPath := RuntimeMasterSchemaPath()
	templatePath := filepath.Join("templates", "tenant_schema_template.sql")
	genTemplatePath := generatedTemplatePath()
	auditReportPath := filepath.Join(schemaDir, runtimeAuditReport)

	dumpBuf := &bytes.Buffer{}
	cmd := exec.Command("pg_dump",
		"-h", cfg.DBHost,
		"-p", cfg.DBPort,
		"-U", cfg.DBUser,
		"--schema-only",
		"--no-owner",
		"--no-privileges",
		cfg.DBName,
	)
	cmd.Stdout = dumpBuf
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", cfg.DBPassword))

	if err := cmd.Run(); err != nil {
		log.Printf("tenant schema sync: pg_dump failed: %v", err)
		return
	}

	sanitized := sanitizeDump(dumpBuf.String())

	if err := os.WriteFile(masterDumpPath, []byte(sanitized), 0o644); err != nil {
		log.Printf("tenant schema sync: unable to write master schema dump: %v", err)
		return
	}

	masterSchemaCache = sanitized

	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		log.Printf("tenant schema sync: unable to read tenant template: %v", err)
		return
	}

	diff := schemaaudit.Analyze(sanitized, string(templateBytes))
	report := diff.Format()

	if err := os.WriteFile(auditReportPath, []byte(report), 0o644); err != nil {
		log.Printf("tenant schema sync: unable to write audit report: %v", err)
	}

	if !diff.HasDifferences() {
		log.Println("tenant schema sync: tenant template already aligned with master schema")
		return
	}

	log.Println("tenant schema sync: differences detected between master schema and tenant template")

	if err := os.WriteFile(genTemplatePath, []byte(sanitized), 0o644); err != nil {
		log.Printf("tenant schema sync: unable to write generated template copy: %v", err)
	}

	if !strings.EqualFold(os.Getenv("DISABLE_AUTO_TENANT_TEMPLATE_UPDATE"), "1") {
		if err := os.WriteFile(templatePath, []byte(sanitized), 0o644); err != nil {
			log.Printf("tenant schema sync: failed to overwrite tenant template: %v", err)
		} else {
			log.Printf("tenant schema sync: tenant schema template updated from master schema")
		}
	} else {
		log.Printf("tenant schema sync: leaving template unchanged. Generated copy at %s", genTemplatePath)
	}
}
