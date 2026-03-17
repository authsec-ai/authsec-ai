package shared

import (
	"net/http"
	"runtime"
	"time"

	amMiddlewares "github.com/authsec-ai/auth-manager/pkg/middlewares"
	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/monitoring"
	"github.com/gin-gonic/gin"
)

type HealthController struct{}

// ComprehensiveHealthCheck godoc
// @Summary Comprehensive system health check
// @Description Performs comprehensive health checks on all system components
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Router /uflow/health [get]
func (hc *HealthController) ComprehensiveHealthCheck(c *gin.Context) {
	startTime := time.Now()
	healthStatus := map[string]interface{}{
		"status":    "healthy",
		"timestamp": startTime.UTC(),
		"version":   "4.0.0",
		"checks":    make(map[string]interface{}),
	}

	checks := healthStatus["checks"].(map[string]interface{})
	isHealthy := true

	// 1. Database Health Check
	dbHealthy := hc.checkDatabaseHealth()
	checks["database"] = dbHealthy
	if !dbHealthy["healthy"].(bool) {
		isHealthy = false
	}

	// 2. Redis Cache Health Check (if configured)
	if config.CacheManager != nil {
		cacheHealthy := hc.checkCacheHealth()
		checks["redis_cache"] = cacheHealthy
		if !cacheHealthy["healthy"].(bool) {
			isHealthy = false
		}
	} else {
		checks["redis_cache"] = map[string]interface{}{
			"healthy": false,
			"message": "Redis not configured",
		}
	}

	// 3. Vault Health Check (if configured)
	vaultHealthy := hc.checkVaultHealth()
	checks["vault"] = vaultHealthy
	if !vaultHealthy["healthy"].(bool) {
		isHealthy = false
	}

	// 4. System Resources Check
	systemHealthy := hc.checkSystemHealth()
	checks["system"] = systemHealthy
	if !systemHealthy["healthy"].(bool) {
		isHealthy = false
	}

	// 5. Metrics Health Check
	metricsHealthy := hc.checkMetricsHealth()
	checks["metrics"] = metricsHealthy
	if !metricsHealthy["healthy"].(bool) {
		isHealthy = false
	}

	// Set overall status
	if !isHealthy {
		healthStatus["status"] = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, healthStatus)
		return
	}

	// Add response time
	healthStatus["response_time_ms"] = time.Since(startTime).Milliseconds()
	c.JSON(http.StatusOK, healthStatus)
}

// checkDatabaseHealth checks the health of the global database
func (hc *HealthController) checkDatabaseHealth() map[string]interface{} {
	result := map[string]interface{}{
		"healthy": true,
		"message": "Database is healthy",
	}

	if config.DB == nil {
		result["healthy"] = false
		result["message"] = "Database connection not initialized"
		return result
	}

	// Test database connectivity
	sqlDB, err := config.DB.DB()
	if err != nil {
		result["healthy"] = false
		result["message"] = "Failed to get database instance: " + err.Error()
		return result
	}

	if err := sqlDB.Ping(); err != nil {
		result["healthy"] = false
		result["message"] = "Database ping failed: " + err.Error()
		return result
	}

	// Check if we can execute a simple query
	var count int64
	if err := config.DB.Raw("SELECT 1").Scan(&count).Error; err != nil {
		result["healthy"] = false
		result["message"] = "Database query failed: " + err.Error()
		return result
	}

	result["connection_pool_stats"] = map[string]interface{}{
		"open_connections": sqlDB.Stats().OpenConnections,
		"in_use":           sqlDB.Stats().InUse,
		"idle":             sqlDB.Stats().Idle,
		"max_open":         sqlDB.Stats().MaxOpenConnections,
	}

	return result
}

// checkCacheHealth checks Redis cache health
func (hc *HealthController) checkCacheHealth() map[string]interface{} {
	result := map[string]interface{}{
		"healthy": true,
		"message": "Redis cache is healthy",
	}

	cacheMgr := config.CacheManager
	if cacheMgr == nil {
		result["healthy"] = false
		result["message"] = "Cache manager not initialized"
		return result
	}

	if err := cacheMgr.HealthCheck(); err != nil {
		result["healthy"] = false
		result["message"] = "Redis ping failed: " + err.Error()
		return result
	}

	// Test basic operations
	testKey := "health_check_" + time.Now().Format("20060102150405")
	if err := cacheMgr.Set(testKey, "test_value", 10*time.Second); err != nil {
		result["healthy"] = false
		result["message"] = "Redis set operation failed: " + err.Error()
		return result
	}

	var value string
	found, err := cacheMgr.Get(testKey, &value)
	if err != nil || !found || value != "test_value" {
		result["healthy"] = false
		result["message"] = "Redis get operation failed"
		return result
	}

	// Cleanup
	cacheMgr.Delete(testKey)

	return result
}

