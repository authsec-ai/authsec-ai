package services

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
)

// RiskEngineService evaluates risk for agent actions using tenant-configurable policies
type RiskEngineService struct {
	actionRepo *database.AgentActionRepository
}

// NewRiskEngineService creates a new risk engine service
func NewRiskEngineService(actionRepo *database.AgentActionRepository) *RiskEngineService {
	return &RiskEngineService{
		actionRepo: actionRepo,
	}
}

// Evaluate scores an agent action against tenant risk policies and settings
func (s *RiskEngineService) Evaluate(
	tenantID uuid.UUID,
	agentID string,
	action string,
	resource string,
	metadata map[string]interface{},
	settings *models.AgentGuardSettings,
) (*models.RiskEvaluation, error) {

	// Load tenant policies
	policies, err := s.actionRepo.GetRiskPoliciesByTenant(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to load risk policies: %w", err)
	}

	// Find the best matching policy
	matchedPolicy := s.matchPolicy(policies, action, resource, metadata)

	// Calculate risk score
	var factors []models.RiskFactor
	totalScore := 0

	if matchedPolicy != nil {
		// Base score from matched policy
		totalScore += matchedPolicy.BaseScore
		factors = append(factors, models.RiskFactor{
			Factor: "base",
			Score:  matchedPolicy.BaseScore,
			Reason: fmt.Sprintf("Policy '%s' matched action=%s resource=%s", matchedPolicy.Name, action, resource),
		})

		// Bulk modifier
		if rowCount := getIntFromMetadata(metadata, "row_count"); rowCount > matchedPolicy.ScopeBulkThreshold {
			totalScore += matchedPolicy.ScopeBulkModifier
			factors = append(factors, models.RiskFactor{
				Factor: "bulk",
				Score:  matchedPolicy.ScopeBulkModifier,
				Reason: fmt.Sprintf("Bulk operation: %d items exceeds threshold of %d", rowCount, matchedPolicy.ScopeBulkThreshold),
			})
		}

		// PII modifier
		if isFlagged(metadata, "pii") || s.isKnownPIIResource(resource) {
			totalScore += matchedPolicy.PIIModifier
			factors = append(factors, models.RiskFactor{
				Factor: "pii",
				Score:  matchedPolicy.PIIModifier,
				Reason: fmt.Sprintf("Resource '%s' contains personally identifiable information", resource),
			})
		}

		// Financial modifier
		if isFlagged(metadata, "financial") || s.isKnownFinancialResource(resource) {
			totalScore += matchedPolicy.FinancialModifier
			factors = append(factors, models.RiskFactor{
				Factor: "financial",
				Score:  matchedPolicy.FinancialModifier,
				Reason: fmt.Sprintf("Resource '%s' involves financial data", resource),
			})
		}

		// Off-hours modifier
		if s.isOffHours(settings) {
			totalScore += matchedPolicy.OffHoursModifier
			factors = append(factors, models.RiskFactor{
				Factor: "off_hours",
				Score:  matchedPolicy.OffHoursModifier,
				Reason: fmt.Sprintf("Action requested outside business hours (%02d:00-%02d:00 %s)",
					settings.BusinessHoursStart, settings.BusinessHoursEnd, settings.BusinessHoursTimezone),
			})
		}

		// First-time modifier
		hasPrior, err := s.actionRepo.HasPriorAction(agentID, action, resource, tenantID)
		if err == nil && !hasPrior {
			totalScore += matchedPolicy.FirstTimeModifier
			factors = append(factors, models.RiskFactor{
				Factor: "first_time",
				Score:  matchedPolicy.FirstTimeModifier,
				Reason: fmt.Sprintf("First time agent '%s' performs %s on %s", agentID, action, resource),
			})
		}

		// Environment modifier
		if env := getStringFromMetadata(metadata, "env"); env == "production" || env == "prod" {
			prodModifier := 20
			totalScore += prodModifier
			factors = append(factors, models.RiskFactor{
				Factor: "production",
				Score:  prodModifier,
				Reason: "Action targets production environment",
			})
		}
	} else {
		// No matching policy — use default scoring based on action verb
		defaultScore := s.defaultActionScore(action)
		totalScore += defaultScore
		factors = append(factors, models.RiskFactor{
			Factor: "default",
			Score:  defaultScore,
			Reason: fmt.Sprintf("No specific policy matched; default score for action '%s'", action),
		})
	}

	// Cap score at 100
	if totalScore > 100 {
		totalScore = 100
	}
	if totalScore < 0 {
		totalScore = 0
	}

	// Determine risk level
	level := s.scoreToLevel(totalScore)

	// Determine approval type using policy-specific overrides or tenant defaults
	approvalType, requiredApprovals := s.determineApproval(totalScore, matchedPolicy, settings)

	var matchedPolicyID *uuid.UUID
	if matchedPolicy != nil {
		matchedPolicyID = &matchedPolicy.ID
	}

	return &models.RiskEvaluation{
		Score:             totalScore,
		Level:             level,
		Factors:           factors,
		ApprovalType:      approvalType,
		RequiredApprovals: requiredApprovals,
		MatchedPolicyID:   matchedPolicyID,
	}, nil
}

