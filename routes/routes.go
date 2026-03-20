// Package routes wires together all HTTP routes for the merged authsec monolith.
//
// All API routes are served under the /authsec prefix:
//
//	/authsec/auth/*          – admin and end-user authentication
//	/authsec/webauthn/*      – WebAuthn/FIDO2 passkey flows
//	/authsec/admin/*         – admin management (tenants, users, RBAC, OIDC, …)
//	/authsec/user/*          – end-user self-service
//	/authsec/oidc/*          – OIDC federation
//	/authsec/scim/v2/*       – SCIM 2.0 provisioning
//	/authsec/health          – health checks
//	/authsec/debug/*         – debug helpers (dev only)
//
// The well-known OIDC discovery endpoints remain at the root as required by RFC 8414:
//
//	/.well-known/openid-configuration
//	/.well-known/jwks.json
//
// All merged microservice routes are under /authsec:
//
//	/authsec/uflow/*      – user flow (formerly user-flow)
//	/authsec/webauthn/*   – WebAuthn/passkeys (formerly webauthn-service)
//	/authsec/exsvc/*      – external services (formerly mcp-service/external-service)
//	/authsec/clientms/*   – client management (formerly clients-microservice)
//	/authsec/hmgr/*       – Hydra manager (formerly hydra-service)
//	/authsec/oocmgr/*     – OIDC config manager (formerly oath_oidc_configuration_manager)
//	/authsec/authmgr/*    – Auth manager (formerly auth-manager)
//	/authsec/sdkmgr/*     – SDK manager (formerly sdk-manager Python service)
package routes

