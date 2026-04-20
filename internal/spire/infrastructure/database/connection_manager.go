package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"github.com/authsec-ai/authsec/internal/spire/domain/repositories"
	spireerrors "github.com/authsec-ai/authsec/internal/spire/errors"
)

// ConnectionManager manages database connections for multiple tenants.
// It can operate in two modes:
// 1. Adapter mode: wraps an existing master *sql.DB (e.g. from GORM)
// 2. Standalone mode: manages its own connections
type ConnectionManager struct {
	masterDB          *sql.DB
	tenantConnections map[string]*sql.DB
	mu                sync.RWMutex
	logger            *logrus.Entry
	tenantRepo        repositories.TenantRepository
	maxOpenConns      int
	maxIdleConns      int
	connMaxLifetime   time.Duration
	dbHost            string
	dbPort            int
	dbUsername        string
	dbPassword        string
	dbSSLMode         string
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(
	masterDB *sql.DB,
	logger *logrus.Entry,
	tenantRepo repositories.TenantRepository,
	maxOpenConns, maxIdleConns int,
	connMaxLifetime time.Duration,
	dbHost string,
	dbPort int,
	dbUsername, dbPassword, dbSSLMode string,
) *ConnectionManager {
	return &ConnectionManager{
		masterDB:          masterDB,
		tenantConnections: make(map[string]*sql.DB),
		logger:            logger,
		tenantRepo:        tenantRepo,
		maxOpenConns:      maxOpenConns,
		maxIdleConns:      maxIdleConns,
		connMaxLifetime:   connMaxLifetime,
		dbHost:            dbHost,
		dbPort:            dbPort,
		dbUsername:        dbUsername,
		dbPassword:        dbPassword,
		dbSSLMode:         dbSSLMode,
	}
}

// GetTenantDB returns a database connection for the given tenant
func (cm *ConnectionManager) GetTenantDB(ctx context.Context, tenantID string) (*sql.DB, error) {
	cm.mu.RLock()
	db, exists := cm.tenantConnections[tenantID]
	cm.mu.RUnlock()

	if exists {
		if err := db.PingContext(ctx); err == nil {
			return db, nil
		}
		cm.removeTenantConnection(tenantID)
	}

	return cm.createTenantConnection(ctx, tenantID)
}

// GetTenantDBByName returns a database connection using the tenant database name directly
func (cm *ConnectionManager) GetTenantDBByName(ctx context.Context, tenantID, dbName string) (*sql.DB, error) {
	cm.mu.RLock()
	db, exists := cm.tenantConnections[tenantID]
	cm.mu.RUnlock()

	if exists {
		if err := db.PingContext(ctx); err == nil {
			return db, nil
		}
		cm.removeTenantConnection(tenantID)
	}

	return cm.createTenantConnectionByName(ctx, tenantID, dbName)
}

func (cm *ConnectionManager) createTenantConnection(ctx context.Context, tenantID string) (*sql.DB, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if db, exists := cm.tenantConnections[tenantID]; exists {
		return db, nil
	}

	tenant, err := cm.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, spireerrors.NewNotFoundError("Tenant not found", err)
	}

	if !tenant.IsActive() {
		return nil, spireerrors.NewForbiddenError("Tenant is not active", nil)
	}

	tenantDBName := "tenant_" + strings.ReplaceAll(tenantID, "-", "_")

	dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		cm.dbHost, cm.dbPort, tenantDBName, cm.dbUsername, cm.dbPassword, cm.dbSSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		cm.logger.WithFields(logrus.Fields{"tenant_id": tenantID, "db_name": tenantDBName}).WithError(err).Error("Failed to open tenant database")
		return nil, spireerrors.NewInternalError("Failed to connect to tenant database", err)
	}

	db.SetMaxOpenConns(cm.maxOpenConns)
	db.SetMaxIdleConns(cm.maxIdleConns)
	db.SetConnMaxLifetime(cm.connMaxLifetime)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		cm.logger.WithFields(logrus.Fields{"tenant_id": tenantID, "db_name": tenantDBName}).WithError(err).Error("Failed to ping tenant database")
		return nil, spireerrors.NewInternalError("Failed to connect to tenant database", err)
	}

	cm.tenantConnections[tenantID] = db
	cm.logger.WithFields(logrus.Fields{"tenant_id": tenantID, "db_name": tenantDBName}).Info("Created tenant database connection")
	return db, nil
}

func (cm *ConnectionManager) createTenantConnectionByName(ctx context.Context, tenantID, dbName string) (*sql.DB, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if db, exists := cm.tenantConnections[tenantID]; exists {
		return db, nil
	}

	dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		cm.dbHost, cm.dbPort, dbName, cm.dbUsername, cm.dbPassword, cm.dbSSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		cm.logger.WithFields(logrus.Fields{"tenant_id": tenantID, "db_name": dbName}).WithError(err).Error("Failed to open tenant database")
		return nil, spireerrors.NewInternalError("Failed to connect to tenant database", err)
	}

	db.SetMaxOpenConns(cm.maxOpenConns)
	db.SetMaxIdleConns(cm.maxIdleConns)
	db.SetConnMaxLifetime(cm.connMaxLifetime)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		cm.logger.WithFields(logrus.Fields{"tenant_id": tenantID, "db_name": dbName}).WithError(err).Error("Failed to ping tenant database")
		return nil, spireerrors.NewInternalError("Failed to connect to tenant database", err)
	}

	cm.tenantConnections[tenantID] = db
	cm.logger.WithFields(logrus.Fields{"tenant_id": tenantID, "db_name": dbName}).Info("Created tenant database connection by name")
	return db, nil
}

func (cm *ConnectionManager) removeTenantConnection(tenantID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if db, exists := cm.tenantConnections[tenantID]; exists {
		db.Close()
		delete(cm.tenantConnections, tenantID)
		cm.logger.WithField("tenant_id", tenantID).Info("Removed tenant database connection")
	}
}

// Close closes all connections
func (cm *ConnectionManager) Close() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var lastErr error
	for tenantID, db := range cm.tenantConnections {
		if err := db.Close(); err != nil {
			cm.logger.WithField("tenant_id", tenantID).WithError(err).Error("Failed to close tenant connection")
			lastErr = err
		}
	}
	return lastErr
}

// GetMasterDB returns the master database connection
func (cm *ConnectionManager) GetMasterDB() *sql.DB {
	return cm.masterDB
}
