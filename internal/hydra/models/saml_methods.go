package hydramodels

import (
	"bytes"
	"compress/flate"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SAML XML structures

type SAMLAuthnRequest struct {
	XMLName                     xml.Name         `xml:"urn:oasis:names:tc:SAML:2.0:protocol AuthnRequest"`
	ID                          string           `xml:"ID,attr"`
	Version                     string           `xml:"Version,attr"`
	IssueInstant                string           `xml:"IssueInstant,attr"`
	Destination                 string           `xml:"Destination,attr"`
	AssertionConsumerServiceURL string           `xml:"AssertionConsumerServiceURL,attr"`
	ProtocolBinding             string           `xml:"ProtocolBinding,attr"`
	Issuer                      SAMLIssuer       `xml:"urn:oasis:names:tc:SAML:2.0:assertion Issuer"`
	NameIDPolicy                SAMLNameIDPolicy `xml:"urn:oasis:names:tc:SAML:2.0:protocol NameIDPolicy"`
}

type SAMLIssuer struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion Issuer"`
	Value   string   `xml:",chardata"`
}

type SAMLNameIDPolicy struct {
	XMLName     xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:protocol NameIDPolicy"`
	Format      string   `xml:"Format,attr"`
	AllowCreate bool     `xml:"AllowCreate,attr"`
}

type SAMLResponseEnvelope struct {
	XMLName      xml.Name              `xml:"urn:oasis:names:tc:SAML:2.0:protocol Response"`
	ID           string                `xml:"ID,attr"`
	InResponseTo string                `xml:"InResponseTo,attr"`
	Version      string                `xml:"Version,attr"`
	IssueInstant string                `xml:"IssueInstant,attr"`
	Destination  string                `xml:"Destination,attr"`
	Status       SAMLStatus            `xml:"urn:oasis:names:tc:SAML:2.0:protocol Status"`
	Assertion    SAMLAssertionEnvelope `xml:"urn:oasis:names:tc:SAML:2.0:assertion Assertion"`
}

type SAMLStatus struct {
	StatusCode SAMLStatusCode `xml:"urn:oasis:names:tc:SAML:2.0:protocol StatusCode"`
}

type SAMLStatusCode struct {
	Value string `xml:"Value,attr"`
}

type SAMLAssertionEnvelope struct {
	XMLName            xml.Name               `xml:"urn:oasis:names:tc:SAML:2.0:assertion Assertion"`
	ID                 string                 `xml:"ID,attr"`
	Version            string                 `xml:"Version,attr"`
	IssueInstant       string                 `xml:"IssueInstant,attr"`
	Issuer             SAMLIssuer             `xml:"urn:oasis:names:tc:SAML:2.0:assertion Issuer"`
	Subject            SAMLSubject            `xml:"urn:oasis:names:tc:SAML:2.0:assertion Subject"`
	Conditions         SAMLConditions         `xml:"urn:oasis:names:tc:SAML:2.0:assertion Conditions"`
	AttributeStatement SAMLAttributeStatement `xml:"urn:oasis:names:tc:SAML:2.0:assertion AttributeStatement"`
}

type SAMLSubject struct {
	NameID              SAMLNameID              `xml:"urn:oasis:names:tc:SAML:2.0:assertion NameID"`
	SubjectConfirmation SAMLSubjectConfirmation `xml:"urn:oasis:names:tc:SAML:2.0:assertion SubjectConfirmation"`
}

type SAMLNameID struct {
	Format string `xml:"Format,attr"`
	Value  string `xml:",chardata"`
}

type SAMLSubjectConfirmation struct {
	Method                  string                      `xml:"Method,attr"`
	SubjectConfirmationData SAMLSubjectConfirmationData `xml:"urn:oasis:names:tc:SAML:2.0:assertion SubjectConfirmationData"`
}

type SAMLSubjectConfirmationData struct {
	NotOnOrAfter string `xml:"NotOnOrAfter,attr"`
	Recipient    string `xml:"Recipient,attr"`
	InResponseTo string `xml:"InResponseTo,attr"`
}