import (
	"log"
	"net/http"
	"time"

	amMiddlewares "github.com/authsec-ai/auth-manager/pkg/middlewares"
	"github.com/authsec-ai/authsec/config"
	adminCtrl "github.com/authsec-ai/authsec/controllers/admin"
	userCtrl "github.com/authsec-ai/authsec/controllers/enduser"
	platformCtrl "github.com/authsec-ai/authsec/controllers/platform"
	sdkmgrCtrl "github.com/authsec-ai/authsec/controllers/sdkmgr"
	sharedCtrl "github.com/authsec-ai/authsec/controllers/shared"
	"github.com/authsec-ai/authsec/handlers"
	"github.com/authsec-ai/authsec/middlewares"
	sdkmgrSvc "github.com/authsec-ai/authsec/services/sdkmgr"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRoutes registers all HTTP routes on the provided Gin engine.
// It accepts the initialised WebAuthn handler structs so the caller
// (main.go) controls their lifecycle.
func SetupRoutes(
	r *gin.Engine,
	webAuthnHandler *handlers.WebAuthnHandler,
	adminWebAuthnHandler *handlers.AdminWebAuthnHandler,
	endUserWebAuthnHandler *handlers.EndUserWebAuthnHandler,
) {
	// ────────────────────────────────────────────────────────
	// CORS is already applied by the caller (main.go)
	// ────────────────────────────────────────────────────────

	// ────────────────────────────────────────────────────────
	// Initialise controllers
	// ────────────────────────────────────────────────────────
	userController, err := adminCtrl.NewUserController()
	if err != nil {
		log.Fatalf("Failed to initialize user controller: %v", err)
	}
	adminAuthController, err := adminCtrl.NewAdminAuthController()
	if err != nil {
		log.Fatalf("Failed to initialize admin auth controller: %v", err)
	}
	adminUserController, err := adminCtrl.NewAdminUserController()
	if err != nil {
		log.Fatalf("Failed to initialize admin user controller: %v", err)
	}
	endUserAuthController, err := userCtrl.NewEndUserAuthController()
	if err != nil {
		log.Fatalf("Failed to initialize end-user auth controller: %v", err)
	}
	projectController := &adminCtrl.ProjectController{}

	// Scoped RBAC Controllers
	rolesScopedBindingsController := adminCtrl.NewRolesScopedBindingsController()
	authController := platformCtrl.NewAuthorizationController()
	permissionController := adminCtrl.NewPermissionController()
	scopeController := adminCtrl.NewScopeController()
	apiScopesController := adminCtrl.NewAPIScopesController()

	// AI Agent Delegation controllers
	agentController := adminCtrl.NewAgentController()
	delegationPolicyController := adminCtrl.NewDelegationPolicyController()
	sdkTokenController := adminCtrl.NewSDKTokenController()

	// Legacy / existing controllers
	groupController := &adminCtrl.GroupController{}
	endUserController := &userCtrl.EndUserController{}
	adSyncController := &sharedCtrl.ADSyncController{}
	entraIDController := &sharedCtrl.EntraIDController{}
	syncConfigController := &adminCtrl.SyncConfigController{}
	healthController := &sharedCtrl.HealthController{}
	adminInviteController, err := adminCtrl.NewAdminInviteController()
	if err != nil {
		log.Fatalf("Failed to initialize admin invite controller: %v", err)
	}

	domainController := adminCtrl.NewDomainController(config.GetDatabase())
	hubspotController := platformCtrl.NewHubSpotController()

	scimController := &platformCtrl.SCIMController{}
	scimAdminController, err := adminCtrl.NewSCIMAdminController()
	if err != nil {
		log.Fatalf("Failed to initialize SCIM admin controller: %v", err)
	}

	_ = middlewares.NewTenantResolutionMiddleware(config.GetDatabase())

	oidcController, err := platformCtrl.NewOIDCController()
	if err != nil {
		log.Fatalf("Failed to initialize OIDC controller: %v", err)
	}
	adminSyncController, err := adminCtrl.NewAdminSyncController()
	if err != nil {
		log.Fatalf("Failed to initialize admin sync controller: %v", err)
	}

	deviceAuthController, err := userCtrl.NewDeviceAuthController()
	if err != nil {
		log.Fatalf("Failed to initialize device auth controller: %v", err)
	}

	voiceAuthController, err := userCtrl.NewVoiceAuthController()
	if err != nil {
		log.Fatalf("Failed to initialize voice auth controller: %v", err)
	}

	totpController, err := userCtrl.NewTOTPController()
	if err != nil {
		log.Fatalf("Failed to initialize TOTP controller: %v", err)
	}

	cibaAuthController, err := userCtrl.NewCIBAAuthController()
	if err != nil {
		log.Fatalf("Failed to initialize CIBA auth controller: %v", err)
	}

	tenantCIBAController, err := userCtrl.NewTenantCIBAController()
	if err != nil {
		log.Fatalf("Failed to initialize tenant CIBA auth controller: %v", err)
	}

	tenantTOTPController := userCtrl.NewTenantTOTPController()

	spiffeDelegateController, err := platformCtrl.NewSpiffeDelegateController()
	if err != nil {
		log.Fatalf("Failed to initialize SPIFFE delegate controller: %v", err)
	}

	// ────────────────────────────────────────────────────────
	// Well-known OIDC discovery – must remain at root (RFC 8414)
	// ────────────────────────────────────────────────────────
	r.GET("/.well-known/openid-configuration", spiffeDelegateController.OIDCDiscovery)
	r.GET("/.well-known/jwks.json", spiffeDelegateController.GetJWKS)

	// Catch-all OPTIONS handler so CORS preflight requests are answered for every
	// path regardless of which method-specific route is registered.
	r.OPTIONS("/*path", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	// Backward-compat: user-flow previously exposed this at the bare root so that
	// existing webauthn-service clients did not need to change their URLs.
	r.POST("/webauthn/mfa/loginStatus", userController.WebAuthnMFALoginStatus)

	// ════════════════════════════════════════════════════════
	// ALL ROUTES UNDER /authsec
	// ════════════════════════════════════════════════════════
	authsec := r.Group("/authsec")
	{
		// ────────────────────────────────────────────────────
		// WebAuthn routes  (/authsec/webauthn/*)
		// Served under /authsec/webauthn (formerly webauthn-service).
		// ────────────────────────────────────────────────────
		registerWebAuthnRoutes(authsec.Group("/webauthn"), webAuthnHandler, adminWebAuthnHandler, endUserWebAuthnHandler)

		// ────────────────────────────────────────────────────
		// User Flow (formerly user-flow)
		// Served under /authsec/uflow.
		// ────────────────────────────────────────────────────
		uflow := authsec.Group("/uflow")

		// Device activation page (public)
		uflow.GET("/activate", deviceAuthController.ShowActivationPage)

		// ────────────────────────────────────────────────────
		// API docs
		// ────────────────────────────────────────────────────
		uflow.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		uflow.GET("/docs", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			html := `<!DOCTYPE html>
						<html>
						<head>
							<title>AuthSec API Documentation</title>
							<meta charset="utf-8"/>
							<meta name="viewport" content="width=device-width, initial-scale=1">
							<link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
							<style>body { margin: 0; padding: 0; }</style>
						</head>
						<body>
							<redoc spec-url='/authsec/uflow/swagger/doc.json'></redoc>
							<script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
						</body>
						</html>`
			c.String(http.StatusOK, html)
		})
		uflow.GET("/apidocs", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"title":   "AuthSec API",
				"version": "5.0.0",
				"status":  "available",
			})
		})
		uflow.GET("/apidocs/*any", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "API documentation available at /authsec/uflow/docs"})
		})

		// ────────────────────────────────────────────────────
		// Admin RBAC routes
		// ────────────────────────────────────────────────────
		adminRBAC := uflow.Group("/admin")
		adminRBAC.Use(
			middlewares.AuthMiddleware(),
			middlewares.Require("admin", "access"),
			amMiddlewares.ValidateTenantFromToken(),
		)
		{
			adminRBAC.POST("/roles", rolesScopedBindingsController.CreateRoleCompositeAdmin)
			adminRBAC.GET("/roles", rolesScopedBindingsController.ListRolesAdmin)
			adminRBAC.GET("/roles/:role_id", rolesScopedBindingsController.GetRoleAdmin)
			adminRBAC.PUT("/roles/:role_id", rolesScopedBindingsController.UpdateRoleCompositeAdmin)
			adminRBAC.DELETE("/roles/:role_id", rolesScopedBindingsController.DeleteRoleAdmin)
			adminRBAC.POST("/bindings", rolesScopedBindingsController.AssignRoleScopedAdmin)
			adminRBAC.GET("/bindings", rolesScopedBindingsController.ListRoleBindingsAdmin)
			adminRBAC.POST("/permissions", permissionController.RegisterAtomicPermission)
			adminRBAC.GET("/permissions", permissionController.ListPermissions)
			adminRBAC.DELETE("/permissions/:id", permissionController.DeletePermission)
			adminRBAC.DELETE("/permissions", permissionController.DeletePermissionByBody)
			adminRBAC.GET("/permissions/resources", permissionController.ShowResources)
			adminRBAC.GET("/scopes", scopeController.ListScopes)
			adminRBAC.GET("/scopes/mappings", scopeController.GetMappings)
			adminRBAC.POST("/scopes", scopeController.AddScope)
			adminRBAC.PUT("/scopes/:scope_name", scopeController.EditScope)
			adminRBAC.DELETE("/scopes/:scope_name", scopeController.DeleteScope)
			adminRBAC.POST("/policy/check", authController.PolicyDecisionPointCheckAdmin)
			adminRBAC.POST("/api_scopes", apiScopesController.CreateAPIScopeAdmin)
			adminRBAC.GET("/api_scopes", apiScopesController.ListAPIScopesAdmin)
			adminRBAC.GET("/api_scopes/:scope_id", apiScopesController.GetAPIScopeAdmin)
			adminRBAC.PUT("/api_scopes/:scope_id", apiScopesController.UpdateAPIScopeAdmin)
			adminRBAC.DELETE("/api_scopes/:scope_id", apiScopesController.DeleteAPIScopeAdmin)

			// AI Agent Management
			adminRBAC.GET("/agents", agentController.ListAgents)
			adminRBAC.GET("/agents/:id", agentController.GetAgent)
			adminRBAC.POST("/agents/:id/provision-identity", agentController.ProvisionIdentity)
			adminRBAC.DELETE("/agents/:id/revoke-identity", agentController.RevokeIdentity)
			adminRBAC.POST("/agents/:id/delegate-token", agentController.DelegateToken)
			adminRBAC.POST("/agents/:id/revoke-token", sdkTokenController.RevokeDelegationToken)

			// Admin self-introspection (delegation UI)
			adminRBAC.GET("/me/roles-permissions", delegationPolicyController.GetMyRolesAndPermissions)
		}

		// ────────────────────────────────────────────────────
		// OIDC public endpoints
		// ────────────────────────────────────────────────────
		oidcPublic := uflow.Group("/oidc")
		{
			oidcPublic.GET("/providers", oidcController.GetProviders)
			oidcPublic.POST("/initiate", oidcController.Initiate)
			oidcPublic.POST("/register/initiate", oidcController.InitiateRegistration)
			oidcPublic.POST("/login/initiate", oidcController.InitiateLogin)
			oidcPublic.GET("/callback", oidcController.Callback)
			oidcPublic.POST("/exchange-code", oidcController.ExchangeCode)
			oidcPublic.POST("/complete-registration", oidcController.CompleteRegistration)
			oidcPublic.GET("/check-tenant", oidcController.CheckTenantExists)
			oidcPublic.POST("/auth-url", oidcController.GetAuthURL)
		}

		// Authenticated OIDC endpoints
		oidcAuth := uflow.Group("/oidc")
		oidcAuth.Use(middlewares.AuthMiddleware(), amMiddlewares.ValidateTenantFromToken())
		{
			oidcAuth.POST("/link", oidcController.LinkIdentity)
			oidcAuth.GET("/identities", oidcController.GetLinkedIdentities)
			oidcAuth.DELETE("/unlink/:provider", oidcController.UnlinkIdentity)
		}

		// ────────────────────────────────────────────────────
		// Authentication routes
		// ────────────────────────────────────────────────────
		auth := uflow.Group("/auth")
		{
			notify := auth.Group("/notify")
			notify.Use(middlewares.AuthMiddleware(), amMiddlewares.ValidateTenantFromToken())
			{
				notify.POST("/new-user-registration", endUserController.NotifyOwnerNewRegistration)
			}

			// Admin authentication (strict rate limit: 5 req/min)
			adminAuth := auth.Group("/admin")
			adminAuth.Use(middlewares.StrictAuthRateLimitMiddleware(5, time.Minute))
			{
				adminAuth.GET("/challenge", adminAuthController.GetAuthChallenge)
				adminAuth.POST("/login/precheck", adminAuthController.AdminLoginPrecheck)
				adminAuth.POST("/login/bootstrap", adminAuthController.AdminBootstrap)
				adminAuth.POST("/login", adminAuthController.AdminLogin)
				adminAuth.POST("/login-hybrid", adminAuthController.AdminLoginHybrid)
				adminAuth.POST("/register", adminAuthController.AdminRegister)
				adminAuth.POST("/complete-registration", adminAuthController.AdminCompleteRegistration)
				adminAuth.POST("/forgot-password", adminAuthController.AdminForgotPassword)
				adminAuth.POST("/forgot-password/verify-otp", adminAuthController.AdminVerifyOTP)
				adminAuth.POST("/forgot-password/reset", adminAuthController.AdminResetPassword)
			}

			// End-user authentication (strict rate limit: 10 req/min)
			enduserAuth := auth.Group("/enduser")
			enduserAuth.Use(middlewares.StrictAuthRateLimitMiddleware(10, time.Minute))
			{
				enduserAuth.GET("/challenge", endUserAuthController.GetAuthChallenge)
				enduserAuth.POST("/initiate-registration", endUserAuthController.InitiateRegistration)
				enduserAuth.POST("/verify-otp", endUserAuthController.VerifyOTPAndCompleteRegistration)
				enduserAuth.POST("/login/precheck", endUserAuthController.EndUserLoginPrecheck)
				enduserAuth.POST("/webauthn-callback", endUserAuthController.WebAuthnCallback)
				enduserAuth.POST("/delegate-svid", spiffeDelegateController.DelegateSVID)
			}

			// Device Authorization Grant (RFC 8628)
			deviceAuth := auth.Group("/device")
			{
				deviceAuth.POST("/code", deviceAuthController.RequestDeviceCode)
				deviceAuth.POST("/token", deviceAuthController.PollDeviceToken)
				deviceAuth.GET("/activate/info", deviceAuthController.GetActivationInfo)
				deviceAuth.POST("/verify", middlewares.AuthMiddleware(), deviceAuthController.VerifyDeviceCode)
			}

			// Voice Authentication
			voiceAuth := auth.Group("/voice")
			{
				voiceAuth.POST("/initiate", voiceAuthController.InitiateVoiceAuth)
				voiceAuth.POST("/verify", voiceAuthController.VerifyVoiceOTP)
				voiceAuth.POST("/token", voiceAuthController.GetTokenWithCredentials)
				voiceAuth.POST("/link", middlewares.AuthMiddleware(), voiceAuthController.LinkVoiceAssistant)
				voiceAuth.POST("/unlink", middlewares.AuthMiddleware(), voiceAuthController.UnlinkVoiceAssistant)
				voiceAuth.GET("/links", middlewares.AuthMiddleware(), voiceAuthController.ListVoiceLinks)
				voiceAuth.GET("/device-pending", middlewares.AuthMiddleware(), voiceAuthController.GetPendingDeviceCodes)
				voiceAuth.POST("/device-approve", middlewares.AuthMiddleware(), voiceAuthController.ApproveDeviceCode)
			}

			// TOTP
			totp := auth.Group("/totp")
			{
				totp.POST("/login", totpController.LoginWithTOTP)
				totp.POST("/device-approve", totpController.ApproveDeviceCodeWithTOTP)
				totp.POST("/register", middlewares.AuthMiddleware(), totpController.RegisterDevice)
				totp.POST("/confirm", middlewares.AuthMiddleware(), totpController.ConfirmRegistration)
				totp.POST("/verify", middlewares.AuthMiddleware(), totpController.VerifyTOTP)
				totp.GET("/devices", middlewares.AuthMiddleware(), totpController.GetUserDevices)
				totp.POST("/device/delete", middlewares.AuthMiddleware(), totpController.DeleteDevice)
				totp.POST("/device/primary", middlewares.AuthMiddleware(), totpController.SetPrimaryDevice)
				totp.POST("/backup/regenerate", middlewares.AuthMiddleware(), totpController.RegenerateBackupCodes)
			}

			// CIBA
			ciba := auth.Group("/ciba")
			{
				ciba.POST("/initiate", cibaAuthController.InitiateCIBAAuth)
				ciba.POST("/token", cibaAuthController.PollCIBAToken)
				ciba.POST("/respond", middlewares.AuthMiddleware(), cibaAuthController.RespondToCIBA)
				ciba.POST("/register-device", middlewares.AuthMiddleware(), cibaAuthController.RegisterDevice)
				ciba.GET("/devices", middlewares.AuthMiddleware(), cibaAuthController.GetDevices)
				ciba.DELETE("/devices/:device_id", middlewares.AuthMiddleware(), cibaAuthController.DeleteDevice)
			}
		}

		// ────────────────────────────────────────────────────
		// Tenant auth routes
		// ────────────────────────────────────────────────────
		tenantAuth := uflow.Group("/auth/tenant")
		{
			tenantCIBA := tenantAuth.Group("/ciba")
			{
				tenantCIBA.POST("/initiate", tenantCIBAController.InitiateTenantCIBA)
				tenantCIBA.POST("/token", tenantCIBAController.PollTenantCIBAToken)
				tenantCIBA.POST("/respond", middlewares.AuthMiddleware(), amMiddlewares.ValidateTenantFromToken(), tenantCIBAController.RespondToTenantCIBA)
				tenantCIBA.POST("/register-device", middlewares.AuthMiddleware(), amMiddlewares.ValidateTenantFromToken(), tenantCIBAController.RegisterTenantDevice)
				tenantCIBA.GET("/requests", middlewares.AuthMiddleware(), amMiddlewares.ValidateTenantFromToken(), tenantCIBAController.GetTenantCIBARequests)
				tenantCIBA.GET("/devices", middlewares.AuthMiddleware(), amMiddlewares.ValidateTenantFromToken(), tenantCIBAController.ListTenantDevices)
				tenantCIBA.DELETE("/devices/:device_id", middlewares.AuthMiddleware(), amMiddlewares.ValidateTenantFromToken(), tenantCIBAController.DeleteTenantDevice)
			}

			tenantTOTP := tenantAuth.Group("/totp")
			{
				tenantTOTP.POST("/login", tenantTOTPController.LoginWithTenantTOTP)
				tenantTOTP.POST("/register", middlewares.AuthMiddleware(), amMiddlewares.ValidateTenantFromToken(), tenantTOTPController.RegisterTenantTOTPDevice)
				tenantTOTP.POST("/confirm", middlewares.AuthMiddleware(), amMiddlewares.ValidateTenantFromToken(), tenantTOTPController.ConfirmTenantTOTPDevice)
				tenantTOTP.GET("/devices", middlewares.AuthMiddleware(), amMiddlewares.ValidateTenantFromToken(), tenantTOTPController.GetTenantTOTPDevices)
				tenantTOTP.POST("/devices/delete", middlewares.AuthMiddleware(), amMiddlewares.ValidateTenantFromToken(), tenantTOTPController.DeleteTenantTOTPDevice)
				tenantTOTP.POST("/devices/primary", middlewares.AuthMiddleware(), amMiddlewares.ValidateTenantFromToken(), tenantTOTPController.SetTenantPrimaryTOTPDevice)
			}
		}

		// ────────────────────────────────────────────────────
		// Admin management routes
		// ────────────────────────────────────────────────────
		admin := uflow.Group("/admin")
		admin.Use(
			middlewares.AuthMiddleware(),
			middlewares.Require("admin", "access"),
			amMiddlewares.ValidateTenantFromToken(),
		)
		{
			admin.GET("/tenants", adminUserController.ListTenants)
			admin.POST("/tenants", adminUserController.CreateTenant)
			admin.PUT("/tenants/:tenant_id", adminUserController.UpdateTenant)
			admin.DELETE("/tenants/:tenant_id", middlewares.Require("tenants", "delete"), adminUserController.DeleteTenant)
			admin.GET("/tenants/:tenant_id/users", adminUserController.GetTenantUsers)
			admin.GET("/users/list", adminUserController.ListAdminUsers)
			admin.POST("/users/list", adminUserController.ListAdminUsers)
			admin.DELETE("/users/:user_id", middlewares.Require("users", "delete"), adminUserController.DeleteAdminUser)
			admin.DELETE("/users/delete_all/:user_id", middlewares.Require("users", "delete"), adminUserController.DeleteAdminUserAll)
			admin.POST("/enduser/list", adminUserController.ListEndUsersByTenant)
			admin.POST("/invite", adminInviteController.InviteAdmin)
			admin.POST("/invite/cancel", adminInviteController.CancelInvite)
			admin.POST("/invite/resend", adminInviteController.ResendInvite)
			admin.GET("/invite/pending", adminInviteController.ListPendingInvites)

			adminDomains := admin.Group("/tenants/:tenant_id/domains")
			adminDomains.Use(middlewares.ExtractTenantFromPath())
			{
				adminDomains.POST("", domainController.CreateDomain)
				adminDomains.GET("", domainController.ListDomains)
				adminDomains.POST("/:domain_id/verify", domainController.VerifyDomain)
				adminDomains.POST("/:domain_id/set-primary", domainController.SetPrimaryDomain)
				adminDomains.GET("/:domain_id", domainController.GetDomainByID)
				adminDomains.DELETE("/:domain_id", domainController.DeleteDomain)
			}
		}

		// Platform admin routes
		adminPlatform := uflow.Group("/admin")
		adminPlatform.Use(
			middlewares.AuthMiddleware(),
			middlewares.Require("admin", "access"),
			amMiddlewares.ValidateTenantFromToken(),
		)
		{
			adminPlatform.GET("/oidc/providers", oidcController.GetAllProviders)
			adminPlatform.PUT("/oidc/providers/:provider", oidcController.UpdateProvider)
			adminPlatform.POST("/projects", projectController.CreateProject)
			adminPlatform.GET("/projects", projectController.ListProjects)
			adminPlatform.POST("/users/active", adminUserController.ToggleAdminUserActive)
			adminPlatform.POST("/groups", groupController.AddUserDefinedGroups)
			adminPlatform.POST("/groups/map", groupController.MapGroupsToClient)
			adminPlatform.POST("/groups/list", groupController.ListTenantGroupsForAdmin)
			adminPlatform.DELETE("/groups/map", groupController.RemoveGroupsFromClient)
			adminPlatform.GET("/groups/:tenant_id", groupController.GetUserDefinedGroups)
			adminPlatform.PUT("/groups/:id", groupController.UpdateUserDefinedGroup)
			adminPlatform.DELETE("/groups", groupController.DeleteUserDefinedGroups)
			adminPlatform.POST("/groups/:tenant_id/users/bulk", groupController.AddUsersToGroup)
			adminPlatform.DELETE("/groups/:tenant_id/users/bulk", groupController.RemoveUsersFromGroup)
			adminPlatform.POST("/enduser/active", adminUserController.ToggleEndUserActive)
			adminPlatform.POST("/ad/sync", adSyncController.SyncADUsers)
			adminPlatform.POST("/ad/test-connection", adSyncController.TestADConnection)
			adminPlatform.POST("/ad/test-network", adSyncController.TestNetworkConnection)
			adminPlatform.POST("/ad/agent-sync", adSyncController.AgentSyncUsers)
			adminPlatform.POST("/entra/sync", entraIDController.SyncEntraIDUsers)
			adminPlatform.POST("/entra/test-connection", entraIDController.TestEntraIDConnection)
			adminPlatform.POST("/entra/check-permissions", entraIDController.GetEntraIDPermissions)
			adminPlatform.POST("/sync-configs/create", syncConfigController.CreateSyncConfig)
			adminPlatform.POST("/sync-configs/list", syncConfigController.ListSyncConfigs)
			adminPlatform.POST("/sync-configs/update", syncConfigController.UpdateSyncConfig)
			adminPlatform.POST("/sync-configs/delete", syncConfigController.DeleteSyncConfig)
			adminPlatform.POST("/admin-users/ad/sync", adminSyncController.SyncADAdminUsers)
			adminPlatform.POST("/admin-users/entra/sync", adminSyncController.SyncEntraAdminUsers)
		}

		// SCIM token
		scimToken := uflow.Group("/admin/scim")
		scimToken.Use(middlewares.AuthMiddleware(), amMiddlewares.ValidateTenantFromToken())
		{
			scimToken.POST("/generate-token", scimController.GenerateSCIMToken)
		}

		// ────────────────────────────────────────────────────
		// End-user admin scopes
		// ────────────────────────────────────────────────────
		enduserAdmin := uflow.Group("/enduser")
		enduserAdmin.Use(
			middlewares.AuthMiddleware(),
			middlewares.Require("admin", "access"),
			amMiddlewares.ValidateTenantFromToken(),
		)
		{
			enduserAdmin.GET("/scopes", scopeController.ListUserScopes)
			enduserAdmin.GET("/scopes/:tenant_id", scopeController.ListUserScopes)
			enduserAdmin.GET("/scopes/mappings", scopeController.GetUserMappings)
			enduserAdmin.POST("/scopes", scopeController.AddUserScope)
			enduserAdmin.PUT("/scopes/:scope_name", scopeController.EditUserScope)
			enduserAdmin.DELETE("/scopes/:scope_name", scopeController.DeleteUserScope)
		}

		// ────────────────────────────────────────────────────
		// End-user self-service routes
		// ────────────────────────────────────────────────────
		user := uflow.Group("/user")
		{
			user.POST("/login", endUserController.CustomLogin)
			user.POST("/login/status", endUserController.CustomLoginStatus)
			user.POST("/saml/login", endUserAuthController.SAMLLogin)
			user.POST("/register/initiate", endUserController.InitiateCustomLoginRegister)
			user.POST("/register/complete", endUserController.CompleteCustomLoginRegister)
			user.POST("/register", endUserController.CustomLoginRegister)
			user.POST("/forgot-password", endUserController.CustomForgotPassword)
			user.POST("/forgot-password/verify-otp", endUserController.CustomVerifyPasswordResetOTP)
			user.POST("/forgot-password/reset", endUserController.CustomResetPassword)
			user.POST("/oidc/login", endUserController.OIDCLogin)
		}

		user.Use(middlewares.AuthMiddleware(), amMiddlewares.ValidateTenantFromToken())
		{
			user.POST("/clients/register", endUserController.RegisterClient)
			user.GET("/clients", endUserController.GetClients)
			user.POST("/clients/get", endUserController.GetClientsPost)
			user.GET("/enduser/:tenant_id/:user_id", endUserController.GetEndUser)
			user.POST("/enduser/list", endUserController.GetEndUsers)
			user.GET("/enduser/list", endUserController.GetEndUsers)
			user.GET("/enduser/databases", endUserController.GetTenantDatabases)
			user.PUT("/enduser/:tenant_id/:user_id", endUserController.UpdateUser)
			user.PUT("/enduser/:tenant_id/:user_id/status", endUserController.UpdateEndUserStatus)
			user.POST("/enduser/active", endUserController.ActiveOrDeactiveEndUser)
			user.POST("/enduser/delete", endUserController.DeleteEndUser)
			user.DELETE("/enduser/:tenant_id/:user_id", middlewares.Require("users", "delete"), endUserController.DeleteEndUser)
			user.DELETE("/enduser/delete_all/:tenant_id/:user_id", middlewares.Require("users", "delete"), endUserController.DeleteUserAll)
			user.POST("/rbac/roles", rolesScopedBindingsController.CreateRoleCompositeEndUser)
			user.GET("/rbac/roles", rolesScopedBindingsController.ListRolesEndUser)
			user.PUT("/rbac/roles/:role_id", rolesScopedBindingsController.UpdateRoleCompositeEndUser)
			user.DELETE("/rbac/roles/:role_id", rolesScopedBindingsController.DeleteRoleEndUser)
			user.POST("/rbac/bindings", rolesScopedBindingsController.AssignRoleScopedEndUser)
			user.GET("/rbac/bindings", rolesScopedBindingsController.ListRoleBindingsEndUser)
			user.GET("/rbac/permissions", permissionController.ListPermissionsEndUser)
			user.POST("/rbac/permissions", permissionController.RegisterAtomicPermissionEndUser)
			user.DELETE("/rbac/permissions/:id", permissionController.DeletePermissionEndUser)
			user.DELETE("/rbac/permissions", permissionController.DeletePermissionEndUserByBody)
			user.GET("/rbac/permissions/resources", permissionController.ShowResourcesEndUser)
			user.GET("/scopes", scopeController.ListUserScopes)
			user.GET("/scopes/mappings", scopeController.GetUserMappings)
			user.POST("/scopes", scopeController.AddUserScope)
			user.PUT("/scopes/:scope_name", scopeController.EditUserScope)
			user.DELETE("/scopes/:scope_name", scopeController.DeleteUserScope)
			user.POST("/rbac/policy/check", authController.PolicyDecisionPointCheckUser)
			user.POST("/api_scopes", apiScopesController.CreateAPIScopeEndUser)
			user.GET("/api_scopes", apiScopesController.ListAPIScopesEndUser)
			user.GET("/api_scopes/:scope_id", apiScopesController.GetAPIScopeEndUser)
			user.PUT("/api_scopes/:scope_id", apiScopesController.UpdateAPIScopeEndUser)
			user.DELETE("/api_scopes/:scope_id", apiScopesController.DeleteAPIScopeEndUser)
			user.GET("/permissions", permissionController.GetMyPermissions)
			user.GET("/permissions/effective", permissionController.GetMyEffectivePermissions)
			user.GET("/permissions/check", permissionController.CheckPermission)
			user.POST("/groups/users/add", groupController.AddUserToGroups)
			user.POST("/groups/users/remove", groupController.RemoveUserFromGroups)
			user.GET("/groups/users", groupController.GetMyGroups)
			user.GET("/groups/:tenant_id/:group_id/users", groupController.GetGroupUsers)
			user.POST("/admin/change-password", endUserController.AdminChangeUserPassword)
			user.POST("/admin/reset-password", endUserController.AdminResetUserPassword)
		}

		// ────────────────────────────────────────────────────
		// HubSpot integration
		// ────────────────────────────────────────────────────
		hubspot := uflow.Group("/hubspot")
		hubspot.Use(middlewares.AuthMiddleware(), amMiddlewares.ValidateTenantFromToken())
		{
			hubspot.POST("/contacts/sync", hubspotController.SyncContact)
		}

		// ────────────────────────────────────────────────────
		// SCIM 2.0
		// ────────────────────────────────────────────────────

		// Discovery (public)
		scimDiscovery := uflow.Group("/scim/v2")
		{
			scimDiscovery.GET("/ServiceProviderConfig", scimController.GetServiceProviderConfig)
			scimDiscovery.GET("/Schemas", scimController.GetSchemas)
			scimDiscovery.GET("/ResourceTypes", scimController.GetResourceTypes)
		}

		// End-user provisioning
		scimEndUser := uflow.Group("/scim/v2/:client_id/:project_id")
		scimEndUser.Use(middlewares.AuthMiddleware(), amMiddlewares.ValidateTenantFromToken())
		{
			scimEndUser.GET("/Users", scimController.ListUsers)
			scimEndUser.GET("/Users/:id", scimController.GetUser)
			scimEndUser.POST("/Users", scimController.CreateUser)
			scimEndUser.PUT("/Users/:id", scimController.ReplaceUser)
			scimEndUser.PATCH("/Users/:id", scimController.PatchUser)
			scimEndUser.DELETE("/Users/:id", scimController.DeleteUser)
			scimEndUser.GET("/Groups", scimController.ListGroups)
			scimEndUser.GET("/Groups/:id", scimController.GetGroup)
			scimEndUser.POST("/Groups", scimController.CreateGroup)
			scimEndUser.PUT("/Groups/:id", scimController.ReplaceGroup)
			scimEndUser.PATCH("/Groups/:id", scimController.PatchGroup)
			scimEndUser.DELETE("/Groups/:id", scimController.DeleteGroup)
		}

		// Admin provisioning
		scimAdmin := uflow.Group("/scim/v2/admin")
		scimAdmin.Use(
			middlewares.AuthMiddleware(),
			middlewares.Require("admin", "access"),
			amMiddlewares.ValidateTenantFromToken(),
		)
		{
			scimAdmin.GET("/Users", scimAdminController.ListAdminUsers)
			scimAdmin.GET("/Users/:id", scimAdminController.GetAdminUser)
			scimAdmin.POST("/Users", scimAdminController.CreateAdminUser)
			scimAdmin.PUT("/Users/:id", scimAdminController.ReplaceAdminUser)
			scimAdmin.PATCH("/Users/:id", scimAdminController.PatchAdminUser)
			scimAdmin.DELETE("/Users/:id", scimAdminController.DeleteAdminUser)
		}

		// ────────────────────────────────────────────────────
		// Delegation Policy CRUD (admin-authenticated)
		// ────────────────────────────────────────────────────
		delegationPolicies := uflow.Group("/delegation-policies")
		delegationPolicies.Use(
			middlewares.AuthMiddleware(),
			middlewares.Require("admin", "access"),
			amMiddlewares.ValidateTenantFromToken(),
		)
		{
			delegationPolicies.POST("", delegationPolicyController.CreateDelegationPolicy)
			delegationPolicies.GET("", delegationPolicyController.ListDelegationPolicies)
			delegationPolicies.GET("/:id", delegationPolicyController.GetDelegationPolicy)
			delegationPolicies.PUT("/:id", delegationPolicyController.UpdateDelegationPolicy)
			delegationPolicies.DELETE("/:id", delegationPolicyController.DeleteDelegationPolicy)
		}

		// ────────────────────────────────────────────────────
		// SDK Token Pull (public, authenticated by client_id)
		// ────────────────────────────────────────────────────
		sdk := uflow.Group("/sdk")
		{
			sdk.GET("/delegation-token", sdkTokenController.GetDelegationToken)
		}

		// ────────────────────────────────────────────────────
		// Health checks
		// ────────────────────────────────────────────────────
		health := uflow.Group("/health")
		{
			health.GET("", healthController.ComprehensiveHealthCheck)
			health.GET("/tenant/:tenant_id", healthController.CheckTenantDatabase)
			health.GET("/tenants", healthController.CheckAllTenantDatabases)
		}

		// ────────────────────────────────────────────────────
		// Client Management (formerly clients-microservice)
		// Served under /clientms to match the original service prefix.
		// ────────────────────────────────────────────────────
		registerClientsRoutes(authsec)

		// ────────────────────────────────────────────────────
		// Hydra Manager (formerly hydra-service)
		// Served under /authsec/hmgr.
		// ────────────────────────────────────────────────────
		registerHmgrRoutes(authsec)

		// ────────────────────────────────────────────────────
		// OIDC Configuration Manager (formerly oath_oidc_configuration_manager)
		// Served under /authsec/oocmgr.
		// ────────────────────────────────────────────────────
		registerOocmgrRoutes(authsec)

		// ────────────────────────────────────────────────────
		// Auth Manager (formerly auth-manager)
		// Served under /authsec/authmgr.
		// ────────────────────────────────────────────────────
		registerAuthmgrRoutes(authsec)

		// ────────────────────────────────────────────────────
		// SDK Manager (formerly sdk-manager Python service)
		// Served under /authsec/sdkmgr.
		// Backward-compat alias at bare /sdkmgr/* for existing SDKs.
		// ────────────────────────────────────────────────────
		registerSdkmgrRoutes(authsec, r)

		// ────────────────────────────────────────────────────
		// SPIRE Headless (formerly spire-headless microservice)
		// Served under /authsec/spire.
		// ────────────────────────────────────────────────────
		registerSpireRoutes(authsec)

		// ────────────────────────────────────────────────────
		// External Service (formerly exsvc / mcp-service)
		// Served under /authsec/exsvc.
		// ────────────────────────────────────────────────────
		extSvcController := platformCtrl.NewExternalServiceController(config.DB)

		exsvc := authsec.Group("/exsvc")
		exsvc.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok", "service": "external-service"})
		})
		exsvc.GET("/debug/auth", middlewares.AuthMiddleware(), platformCtrl.DebugExternalServiceAuth)
		exsvc.GET("/debug/test", middlewares.AuthMiddleware(), func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "authenticated", "path": "/debug/test"})
		})
		exsvc.GET("/debug/token", middlewares.AuthMiddleware(), func(c *gin.Context) {
			contextData := make(map[string]interface{})
			if claims, exists := c.Get("claims"); exists {
				contextData["claims"] = claims
			}
			if perms, exists := c.Get("perms"); exists {
				contextData["perms"] = perms
			}
			if scope, exists := c.Get("scope"); exists {
				contextData["scope"] = scope
			}
			if user, exists := c.Get("user"); exists {
				contextData["user"] = user
			}
			contextData["all_context_keys"] = c.Keys
			c.JSON(200, gin.H{"status": "authenticated", "context_data": contextData})
		})

		extSvcs := exsvc.Group("/services")
		extSvcs.Use(middlewares.AuthMiddleware())
		{
			extSvcs.POST("", middlewares.Require("external-service", "create"), extSvcController.CreateExternalService)
			extSvcs.GET("", middlewares.Require("external-service", "read"), extSvcController.ListExternalServices)
			extSvcs.GET("/:id", middlewares.Require("external-service", "read"), extSvcController.GetExternalService)
			extSvcs.PUT("/:id", middlewares.Require("external-service", "update"), extSvcController.UpdateExternalService)
			extSvcs.DELETE("/:id", middlewares.Require("external-service", "delete"), extSvcController.DeleteExternalService)
			extSvcs.GET("/:id/credentials", middlewares.Require("external-service", "credentials"), extSvcController.GetExternalServiceCredentials)
		}

		// Legacy login/register endpoints
		uflow.POST("/register/verify", userController.VerifyOTPAndCompleteRegistration)
		uflow.POST("/login/webauthn-callback", userController.WebAuthnCallback)
		uflow.POST("/login", userController.Login)

		// Misrouted health-check helpers for monitoring
		uflow.GET("/spire/health", func(c *gin.Context) {
			c.JSON(404, gin.H{"error": "Health check URL misconfigured", "correct_url": "/spiresvc/health"})
		})
		uflow.GET("/clients/clients/api/v1/health", func(c *gin.Context) {
			c.JSON(404, gin.H{"error": "Health check URL misconfigured", "correct_url": "/clientms/api/v1/health"})
		})
	}
}

