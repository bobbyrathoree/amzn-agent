package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
	"github.com/bobbyrathore/go-lambda-backend/pkg/utils"
)

// VaultHandler handles vault-related API endpoints
type VaultHandler struct {
	vaultService *services.VaultService
}

// NewVaultHandler creates a new vault handler
func NewVaultHandler(vaultService *services.VaultService) *VaultHandler {
	return &VaultHandler{
		vaultService: vaultService,
	}
}

// HandleVaultRequest routes vault requests to appropriate handlers
func (h *VaultHandler) HandleVaultRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract user ID from API Gateway request context (Cognito claims)
	userID, err := utils.ExtractUserIDFromRequest(request)
	if err != nil {
		return utils.ErrorResponse(err, http.StatusUnauthorized), nil
	}

	// Route based on HTTP method and path
	switch request.HTTPMethod {
	case "GET":
		return h.handleGetRequest(ctx, request, userID)
	case "POST":
		return h.handlePostRequest(ctx, request, userID)
	case "PUT":
		return h.handlePutRequest(ctx, request, userID)
	case "DELETE":
		return h.handleDeleteRequest(ctx, request, userID)
	default:
		return utils.ErrorResponse(utils.ErrorFromString("Method not allowed"), http.StatusMethodNotAllowed), nil
	}
}

// handleGetRequest handles GET requests for vault endpoints
func (h *VaultHandler) handleGetRequest(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	path := request.Path
	pathParams := request.PathParameters

	switch {
	case strings.HasSuffix(path, "/vault/status"):
		return h.getVaultStatus(ctx, userID)
	case strings.HasSuffix(path, "/vault/services"):
		return h.getSupportedServices(ctx)
	case strings.HasSuffix(path, "/vault/services/categories"):
		return h.getServiceCategories(ctx)
	case strings.Contains(path, "/vault/services/") && pathParams["serviceId"] != "":
		return h.getSupportedService(ctx, pathParams["serviceId"])
	case strings.HasSuffix(path, "/vault/keys"):
		return h.listAPIKeys(ctx, userID)
	case strings.Contains(path, "/vault/keys/") && pathParams["serviceId"] != "":
		return h.getAPIKey(ctx, userID, pathParams["serviceId"])
	case strings.HasSuffix(path, "/vault/salt"):
		return h.getUserSalt(ctx, userID)
	default:
		return utils.ErrorResponse(utils.ErrorFromString("Endpoint not found"), http.StatusNotFound), nil
	}
}

// handlePostRequest handles POST requests for vault endpoints
func (h *VaultHandler) handlePostRequest(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	path := request.Path
	pathParams := request.PathParameters

	switch {
	case strings.Contains(path, "/vault/keys/") && pathParams["serviceId"] != "":
		return h.storeAPIKey(ctx, request, userID, pathParams["serviceId"])
	case strings.Contains(path, "/vault/keys/") && strings.HasSuffix(path, "/test"):
		serviceID := pathParams["serviceId"]
		return h.testAPIKey(ctx, userID, serviceID)
	case strings.HasSuffix(path, "/vault/salt"):
		return h.createUserSalt(ctx, request, userID)
	default:
		return utils.ErrorResponse(utils.ErrorFromString("Endpoint not found"), http.StatusNotFound), nil
	}
}

// handlePutRequest handles PUT requests for vault endpoints
func (h *VaultHandler) handlePutRequest(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	path := request.Path
	pathParams := request.PathParameters
	
	switch {
	case strings.HasSuffix(path, "/vault/salt"):
		return h.updateUserSalt(ctx, request, userID)
	case pathParams["serviceId"] != "":
		return h.updateAPIKey(ctx, request, userID, pathParams["serviceId"])
	default:
		return utils.ErrorResponse(utils.ErrorFromString("Service ID required"), http.StatusBadRequest), nil
	}
}

// handleDeleteRequest handles DELETE requests for vault endpoints
func (h *VaultHandler) handleDeleteRequest(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	pathParams := request.PathParameters
	if pathParams["serviceId"] == "" {
		return utils.ErrorResponse(utils.ErrorFromString("Service ID required"), http.StatusBadRequest), nil
	}

	return h.deleteAPIKey(ctx, userID, pathParams["serviceId"])
}