type SAMLConditions struct {
	NotBefore    string `xml:"NotBefore,attr"`
	NotOnOrAfter string `xml:"NotOnOrAfter,attr"`
}

type SAMLAttributeStatement struct {
	Attributes []SAMLAttribute `xml:"urn:oasis:names:tc:SAML:2.0:assertion Attribute"`
}

type SAMLAttribute struct {
	Name       string               `xml:"Name,attr"`
	NameFormat string               `xml:"NameFormat,attr"`
	Values     []SAMLAttributeValue `xml:"urn:oasis:names:tc:SAML:2.0:assertion AttributeValue"`
}

type SAMLAttributeValue struct {
	Type  string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
	Value string `xml:",chardata"`
}

// GetSAMLProvidersForTenant retrieves SAML providers for a tenant from tenant database
func (s *OAuthLoginService) GetSAMLProvidersForTenant(tenantID string, clientID ...string) ([]Provider, error) {
	db, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant database: %w", err)
	}

	query := db.Where("is_active = ?", true)

	if len(clientID) > 0 && clientID[0] != "" {
		trimmedClientID := strings.TrimSuffix(clientID[0], "-main-client")
		query = query.Where("client_id = ?", trimmedClientID)
	}

	var samlProviders []SAMLProvider
	if err := query.Order("sort_order ASC").Find(&samlProviders).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch SAML providers: %w", err)
	}

	providers := make([]Provider, 0, len(samlProviders))
	for _, sp := range samlProviders {
		providers = append(providers, Provider{
			ProviderName: sp.ProviderName,
			DisplayName:  sp.DisplayName,
			Type:         "saml",
			IsActive:     sp.IsActive,
			SortOrder:    sp.SortOrder,
			Config: map[string]interface{}{
				"entity_id":      sp.EntityID,
				"sso_url":        sp.SSOURL,
				"slo_url":        sp.SLOURL,
				"name_id_format": sp.NameIDFormat,
				"client_id":      sp.ClientID.String(),
			},
		})
	}
	return providers, nil
}

// GetAllProvidersForTenant returns both OIDC and SAML providers
func (s *OAuthLoginService) GetAllProvidersForTenant(tenantIDForOIDC string, realTenantID string, clientID ...string) ([]Provider, error) {
	var allProviders []Provider

	oidcProviders, err := s.GetOIDCProvidersForTenant(tenantIDForOIDC)
	if err != nil {
		log.Printf("Warning: Failed to get OIDC providers: %v", err)
	} else {
		for _, op := range oidcProviders {
			allProviders = append(allProviders, Provider{
				ProviderName: op.ProviderName,
				DisplayName:  op.DisplayName,
				Type:         "oidc",
				IsActive:     op.IsActive,
				SortOrder:    op.SortOrder,
				Config:       op.Config,
			})
		}
	}

	samlProviders, err := s.GetSAMLProvidersForTenant(realTenantID, clientID...)
	if err != nil {
		log.Printf("Warning: Failed to get SAML providers: %v", err)
	} else {
		allProviders = append(allProviders, samlProviders...)
	}

	for i := 0; i < len(allProviders)-1; i++ {
		for j := i + 1; j < len(allProviders); j++ {
			if allProviders[i].SortOrder > allProviders[j].SortOrder {
				allProviders[i], allProviders[j] = allProviders[j], allProviders[i]
			}
		}
	}
	return allProviders, nil
}

// GetSAMLProvider retrieves a specific SAML provider by name and optionally client_id
func (s *OAuthLoginService) GetSAMLProvider(tenantID, providerName string, clientID ...string) (*SAMLProvider, error) {
	db, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant database: %w", err)
	}

	providerName = strings.ToLower(strings.TrimSpace(providerName))
	query := db.Where("tenant_id = ? AND provider_name = ?", tenantID, providerName)

	if len(clientID) > 0 && clientID[0] != "" {
		cid := strings.TrimSuffix(clientID[0], "-main-client")
		query = query.Where("client_id = ?", cid)
	}

	var provider SAMLProvider
	if err := query.First(&provider).Error; err != nil {
		return nil, fmt.Errorf("SAML provider not found: %w", err)
	}
	return &provider, nil
}