// matchPolicy finds the highest-priority policy that matches the action + resource
func (s *RiskEngineService) matchPolicy(policies []models.RiskPolicy, action, resource string, metadata map[string]interface{}) *models.RiskPolicy {
	env := getStringFromMetadata(metadata, "env")
	if env == "" {
		env = "default"
	}

	for i := range policies {
		p := &policies[i]

		actionMatch, _ := filepath.Match(strings.ToUpper(p.ActionPattern), strings.ToUpper(action))
		if !actionMatch {
			actionMatch = strings.EqualFold(p.ActionPattern, action) ||
				strings.EqualFold(p.ActionPattern, "*")
		}

		resourceMatch, _ := filepath.Match(strings.ToLower(p.ResourcePattern), strings.ToLower(resource))
		if !resourceMatch {
			resourceMatch = strings.EqualFold(p.ResourcePattern, resource) ||
				strings.EqualFold(p.ResourcePattern, "*")
		}

		envMatch, _ := filepath.Match(strings.ToLower(p.EnvironmentPattern), strings.ToLower(env))
		if !envMatch {
			envMatch = strings.EqualFold(p.EnvironmentPattern, env) ||
				strings.EqualFold(p.EnvironmentPattern, "*")
		}

		if actionMatch && resourceMatch && envMatch {
			return p
		}
	}

	return nil
}

// defaultActionScore returns a baseline score when no policy matches
func (s *RiskEngineService) defaultActionScore(action string) int {
	upper := strings.ToUpper(action)
	switch {
	case strings.Contains(upper, "READ") || strings.Contains(upper, "GET") || strings.Contains(upper, "LIST") || strings.Contains(upper, "SELECT"):
		return 10
	case strings.Contains(upper, "CREATE") || strings.Contains(upper, "INSERT") || strings.Contains(upper, "ADD"):
		return 30
	case strings.Contains(upper, "UPDATE") || strings.Contains(upper, "MODIFY") || strings.Contains(upper, "EDIT"):
		return 45
	case strings.Contains(upper, "SEND") || strings.Contains(upper, "EMAIL") || strings.Contains(upper, "NOTIFY"):
		return 50
	case strings.Contains(upper, "DELETE") || strings.Contains(upper, "DROP") || strings.Contains(upper, "REMOVE"):
		return 70
	case strings.Contains(upper, "TRANSFER") || strings.Contains(upper, "PAY") || strings.Contains(upper, "CHARGE"):
		return 80
	case strings.Contains(upper, "DEPLOY") || strings.Contains(upper, "MIGRATE") || strings.Contains(upper, "ROLLBACK"):
		return 75
	default:
		return 50
	}
}

// scoreToLevel converts a numeric score to a risk level string
func (s *RiskEngineService) scoreToLevel(score int) string {
	switch {
	case score <= 30:
		return models.RiskLevelLow
	case score <= 60:
		return models.RiskLevelMedium
	case score <= 80:
		return models.RiskLevelHigh
	default:
		return models.RiskLevelCritical
	}
}

// determineApproval decides the approval type based on score and thresholds
func (s *RiskEngineService) determineApproval(score int, policy *models.RiskPolicy, settings *models.AgentGuardSettings) (string, int) {
	autoBelow := settings.AutoApproveBelow
	approvalAbove := settings.RequireApprovalAbove
	multiAbove := settings.RequireMultiApprovalAbove

	if policy != nil {
		if policy.AutoApproveBelow != nil {
			autoBelow = *policy.AutoApproveBelow
		}
		if policy.RequireApprovalAbove != nil {
			approvalAbove = *policy.RequireApprovalAbove
		}
		if policy.RequireMultiApprovalAbove != nil {
			multiAbove = *policy.RequireMultiApprovalAbove
		}
	}

	switch {
	case score <= autoBelow:
		return "auto", 0
	case score >= multiAbove:
		return "multi", 2
	case score >= approvalAbove:
		return "single", 1
	default:
		return "auto", 0
	}
}

// isOffHours checks if the current time is outside business hours
func (s *RiskEngineService) isOffHours(settings *models.AgentGuardSettings) bool {
	loc, err := time.LoadLocation(settings.BusinessHoursTimezone)
	if err != nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)
	hour := now.Hour()
	weekday := now.Weekday()

	if weekday == time.Saturday || weekday == time.Sunday {
		return true
	}

	return hour < settings.BusinessHoursStart || hour >= settings.BusinessHoursEnd
}

// isKnownPIIResource checks if a resource name suggests PII
func (s *RiskEngineService) isKnownPIIResource(resource string) bool {
	lower := strings.ToLower(resource)
	piiKeywords := []string{"user", "customer", "patient", "employee", "person", "profile", "identity", "credential", "password", "ssn", "passport"}
	for _, kw := range piiKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// isKnownFinancialResource checks if a resource name suggests financial data
func (s *RiskEngineService) isKnownFinancialResource(resource string) bool {
	lower := strings.ToLower(resource)
	finKeywords := []string{"billing", "payment", "invoice", "transaction", "transfer", "account", "balance", "wallet", "credit", "subscription"}
	for _, kw := range finKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// ========================================
// Metadata helpers
// ========================================

func getIntFromMetadata(metadata map[string]interface{}, key string) int {
	if metadata == nil {
		return 0
	}
	v, ok := metadata[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}

func getStringFromMetadata(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	v, ok := metadata[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func isFlagged(metadata map[string]interface{}, flag string) bool {
	if metadata == nil {
		return false
	}

	if v, ok := metadata[flag]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}

	if tags, ok := metadata["tags"]; ok {
		switch t := tags.(type) {
		case []interface{}:
			for _, tag := range t {
				if s, ok := tag.(string); ok && strings.EqualFold(s, flag) {
					return true
				}
			}
		case []string:
			for _, tag := range t {
				if strings.EqualFold(tag, flag) {
					return true
				}
			}
		}
	}

	return false
}
