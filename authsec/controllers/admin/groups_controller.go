package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	amMiddlewares "github.com/authsec-ai/auth-manager/pkg/middlewares"
	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/controllers/shared"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GroupController struct{}

// AdminGroupListRequest represents the payload for admin tenant group listing
type AdminGroupListRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	UserID   string `json:"user_id"`
}

// GroupRequest handles both string and object formats for groups
type GroupRequest struct {
	TenantID  string          `json:"tenant_id" binding:"required"`
	ClientID  string          `json:"client_id,omitempty"`
	ProjectID string          `json:"project_id,omitempty"`
	Groups    json.RawMessage `json:"groups" binding:"required"`
}

// GroupItem represents a group with a name
type GroupItem struct {
	Name string `json:"name" binding:"required"`
}

// function to add user defined groups to the groups table in db

// AddUserDefinedGroups godoc
// @Summary Add user-defined groups
// @Description Adds custom groups under a tenant
// @Tags Groups
// @Accept json
// @Produce json
// @Param input body object true "Groups payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/groups [post]
func (gc *GroupController) AddUserDefinedGroups(c *gin.Context) {
	var req GroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if req.TenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TenantID is required"})
		return
	}

	// Parse groups - handle both string array and object array formats
	var groupNames []string

	// First try to parse as array of strings
	var stringGroups []string
	if err := json.Unmarshal(req.Groups, &stringGroups); err == nil {
		groupNames = stringGroups
	} else {
		// Try to parse as array of objects with "name" field
		var objectGroups []GroupItem
		if err := json.Unmarshal(req.Groups, &objectGroups); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Groups must be either array of strings or array of objects with 'name' field"})
			return
		}
		// Extract names from objects
		for _, group := range objectGroups {
			if group.Name == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Group name cannot be empty"})
				return
			}
			groupNames = append(groupNames, group.Name)
		}
	}

	if len(groupNames) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one group is required"})
		return
	}

	createdGroups, err := AddUserDefinedGroups(req.TenantID, groupNames)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add groups: " + err.Error()})
		return
	}

	// Audit log: Groups created
	middlewares.Audit(c, "group", req.TenantID, "create", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"tenant_id":    req.TenantID,
			"groups_count": len(createdGroups),
			"group_names":  groupNames,
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Groups added successfully",
		"groups":  createdGroups,
	})
}

// function to map groups to client in client_groups table

// MapGroupsToClient godoc
// @Summary Map groups to client
// @Description Maps groups to a client under a tenant
// @Tags Groups
// @Accept json
// @Produce json
// @Param input body object true "Mapping payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/groups/map [post]
func (gc *GroupController) MapGroupsToClient(c *gin.Context) {
	var req models.MapGroupsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if req.TenantID == "" || req.ClientID == "" || len(req.Groups) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TenantID, ClientID and Groups are required"})
		return
	}

	if err := MapGroupsToClient(req.TenantID, req.ClientID, req.Groups); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to map groups to client: " + err.Error()})
		return
	}

	// Audit log: Groups mapped to client
	middlewares.Audit(c, "group", req.ClientID, "map_to_client", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"tenant_id": req.TenantID,
			"client_id": req.ClientID,
			"groups":    req.Groups,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Groups mapped to client successfully"})
}

// RemoveGroupsFromClient godoc
// @Summary Remove groups from client
// @Description Removes group associations from a client under a tenant
// @Tags Groups
// @Accept json
// @Produce json
// @Param input body object true "Unmapping payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/groups/map [delete]
func (gc *GroupController) RemoveGroupsFromClient(c *gin.Context) {
	var req models.RemoveGroupsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if req.TenantID == "" || req.ClientID == "" || len(req.Groups) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TenantID, ClientID and Groups are required"})
		return
	}

	if err := RemoveGroupsFromClient(req.TenantID, req.ClientID, req.Groups); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove groups from client: " + err.Error()})
		return
	}

	// Audit log: Groups removed from client
	middlewares.Audit(c, "group", req.ClientID, "unmap_from_client", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"tenant_id": req.TenantID,
			"client_id": req.ClientID,
			"groups":    req.Groups,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Groups removed from client successfully"})
}