// registerClientsRoutes registers all client management routes under /clientms.
// Previously served by the standalone clients-microservice.
// Auth middleware is applied to all routes inside the /clientms group.
func registerClientsRoutes(r gin.IRouter) {
	redoclyHandler := func(c *gin.Context) {
		html := `<!DOCTYPE html>
					<html>
					<head>
						<title>Clients API Documentation</title>
						<meta charset="utf-8"/>
						<meta name="viewport" content="width=device-width, initial-scale=1">
						<link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
						<style>body { margin: 0; padding: 0; }</style>
					</head>
					<body>
						<redoc spec-url='/clientms/swagger/doc.json'></redoc>
						<script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"> </script>
					</body>
					</html>`
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, html)
	}

	// Documentation endpoints (no auth required)
	r.GET("/clientms/swagger", redoclyHandler)
	r.GET("/clientms/swagger/index.html", redoclyHandler)
	r.GET("/clientms/swagger/doc.json", ginSwagger.WrapHandler(swaggerFiles.Handler))

	clientms := r.Group("/clientms")

	// Health check (no auth required)
	clientms.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "clients-microservice"})
	})

	// All API routes require JWT authentication
	clientms.Use(middlewares.AuthMiddleware())
	{
		// Tenant-scoped client routes
		tenantRoutes := clientms.Group("/tenants/:tenantId")
		{
			clients := tenantRoutes.Group("/clients")
			{
				clients.GET("/getClients", platformCtrl.GetClients)
				clients.POST("/getClients", platformCtrl.GetClientsByTenant)

				clients.GET("/:id", platformCtrl.GetClient)

				clients.POST("/create", platformCtrl.RegisterClient)

				clients.PUT("/:id", platformCtrl.UpdateClient)
				clients.PATCH("/:id", platformCtrl.EditClient)

				clients.PATCH("/:id/soft-delete", platformCtrl.SoftDeleteClient)
				clients.DELETE("/:id", platformCtrl.DeleteClient)

				clients.POST("/delete-complete", platformCtrl.DeleteCompleteClient)

				clients.PATCH("/:id/activate", platformCtrl.ActivateClient)
				clients.PATCH("/:id/deactivate", platformCtrl.DeactivateClient)

				clients.POST("/set-status", platformCtrl.SetClientStatus)
			}
		}

		// Admin cross-tenant route (requires admin access)
		adminClients := clientms.Group("/admin/clients")
		adminClients.Use(middlewares.Require("clients", "admin"))
		{
			adminClients.GET("/", platformCtrl.GetClients)
		}

		// OOC Manager integration routes (internal service-to-service)
		oocmgr := clientms.Group("/oocmgr")
		{
			oocmgr.POST("/tenant/delete-complete", platformCtrl.DeleteCompleteClient)
		}
	}
}

