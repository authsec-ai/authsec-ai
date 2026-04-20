// Package spire provides the merged authsec-spire service as a sub-module
// within the authsec monolith. It registers all SPIRE identity service routes
// under /authsec/spiresvc.
package spire

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/authsec-ai/authsec/internal/spire/controllers"
	"github.com/authsec-ai/authsec/internal/spire/middleware"
	"github.com/authsec-ai/authsec/internal/spire/services"
)

// Dependencies holds all controller and middleware instances needed for routing.
type Dependencies struct {
	// Controllers
	Health          *controllers.HealthController
	NodeAttestation *controllers.NodeAttestationController
	Agent           *controllers.AgentController
	Attestation     *controllers.AttestationController
	Workload        *controllers.WorkloadController
	Certificate     *controllers.CertificateController
	JWTSVID         *controllers.JWTSVIDController
	Bundle          *controllers.BundleController
	PKIAdmin        *controllers.PKIAdminController

	// Services (exposed for injection into existing monolith controllers)
	PKIProvisioningSvc  *services.PKIProvisioningService
	WorkloadEntrySvc    *services.WorkloadEntryService
	JWTSVIDSvc          *services.JWTSVIDService
	AgentSvc            *services.AgentService

	// Middleware
	JWTAuth   gin.HandlerFunc
	AgentCert gin.HandlerFunc
	MTLSAuth  gin.HandlerFunc

	Logger *logrus.Entry
}

// RegisterRoutes registers all SPIRE identity service routes on the given router group.
// The caller is expected to pass a group like router.Group("/authsec/spiresvc").
func RegisterRoutes(rg *gin.RouterGroup, deps *Dependencies) {
	// ────────────────────────────────────────
	// Public / Bootstrap endpoints (no auth)
	// ────────────────────────────────────────
	rg.GET("/health", deps.Health.Health)
	rg.POST("/v1/node/attest", middleware.BootstrapLimiter.Middleware(), deps.NodeAttestation.Attest)
	rg.POST("/v1/agent/renew", middleware.BootstrapLimiter.Middleware(), deps.Agent.RenewAgent)
	rg.GET("/bundle/:tenant", deps.Bundle.GetBundle)
	rg.GET("/v1/jwt/bundle", deps.JWTSVID.GetJWTBundle)
	rg.POST("/v1/jwt/validate", middleware.SensitiveLimiter.Middleware(), deps.JWTSVID.ValidateJWTSVID)
	rg.POST("/v1/jwt/renew", middleware.SensitiveLimiter.Middleware(), deps.JWTSVID.RenewJWTSVID)
	rg.POST("/admin/pki/provision", middleware.SensitiveLimiter.Middleware(), deps.PKIAdmin.ProvisionPKI)
	rg.POST("/admin/pki/provision/:tenant_id", middleware.SensitiveLimiter.Middleware(), deps.PKIAdmin.ProvisionPKIForTenant)

	// ────────────────────────────────────────
	// Agent-protected endpoints
	// ────────────────────────────────────────
	agentGroup := rg.Group("/v1", deps.AgentCert, middleware.StandardLimiter.Middleware())
	{
		agentGroup.GET("/entries/by-parent", deps.Workload.ListEntriesByParent)
		agentGroup.POST("/workload/attest", deps.Workload.AttestWorkload)
		agentGroup.POST("/workload/revoke", deps.Workload.RevokeWorkloadSVID)
	}

	// ────────────────────────────────────────
	// JWT-protected endpoints (admin / user-flow UI)
	// ────────────────────────────────────────
	jwtGroup := rg.Group("/v1", deps.JWTAuth)
	{
		// Agents
		jwtGroup.GET("/agents", deps.Agent.ListAgents)

		// Workload entries CRUD
		jwtGroup.POST("/entries", deps.Workload.CreateEntry)
		jwtGroup.GET("/entries", deps.Workload.ListEntries)
		jwtGroup.GET("/entries/:id", deps.Workload.GetEntry)
		jwtGroup.PUT("/entries/:id", deps.Workload.UpdateEntry)
		jwtGroup.DELETE("/entries/:id", deps.Workload.DeleteEntry)
		jwtGroup.POST("/entries/agent", deps.Workload.CreateAgentEntry)

		// Delegated JWT-SVID
		jwtGroup.POST("/jwt/issue-delegated", deps.JWTSVID.IssueDelegatedJWTSVID)
	}

	// ────────────────────────────────────────
	// mTLS-protected endpoints (service-to-service)
	// ────────────────────────────────────────
	mtlsGroup := rg.Group("/v1", deps.MTLSAuth)
	{
		mtlsGroup.POST("/attest", deps.Attestation.Attest)
		mtlsGroup.POST("/renew", deps.Certificate.Renew)
		mtlsGroup.POST("/revoke", deps.Certificate.Revoke)
		mtlsGroup.POST("/jwt/issue", deps.JWTSVID.IssueJWTSVID)
	}

	deps.Logger.Info("SPIRE identity service routes registered under /authsec/spiresvc")
}