// checkVaultHealth checks HashiCorp Vault health
func (hc *HealthController) checkVaultHealth() map[string]interface{} {
	result := map[string]interface{}{
		"healthy": true,
		"message": "Vault is healthy",
	}

	// For now, just check if Vault is configured
	// In a real implementation, you'd make an actual health check call to Vault
	if config.AppConfig.VaultAddr == "" {
		result["healthy"] = false
		result["message"] = "Vault not configured"
		return result
	}

	result["vault_address"] = config.AppConfig.VaultAddr
	result["has_token"] = config.AppConfig.VaultToken != ""

	return result
}

// checkSystemHealth checks system resources
func (hc *HealthController) checkSystemHealth() map[string]interface{} {
	result := map[string]interface{}{
		"healthy": true,
		"message": "System resources are healthy",
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	result["memory"] = map[string]interface{}{
		"alloc_mb":       m.Alloc / 1024 / 1024,
		"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
		"sys_mb":         m.Sys / 1024 / 1024,
		"gc_cycles":      m.NumGC,
	}

	result["goroutines"] = runtime.NumGoroutine()
	result["cpu_count"] = runtime.NumCPU()

	// Check if memory usage is too high (> 80% of system memory would be concerning)
	// For now, just log the values - in production you'd set thresholds

	return result
}

// checkMetricsHealth checks if metrics collection is working
func (hc *HealthController) checkMetricsHealth() map[string]interface{} {
	result := map[string]interface{}{
		"healthy": true,
		"message": "Metrics collection is healthy",
	}

	// Check if metrics are initialized
	if monitoring.GetMetrics() == nil {
		result["healthy"] = false
		result["message"] = "Metrics not initialized"
		return result
	}

	// Test a metric update
	monitoring.GetMetrics().HTTPRequestTotal.WithLabelValues("GET", "/health", "200", "system").Inc()

	result["metrics_enabled"] = true
	result["logger_initialized"] = monitoring.GetLogger() != nil

	return result
}

// CheckTenantDatabase godoc
// @Summary Check tenant database health
// @Description Verifies that a tenant database is accessible and healthy
// @Tags Health
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/health/tenant/{tenant_id} [get]
func (hc *HealthController) CheckTenantDatabase(c *gin.Context) {
	tenantID, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in authentication token"})
		return
	}

	// Try to get tenant database connection
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":    "unhealthy",
			"tenant_id": tenantID,
			"error":     err.Error(),
		})
		return
	}

	// Test the connection with a simple query
	if sqlDB, err := tenantDB.DB(); err == nil {
		if err := sqlDB.Ping(); err == nil {
			// Connection is healthy, check if we can query tables
			var count int64
			if err := tenantDB.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&count).Error; err == nil {
				c.JSON(http.StatusOK, gin.H{
					"status":      "healthy",
					"tenant_id":   tenantID,
					"table_count": count,
					"message":     "Tenant database is accessible and healthy",
				})
				return
			}
		}
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"status":    "unhealthy",
		"tenant_id": tenantID,
		"error":     "Failed to ping tenant database",
	})
}

// CheckAllTenantDatabases godoc
// @Summary Check all tenant databases health
// @Description Verifies that all configured tenant databases are accessible
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /uflow/health/tenants [get]
func (hc *HealthController) CheckAllTenantDatabases(c *gin.Context) {
	// Get all tenants with configured databases
	var tenants []struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		TenantDB string `json:"tenant_db"`
	}

	if err := config.DB.Table("tenants").
		Select("id, email, tenant_db").
		Where("tenant_db IS NOT NULL AND tenant_db != '' AND active = true").
		Find(&tenants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "Failed to query tenants",
		})
		return
	}

	if len(tenants) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"message": "No tenant databases configured",
			"tenants": []interface{}{},
		})
		return
	}

	// Create database service for health checks
	db := config.GetDatabase()
	cfg := config.AppConfig
	dbService, err := database.NewTenantDBService(db, cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBPort)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "Failed to create database service",
		})
		return
	}
	defer dbService.Close()

	healthyCount := 0
	unhealthyCount := 0
	tenantStatus := make([]map[string]interface{}, 0, len(tenants))

	// Check each tenant database
	for _, tenant := range tenants {
		status := map[string]interface{}{
			"tenant_id": tenant.ID,
			"email":     tenant.Email,
			"database":  tenant.TenantDB,
		}

		if err := dbService.HealthCheck(tenant.TenantDB); err != nil {
			status["status"] = "unhealthy"
			status["error"] = err.Error()
			unhealthyCount++
		} else {
			status["status"] = "healthy"
			healthyCount++
		}

		tenantStatus = append(tenantStatus, status)
	}

	overallStatus := "healthy"
	if unhealthyCount > 0 {
		overallStatus = "degraded"
		if healthyCount == 0 {
			overallStatus = "unhealthy"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":          overallStatus,
		"total_tenants":   len(tenants),
		"healthy_count":   healthyCount,
		"unhealthy_count": unhealthyCount,
		"tenant_status":   tenantStatus,
	})
}