// registerWebAuthnRoutes registers WebAuthn routes on the provided router group.
// Previously served by the standalone webauthn-service under /webauthn/*.
// Now served under /authsec/webauthn/*.
func registerWebAuthnRoutes(
	router gin.IRouter,
	webAuthnHandler *handlers.WebAuthnHandler,
	adminHandler *handlers.AdminWebAuthnHandler,
	endUserHandler *handlers.EndUserWebAuthnHandler,
) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "webauthn-service"})
	})

	// Admin WebAuthn (uses global DB)  →  /authsec/webauthn/admin/*
	admin := router.Group("/admin")
	{
		admin.POST("/mfa/status", adminHandler.GetMFAStatus)
		admin.POST("/mfa/loginStatus", adminHandler.GetMFAStatusForLogin)
		admin.GET("/mfa/loginStatus", adminHandler.GetMFAStatusForLoginGET)
		admin.POST("/beginRegistration", adminHandler.BeginRegistration)
		admin.POST("/finishRegistration", adminHandler.FinishRegistration)
		admin.POST("/beginAuthentication", adminHandler.BeginAuthentication)
		admin.POST("/finishAuthentication", adminHandler.FinishAuthentication)
	}

	// End-user WebAuthn (uses tenant-specific DBs)  →  /authsec/webauthn/enduser/*
	enduser := router.Group("/enduser")
	{
		enduser.POST("/mfa/status", endUserHandler.GetMFAStatus)
		enduser.POST("/mfa/loginStatus", endUserHandler.GetMFAStatusForLogin)
		enduser.GET("/mfa/loginStatus", endUserHandler.GetMFAStatusForLoginGET)
		enduser.POST("/beginRegistration", endUserHandler.BeginRegistration)
		enduser.POST("/finishRegistration", endUserHandler.FinishRegistration)
		enduser.POST("/beginAuthentication", endUserHandler.BeginAuthentication)
		enduser.POST("/finishAuthentication", endUserHandler.FinishAuthentication)
	}

	// Legacy flat routes  →  /authsec/webauthn/*
	router.POST("/mfa/status", webAuthnHandler.GetMFAStatus)
	router.POST("/mfa/loginStatus", webAuthnHandler.GetMFAStatusForLogin)
	router.GET("/mfa/loginStatus", webAuthnHandler.GetMFAStatusForLoginGET)
	router.POST("/beginRegistration", webAuthnHandler.BeginRegistration)
	router.POST("/beginAuthRegistration", webAuthnHandler.BeginWebAuthnRegistration)
	router.POST("/finishRegistration", webAuthnHandler.FinishRegistration)
	router.POST("/beginAuthentication", webAuthnHandler.BeginAuthentication)
	router.POST("/finishAuthentication", webAuthnHandler.FinishAuthentication)

	// Biometric (alias flows)
	router.POST("/biometric/verifyBegin", webAuthnHandler.BeginBiometricVerify)
	router.POST("/biometric/verifyFinish", webAuthnHandler.FinishBiometricVerify)
	router.POST("/biometric/beginSetup", webAuthnHandler.BeginBiometricSetup)
	router.POST("/biometric/confirmSetup", webAuthnHandler.ConfirmBiometricSetup)
	router.POST("/biometric/beginLoginSetup", webAuthnHandler.BeginBiometricLoginSetup)
	router.POST("/biometric/confirmLoginSetup", webAuthnHandler.ConfirmBiometricLoginSetup)
	router.POST("/biometric/verifyLoginBegin", webAuthnHandler.BeginBiometricLoginVerify)
	router.POST("/biometric/verifyLoginFinish", webAuthnHandler.FinishBiometricLoginVerify)

	// TOTP (legacy)
	totpHandler := handlers.NewTOTPHandler()
	router.POST("/totp/beginLoginSetup", totpHandler.BeginSetup)
	router.POST("/totp/beginSetup", totpHandler.BeginTOTPSetup)
	router.POST("/totp/confirmLoginSetup", totpHandler.ConfirmSetup)
	router.POST("/totp/confirmSetup", totpHandler.ConfirmTOTPSetup)
	router.POST("/totp/verifyLogin", totpHandler.VerifyLoginTOTP)
	router.POST("/totp/verify", totpHandler.VerifyTOTP)

	// SMS (legacy)
	smsHandler := handlers.NewSMSHandler()
	router.POST("/sms/beginSetup", smsHandler.BeginSMSSetup)
	router.POST("/sms/confirmSetup", smsHandler.ConfirmSMSSetup)
	router.POST("/sms/requestCode", smsHandler.RequestSMSCode)
	router.POST("/sms/verify", smsHandler.VerifySMS)
}

