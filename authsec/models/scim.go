package models

import (
	"time"

	"github.com/google/uuid"
)

// SCIM Schema URIs
const (
	SCIMSchemaUser              = "urn:ietf:params:scim:schemas:core:2.0:User"
	SCIMSchemaGroup             = "urn:ietf:params:scim:schemas:core:2.0:Group"
	SCIMSchemaListResponse      = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	SCIMSchemaPatchOp           = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
	SCIMSchemaError             = "urn:ietf:params:scim:api:messages:2.0:Error"
	SCIMSchemaServiceProvider   = "urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"
	SCIMSchemaSchema            = "urn:ietf:params:scim:schemas:core:2.0:Schema"
	SCIMSchemaResourceType      = "urn:ietf:params:scim:schemas:core:2.0:ResourceType"
)

// SCIMMeta represents SCIM resource metadata
type SCIMMeta struct {
	ResourceType string    `json:"resourceType"`
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"lastModified"`
	Location     string    `json:"location,omitempty"`
}

// SCIMName represents a user's name in SCIM format
type SCIMName struct {
	Formatted  string `json:"formatted,omitempty"`
	FamilyName string `json:"familyName,omitempty"`
	GivenName  string `json:"givenName,omitempty"`
}

// SCIMEmail represents an email in SCIM format
type SCIMEmail struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

// SCIMGroupRef represents a group reference on a user
type SCIMGroupRef struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Ref     string `json:"$ref,omitempty"`
}

// SCIMMemberRef represents a member reference on a group
type SCIMMemberRef struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Ref     string `json:"$ref,omitempty"`
}

// SCIMUser represents a SCIM 2.0 User resource
type SCIMUser struct {
	Schemas    []string       `json:"schemas"`
	ID         string         `json:"id"`
	ExternalID string         `json:"externalId,omitempty"`
	UserName   string         `json:"userName"`
	Name       *SCIMName      `json:"name,omitempty"`
	DisplayName string        `json:"displayName,omitempty"`
	Emails     []SCIMEmail    `json:"emails,omitempty"`
	Active     bool           `json:"active"`
	Groups     []SCIMGroupRef `json:"groups,omitempty"`
	Title      string         `json:"title,omitempty"`
	Department string         `json:"department,omitempty"`
	Meta       *SCIMMeta      `json:"meta,omitempty"`
}

// SCIMGroup represents a SCIM 2.0 Group resource
type SCIMGroup struct {
	Schemas     []string        `json:"schemas"`
	ID          string          `json:"id"`
	ExternalID  string          `json:"externalId,omitempty"`
	DisplayName string          `json:"displayName"`
	Members     []SCIMMemberRef `json:"members,omitempty"`
	Meta        *SCIMMeta       `json:"meta,omitempty"`
}

// SCIMListResponse represents a SCIM 2.0 ListResponse
type SCIMListResponse struct {
	Schemas      []string      `json:"schemas"`
	TotalResults int           `json:"totalResults"`
	StartIndex   int           `json:"startIndex"`
	ItemsPerPage int           `json:"itemsPerPage"`
	Resources    []interface{} `json:"Resources"`
}

// SCIMError represents a SCIM 2.0 error response
type SCIMError struct {
	Schemas  []string `json:"schemas"`
	Detail   string   `json:"detail"`
	Status   string   `json:"status"`
	ScimType string   `json:"scimType,omitempty"`
}

// SCIMPatchOp represents a single SCIM PATCH operation
type SCIMPatchOp struct {
	Op    string      `json:"op"`
	Path  string      `json:"path,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

// SCIMPatchRequest represents a SCIM PATCH request
type SCIMPatchRequest struct {
	Schemas    []string      `json:"schemas"`
	Operations []SCIMPatchOp `json:"Operations"`
}

// SCIMServiceProviderConfig represents the SCIM ServiceProviderConfig response
type SCIMServiceProviderConfig struct {
	Schemas               []string               `json:"schemas"`
	DocumentationURI      string                 `json:"documentationUri,omitempty"`
	Patch                 SCIMSupported           `json:"patch"`
	Bulk                  SCIMBulkConfig          `json:"bulk"`
	Filter                SCIMFilterConfig        `json:"filter"`
	ChangePassword        SCIMSupported           `json:"changePassword"`
	Sort                  SCIMSupported           `json:"sort"`
	ETag                  SCIMSupported           `json:"etag"`
	AuthenticationSchemes []SCIMAuthScheme        `json:"authenticationSchemes"`
	Meta                  *SCIMMeta               `json:"meta,omitempty"`
}

// SCIMSupported represents a boolean-supported feature
type SCIMSupported struct {
	Supported bool `json:"supported"`
}

// SCIMBulkConfig represents bulk operation configuration
type SCIMBulkConfig struct {
	Supported      bool `json:"supported"`
	MaxOperations  int  `json:"maxOperations"`
	MaxPayloadSize int  `json:"maxPayloadSize"`
}

// SCIMFilterConfig represents filter configuration
type SCIMFilterConfig struct {
	Supported  bool `json:"supported"`
	MaxResults int  `json:"maxResults"`
}

// SCIMAuthScheme represents an authentication scheme
type SCIMAuthScheme struct {
	Type             string `json:"type"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	SpecURI          string `json:"specUri,omitempty"`
	DocumentationURI string `json:"documentationUri,omitempty"`
	Primary          bool   `json:"primary,omitempty"`
}

// SCIMSchemaDefinition represents a SCIM Schema definition for discovery
type SCIMSchemaDefinition struct {
	Schemas     []string              `json:"schemas"`
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Attributes  []SCIMSchemaAttribute `json:"attributes"`
	Meta        *SCIMMeta             `json:"meta,omitempty"`
}

// SCIMSchemaAttribute represents a SCIM schema attribute
type SCIMSchemaAttribute struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	MultiValued bool   `json:"multiValued"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Mutability  string `json:"mutability"`
	Returned    string `json:"returned"`
	Uniqueness  string `json:"uniqueness"`
}

// SCIMResourceTypeDefinition represents a SCIM ResourceType definition
type SCIMResourceTypeDefinition struct {
	Schemas     []string  `json:"schemas"`
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Endpoint    string    `json:"endpoint"`
	Schema      string    `json:"schema"`
	Meta        *SCIMMeta `json:"meta,omitempty"`
}

// Helper: NewSCIMListResponse creates a properly formatted list response
func NewSCIMListResponse(resources []interface{}, totalResults, startIndex, itemsPerPage int) SCIMListResponse {
	return SCIMListResponse{
		Schemas:      []string{SCIMSchemaListResponse},
		TotalResults: totalResults,
		StartIndex:   startIndex,
		ItemsPerPage: itemsPerPage,
		Resources:    resources,
	}
}

// Helper: NewSCIMError creates a properly formatted error response
func NewSCIMError(status, detail, scimType string) SCIMError {
	return SCIMError{
		Schemas:  []string{SCIMSchemaError},
		Detail:   detail,
		Status:   status,
		ScimType: scimType,
	}
}

// Helper: UserToSCIMUser converts an ExtendedUser to a SCIM User
func UserToSCIMUser(user ExtendedUser, baseURL string) SCIMUser {
	scimUser := SCIMUser{
		Schemas:     []string{SCIMSchemaUser},
		ID:          user.ID.String(),
		UserName:    user.Email,
		DisplayName: user.Name,
		Active:      user.Active,
		Meta: &SCIMMeta{
			ResourceType: "User",
			Created:      user.CreatedAt,
			LastModified: user.UpdatedAt,
			Location:     baseURL + "/Users/" + user.ID.String(),
		},
	}

	if user.ExternalID != nil {
		scimUser.ExternalID = *user.ExternalID
	}

	if user.Username != nil {
		scimUser.UserName = *user.Username
	}

	if user.Email != "" {
		scimUser.Emails = []SCIMEmail{
			{Value: user.Email, Type: "work", Primary: true},
		}
	}

	return scimUser
}

// Helper: AdminUserToSCIMUser converts an AdminUser to a SCIM User
func AdminUserToSCIMUser(user AdminUser, baseURL string) SCIMUser {
	scimUser := SCIMUser{
		Schemas:     []string{SCIMSchemaUser},
		ID:          user.ID.String(),
		ExternalID:  user.ExternalID,
		UserName:    user.Email,
		DisplayName: user.Name,
		Active:      user.Active,
		Meta: &SCIMMeta{
			ResourceType: "User",
			Created:      user.CreatedAt,
			LastModified: user.UpdatedAt,
			Location:     baseURL + "/Users/" + user.ID.String(),
		},
	}

	if user.Email != "" {
		scimUser.Emails = []SCIMEmail{
			{Value: user.Email, Type: "work", Primary: true},
		}
	}

	if user.Username != "" {
		scimUser.UserName = user.Username
	}

	return scimUser
}

// Helper: TenantGroupToSCIMGroup converts a TenantGroup to a SCIM Group
func TenantGroupToSCIMGroup(group TenantGroup, members []SCIMMemberRef, baseURL string) SCIMGroup {
	return SCIMGroup{
		Schemas:     []string{SCIMSchemaGroup},
		ID:          group.ID.String(),
		DisplayName: group.Name,
		Members:     members,
		Meta: &SCIMMeta{
			ResourceType: "Group",
			Created:      group.CreatedAt,
			LastModified: group.UpdatedAt,
			Location:     baseURL + "/Groups/" + group.ID.String(),
		},
	}
}

// SCIMCreateUserInput is the input shape for creating a user via SCIM
type SCIMCreateUserInput struct {
	Schemas     []string    `json:"schemas"`
	ExternalID  string      `json:"externalId,omitempty"`
	UserName    string      `json:"userName"`
	Name        *SCIMName   `json:"name,omitempty"`
	DisplayName string      `json:"displayName,omitempty"`
	Emails      []SCIMEmail `json:"emails,omitempty"`
	Active      *bool       `json:"active,omitempty"`
	Title       string      `json:"title,omitempty"`
	Department  string      `json:"department,omitempty"`
	Password    string      `json:"password,omitempty"`
}

// GetPrimaryEmail returns the primary email from the SCIM user input
func (s *SCIMCreateUserInput) GetPrimaryEmail() string {
	for _, email := range s.Emails {
		if email.Primary {
			return email.Value
		}
	}
	if len(s.Emails) > 0 {
		return s.Emails[0].Value
	}
	return s.UserName
}

// GetDisplayName returns the display name, falling back to name parts or username
func (s *SCIMCreateUserInput) GetDisplayName() string {
	if s.DisplayName != "" {
		return s.DisplayName
	}
	if s.Name != nil {
		if s.Name.Formatted != "" {
			return s.Name.Formatted
		}
		if s.Name.GivenName != "" || s.Name.FamilyName != "" {
			return s.Name.GivenName + " " + s.Name.FamilyName
		}
	}
	return s.UserName
}

// GetActive returns whether the user should be active (defaults to true)
func (s *SCIMCreateUserInput) GetActive() bool {
	if s.Active != nil {
		return *s.Active
	}
	return true
}

// SCIMCreateGroupInput is the input shape for creating a group via SCIM
type SCIMCreateGroupInput struct {
	Schemas     []string        `json:"schemas"`
	ExternalID  string          `json:"externalId,omitempty"`
	DisplayName string          `json:"displayName"`
	Members     []SCIMMemberRef `json:"members,omitempty"`
}

// Helper to generate a new UUID string
func NewSCIMID() string {
	return uuid.New().String()
}