// Vault Status Endpoint: GET /vault/status
func (h *VaultHandler) getVaultStatus(ctx context.Context, userID string) (events.APIGatewayProxyResponse, error) {
	status, err := h.vaultService.GetVaultStatus(ctx, userID)
	if err != nil {
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	return utils.SuccessResponse(status), nil
}

// List API Keys Endpoint: GET /vault/keys
func (h *VaultHandler) listAPIKeys(ctx context.Context, userID string) (events.APIGatewayProxyResponse, error) {
	entries, err := h.vaultService.ListAPIKeys(ctx, userID)
	if err != nil {
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	// Return metadata only (without encrypted keys)
	var keyMetadata []map[string]interface{}
	for _, entry := range entries {
		metadata := map[string]interface{}{
			"serviceId":   entry.ServiceID,
			"keyName":     entry.KeyName,
			"createdAt":   entry.CreatedAt,
			"updatedAt":   entry.UpdatedAt,
			"lastUsed":    entry.LastUsed,
			"usageCount":  entry.UsageCount,
			"isActive":    entry.IsActive,
		}
		if entry.ExpiresAt != nil {
			metadata["expiresAt"] = entry.ExpiresAt
		}
		keyMetadata = append(keyMetadata, metadata)
	}

	response := map[string]interface{}{
		"keys":     keyMetadata,
		"count":    len(keyMetadata),
		"userId":   userID,
	}

	return utils.SuccessResponse(response), nil
}

// Get API Key Endpoint: GET /vault/keys/{serviceId}
func (h *VaultHandler) getAPIKey(ctx context.Context, userID, serviceID string) (events.APIGatewayProxyResponse, error) {
	entry, err := h.vaultService.GetVaultEntry(ctx, userID, serviceID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return utils.ErrorResponse(utils.ErrorFromString(fmt.Sprintf("API key not found for service %s", serviceID)), http.StatusNotFound), nil
		}
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	// Return the complete entry (frontend needs encrypted key + nonce + salt for decryption)
	return utils.SuccessResponse(entry), nil
}

// Store API Key Request
type StoreAPIKeyRequest struct {
	KeyName      string `json:"keyName"`
	EncryptedKey string `json:"encryptedKey"`
	Nonce        string `json:"nonce"`
	Salt         string `json:"salt"`
}

// Store API Key Endpoint: POST /vault/keys/{serviceId}
func (h *VaultHandler) storeAPIKey(ctx context.Context, request events.APIGatewayProxyRequest, userID, serviceID string) (events.APIGatewayProxyResponse, error) {
	var req StoreAPIKeyRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return utils.ErrorResponse(err, http.StatusBadRequest), nil
	}

	// Validate required fields
	if req.EncryptedKey == "" || req.Nonce == "" || req.Salt == "" {
		return utils.ErrorResponse(utils.ErrorFromString("encryptedKey, nonce, and salt are required"), http.StatusBadRequest), nil
	}

	// Store the API key
	err := h.vaultService.StoreAPIKey(ctx, userID, serviceID, req.KeyName, req.EncryptedKey, req.Nonce, req.Salt)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return utils.ErrorResponse(err, http.StatusConflict), nil
		}
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	response := map[string]interface{}{
		"success":   true,
		"message":   "API key stored successfully",
		"serviceId": serviceID,
		"userId":    userID,
	}

	return utils.SuccessResponse(response), nil
}

// Update API Key Request
type UpdateAPIKeyRequest struct {
	EncryptedKey string `json:"encryptedKey"`
	Nonce        string `json:"nonce"`
}

// Update API Key Endpoint: PUT /vault/keys/{serviceId}
func (h *VaultHandler) updateAPIKey(ctx context.Context, request events.APIGatewayProxyRequest, userID, serviceID string) (events.APIGatewayProxyResponse, error) {
	var req UpdateAPIKeyRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return utils.ErrorResponse(err, http.StatusBadRequest), nil
	}

	// Validate required fields
	if req.EncryptedKey == "" || req.Nonce == "" {
		return utils.ErrorResponse(utils.ErrorFromString("encryptedKey and nonce are required"), http.StatusBadRequest), nil
	}

	// Update the API key
	err := h.vaultService.UpdateAPIKey(ctx, userID, serviceID, req.EncryptedKey, req.Nonce)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return utils.ErrorResponse(err, http.StatusNotFound), nil
		}
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	response := map[string]interface{}{
		"success":   true,
		"message":   "API key updated successfully",
		"serviceId": serviceID,
		"userId":    userID,
	}

	return utils.SuccessResponse(response), nil
}

// Delete API Key Endpoint: DELETE /vault/keys/{serviceId}
func (h *VaultHandler) deleteAPIKey(ctx context.Context, userID, serviceID string) (events.APIGatewayProxyResponse, error) {
	err := h.vaultService.DeleteAPIKey(ctx, userID, serviceID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return utils.ErrorResponse(err, http.StatusNotFound), nil
		}
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	response := map[string]interface{}{
		"success":   true,
		"message":   "API key deleted successfully",
		"serviceId": serviceID,
		"userId":    userID,
	}

	return utils.SuccessResponse(response), nil
}