// registerHmgrRoutes registers all Hydra Manager routes under /hmgr.
// Previously served by the standalone hydra-service.
func registerHmgrRoutes(r gin.IRouter) {
	hmgrController := platformCtrl.NewHmgrController(*config.AppConfig)

	// ── Public routes (no authentication required) ──
	pub := r.Group("/hmgr")
	{
		// OIDC endpoints
		pub.GET("/login/page-data", hmgrController.GetLoginPageDataHandler)
		pub.POST("/auth/initiate/:provider", hmgrController.InitiateAuthHandler)
		pub.POST("/auth/callback", hmgrController.HandleCallbackHandler)
		pub.POST("/auth/exchange-token", hmgrController.ExchangeTokenHandler)

		// SAML endpoints
		pub.POST("/saml/initiate/:provider", hmgrController.InitiateSAMLAuthHandler)
		pub.POST("/saml/acs", hmgrController.HandleSAMLACSHandler)
		pub.POST("/saml/acs/:tenant_id/:client_id", hmgrController.HandleSAMLACSClientHandler)
		pub.GET("/saml/metadata/:tenant_id/:client_id", hmgrController.GetSAMLMetadataHandler)
		pub.POST("/saml/test-provider", hmgrController.TestSAMLProviderHandler)

		// DEV ONLY: Temporary unauthenticated SAML provider management
		// TODO: Remove in production - use /hmgr/admin/saml-providers instead
		dev := pub.Group("/dev")
		{
			dev.GET("/saml-providers", hmgrController.GetSAMLProvidersHandler)
			dev.POST("/saml-providers", hmgrController.CreateSAMLProviderHandler)
			dev.PUT("/saml-providers/:id", hmgrController.UpdateSAMLProviderHandler)
			dev.DELETE("/saml-providers/:id", hmgrController.DeleteSAMLProviderHandler)
		}

		// Common endpoints
		pub.GET("/login", hmgrController.LoginRedirectHandler)
		pub.GET("/consent", hmgrController.ConsentHandler)
		pub.GET("/health", hmgrController.HealthHandler)
		pub.GET("/challenge", hmgrController.LoginChallengeHandler)
	}

	// ── Protected routes requiring authentication ──
	prot := r.Group("/hmgr/admin")
	prot.Use(middlewares.AuthMiddleware())
	{
		// Admin-only routes
		admin := prot.Group("/")
		admin.Use(middlewares.Require("admin", "manage"))
		{
			// User management
			admin.GET("/users", hmgrController.GetUsersHandler)
			admin.POST("/users", hmgrController.CreateUserHandler)
			admin.PUT("/users/:id", hmgrController.UpdateUserHandler)
			admin.DELETE("/users/:id", hmgrController.DeleteUserHandler)

			// Tenant management
			admin.GET("/tenants", hmgrController.GetTenantsHandler)
			admin.POST("/tenants", hmgrController.CreateTenantHandler)
			admin.PUT("/tenants/:id", hmgrController.UpdateTenantHandler)
			admin.DELETE("/tenants/:id", hmgrController.DeleteTenantHandler)

			// SAML provider management
			admin.GET("/saml-providers", hmgrController.GetSAMLProvidersHandler)
			admin.POST("/saml-providers", hmgrController.CreateSAMLProviderHandler)
			admin.PUT("/saml-providers/:id", hmgrController.UpdateSAMLProviderHandler)
			admin.DELETE("/saml-providers/:id", hmgrController.DeleteSAMLProviderHandler)

			// Role and permission management
			admin.GET("/roles", hmgrController.GetRolesHandler)
			admin.POST("/roles", hmgrController.CreateRoleHandler)
			admin.PUT("/roles/:id", hmgrController.UpdateRoleHandler)
			admin.DELETE("/roles/:id", hmgrController.DeleteRoleHandler)
			admin.GET("/permissions", hmgrController.GetPermissionsHandler)
			admin.POST("/permissions", hmgrController.CreatePermissionHandler)

			// User role assignments
			admin.POST("/users/:id/roles", hmgrController.AssignUserRoleHandler)
			admin.DELETE("/users/:id/roles/:role_id", hmgrController.RemoveUserRoleHandler)
		}

		// Authenticated user routes (own profile)
		user := prot.Group("/")
		{
			user.GET("/profile", hmgrController.GetProfileHandler)
			user.PUT("/profile", hmgrController.UpdateProfileHandler)
		}
	}
}

