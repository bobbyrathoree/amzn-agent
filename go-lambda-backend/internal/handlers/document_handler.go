package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator/v10"

	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
	"github.com/bobbyrathore/go-lambda-backend/pkg/utils"
)

// DocumentHandler handles document upload and management
type DocumentHandler struct {
	documentService *services.DocumentService
	botService      *services.BotService
	validator       *validator.Validate
	bucketName      string
}

// NewDocumentHandler creates a new document handler
func NewDocumentHandler(documentService *services.DocumentService, botService *services.BotService, bucketName string) *DocumentHandler {
	return &DocumentHandler{
		documentService: documentService,
		botService:      botService,
		validator:       validator.New(),
		bucketName:      bucketName,
	}
}

// HandleDocumentRequest routes document-related HTTP requests
func (h *DocumentHandler) HandleDocumentRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	method := request.HTTPMethod
	pathSegments := strings.Split(strings.Trim(request.Path, "/"), "/")
	
	// Extract user ID from JWT token
	userID, userGroups := h.extractUserInfoFromJWT(request)
	if userID == "" {
		return utils.ErrorResponse(fmt.Errorf("unauthorized"), http.StatusUnauthorized), nil
	}
	
	// Route based on path structure
	// Expected paths:
	// POST   /bots/{id}/documents/presigned-url
	// GET    /bots/{id}/documents
	// DELETE /bots/{id}/documents/{s3Key}
	// GET    /bots/{id}/documents/{s3Key}/download-url
	
	if len(pathSegments) < 4 || pathSegments[0] != "bots" || pathSegments[2] != "documents" {
		return utils.ErrorResponse(fmt.Errorf("invalid path"), http.StatusNotFound), nil
	}
	
	botID := pathSegments[1]
	
	// Verify user has access to the bot
	if err := h.validateBotAccess(ctx, botID, userID, userGroups); err != nil {
		return utils.ErrorResponse(err, http.StatusForbidden), nil
	}
	
	switch method {
	case "POST":
		if len(pathSegments) == 4 && pathSegments[3] == "presigned-url" {
			return h.generatePresignedUploadURL(ctx, request, botID)
		}
	case "GET":
		if len(pathSegments) == 3 {
			return h.listDocuments(ctx, botID)
		} else if len(pathSegments) == 5 && pathSegments[4] == "download-url" {
			return h.generatePresignedDownloadURL(ctx, pathSegments[3], botID)
		}
	case "DELETE":
		if len(pathSegments) == 4 {
			return h.deleteDocument(ctx, pathSegments[3], botID)
		}
	}
	
	return utils.ErrorResponse(fmt.Errorf("endpoint not found"), http.StatusNotFound), nil
}

// generatePresignedUploadURL handles POST /bots/{id}/documents/presigned-url
func (h *DocumentHandler) generatePresignedUploadURL(ctx context.Context, request events.APIGatewayProxyRequest, botID string) (events.APIGatewayProxyResponse, error) {
	var req services.PresignedUploadRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return utils.ErrorResponse(fmt.Errorf("invalid JSON"), http.StatusBadRequest), nil
	}
	
	// Set bot ID from path
	req.BotID = botID
	
	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		return utils.ErrorResponse(fmt.Errorf("validation failed: %v", err), http.StatusBadRequest), nil
	}
	
	// Generate presigned URL
	response, err := h.documentService.GeneratePresignedUploadURL(ctx, h.bucketName, req)
	if err != nil {
		return utils.ErrorResponse(fmt.Errorf("failed to generate upload URL: %v", err), http.StatusInternalServerError), nil
	}
	
	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// generatePresignedDownloadURL handles GET /bots/{id}/documents/{s3Key}/download-url
func (h *DocumentHandler) generatePresignedDownloadURL(ctx context.Context, s3Key, botID string) (events.APIGatewayProxyResponse, error) {
	req := services.PresignedDownloadRequest{
		BotID: botID,
		S3Key: s3Key,
	}
	
	// Generate presigned URL
	response, err := h.documentService.GeneratePresignedDownloadURL(ctx, h.bucketName, req)
	if err != nil {
		return utils.ErrorResponse(fmt.Errorf("failed to generate download URL: %v", err), http.StatusInternalServerError), nil
	}
	
	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// listDocuments handles GET /bots/{id}/documents
func (h *DocumentHandler) listDocuments(ctx context.Context, botID string) (events.APIGatewayProxyResponse, error) {
	response, err := h.documentService.ListDocuments(ctx, h.bucketName, botID)
	if err != nil {
		return utils.ErrorResponse(fmt.Errorf("failed to list documents: %v", err), http.StatusInternalServerError), nil
	}
	
	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// deleteDocument handles DELETE /bots/{id}/documents/{s3Key}
func (h *DocumentHandler) deleteDocument(ctx context.Context, s3Key, botID string) (events.APIGatewayProxyResponse, error) {
	err := h.documentService.DeleteDocument(ctx, h.bucketName, botID, s3Key)
	if err != nil {
		return utils.ErrorResponse(fmt.Errorf("failed to delete document: %v", err), http.StatusInternalServerError), nil
	}
	
	response := map[string]interface{}{
		"message": "Document deleted successfully",
		"s3Key":   s3Key,
	}
	
	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// validateBotAccess checks if user has access to the specified bot
func (h *DocumentHandler) validateBotAccess(ctx context.Context, botID, userID string, userGroups []string) error {
	// Get the bot to check permissions
	bot, err := h.botService.GetBot(ctx, botID, userID, userGroups, false)
	if err != nil {
		if err == services.ErrBotNotFound {
			return fmt.Errorf("bot not found")
		}
		return fmt.Errorf("failed to validate bot access: %v", err)
	}
	
	// Check if user has edit access (required for document management)
	if !bot.IsEditableByUser(userID, userGroups, false) {
		return fmt.Errorf("insufficient permissions to manage documents for this bot")
	}
	
	return nil
}

// extractUserInfoFromJWT extracts user ID and groups from JWT claims
func (h *DocumentHandler) extractUserInfoFromJWT(request events.APIGatewayProxyRequest) (string, []string) {
	userID := ""
	var userGroups []string
	
	// Extract from JWT authorizer context
	if request.RequestContext.Authorizer != nil {
		if claims, ok := request.RequestContext.Authorizer["claims"].(map[string]interface{}); ok {
			if sub, ok := claims["sub"].(string); ok {
				userID = sub
			}
			
			if groups, ok := claims["cognito:groups"].([]interface{}); ok {
				for _, group := range groups {
					if groupStr, ok := group.(string); ok {
						userGroups = append(userGroups, groupStr)
					}
				}
			}
		}
	}
	
	return userID, userGroups
}