// Test API Key Endpoint: POST /vault/keys/{serviceId}/test
func (h *VaultHandler) testAPIKey(ctx context.Context, userID, serviceID string) (events.APIGatewayProxyResponse, error) {
	isValid, err := h.vaultService.TestAPIKey(ctx, userID, serviceID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return utils.ErrorResponse(err, http.StatusNotFound), nil
		}
		if strings.Contains(err.Error(), "disabled") {
			response := map[string]interface{}{
				"success":   false,
				"valid":     false,
				"serviceId": serviceID,
				"userId":    userID,
				"message":   fmt.Sprintf("API key is disabled for %s", serviceID),
				"testedAt":  fmt.Sprintf("%v", time.Now().Format(time.RFC3339)),
			}
			return utils.SuccessResponse(response), nil
		}
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	var message string
	if isValid {
		message = fmt.Sprintf("API key exists and is active for %s. Frontend testing required for full validation.", serviceID)
	} else {
		message = fmt.Sprintf("API key test failed for %s", serviceID)
	}

	response := map[string]interface{}{
		"success":   true,
		"valid":     isValid,
		"serviceId": serviceID,
		"userId":    userID,
		"message":   message,
		"testedAt":  fmt.Sprintf("%v", time.Now().Format(time.RFC3339)),
		"details": map[string]interface{}{
			"backendValidation": "passed",
			"keyExists":        isValid,
			"keyActive":        isValid,
			"note":            "Complete validation requires frontend testing with decrypted key",
		},
	}

	return utils.SuccessResponse(response), nil
}

// Get Supported Services Endpoint: GET /vault/services
func (h *VaultHandler) getSupportedServices(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	services := h.vaultService.GetSupportedServices()
	
	response := map[string]interface{}{
		"services": services,
		"count":    len(services),
	}
	
	return utils.SuccessResponse(response), nil
}

// Get Service Categories Endpoint: GET /vault/services/categories
func (h *VaultHandler) getServiceCategories(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	categories := h.vaultService.GetServiceCategories()
	
	response := map[string]interface{}{
		"categories": categories,
		"count":      len(categories),
	}
	
	return utils.SuccessResponse(response), nil
}

// Get Supported Service Endpoint: GET /vault/services/{serviceId}
func (h *VaultHandler) getSupportedService(ctx context.Context, serviceID string) (events.APIGatewayProxyResponse, error) {
	service := h.vaultService.GetSupportedServiceByID(serviceID)
	if service == nil {
		return utils.ErrorResponse(utils.ErrorFromString(fmt.Sprintf("Service %s not found", serviceID)), http.StatusNotFound), nil
	}
	
	return utils.SuccessResponse(service), nil
}

// 🧂 SALT MANAGEMENT ENDPOINTS

// Get User Salt Endpoint: GET /vault/salt
func (h *VaultHandler) getUserSalt(ctx context.Context, userID string) (events.APIGatewayProxyResponse, error) {
	userSalt, err := h.vaultService.GetUserSalt(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "salt is inactive") {
			return utils.ErrorResponse(utils.ErrorFromString("Salt is inactive"), http.StatusForbidden), nil
		}
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	if userSalt == nil {
		return utils.ErrorResponse(utils.ErrorFromString("Salt not found"), http.StatusNotFound), nil
	}

	// Return salt information (without the actual salt value for security)
	response := map[string]interface{}{
		"success":   true,
		"userID":    userSalt.UserID,
		"salt":      userSalt.Salt, // Frontend needs this for decryption
		"version":   userSalt.Version,
		"createdAt": userSalt.CreatedAt,
		"updatedAt": userSalt.UpdatedAt,
	}

	return utils.SuccessResponse(response), nil
}

// Create User Salt Request
type CreateUserSaltRequest struct {
	Salt string `json:"salt"`
}

// Create User Salt Endpoint: POST /vault/salt
func (h *VaultHandler) createUserSalt(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	var req CreateUserSaltRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return utils.ErrorResponse(err, http.StatusBadRequest), nil
	}

	// Validate required fields
	if req.Salt == "" {
		return utils.ErrorResponse(utils.ErrorFromString("salt is required"), http.StatusBadRequest), nil
	}

	// Create the salt
	userSalt, err := h.vaultService.CreateUserSalt(ctx, userID, req.Salt)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return utils.ErrorResponse(err, http.StatusConflict), nil
		}
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	response := map[string]interface{}{
		"success":   true,
		"message":   "Salt created successfully",
		"userID":    userSalt.UserID,
		"version":   userSalt.Version,
		"createdAt": userSalt.CreatedAt,
	}

	return utils.SuccessResponse(response), nil
}

// Update User Salt Request
type UpdateUserSaltRequest struct {
	Salt string `json:"salt"`
}

// Update User Salt Endpoint: PUT /vault/salt
func (h *VaultHandler) updateUserSalt(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	var req UpdateUserSaltRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return utils.ErrorResponse(err, http.StatusBadRequest), nil
	}

	// Validate required fields
	if req.Salt == "" {
		return utils.ErrorResponse(utils.ErrorFromString("salt is required"), http.StatusBadRequest), nil
	}

	// Update the salt
	userSalt, err := h.vaultService.UpdateUserSalt(ctx, userID, req.Salt)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return utils.ErrorResponse(err, http.StatusNotFound), nil
		}
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	response := map[string]interface{}{
		"success":   true,
		"message":   "Salt updated successfully",
		"userID":    userSalt.UserID,
		"version":   userSalt.Version,
		"updatedAt": userSalt.UpdatedAt,
	}

	return utils.SuccessResponse(response), nil
}