// registerOocmgrRoutes registers all OIDC Configuration Manager routes under /oocmgr.
// Previously served by the standalone oath_oidc_configuration_manager microservice.
func registerOocmgrRoutes(r gin.IRouter) {
	ac := platformCtrl.NewOocmgrController()

	v1 := r.Group("/oocmgr")

	v1.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "oidc-config-manager", "version": "2.0.0"})
	})

	secured := v1.Group("/")

	// ── Main configuration ──
	secured.POST("/configure-complete-oidc", ac.CompleteOIDCConfiguration)

	// ── Tenant management ──
	tenant := secured.Group("/tenant")
	{
		tenant.POST("/create-base-client", ac.CreateBaseTenantClient)
		tenant.POST("/check-exists", ac.CheckTenantExists)
		tenant.POST("/list-all", ac.ListAllTenants)
		tenant.POST("/delete-complete", ac.DeleteCompleteTenantConfig)
		tenant.POST("/update-complete", ac.UpdateCompleteTenantConfig)
		tenant.POST("/login-page-data", ac.GetTenantLoginPageData)
	}

	// ── Config management ──
	configs := secured.Group("/config")
	{
		configs.POST("/edit", ac.EditConfig)
	}

	// ── OIDC management ──
	oidc := secured.Group("/oidc")
	{
		oidc.POST("/add-provider", ac.AddOIDCProviderToTenant)
		oidc.POST("/get-config", ac.GetTenantOIDCConfig)
		oidc.POST("/get-provider", ac.GetOIDCProvider)
		oidc.POST("/get-provider-secret", ac.GetProviderSecret)
		oidc.POST("/update-provider", ac.UpdateOIDCProvider)
		oidc.POST("/delete-provider", ac.DeleteOIDCProvider)
		oidc.POST("/templates", ac.GetProviderTemplates)
		oidc.POST("/validate", ac.ValidateOIDCConfig)
		oidc.GET("/show-auth-providers", ac.ShowAuthProviders)
		oidc.POST("/show-auth-providers", ac.ShowAuthProviders)
		oidc.POST("/raw-hydra-dump", middlewares.AuthMiddleware(), ac.DumpHydraRawData)
		oidc.POST("/edit-client-auth-provider", ac.EditAuthProvider)
	}

	// ── SAML management ──
	saml := secured.Group("/saml")
	{
		saml.POST("/add-provider", ac.AddSAMLProvider)
		saml.POST("/list-providers", ac.ListSAMLProviders)
		saml.POST("/get-provider", ac.GetSAMLProvider)
		saml.POST("/update-provider", ac.UpdateSAMLProvider)
		saml.POST("/delete-provider", ac.DeleteSAMLProvider)
		saml.POST("/templates", ac.GetSAMLProviderTemplates)
	}

	// ── Hydra client management ──
	hydraClients := secured.Group("/hydra-clients")
	{
		hydraClients.POST("/list", ac.ListTenantHydraClients)
		hydraClients.POST("/get-by-tenant", ac.GetTenantHydraClients)
		hydraClients.POST("/sync", ac.SyncHydraClients)
	}

	// ── Testing ──
	test := v1.Group("/test")
	{
		test.POST("/oidc-flow", ac.TestOIDCFlow)
	}

	// ── Stats ──
	stats := v1.Group("/stats")
	{
		stats.POST("/tenant", ac.GetTenantStats)
	}

	// ── Clients ──
	oocmgrClients := v1.Group("/clients")
	{
		oocmgrClients.POST("/getClients", ac.GetClientsByTenant)
	}
}

