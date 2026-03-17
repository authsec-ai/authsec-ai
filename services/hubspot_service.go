package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const hubSpotBaseURL = "https://api.hubapi.com"

// HubSpotService handles communication with the HubSpot CRM API
type HubSpotService struct {
	accessToken string
	httpClient  *http.Client
}

// NewHubSpotService creates a new HubSpot service instance
func NewHubSpotService(accessToken string) *HubSpotService {
	return &HubSpotService{
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// hubSpotContactProperties represents the properties sent to HubSpot
type hubSpotContactProperties struct {
	Email                string `json:"email,omitempty"`
	TenantDomain         string `json:"tenant_domain,omitempty"`
	TenantID             string `json:"tenant_id,omitempty"`
	RegistrationDate string `json:"registration_date,omitempty"`
	LifecycleStage   string `json:"lifecyclestage,omitempty"`
}

type hubSpotCreateRequest struct {
	Properties hubSpotContactProperties `json:"properties"`
}

type hubSpotCreateResponse struct {
	ID         string                   `json:"id"`
	Properties hubSpotContactProperties `json:"properties"`
}

type hubSpotSearchRequest struct {
	FilterGroups []hubSpotFilterGroup `json:"filterGroups"`
}

type hubSpotFilterGroup struct {
	Filters []hubSpotFilter `json:"filters"`
}

type hubSpotFilter struct {
	PropertyName string `json:"propertyName"`
	Operator     string `json:"operator"`
	Value        string `json:"value"`
}

type hubSpotSearchResponse struct {
	Total   int `json:"total"`
	Results []struct {
		ID string `json:"id"`
	} `json:"results"`
}

// SyncContact creates or updates a contact in HubSpot.
// Returns the HubSpot contact ID on success.
func (s *HubSpotService) SyncContact(email, tenantDomain, tenantID string) (string, error) {
	today := time.Now().Format("2006-01-02")

	// Step 1: Try to create the contact
	createReq := hubSpotCreateRequest{
		Properties: hubSpotContactProperties{
			Email:            email,
			TenantDomain:     tenantDomain,
			TenantID:         tenantID,
			RegistrationDate: today,
			LifecycleStage:   "lead",
		},
	}

	contactID, err := s.createContact(createReq)
	if err == nil {
		log.Printf("[HubSpot] Created new contact %s for email %s", contactID, email)
		return contactID, nil
	}

	// Step 2: If conflict (contact exists), search and update
	if err.Error() == "conflict" {
		log.Printf("[HubSpot] Contact already exists for %s, searching to update", email)

		contactID, err = s.searchContactByEmail(email)
		if err != nil {
			return "", fmt.Errorf("failed to search existing contact: %w", err)
		}

		err = s.updateContact(contactID, hubSpotContactProperties{
			TenantDomain:     tenantDomain,
			TenantID:         tenantID,
			RegistrationDate: today,
		})
		if err != nil {
			return "", fmt.Errorf("failed to update existing contact: %w", err)
		}

		log.Printf("[HubSpot] Updated existing contact %s for email %s", contactID, email)
		return contactID, nil
	}

	return "", fmt.Errorf("failed to create contact: %w", err)
}

func (s *HubSpotService) createContact(reqBody hubSpotCreateRequest) (string, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", hubSpotBaseURL+"/crm/v3/objects/contacts", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	s.setHeaders(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call HubSpot API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return "", fmt.Errorf("conflict")
	}

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HubSpot create returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result hubSpotCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.ID, nil
}

func (s *HubSpotService) searchContactByEmail(email string) (string, error) {
	searchReq := hubSpotSearchRequest{
		FilterGroups: []hubSpotFilterGroup{
			{
				Filters: []hubSpotFilter{
					{
						PropertyName: "email",
						Operator:     "EQ",
						Value:        email,
					},
				},
			},
		},
	}

	body, err := json.Marshal(searchReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal search request: %w", err)
	}

	req, err := http.NewRequest("POST", hubSpotBaseURL+"/crm/v3/objects/contacts/search", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create search request: %w", err)
	}
	s.setHeaders(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call HubSpot search API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HubSpot search returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result hubSpotSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode search response: %w", err)
	}

	if result.Total == 0 || len(result.Results) == 0 {
		return "", fmt.Errorf("no contact found for email %s", email)
	}

	return result.Results[0].ID, nil
}

func (s *HubSpotService) updateContact(contactID string, props hubSpotContactProperties) error {
	updateReq := struct {
		Properties hubSpotContactProperties `json:"properties"`
	}{
		Properties: props,
	}

	body, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("failed to marshal update request: %w", err)
	}

	req, err := http.NewRequest("PATCH", hubSpotBaseURL+"/crm/v3/objects/contacts/"+contactID, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create update request: %w", err)
	}
	s.setHeaders(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call HubSpot update API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HubSpot update returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (s *HubSpotService) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.accessToken)
}