// function to get user defined groups for a tenant

// GetUserDefinedGroups godoc
// @Summary Get user-defined groups
// @Description Retrieves all groups created by the tenant
// @Tags Groups
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/groups/{tenant_id} [get]
func (gc *GroupController) GetUserDefinedGroups(c *gin.Context) {
	// Get tenant_id from validated JWT token (not URL parameter to prevent spoofing)
	tenantID, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in authentication token"})
		return
	}

	groups, err := GetUserDefinedGroups(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

// AddUsersToGroup godoc
// @Summary Add multiple users to a group
// @Description Adds multiple users to a specified group within a tenant. Supports large arrays of user IDs.
// @Tags Groups
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Param input body models.AddUsersToGroupRequest true "Add users to group payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/groups/{tenant_id}/users/bulk [post]
func (gc *GroupController) AddUsersToGroup(c *gin.Context) {
	// Get tenant_id from validated JWT token (not URL parameter to prevent spoofing)
	tenantID, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in authentication token"})
		return
	}

	var req models.AddUsersToGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if len(req.UserIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one user ID is required"})
		return
	}

	if err := AddUsersToGroupBulk(tenantID, req.GroupID, req.UserIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add users to group: " + err.Error()})
		return
	}

	// Audit log: Users added to group
	middlewares.Audit(c, "group", req.GroupID.String(), "add_users", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"tenant_id":  tenantID,
			"group_id":   req.GroupID.String(),
			"user_count": len(req.UserIDs),
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"message":    "Users added to group successfully",
		"group_id":   req.GroupID.String(),
		"user_count": len(req.UserIDs),
	})
}

// RemoveUsersFromGroup godoc
// @Summary Remove multiple users from a group
// @Description Removes multiple users from a specified group within a tenant. Supports large arrays of user IDs.
// @Tags Groups
// @Accept json
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Param input body models.RemoveUsersFromGroupRequest true "Remove users from group payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/groups/{tenant_id}/users/bulk [delete]
func (gc *GroupController) RemoveUsersFromGroup(c *gin.Context) {
	// Get tenant_id from validated JWT token (not URL parameter to prevent spoofing)
	tenantID, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in authentication token"})
		return
	}

	var req models.RemoveUsersFromGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if len(req.UserIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one user ID is required"})
		return
	}

	if err := RemoveUsersFromGroupBulk(tenantID, req.GroupID, req.UserIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove users from group: " + err.Error()})
		return
	}

	// Audit log: Users removed from group
	middlewares.Audit(c, "group", req.GroupID.String(), "remove_users", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"tenant_id":  tenantID,
			"group_id":   req.GroupID.String(),
			"user_count": len(req.UserIDs),
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"message":    "Users removed from group successfully",
		"group_id":   req.GroupID.String(),
		"user_count": len(req.UserIDs),
	})
}

// function to delete user defined groups from the groups table

// DeleteUserDefinedGroups godoc
// @Summary Delete user-defined groups
// @Description Deletes groups from the database for a tenant
// @Tags Groups
// @Accept json
// @Produce json
// @Param input body object true "Delete groups payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/groups [delete]
func (gc *GroupController) DeleteUserDefinedGroups(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: unable to read body"})
		return
	}

	// Get tenant_id from validated JWT token
	tenantID, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found in authentication token"})
		return
	}

	tenantID, groups, parseErr := parseDeleteGroupsPayload(body)
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + parseErr.Error()})
		return
	}

	// Override with token tenant_id for security
	tenantID, _ = amMiddlewares.GetTenantIDFromToken(c)

	queryGroups := []string{}
	queryGroups = append(queryGroups, c.QueryArray("group_ids")...)

	if single := c.Query("group"); single != "" {
		queryGroups = append(queryGroups, single)
	}
	groups = uniqueStrings(append(groups, queryGroups...))

	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID is missing"})
		return
	}

	if len(groups) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one group-id is required"})
		return
	}

	if err := DeleteUserDefinedGroups(tenantID, groups); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete groups: " + err.Error()})
		return
	}

	// Audit log: Groups deleted
	middlewares.Audit(c, "group", tenantID, "delete", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"tenant_id":   tenantID,
			"group_ids":   groups,
			"group_count": len(groups),
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Groups deleted successfully"})
}