// registerAuthmgrRoutes registers all Auth Manager routes under /authmgr.
// Previously served by the standalone auth-manager microservice.
func registerAuthmgrRoutes(r gin.IRouter) {
	ac := platformCtrl.NewAuthmgrController()

	// Public / unauthenticated endpoints
	authmgr := r.Group("/authmgr")
	{
		authmgr.GET("/health", ac.HealthCheck)
		authmgr.POST("/token/verify", ac.VerifyToken)
		authmgr.POST("/token/generate", ac.GenerateToken)
		authmgr.POST("/token/oidc", ac.OIDCToken)
	}

	// Admin endpoints (protected by authsec AuthMiddleware)
	admin := r.Group("/authmgr/admin")
	admin.Use(middlewares.AuthMiddleware())
	{
		admin.GET("/profile", ac.GetProfile)
		admin.GET("/auth-status", ac.GetAuthStatus)

		// Validation
		admin.GET("/validate/token", ac.ValidateToken)
		admin.GET("/validate/scope", ac.ValidateScope)
		admin.GET("/validate/resource", ac.ValidateResource)
		admin.POST("/validate/permissions", ac.ValidatePermissions)

		// RBAC permission checks
		admin.GET("/check/permission", ac.CheckPermission)
		admin.GET("/check/role", ac.CheckRole)
		admin.GET("/check/role-resource", ac.CheckRoleResource)
		admin.GET("/check/permission-scoped", ac.CheckPermissionScoped)
		admin.GET("/check/oauth-scope", ac.CheckOAuthScopePermission)
		admin.GET("/permissions", ac.ListUserPermissions)

		// Group management
		admin.POST("/groups", ac.CreateGroup)
		admin.GET("/groups", ac.ListGroups)
		admin.GET("/groups/:id", ac.GetGroup)
		admin.PUT("/groups/:id", ac.UpdateGroup)
		admin.DELETE("/groups/:id", ac.DeleteGroup)
		admin.POST("/groups/:id/users", ac.AddUsersToGroup)
		admin.DELETE("/groups/:id/users", ac.RemoveUsersFromGroup)
		admin.GET("/groups/:id/users", ac.ListGroupUsers)
	}

	// User endpoints (protected by authsec AuthMiddleware)
	user := r.Group("/authmgr/user")
	user.Use(middlewares.AuthMiddleware())
	{
		user.GET("/profile", ac.GetProfile)
		user.GET("/auth-status", ac.GetAuthStatus)

		// Validation
		user.GET("/validate/token", ac.ValidateToken)
		user.GET("/validate/scope", ac.ValidateScope)
		user.GET("/validate/resource", ac.ValidateResource)
		user.POST("/validate/permissions", ac.ValidatePermissions)

		// RBAC permission checks
		user.GET("/check/permission", ac.CheckPermission)
		user.GET("/check/role", ac.CheckRole)
		user.GET("/check/role-resource", ac.CheckRoleResource)
		user.GET("/check/permission-scoped", ac.CheckPermissionScoped)
		user.GET("/check/oauth-scope", ac.CheckOAuthScopePermission)
		user.GET("/permissions", ac.ListUserPermissions)
	}

	// ────────────────────────────────────────────────────────
	// migration – database migration management API
	// Formerly the standalone authsec-migration microservice.
	// ────────────────────────────────────────────────────────
	migCtrl := adminCtrl.NewMigrationController()

	mig := r.Group("/migration")
	{
		// Master database migrations (admin JWT required)
		master := mig.Group("/migrations/master")
		master.Use(middlewares.AuthMiddleware())
		{
			master.POST("/run", migCtrl.RunMasterMigrations)
			master.GET("/status", migCtrl.GetMasterMigrationStatus)
		}

		// Tenant database management
		tenants := mig.Group("/tenants")
		tenants.Use(middlewares.AuthMiddleware())
		{
			tenants.GET("", migCtrl.ListTenants)
			tenants.POST("/create-db", migCtrl.CreateTenantDB)
			tenants.POST("/migrate-all", migCtrl.MigrateAllTenants)
			tenants.POST("/:tenant_id/migrations/run", migCtrl.RunTenantMigrations)
			tenants.GET("/:tenant_id/migrations/status", migCtrl.GetTenantMigrationStatus)
		}
	}
}

// registerSdkmgrRoutes registers all SDK Manager routes under /sdkmgr.
// Previously served by the standalone sdk-manager Python service.
// Routes are registered on both the primary router (under /authsec) and
// a backward-compatibility alias at the bare root so that existing SDKs
// that target /sdkmgr/* continue to work during migration.
func registerSdkmgrRoutes(r gin.IRouter, aliases ...gin.IRouter) {
	// Initialise the MCP Auth service and run startup tasks.
	mcpAuthSvc := sdkmgrSvc.NewMCPAuthService()
	mcpAuthSvc.Initialize()
	mcpAuthCtrl := sdkmgrCtrl.NewMCPAuthController(mcpAuthSvc)

	servicesSvc := sdkmgrSvc.NewServicesService(mcpAuthSvc.SessionStore)
	servicesCtrl := sdkmgrCtrl.NewServicesController(servicesSvc)

	spireSvc := sdkmgrSvc.NewSPIREProxyService()
	spireSvc.Initialize()
	spireCtrl := sdkmgrCtrl.NewSPIREController(spireSvc)

	dashSvc := sdkmgrSvc.NewDashboardService()
	dashCtrl := sdkmgrCtrl.NewDashboardController(dashSvc)

	mcpOAuthSvc := sdkmgrSvc.NewMCPOAuthService()
	mcpOAuthCtrl := sdkmgrCtrl.NewMCPOAuthController(mcpOAuthSvc)

	playgroundSvc := sdkmgrSvc.NewMCPPlaygroundService()
	playgroundCtrl := sdkmgrCtrl.NewMCPPlaygroundController(playgroundSvc)

	voiceSvc := sdkmgrSvc.NewVoiceClientService()
	voiceCtrl := sdkmgrCtrl.NewVoiceController(voiceSvc)

	devServerSvc := sdkmgrSvc.NewDevServerService(playgroundSvc)
	devServerCtrl := sdkmgrCtrl.NewDevServerController(devServerSvc)

	// Bind routes on the primary router and any backward-compat aliases.
	routers := append([]gin.IRouter{r}, aliases...)
	for _, router := range routers {
		bindSdkmgrRoutes(router, mcpAuthCtrl, servicesCtrl, spireCtrl, dashCtrl,
			mcpOAuthCtrl, playgroundCtrl, voiceCtrl, devServerCtrl)
	}
}

