package services

import (
	"context"
	"log"
	"time"

	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/internal/clients/icp"
)

// PKIRetryWorker periodically retries PKI provisioning for tenants with failed status
type PKIRetryWorker struct {
	icpProvisioningService *ICPProvisioningService
	db                     *database.DBConnection
	interval               time.Duration
}

// NewPKIRetryWorker creates a new PKI retry worker
func NewPKIRetryWorker(db *database.DBConnection, icpProvisioningService *ICPProvisioningService, interval time.Duration) *PKIRetryWorker {
	return &PKIRetryWorker{
		icpProvisioningService: icpProvisioningService,
		db:                     db,
		interval:               interval,
	}
}

// Start launches the background retry loop
func (w *PKIRetryWorker) Start() {
	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for range ticker.C {
			w.retryFailedTenants()
		}
	}()
}

// retryFailedTenants queries tenants with pki_provisioning_failed status and retries each
func (w *PKIRetryWorker) retryFailedTenants() {
	rows, err := w.db.Query("SELECT tenant_id, name, tenant_domain FROM tenants WHERE status = 'pki_provisioning_failed'")
	if err != nil {
		log.Printf("PKI retry worker: failed to query failed tenants: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var tenantID, name, domain string
		if err := rows.Scan(&tenantID, &name, &domain); err != nil {
			log.Printf("PKI retry worker: failed to scan tenant row: %v", err)
			continue
		}
		w.retryTenantPKI(tenantID, name, domain)
	}

	if err := rows.Err(); err != nil {
		log.Printf("PKI retry worker: error iterating tenant rows: %v", err)
	}
}

// retryTenantPKI retries PKI provisioning for a single tenant
func (w *PKIRetryWorker) retryTenantPKI(tenantID, name, domain string) {
	log.Printf("PKI retry worker: retrying PKI provisioning for tenant %s (%s)", tenantID, name)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	resp, err := w.icpProvisioningService.ProvisionPKI(ctx, &icp.ProvisionPKIRequest{
		TenantID:   tenantID,
		CommonName: name,
		Domain:     domain,
		TTL:        "87600h",
		MaxTTL:     "24h",
	})
	if err != nil {
		log.Printf("PKI retry worker: failed to provision PKI for tenant %s: %v", tenantID, err)
		return
	}

	_, err = w.db.Exec(
		"UPDATE tenants SET vault_mount = $1, ca_cert = $2, status = 'active', updated_at = NOW() WHERE tenant_id = $3",
		resp.PKIMount, resp.CACert, tenantID,
	)
	if err != nil {
		log.Printf("PKI retry worker: failed to update tenant %s after successful provisioning: %v", tenantID, err)
		return
	}

	log.Printf("PKI retry worker: successfully provisioned PKI for tenant %s", tenantID)
}