// CreateSAMLRequest creates a SAML authentication request
func (s *OAuthLoginService) CreateSAMLRequest(provider *SAMLProvider, loginChallenge string) (string, string, error) {
	requestID := fmt.Sprintf("_%s", uuid.New().String())
	issueInstant := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	spEntityID := fmt.Sprintf("%s/saml/metadata/%s/%s", s.cfg.BaseURL, provider.TenantID.String(), provider.ClientID.String())
	acsURL := fmt.Sprintf("%s/saml/acs/%s/%s", s.cfg.BaseURL, provider.TenantID.String(), provider.ClientID.String())

	authnRequest := SAMLAuthnRequest{
		ID:                          requestID,
		Version:                     "2.0",
		IssueInstant:                issueInstant,
		Destination:                 provider.SSOURL,
		AssertionConsumerServiceURL: acsURL,
		ProtocolBinding:             "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
		Issuer:                      SAMLIssuer{Value: spEntityID},
		NameIDPolicy: SAMLNameIDPolicy{
			Format:      provider.NameIDFormat,
			AllowCreate: true,
		},
	}

	xmlBytes, err := xml.MarshalIndent(authnRequest, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal SAML request: %w", err)
	}

	xmlString := xml.Header + string(xmlBytes)

	var deflatedBuf bytes.Buffer
	deflater, err := flate.NewWriter(&deflatedBuf, flate.DefaultCompression)
	if err != nil {
		return "", "", fmt.Errorf("failed to create deflater: %w", err)
	}
	if _, err := deflater.Write([]byte(xmlString)); err != nil {
		return "", "", fmt.Errorf("failed to deflate SAML request: %w", err)
	}
	if err := deflater.Close(); err != nil {
		return "", "", fmt.Errorf("failed to close deflater: %w", err)
	}

	samlRequest := base64.StdEncoding.EncodeToString(deflatedBuf.Bytes())
	relayState := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s:%s:%s",
		loginChallenge, provider.ProviderName, provider.TenantID.String(), provider.ClientID.String())))

	db := config.DB
	samlReq := SAMLRequest{
		ID:             requestID,
		LoginChallenge: loginChallenge,
		TenantID:       provider.TenantID,
		ClientID:       provider.ClientID,
		ProviderName:   provider.ProviderName,
		RelayState:     relayState,
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(10 * time.Minute),
	}

	if err := db.Create(&samlReq).Error; err != nil {
		return "", "", fmt.Errorf("failed to store SAML request: %w", err)
	}
	return samlRequest, relayState, nil
}