// bindSdkmgrRoutes registers all sdkmgr endpoint groups on the given router.
func bindSdkmgrRoutes(
	r gin.IRouter,
	mcpAuthCtrl *sdkmgrCtrl.MCPAuthController,
	servicesCtrl *sdkmgrCtrl.ServicesController,
	spireCtrl *sdkmgrCtrl.SPIREController,
	dashCtrl *sdkmgrCtrl.DashboardController,
	mcpOAuthCtrl *sdkmgrCtrl.MCPOAuthController,
	playgroundCtrl *sdkmgrCtrl.MCPPlaygroundController,
	voiceCtrl *sdkmgrCtrl.VoiceController,
	devServerCtrl *sdkmgrCtrl.DevServerController,
) {
	// ── MCP Auth routes ──
	mcpAuth := r.Group("/sdkmgr/mcp-auth")
	{
		mcpAuth.GET("/health", mcpAuthCtrl.Health)
		mcpAuth.POST("/start", mcpAuthCtrl.Start)
		mcpAuth.POST("/authenticate", mcpAuthCtrl.Authenticate)
		mcpAuth.POST("/callback", mcpAuthCtrl.CallbackJSON)
		mcpAuth.GET("/callback", mcpAuthCtrl.CallbackHTML)
		mcpAuth.GET("/status/:session_id", mcpAuthCtrl.Status)
		mcpAuth.GET("/sessions/status", mcpAuthCtrl.SessionsStatus)
		mcpAuth.POST("/logout", mcpAuthCtrl.Logout)
		mcpAuth.POST("/tools/list", mcpAuthCtrl.ToolsList)
		mcpAuth.POST("/tools/call/:tool_name", mcpAuthCtrl.ToolCall)
		mcpAuth.POST("/protect-tool", mcpAuthCtrl.ProtectTool)
		mcpAuth.POST("/cleanup-sessions", mcpAuthCtrl.CleanupSessions)
	}

	// ── Services routes ──
	services := r.Group("/sdkmgr/services")
	{
		services.GET("/health", servicesCtrl.Health)
		services.POST("/credentials", servicesCtrl.GetCredentials)
		services.POST("/user-details", servicesCtrl.GetUserDetails)
	}

	// ── SPIRE routes ──
	spire := r.Group("/sdkmgr/spire")
	{
		spire.GET("/health", spireCtrl.Health)
		spire.POST("/workload/initialize", spireCtrl.Initialize)
		spire.POST("/workload/renew", spireCtrl.Renew)
		spire.POST("/workload/status", spireCtrl.Status)
		spire.GET("/validate-agent-connection", spireCtrl.ValidateConnection)
	}

	// ── Dashboard routes ──
	dashboard := r.Group("/sdkmgr/dashboard")
	{
		dashboard.GET("/health", dashCtrl.Health)
		dashboard.POST("/sessions", dashCtrl.Sessions)
		dashboard.POST("/statistics", middlewares.AuthMiddleware(), dashCtrl.Statistics)
		dashboard.POST("/users", dashCtrl.Users)
		dashboard.POST("/admin-users", middlewares.AuthMiddleware(), dashCtrl.AdminUsers)
	}

	// ── MCP OAuth routes ──
	playgroundOAuth := r.Group("/sdkmgr/playground/oauth")
	{
		playgroundOAuth.GET("/check-requirements", mcpOAuthCtrl.CheckRequirements)
		playgroundOAuth.GET("/authorize", mcpOAuthCtrl.Authorize)
		playgroundOAuth.GET("/callback", mcpOAuthCtrl.Callback)
		playgroundOAuth.POST("/refresh", mcpOAuthCtrl.Refresh)
	}

	// ── MCP Playground routes ──
	playground := r.Group("/sdkmgr/playground")
	{
		playground.GET("/health", playgroundCtrl.Health)
		playground.POST("/conversations", playgroundCtrl.CreateConversation)
		playground.GET("/conversations", playgroundCtrl.ListConversations)
		playground.GET("/conversations/:id", playgroundCtrl.GetConversation)
		playground.PATCH("/conversations/:id", playgroundCtrl.UpdateConversation)
		playground.DELETE("/conversations/:id", playgroundCtrl.DeleteConversation)
		playground.GET("/conversations/:id/messages", playgroundCtrl.GetMessages)
		playground.POST("/conversations/:id/chat", playgroundCtrl.Chat)
		playground.POST("/chat/stream", playgroundCtrl.ChatStream)
		playground.POST("/conversations/:id/mcp-servers", playgroundCtrl.AddMCPServer)
		playground.GET("/conversations/:id/mcp-servers", playgroundCtrl.ListMCPServers)
		playground.POST("/conversations/:id/mcp-servers/:sid/disconnect", playgroundCtrl.DisconnectMCPServer)
		playground.POST("/conversations/:id/mcp-servers/:sid/reconnect", playgroundCtrl.ReconnectMCPServer)
		playground.DELETE("/conversations/:id/mcp-servers/:sid", playgroundCtrl.RemoveMCPServer)
		playground.GET("/conversations/:id/mcp-servers/:sid/tools", playgroundCtrl.GetMCPTools)
		playground.GET("/conversations/:id/tools", playgroundCtrl.GetAllConversationTools)
	}

	// ── Voice routes ──
	voice := r.Group("/sdkmgr/voice")
	{
		voice.POST("/interact", voiceCtrl.Interact)
		voice.POST("/poll", voiceCtrl.Poll)
		voice.POST("/tts", voiceCtrl.TTS)
	}

	// ── Dev Server routes (auth required) ──
	devServer := r.Group("/sdkmgr/playground/dev-server")
	devServer.Use(middlewares.AuthMiddleware())
	{
		devServer.POST("/start", devServerCtrl.Start)
		devServer.POST("/stop", devServerCtrl.Stop)
		devServer.GET("/status", devServerCtrl.Status)
	}
}

// registerSpireRoutes registers all SPIRE Headless routes under /spire.
// Previously served by the standalone spire-headless microservice.
func registerSpireRoutes(r gin.IRouter) {
	sc := platformCtrl.NewSpireController()

	spire := r.Group("/spire")

	spire.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "spire-headless", "version": "1.0.0"})
	})

	// ── OIDC discovery (no auth required) ──
	spire.GET("/.well-known/openid-configuration", sc.OIDCDiscovery)
	spire.GET("/.well-known/jwks.json", sc.OIDCJWKSHandler)

	// ── Registry ──
	registry := spire.Group("/registry")
	{
		registry.POST("/workloads", sc.RegisterWorkload)
		registry.PUT("/workloads/:id", sc.UpdateWorkload)
		registry.DELETE("/workloads/:id", sc.DeleteWorkload)
		registry.GET("/workloads", sc.ListWorkloads)
	}

	// ── OIDC token operations ──
	oidc := spire.Group("/oidc")
	{
		oidc.POST("/token", sc.OIDCTokenExchange)
		oidc.POST("/introspect", sc.OIDCIntrospect)
		oidc.POST("/revoke", sc.OIDCRevoke)
		oidc.POST("/exchange/spiffe", sc.OIDCExchangeSPIFFE)
		oidc.POST("/issue/jwt-svid", sc.OIDCIssueJWTSVID)
		oidc.POST("/exchange/cloud", sc.OIDCExchangeCloud)
		oidc.POST("/exchange/aws", sc.OIDCExchangeAWS)
		oidc.POST("/exchange/azure", sc.OIDCExchangeAzure)
		oidc.POST("/exchange/gcp", sc.OIDCExchangeGCP)
	}

	// ── Policy engine ──
	policy := spire.Group("/policy")
	{
		policy.POST("", sc.CreatePolicy)
		policy.GET("", sc.ListPolicies)
		policy.GET("/:id", sc.GetPolicy)
		policy.PUT("/:id", sc.UpdatePolicy)
		policy.DELETE("/:id", sc.DeletePolicy)
		policy.POST("/evaluate", sc.EvaluatePolicy)
		policy.POST("/batch-evaluate", sc.BatchEvaluatePolicy)
		policy.POST("/test", sc.TestPolicy)
	}

	// ── Role bindings ──
	roles := spire.Group("/roles")
	{
		roles.POST("/bind", sc.BindRole)
		roles.POST("/unbind", sc.UnbindRole)
		roles.GET("/bindings", sc.ListRoleBindings)
	}

	// ── Audit ──
	audit := spire.Group("/audit")
	{
		audit.GET("/logs", sc.GetAuditLogs)
		audit.GET("/logs/export", sc.ExportAuditLogs)
	}
}
