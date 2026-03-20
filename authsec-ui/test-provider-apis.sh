#!/bin/bash

# Test script for update-provider and delete-provider APIs
# Make sure to replace the placeholder values with actual data

API_BASE="https://stage.api.authsec.dev"

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}================================================${NC}"
echo -e "${YELLOW}Testing Authentication Provider APIs${NC}"
echo -e "${YELLOW}================================================${NC}"
echo ""

# Replace these with actual values from your environment
TENANT_ID="your-tenant-id-here"
ORG_ID="your-org-id-here"
CLIENT_ID="your-client-id-here"
PROVIDER_NAME="google"  # or github, microsoft, etc.
DISPLAY_NAME="Google OAuth"
CLIENT_SECRET="your-client-secret-here"
AUTH_URL="https://accounts.google.com/o/oauth2/v2/auth"
TOKEN_URL="https://oauth2.googleapis.com/token"
USER_INFO_URL="https://www.googleapis.com/oauth2/v3/userinfo"

echo -e "${YELLOW}1. Testing show-auth-providers (to get current state)${NC}"
echo "GET current providers..."
curl -X POST "${API_BASE}/oocmgr/oidc/show-auth-providers" \
  -H "Content-Type: application/json" \
  -H "Client-Id: ${CLIENT_ID}" \
  -d "{
    \"tenant_id\": \"${TENANT_ID}\",
    \"client_id\": \"${CLIENT_ID}\"
  }" | jq '.'

echo ""
echo -e "${YELLOW}2. Testing update-provider (DEACTIVATE - set is_active to false)${NC}"
echo "Deactivating provider..."
UPDATE_RESPONSE=$(curl -s -X POST "${API_BASE}/oocmgr/oidc/update-provider" \
  -H "Content-Type: application/json" \
  -d "{
    \"tenant_id\": \"${TENANT_ID}\",
    \"org_id\": \"${ORG_ID}\",
    \"provider_name\": \"${PROVIDER_NAME}\",
    \"display_name\": \"${DISPLAY_NAME}\",
    \"client_id\": \"${CLIENT_ID}\",
    \"client_secret\": \"${CLIENT_SECRET}\",
    \"auth_url\": \"${AUTH_URL}\",
    \"token_url\": \"${TOKEN_URL}\",
    \"user_info_url\": \"${USER_INFO_URL}\",
    \"scopes\": [\"openid\", \"profile\", \"email\"],
    \"is_active\": false,
    \"updated_by\": \"test-script\"
  }")

echo "$UPDATE_RESPONSE" | jq '.'

if echo "$UPDATE_RESPONSE" | jq -e '.success == true' > /dev/null 2>&1; then
  echo -e "${GREEN}✓ Provider deactivated successfully${NC}"
else
  echo -e "${RED}✗ Failed to deactivate provider${NC}"
fi

echo ""
echo -e "${YELLOW}3. Testing update-provider (ACTIVATE - set is_active to true)${NC}"
echo "Activating provider..."
ACTIVATE_RESPONSE=$(curl -s -X POST "${API_BASE}/oocmgr/oidc/update-provider" \
  -H "Content-Type: application/json" \
  -d "{
    \"tenant_id\": \"${TENANT_ID}\",
    \"org_id\": \"${ORG_ID}\",
    \"provider_name\": \"${PROVIDER_NAME}\",
    \"display_name\": \"${DISPLAY_NAME}\",
    \"client_id\": \"${CLIENT_ID}\",
    \"client_secret\": \"${CLIENT_SECRET}\",
    \"auth_url\": \"${AUTH_URL}\",
    \"token_url\": \"${TOKEN_URL}\",
    \"user_info_url\": \"${USER_INFO_URL}\",
    \"scopes\": [\"openid\", \"profile\", \"email\"],
    \"is_active\": true,
    \"updated_by\": \"test-script\"
  }")

echo "$ACTIVATE_RESPONSE" | jq '.'

if echo "$ACTIVATE_RESPONSE" | jq -e '.success == true' > /dev/null 2>&1; then
  echo -e "${GREEN}✓ Provider activated successfully${NC}"
else
  echo -e "${RED}✗ Failed to activate provider${NC}"
fi

echo ""
echo -e "${YELLOW}4. Verifying provider status after toggle${NC}"
echo "GET providers to verify status..."
curl -X POST "${API_BASE}/oocmgr/oidc/show-auth-providers" \
  -H "Content-Type: application/json" \
  -H "Client-Id: ${CLIENT_ID}" \
  -d "{
    \"tenant_id\": \"${TENANT_ID}\",
    \"client_id\": \"${CLIENT_ID}\"
  }" | jq '.data.providers[] | select(.provider_name == "'${PROVIDER_NAME}'") | {provider_name, display_name, is_active}'

echo ""
echo -e "${RED}5. Testing delete-provider (CAUTION: This will delete the provider!)${NC}"
read -p "Do you want to test provider deletion? (yes/no): " CONFIRM

if [ "$CONFIRM" = "yes" ]; then
  echo "Deleting provider..."
  DELETE_RESPONSE=$(curl -s -X POST "${API_BASE}/oocmgr/oidc/delete-provider" \
    -H "Content-Type: application/json" \
    -d "{
      \"tenant_id\": \"${TENANT_ID}\",
      \"client_id\": \"${CLIENT_ID}\",
      \"provider_name\": \"${PROVIDER_NAME}\"
    }")
  
  echo "$DELETE_RESPONSE" | jq '.'
  
  if echo "$DELETE_RESPONSE" | jq -e '.success == true' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Provider deleted successfully${NC}"
  else
    echo -e "${RED}✗ Failed to delete provider${NC}"
  fi

  echo ""
  echo -e "${YELLOW}6. Verifying provider is deleted${NC}"
  echo "GET providers to verify deletion..."
  curl -X POST "${API_BASE}/oocmgr/oidc/show-auth-providers" \
    -H "Content-Type: application/json" \
    -H "Client-Id: ${CLIENT_ID}" \
    -d "{
      \"tenant_id\": \"${TENANT_ID}\",
      \"client_id\": \"${CLIENT_ID}\"
    }" | jq '.data.providers[] | select(.provider_name == "'${PROVIDER_NAME}'")'
  
  echo -e "${YELLOW}If no output above, provider was successfully deleted${NC}"
else
  echo -e "${YELLOW}Skipping delete test${NC}"
fi

echo ""
echo -e "${YELLOW}================================================${NC}"
echo -e "${YELLOW}Test completed${NC}"
echo -e "${YELLOW}================================================${NC}"