// ValidateSAMLResponse validates and parses a SAML response
func (s *OAuthLoginService) ValidateSAMLResponse(samlResponse string, relayState string) (*SAMLAssertion, string, string, string, string, error) {
	relayBytes, err := base64.StdEncoding.DecodeString(relayState)
	if err != nil {
		return nil, "", "", "", "", fmt.Errorf("invalid relay state: %w", err)
	}

	parts := []byte(relayBytes)
	relayParts := make([]string, 0)
	current := ""
	for _, b := range parts {
		if b == ':' {
			relayParts = append(relayParts, current)
			current = ""
		} else {
			current += string(b)
		}
	}
	relayParts = append(relayParts, current)

	if len(relayParts) < 4 {
		return nil, "", "", "", "", fmt.Errorf("invalid relay state format, expected 4 parts, got %d", len(relayParts))
	}

	loginChallenge := relayParts[0]
	providerName := relayParts[1]
	tenantID := relayParts[2]
	clientID := relayParts[3]

	responseBytes, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		return nil, "", "", "", "", fmt.Errorf("failed to decode SAML response: %w", err)
	}

	var samlResp SAMLResponseEnvelope
	if err := xml.Unmarshal(responseBytes, &samlResp); err != nil {
		return nil, "", "", "", "", fmt.Errorf("failed to unmarshal SAML response: %w", err)
	}

	if samlResp.Status.StatusCode.Value != "urn:oasis:names:tc:SAML:2.0:status:Success" {
		return nil, "", "", "", "", fmt.Errorf("SAML authentication failed: %s", samlResp.Status.StatusCode.Value)
	}

	provider, err := s.GetSAMLProvider(tenantID, providerName, clientID)
	if err != nil {
		log.Printf("Failed to get SAML provider for entity ID validation: %v", err)
	} else {
		responseEntityID := samlResp.Assertion.Issuer.Value
		if responseEntityID != provider.EntityID {
			return nil, "", "", "", "", fmt.Errorf("SAML entity ID validation failed: response from unexpected identity provider")
		}
	}

	nameID := trimSpace(samlResp.Assertion.Subject.NameID.Value)
	attributes := make(map[string]interface{})
	email, firstName, lastName := "", "", ""

	for _, attr := range samlResp.Assertion.AttributeStatement.Attributes {
		attrName := attr.Name
		var attrValue string
		if len(attr.Values) > 0 {
			attrValue = trimSpace(attr.Values[0].Value)
		}
		attributes[attrName] = attrValue

		switch attrName {
		case "email", "emailAddress", "mail", "urn:oid:0.9.2342.19200300.100.1.3":
			email = attrValue
		case "givenName", "firstName", "urn:oid:2.5.4.42":
			firstName = attrValue
		case "surname", "lastName", "sn", "urn:oid:2.5.4.4":
			lastName = attrValue
		}
	}

	if email == "" {
		email = nameID
	}

	return &SAMLAssertion{
		NameID:     nameID,
		Email:      email,
		FirstName:  firstName,
		LastName:   lastName,
		Attributes: attributes,
	}, loginChallenge, providerName, tenantID, clientID, nil
}

