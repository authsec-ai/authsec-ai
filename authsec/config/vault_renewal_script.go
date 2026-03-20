package config

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// RunVaultRenewalScript executes the helper shell script to renew the Vault token.
func RunVaultRenewalScript() error {
	scriptPath, err := resolveRenewalScriptPath()
	if err != nil {
		return err
	}

	cmd := exec.Command("bash", scriptPath)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if len(output) > 0 {
		log.Printf("Vault renewal script output:\n%s", string(output))
	}
	if err != nil {
		return fmt.Errorf("vault renewal script failed: %w", err)
	}

	return nil
}

func resolveRenewalScriptPath() (string, error) {
	candidates := make([]string, 0, 4)

	candidates = append(candidates, filepath.Join("scripts", "renew_vault_token.sh"))

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "scripts", "renew_vault_token.sh"))
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "scripts", "renew_vault_token.sh"),
			filepath.Join(exeDir, "..", "scripts", "renew_vault_token.sh"),
		)
	}

	for _, candidate := range candidates {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(abs); err == nil && !info.IsDir() {
			return abs, nil
		}
	}

	return "", fmt.Errorf("could not locate renew_vault_token.sh; checked %v", candidates)
}
