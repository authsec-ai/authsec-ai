package admin

import (
	"net/http"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Note: middlewares.Audit is used for audit logging

type ProjectController struct{}

// CreateProject godoc
// @Summary Create a new project
// @Description Creates a project under the authenticated user
// @Tags Projects
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body object true "Project creation payload"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/projects [post]
func (pc *ProjectController) CreateProject(c *gin.Context) {
	userID, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	var input models.ProjectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project := models.Project{
		Name:        input.Name,
		Description: input.Description,
		UserID:      userUUID,
	}

	if config.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})
		return
	}

	if err := config.DB.Create(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	// Reload project with User association, excluding Password
	var reloadedProject models.Project
	if err := config.DB.First(&reloadedProject, project.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load project details"})
		return
	}

	// Create response struct to exclude Password
	response := models.ProjectResponse{
		ID:          reloadedProject.ID,
		CreatedAt:   reloadedProject.CreatedAt,
		UpdatedAt:   reloadedProject.UpdatedAt,
		DeletedAt:   reloadedProject.DeletedAt,
		Name:        reloadedProject.Name,
		Description: reloadedProject.Description,
		UserID:      reloadedProject.UserID,
		TenantID:    reloadedProject.TenantID,
		Active:      reloadedProject.Active,
	}

	// Audit log: Project created
	middlewares.Audit(c, "project", reloadedProject.ID.String(), "create", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"project_id":  reloadedProject.ID.String(),
			"name":        reloadedProject.Name,
			"description": reloadedProject.Description,
			"user_id":     reloadedProject.UserID.String(),
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Project created successfully", "project": response})
}

// ListProjects godoc
// @Summary List all projects
// @Description Lists all projects created by the authenticated user
// @Tags Projects
// @Security BearerAuth
// @Produce json
// @Success 200 {object} object
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /uflow/projects [get]
func (pc *ProjectController) ListProjects(c *gin.Context) {
	userID, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if config.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	var projects []models.Project
	if err := config.DB.Where("user_id = ?", userUUID).Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects"})
		return
	}

	// Create response struct to exclude Password
	response := []models.ProjectResponse{}
	for _, p := range projects {
		projectResponse := models.ProjectResponse{
			ID:          p.ID,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
			DeletedAt:   p.DeletedAt,
			Name:        p.Name,
			Description: p.Description,
			UserID:      p.UserID,
			TenantID:    p.TenantID,
			Active:      p.Active,
		}

		response = append(response, projectResponse)
	}

	c.JSON(http.StatusOK, gin.H{"projects": response})
}