// GetOrCreateSPCertificate gets or creates SP certificate for tenant
func (s *OAuthLoginService) GetOrCreateSPCertificate(tenantID uuid.UUID) (*SAMLSPCertificate, error) {
	db := config.DB

	var cert SAMLSPCertificate
	err := db.Where("tenant_id = ?", tenantID).First(&cert).Error
	if err == nil {
		return &cert, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to query certificate: %w", err)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	encryptedPrivateKey := string(privateKeyPEM)
	if config.VaultClient != nil {
		encrypted, err := encryptPrivateKeyWithVault(tenantID.String(), string(privateKeyPEM))
		if err != nil {
			log.Printf("WARNING: Failed to encrypt private key with Vault: %v. Storing plaintext.", err)
		} else {
			encryptedPrivateKey = encrypted
		}
	} else {
		log.Printf("WARNING: Vault not available, storing SAML private key in plaintext")
	}

	newCert := SAMLSPCertificate{
		TenantID:    tenantID,
		Certificate: string(certPEM),
		PrivateKey:  encryptedPrivateKey,
		ExpiresAt:   time.Now().AddDate(1, 0, 0),
	}

	if err := db.Create(&newCert).Error; err != nil {
		return nil, fmt.Errorf("failed to store certificate: %w", err)
	}
	return &newCert, nil
}

// GenerateSAMLMetadata generates SP metadata XML for a tenant and client
func (s *OAuthLoginService) GenerateSAMLMetadata(tenantID, clientID uuid.UUID) (string, error) {
	cert, err := s.GetOrCreateSPCertificate(tenantID)
	if err != nil {
		return "", fmt.Errorf("failed to get SP certificate: %w", err)
	}

	entityID := fmt.Sprintf("%s/saml/metadata/%s/%s", s.cfg.BaseURL, tenantID.String(), clientID.String())
	acsURLShared := fmt.Sprintf("%s/saml/acs", s.cfg.BaseURL)
	acsURLTenantClient := fmt.Sprintf("%s/saml/acs/%s/%s", s.cfg.BaseURL, tenantID.String(), clientID.String())
	certData := extractCertificateData(cert.Certificate)

	metadata := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata"
                     entityID="%s">
  <md:SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:KeyDescriptor use="signing">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>%s</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </md:KeyDescriptor>
    <md:KeyDescriptor use="encryption">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>%s</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </md:KeyDescriptor>
    <md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</md:NameIDFormat>
    <md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
                                 Location="%s"
                                 index="1"
                                 isDefault="true" />
    <md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
                                 Location="%s"
                                 index="2" />
  </md:SPSSODescriptor>
</md:EntityDescriptor>`, entityID, certData, certData, acsURLTenantClient, acsURLShared)

	return metadata, nil
}

// CreateSAMLProvider creates a new SAML provider for a tenant and client
func (s *OAuthLoginService) CreateSAMLProvider(tenantID string, provider *SAMLProvider) (*SAMLProvider, error) {
	db, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant database: %w", err)
	}

	provider.ProviderName = strings.ToLower(strings.TrimSpace(provider.ProviderName))
	provider.TenantID, _ = uuid.Parse(tenantID)

	if err := db.Create(provider).Error; err != nil {
		return nil, fmt.Errorf("failed to create SAML provider: %w", err)
	}
	return provider, nil
}

// UpdateSAMLProvider updates an existing SAML provider
func (s *OAuthLoginService) UpdateSAMLProvider(tenantID string, providerID uuid.UUID, clientID string, updates *SAMLProvider) (*SAMLProvider, error) {
	db, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant database: %w", err)
	}

	query := db.Where("id = ?", providerID)
	if clientID != "" {
		clientID = strings.TrimSuffix(clientID, "-main-client")
		query = query.Where("client_id = ?", clientID)
	}

	var provider SAMLProvider
	if err := query.First(&provider).Error; err != nil {
		return nil, fmt.Errorf("SAML provider not found: %w", err)
	}

	if updates.ProviderName != "" {
		provider.ProviderName = strings.ToLower(strings.TrimSpace(updates.ProviderName))
	}
	if updates.DisplayName != "" {
		provider.DisplayName = updates.DisplayName
	}
	if updates.EntityID != "" {
		provider.EntityID = updates.EntityID
	}
	if updates.SSOURL != "" {
		provider.SSOURL = updates.SSOURL
	}
	if updates.SLOURL != "" {
		provider.SLOURL = updates.SLOURL
	}
	if updates.Certificate != "" {
		provider.Certificate = updates.Certificate
	}
	if updates.MetadataURL != "" {
		provider.MetadataURL = updates.MetadataURL
	}
	if updates.NameIDFormat != "" {
		provider.NameIDFormat = updates.NameIDFormat
	}
	if updates.AttributeMapping != nil {
		provider.AttributeMapping = updates.AttributeMapping
	}
	provider.IsActive = updates.IsActive
	provider.SortOrder = updates.SortOrder

	if err := db.Save(&provider).Error; err != nil {
		return nil, fmt.Errorf("failed to update SAML provider: %w", err)
	}
	return &provider, nil
}

// DeleteSAMLProvider deletes a SAML provider
func (s *OAuthLoginService) DeleteSAMLProvider(tenantID string, providerID uuid.UUID, clientID string) error {
	db, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant database: %w", err)
	}

	query := db.Where("id = ?", providerID)
	if clientID != "" {
		clientID = strings.TrimSuffix(clientID, "-main-client")
		query = query.Where("client_id = ?", clientID)
	}

	var provider SAMLProvider
	if err := query.First(&provider).Error; err != nil {
		return fmt.Errorf("SAML provider not found: %w", err)
	}

	if err := db.Delete(&provider).Error; err != nil {
		return fmt.Errorf("failed to delete SAML provider: %w", err)
	}
	return nil
}

// XML helper functions

func extractCertificateData(pemCert string) string {
	lines := []string{}
	for _, line := range splitLines(pemCert) {
		trimmed := trimSpace(line)
		if trimmed != "" && !hasPrefix(trimmed, "-----") {
			lines = append(lines, trimmed)
		}
	}
	result := ""
	for _, line := range lines {
		result += line
	}
	return result
}

func splitLines(s string) []string {
	var lines []string
	current := ""
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, current)
			current = ""
		} else if s[i] != '\r' {
			current += string(s[i])
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