func parseDeleteGroupsPayload(body []byte) (string, []string, error) {
	if len(body) == 0 {
		return "", nil, nil
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", nil, err
	}

	var tenantID string
	if value, ok := raw["tenant_id"].(string); ok {
		tenantID = value
	} else if value, ok := raw["tenantId"].(string); ok {
		tenantID = value
	}

	groupKeys := []string{"groups", "group_names", "groupNames", "Groups"}
	var groups []string
	for _, key := range groupKeys {
		if value, ok := raw[key]; ok {
			groups = append(groups, coerceGroupValues(value)...)
		}
	}

	return tenantID, uniqueStrings(groups), nil
}

func coerceGroupValues(value interface{}) []string {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		return splitAndTrim(v)
	case []interface{}:
		var result []string
		for _, item := range v {
			result = append(result, coerceGroupValues(item)...)
		}
		return result
	case map[string]interface{}:
		keys := []string{"name", "value", "id", "group_name", "groupName"}
		var result []string
		for _, key := range keys {
			if str, ok := v[key].(string); ok {
				result = append(result, splitAndTrim(str)...)
			}
		}

		nestedKeys := []string{"values", "names", "items", "groups", "selected"}
		for _, key := range nestedKeys {
			if nested, ok := v[key]; ok {
				result = append(result, coerceGroupValues(nested)...)
			}
		}
		return result
	default:
		// attempt best-effort string conversion for other scalar types
		str := strings.TrimSpace(fmt.Sprint(v))
		if str == "" || str == "0" || str == "<nil>" || str == "false" {
			return nil
		}
		return []string{str}
	}
}

func splitAndTrim(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	if !strings.Contains(value, ",") {
		return []string{value}
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}

// UpdateUserDefinedGroup godoc
// @Summary Update a user-defined group
// @Description Updates the name and/or description of a specific group
// @Tags Groups
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param input body object true "Update group payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/groups/{id} [put]
func (gc *GroupController) UpdateUserDefinedGroup(c *gin.Context) {
	groupID := c.Param("id")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group ID is required"})
		return
	}

	var req struct {
		TenantID    string `json:"tenant_id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if err := UpdateUserDefinedGroup(groupID, req.TenantID, req.Name, req.Description); err != nil {
		if err.Error() == "group not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group: " + err.Error()})
		return
	}

	// Audit log: Group updated
	middlewares.Audit(c, "group", groupID, "update", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"tenant_id":   req.TenantID,
			"group_id":    groupID,
			"name":        req.Name,
			"description": req.Description,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Group updated successfully"})
}

// AddUserToGroups godoc
// @Summary Add user to groups
// @Description Adds a user to specified groups within a tenant
// @Tags Groups
// @Accept json
// @Produce json
// @Param input body object true "Add user to groups payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/groups/users/add [post]
func (gc *GroupController) AddUserToGroups(c *gin.Context) {
	var req struct {
		TenantID string   `json:"tenant_id" binding:"required"`
		UserID   string   `json:"user_id" binding:"required"`
		Groups   []string `json:"groups" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if req.TenantID == "" || req.UserID == "" || len(req.Groups) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TenantID, UserID and Groups are required"})
		return
	}

	if err := AddUserToGroups(req.TenantID, req.UserID, req.Groups); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user to groups: " + err.Error()})
		return
	}

	// Audit log: User added to groups
	middlewares.Audit(c, "group", req.UserID, "add_user_to_groups", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"tenant_id": req.TenantID,
			"user_id":   req.UserID,
			"groups":    req.Groups,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "User added to groups successfully"})
}

// RemoveUserFromGroups godoc
// @Summary Remove user from groups
// @Description Removes a user from specified groups within a tenant
// @Tags Groups
// @Accept json
// @Produce json
// @Param input body object true "Remove user from groups payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/groups/users/remove [post]
func (gc *GroupController) RemoveUserFromGroups(c *gin.Context) {
	var req struct {
		TenantID string   `json:"tenant_id" binding:"required"`
		UserID   string   `json:"user_id" binding:"required"`
		Groups   []string `json:"groups" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if req.TenantID == "" || req.UserID == "" || len(req.Groups) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TenantID, UserID and Groups are required"})
		return
	}

	if err := RemoveUserFromGroups(req.TenantID, req.UserID, req.Groups); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove user from groups: " + err.Error()})
		return
	}

	// Audit log: User removed from groups
	middlewares.Audit(c, "group", req.UserID, "remove_user_from_groups", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"tenant_id": req.TenantID,
			"user_id":   req.UserID,
			"groups":    req.Groups,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "User removed from groups successfully"})
}

// GetMyGroups godoc
// @Summary Get current user's groups
// @Description Retrieves all groups the authenticated user belongs to within their active tenant
// @Tags Groups
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/user/groups/users [get]
func (gc *GroupController) GetMyGroups(c *gin.Context) {
	userID, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	tenantID, err := shared.RequireTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	groups, err := GetUserGroups(tenantID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups: " + err.Error()})
		return
	}

	users, err := fetchTenantGroupUsers(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"groups": groups,
		"users":  users,
	}

	if requestHasAdminRole(c) {
		admins, err := fetchTenantAdmins(tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		response["admins"] = admins
	}

	c.JSON(http.StatusOK, response)
}

// ListTenantGroupsForAdmin godoc
// @Summary List groups within a tenant (admin)
// @Description Retrieves groups for a tenant; optionally filter by user membership
// @Tags Admin-Groups
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body AdminGroupListRequest true "Tenant group listing payload"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/admin/groups/list [post]
func (gc *GroupController) ListTenantGroupsForAdmin(c *gin.Context) {
	var req AdminGroupListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var (
		groups []models.TenantGroup
		err    error
	)

	if strings.TrimSpace(req.UserID) != "" {
		groups, err = GetUserGroups(req.TenantID, req.UserID)
	} else {
		groups, err = GetUserDefinedGroups(req.TenantID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	groupResponses := make([]gin.H, 0, len(groups))
	for _, group := range groups {
		groupID := group.ID.String()
		members, err := GetGroupUsers(req.TenantID, groupID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to fetch users for group %s: %v", groupID, err)})
			return
		}

		groupResponses = append(groupResponses, gin.H{
			"group": group,
			"users": members,
		})
	}

	c.JSON(http.StatusOK, gin.H{"groups": groupResponses})
}

// GetGroupUsers godoc
// @Summary Get group users
// @Description Retrieves all users in a specific group within a tenant
// @Tags Groups
// @Produce json
// @Param tenant_id path string true "Tenant ID"
// @Param group_id path string true "Group ID"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/groups/{tenant_id}/{group_id}/users [get]
func (gc *GroupController) GetGroupUsers(c *gin.Context) {
	// Get tenant_id from validated JWT token (not URL parameter to prevent spoofing)
	tenantID, ok := amMiddlewares.GetTenantIDFromToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in authentication token"})
		return
	}

	groupID := c.Param("group_id")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "GroupID is required"})
		return
	}

	users, err := GetGroupUsers(tenantID, groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group users: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// Database helper functions for group operations

func AddUserDefinedGroups(tenantID string, groups []string) ([]models.TenantGroup, error) {
	// Check if config.DB is available
	if config.DB == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	// Connect to tenant database
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	var createdGroups []models.TenantGroup
	for _, groupName := range groups {
		group := models.TenantGroup{
			Name:     groupName,
			TenantID: uuid.MustParse(tenantID),
		}
		if err := tenantDB.Where("name = ? AND tenant_id = ?", groupName, tenantID).FirstOrCreate(&group).Error; err != nil {
			return nil, err
		}
		createdGroups = append(createdGroups, group)
	}
	return createdGroups, nil
}

func MapGroupsToClient(tenantID, clientID string, groups []string) error {
	// Check if config.DB is available
	if config.DB == nil {
		return fmt.Errorf("database connection not available")
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant ID format: %w", err)
	}

	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		return fmt.Errorf("invalid client ID format: %w", err)
	}

	// Connect to tenant database
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	// Find the user in tenant database
	var user models.User
	if err := tenantDB.Where("client_id = ? AND tenant_id = ?", clientUUID, tenantUUID).First(&user).Error; err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Find the groups in tenant database
	var groupModels []models.TenantGroup
	if err := tenantDB.Where("name IN ? AND tenant_id = ?", groups, tenantUUID).Find(&groupModels).Error; err != nil {
		return fmt.Errorf("failed to find groups: %w", err)
	}

	if len(groupModels) == 0 {
		return fmt.Errorf("no matching groups found in tenant")
	}

	// Insert into user_groups table
	for _, group := range groupModels {
		if err := tenantDB.Exec(
			"INSERT INTO user_groups (user_id, group_id, tenant_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
			user.ID, group.ID, tenantUUID,
		).Error; err != nil {
			return fmt.Errorf("failed to map group to user: %w", err)
		}
	}

	return nil
}

func RemoveGroupsFromClient(tenantID, clientID string, groups []string) error {
	if config.DB == nil {
		return fmt.Errorf("database connection not available")
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant ID format: %w", err)
	}

	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		return fmt.Errorf("invalid client ID format: %w", err)
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	var user models.User
	if err := tenantDB.Where("client_id = ? AND tenant_id = ?", clientUUID, tenantUUID).First(&user).Error; err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	var groupModels []models.TenantGroup
	if err := tenantDB.Where("name IN ? AND tenant_id = ?", groups, tenantUUID).Find(&groupModels).Error; err != nil {
		return fmt.Errorf("failed to find groups: %w", err)
	}

	for _, group := range groupModels {
		if err := tenantDB.Exec(
			"DELETE FROM user_groups WHERE user_id = $1 AND group_id = $2 AND tenant_id = $3",
			user.ID, group.ID, tenantUUID,
		).Error; err != nil {
			return fmt.Errorf("failed to remove group from user: %w", err)
		}
	}

	return nil
}

func GetUserDefinedGroups(tenantID string) ([]models.TenantGroup, error) {
	// Check if config.DB is available
	if config.DB == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID format: %w", err)
	}

	// Connect to tenant database
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	var groups []models.TenantGroup
	// Query groups from tenant database
	if err := tenantDB.Where("tenant_id = ? OR tenant_id IS NULL", tenantUUID).Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("failed to query groups: %w", err)
	}
	return groups, nil
}

func DeleteUserDefinedGroups(tenantID string, groups []string) error {
	// Check if config.DB is available
	if config.DB == nil {
		return fmt.Errorf("database connection not available")
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant ID format: %w", err)
	}

	// Connect to tenant database
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	return tenantDB.Where("id IN ? AND tenant_id = ?", groups, tenantUUID).Delete(&models.TenantGroup{}).Error
}

func UpdateUserDefinedGroup(groupID, tenantID, name, description string) error {
	if config.DB == nil {
		return fmt.Errorf("database connection not available")
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant ID format: %w", err)
	}

	// Connect to tenant database
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	// Parse group ID as UUID
	groupUUID, err := uuid.Parse(groupID)
	if err != nil {
		return fmt.Errorf("invalid group ID format: %w", err)
	}

	// Update the group
	updateData := models.TenantGroup{
		Name:        name,
		Description: &description,
	}

	return tenantDB.Model(&models.TenantGroup{}).Where("id = ? AND tenant_id = ?", groupUUID, tenantUUID).Updates(updateData).Error
}

// AddUserToGroups adds a user to specified groups within a tenant
func AddUserToGroups(tenantID, userID string, groups []string) error {
	if config.DB == nil {
		return fmt.Errorf("database connection not available")
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant ID format: %w", err)
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	// Parse user ID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	var user models.User
	if err := tenantDB.Where("id = ? AND tenant_id = ?", userUUID, tenantUUID).First(&user).Error; err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Find the groups in tenant database
	var groupModels []models.TenantGroup
	if err := tenantDB.Where("name IN ? AND tenant_id = ?", groups, tenantUUID).Find(&groupModels).Error; err != nil {
		return fmt.Errorf("failed to find groups: %w", err)
	}

	if len(groupModels) == 0 {
		return fmt.Errorf("no matching groups found in tenant")
	}

	// Insert into user_groups table
	for _, group := range groupModels {
		if err := tenantDB.Exec(
			"INSERT INTO user_groups (user_id, group_id, tenant_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
			userUUID, group.ID, tenantUUID,
		).Error; err != nil {
			return fmt.Errorf("failed to add user to group: %w", err)
		}
	}

	return nil
}

// RemoveUserFromGroups removes a user from specified groups within a tenant
func RemoveUserFromGroups(tenantID, userID string, groups []string) error {
	if config.DB == nil {
		return fmt.Errorf("database connection not available")
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant ID format: %w", err)
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	// Parse user ID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	var user models.User
	if err := tenantDB.Where("id = ? AND tenant_id = ?", userUUID, tenantUUID).First(&user).Error; err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Find the groups in tenant database
	var groupModels []models.TenantGroup
	if err := tenantDB.Where("name IN ? AND tenant_id = ?", groups, tenantUUID).Find(&groupModels).Error; err != nil {
		return fmt.Errorf("failed to find groups: %w", err)
	}

	// Delete from user_groups table
	for _, group := range groupModels {
		if err := tenantDB.Exec(
			"DELETE FROM user_groups WHERE user_id = $1 AND group_id = $2 AND tenant_id = $3",
			userUUID, group.ID, tenantUUID,
		).Error; err != nil {
			return fmt.Errorf("failed to remove user from group: %w", err)
		}
	}

	return nil
}

// GetUserGroups retrieves all groups a user belongs to within a tenant
func GetUserGroups(tenantID, userID string) ([]models.TenantGroup, error) {
	if config.DB == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID format: %w", err)
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	// Parse user ID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	groups := make([]models.TenantGroup, 0)
	query := `
		SELECT g.* FROM groups g
		INNER JOIN user_groups ug ON g.id = ug.group_id
		WHERE ug.user_id = $1 AND g.tenant_id = $2 AND ug.tenant_id = $2
	`
	if err := tenantDB.Raw(query, userUUID, tenantUUID).Scan(&groups).Error; err != nil {
		return nil, fmt.Errorf("failed to query user groups: %w", err)
	}

	return groups, nil
}

type groupUserSummary struct {
	ID       uuid.UUID  `json:"id"`
	Email    string     `json:"email"`
	Name     string     `json:"name"`
	Provider string     `json:"provider"`
	ClientID *uuid.UUID `json:"client_id,omitempty"`
	Active   bool       `json:"active"`
}

type groupAdminSummary struct {
	ID       uuid.UUID  `json:"id"`
	Email    string     `json:"email"`
	Name     string     `json:"name"`
	ClientID *uuid.UUID `json:"client_id,omitempty"`
	Active   bool       `json:"active"`
}

func fetchTenantGroupUsers(tenantID string) ([]groupUserSummary, error) {
	if config.DB == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	users := make([]groupUserSummary, 0)
	if err := tenantDB.Table("users").
		Select("id, email, name, provider, client_id, active").
		Order("LOWER(email) ASC").
		Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to query tenant users: %w", err)
	}

	return users, nil
}

func fetchTenantAdmins(tenantID string) ([]groupAdminSummary, error) {
	db := config.GetDatabase()
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID format: %w", err)
	}

	adminRepo := database.NewAdminUserRepository(db)
	admins, err := adminRepo.ListAdminUsersByTenant(tenantUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tenant admins: %w", err)
	}

	summaries := make([]groupAdminSummary, 0, len(admins))
	for _, admin := range admins {
		summaries = append(summaries, groupAdminSummary{
			ID:       admin.ID,
			Email:    admin.Email,
			Name:     admin.Name,
			ClientID: admin.ClientID,
			Active:   admin.Active,
		})
	}

	return summaries, nil
}

func requestHasAdminRole(c *gin.Context) bool {
	userInfo := middlewares.GetUserInfo(c)
	if userInfo == nil {
		return false
	}

	for _, role := range userInfo.Roles {
		if strings.EqualFold(role, "admin") || strings.EqualFold(role, "administrator") || strings.EqualFold(role, "super_admin") {
			return true
		}
	}

	return false
}

// GetGroupUsers retrieves all users in a specific group within a tenant
func GetGroupUsers(tenantID, groupID string) ([]models.User, error) {
	if config.DB == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID format: %w", err)
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	// Parse group ID
	groupUUID, err := uuid.Parse(groupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID format: %w", err)
	}

	var users []models.User
	query := `
		SELECT u.* FROM users u
		INNER JOIN user_groups ug ON u.id = ug.user_id
		WHERE ug.group_id = $1 AND ug.tenant_id = $2
	`
	if err := tenantDB.Raw(query, groupUUID, tenantUUID).Scan(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to query group users: %w", err)
	}

	return users, nil
}

// AddUsersToGroupBulk adds multiple users to a group efficiently
func AddUsersToGroupBulk(tenantID string, groupID uuid.UUID, userIDs []uuid.UUID) error {
	if config.DB == nil {
		return fmt.Errorf("database connection not available")
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant ID format: %w", err)
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	// Verify the group exists
	var group models.TenantGroup
	if err := tenantDB.Where("id = ? AND tenant_id = ?", groupID, tenantUUID).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("group not found in tenant")
		}
		return fmt.Errorf("failed to verify group: %w", err)
	}

	// Begin transaction for bulk insert
	tx := tenantDB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Insert user-group associations in batches for better performance
	batchSize := 100
	for i := 0; i < len(userIDs); i += batchSize {
		end := i + batchSize
		if end > len(userIDs) {
			end = len(userIDs)
		}
		batch := userIDs[i:end]

		for _, userID := range batch {
			// Use raw SQL with ON CONFLICT DO NOTHING to handle duplicates
			if err := tx.Exec(
				"INSERT INTO user_groups (user_id, group_id, tenant_id, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW()) ON CONFLICT (user_id, group_id) DO NOTHING",
				userID, groupID, tenantUUID,
			).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to add users to group: %w", err)
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// RemoveUsersFromGroupBulk removes multiple users from a group efficiently
func RemoveUsersFromGroupBulk(tenantID string, groupID uuid.UUID, userIDs []uuid.UUID) error {
	if config.DB == nil {
		return fmt.Errorf("database connection not available")
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant ID format: %w", err)
	}

	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	// Verify the group exists
	var group models.TenantGroup
	if err := tenantDB.Where("id = ? AND tenant_id = ?", groupID, tenantUUID).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("group not found in tenant")
		}
		return fmt.Errorf("failed to verify group: %w", err)
	}

	// Delete user-group associations in batches
	batchSize := 100
	for i := 0; i < len(userIDs); i += batchSize {
		end := i + batchSize
		if end > len(userIDs) {
			end = len(userIDs)
		}
		batch := userIDs[i:end]

		if err := tenantDB.Where("group_id = ? AND tenant_id = ? AND user_id IN ?", groupID, tenantUUID, batch).
			Delete(&models.UserGroup{}).Error; err != nil {
			return fmt.Errorf("failed to remove users from group: %w", err)
		}
	}

	return nil
}
