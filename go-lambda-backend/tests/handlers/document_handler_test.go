package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/bobbyrathore/go-lambda-backend/internal/handlers"
	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
)

// Mock Document Service
type MockDocumentService struct {
	mock.Mock
}

func (m *MockDocumentService) GeneratePresignedUploadURL(ctx context.Context, bucketName string, req services.PresignedUploadRequest) (*services.PresignedUploadResponse, error) {
	args := m.Called(ctx, bucketName, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.PresignedUploadResponse), args.Error(1)
}

func (m *MockDocumentService) ListDocuments(ctx context.Context, bucketName, botID string) (*services.DocumentListResponse, error) {
	args := m.Called(ctx, bucketName, botID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.DocumentListResponse), args.Error(1)
}

func (m *MockDocumentService) DeleteDocument(ctx context.Context, bucketName, s3Key string) error {
	args := m.Called(ctx, bucketName, s3Key)
	return args.Error(0)
}

// Mock Bot Service (minimal for document handler tests)
type MockBotService struct {
	mock.Mock
}

func (m *MockBotService) GetBot(ctx context.Context, botID, userID string, userGroups []string, trackUsage bool) (*models.Bot, error) {
	args := m.Called(ctx, botID, userID, userGroups, trackUsage)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Bot), args.Error(1)
}

// Test Document Handler API Endpoints
func TestDocumentHandler(t *testing.T) {
	t.Run("HandleDocumentRequest_Routing", func(t *testing.T) {
		mockDocService := new(MockDocumentService)
		mockBotService := new(MockBotService)
		handler := handlers.NewDocumentHandler(mockDocService, mockBotService, "test-bucket")

		t.Run("POST_PresignedURL", func(t *testing.T) {
			// Mock bot access check
			mockBotService.On("GetBot", mock.Anything, "bot-123", "user-456", []string{}, false).Return(&models.Bot{
				ID:          "bot-123",
				OwnerUserID: "user-456",
				Title:       "Test Bot",
			}, nil)

			// Mock presigned URL generation
			expectedResponse := &services.PresignedUploadResponse{
				UploadURL: "https://test-bucket.s3.amazonaws.com/documents/bot-123/test.pdf?presigned",
				S3Key:     "documents/bot-123/test.pdf",
				S3Url:     "https://test-bucket.s3.amazonaws.com/documents/bot-123/test.pdf",
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
			}

			mockDocService.On("GeneratePresignedUploadURL", mock.Anything, "test-bucket", mock.MatchedBy(func(req services.PresignedUploadRequest) bool {
				return req.BotID == "bot-123" && req.FileName == "test.pdf"
			})).Return(expectedResponse, nil)

			// Create test request
			requestBody := `{
				"botId": "bot-123",
				"fileName": "test.pdf",
				"contentType": "application/pdf",
				"fileSize": 1024000
			}`

			request := events.APIGatewayProxyRequest{
				HTTPMethod: "POST",
				Path:       "/bots/bot-123/documents/presigned-url",
				Headers: map[string]string{
					"Content-Type":  "application/json",
					"Authorization": "Bearer valid-jwt-token",
					"X-User-ID":     "user-456",
				},
				Body: requestBody,
				PathParameters: map[string]string{
					"id": "bot-123",
				},
			}

			response, err := handler.HandleDocumentRequest(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, response.StatusCode)

			var responseData services.PresignedUploadResponse
			err = json.Unmarshal([]byte(response.Body), &responseData)
			assert.NoError(t, err)
			assert.Equal(t, expectedResponse.UploadURL, responseData.UploadURL)
			assert.Equal(t, expectedResponse.S3Key, responseData.S3Key)

			mockDocService.AssertExpectations(t)
			mockBotService.AssertExpectations(t)
		})

		t.Run("GET_ListDocuments", func(t *testing.T) {
			// Mock bot access check
			mockBotService.On("GetBot", mock.Anything, "bot-789", "user-456", []string{}, false).Return(&models.Bot{
				ID:          "bot-789",
				OwnerUserID: "user-456",
				Title:       "Document Bot",
			}, nil)

			// Mock document listing
			expectedDocs := &services.DocumentListResponse{
				Documents: []services.DocumentInfo{
					{
						S3Key:        "documents/bot-789/doc1.pdf",
						FileName:     "doc1.pdf",
						ContentType:  "application/pdf",
						Size:         2048000,
						LastModified: time.Now(),
						S3Url:        "https://test-bucket.s3.amazonaws.com/documents/bot-789/doc1.pdf",
					},
					{
						S3Key:        "documents/bot-789/doc2.docx",
						FileName:     "doc2.docx",
						ContentType:  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
						Size:         1536000,
						LastModified: time.Now(),
						S3Url:        "https://test-bucket.s3.amazonaws.com/documents/bot-789/doc2.docx",
					},
				},
				Count: 2,
			}

			mockDocService.On("ListDocuments", mock.Anything, "test-bucket", "bot-789").Return(expectedDocs, nil)

			request := events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/bots/bot-789/documents",
				Headers: map[string]string{
					"Authorization": "Bearer valid-jwt-token",
					"X-User-ID":     "user-456",
				},
				PathParameters: map[string]string{
					"id": "bot-789",
				},
			}

			response, err := handler.HandleDocumentRequest(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, response.StatusCode)

			var responseData services.DocumentListResponse
			err = json.Unmarshal([]byte(response.Body), &responseData)
			assert.NoError(t, err)
			assert.Equal(t, 2, responseData.Count)
			assert.Len(t, responseData.Documents, 2)
			assert.Equal(t, "doc1.pdf", responseData.Documents[0].FileName)
			assert.Equal(t, "doc2.docx", responseData.Documents[1].FileName)

			mockDocService.AssertExpectations(t)
			mockBotService.AssertExpectations(t)
		})

		t.Run("DELETE_Document", func(t *testing.T) {
			// Mock bot access check
			mockBotService.On("GetBot", mock.Anything, "bot-321", "user-456", []string{}, false).Return(&models.Bot{
				ID:          "bot-321",
				OwnerUserID: "user-456",
				Title:       "Delete Test Bot",
			}, nil)

			// Mock document deletion
			s3Key := "documents/bot-321/obsolete-doc.pdf"
			mockDocService.On("DeleteDocument", mock.Anything, "test-bucket", s3Key).Return(nil)

			request := events.APIGatewayProxyRequest{
				HTTPMethod: "DELETE",
				Path:       "/bots/bot-321/documents/" + s3Key,
				Headers: map[string]string{
					"Authorization": "Bearer valid-jwt-token",
					"X-User-ID":     "user-456",
				},
				PathParameters: map[string]string{
					"id":    "bot-321",
					"s3Key": s3Key,
				},
			}

			response, err := handler.HandleDocumentRequest(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, response.StatusCode)

			var responseData map[string]string
			err = json.Unmarshal([]byte(response.Body), &responseData)
			assert.NoError(t, err)
			assert.Equal(t, "Document deleted successfully", responseData["message"])

			mockDocService.AssertExpectations(t)
			mockBotService.AssertExpectations(t)
		})
	})

	t.Run("HandleDocumentRequest_Authorization", func(t *testing.T) {
		mockDocService := new(MockDocumentService)
		mockBotService := new(MockBotService)
		handler := handlers.NewDocumentHandler(mockDocService, mockBotService, "test-bucket")

		t.Run("Missing_Authorization_Header", func(t *testing.T) {
			request := events.APIGatewayProxyRequest{
				HTTPMethod: "POST",
				Path:       "/bots/bot-123/documents/presigned-url",
				Headers: map[string]string{
					"Content-Type": "application/json",
					// Missing Authorization header
				},
				Body: `{"botId": "bot-123", "fileName": "test.pdf", "contentType": "application/pdf", "fileSize": 1024000}`,
			}

			response, err := handler.HandleDocumentRequest(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, response.StatusCode)

			var errorResponse map[string]string
			err = json.Unmarshal([]byte(response.Body), &errorResponse)
			assert.NoError(t, err)
			assert.Contains(t, errorResponse["error"], "Authorization header required")

			// No service calls should be made
			mockDocService.AssertNotCalled(t, "GeneratePresignedUploadURL")
			mockBotService.AssertNotCalled(t, "GetBot")
		})

		t.Run("Bot_Not_Found", func(t *testing.T) {
			// Mock bot not found
			mockBotService.On("GetBot", mock.Anything, "nonexistent-bot", "user-456", []string{}, false).Return(nil, errors.New("bot not found"))

			request := events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/bots/nonexistent-bot/documents",
				Headers: map[string]string{
					"Authorization": "Bearer valid-jwt-token",
					"X-User-ID":     "user-456",
				},
				PathParameters: map[string]string{
					"id": "nonexistent-bot",
				},
			}

			response, err := handler.HandleDocumentRequest(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, response.StatusCode)

			var errorResponse map[string]string
			err = json.Unmarshal([]byte(response.Body), &errorResponse)
			assert.NoError(t, err)
			assert.Contains(t, errorResponse["error"], "Bot not found")

			mockBotService.AssertExpectations(t)
			// Document service should not be called if bot doesn't exist
			mockDocService.AssertNotCalled(t, "ListDocuments")
		})

		t.Run("Bot_Access_Denied", func(t *testing.T) {
			// Mock bot owned by different user
			mockBotService.On("GetBot", mock.Anything, "other-user-bot", "user-456", []string{}, false).Return(&models.Bot{
				ID:          "other-user-bot",
				OwnerUserID: "other-user",
				Title:       "Private Bot",
				SharedScope: models.SharedScopePrivate,
			}, nil)

			request := events.APIGatewayProxyRequest{
				HTTPMethod: "POST",
				Path:       "/bots/other-user-bot/documents/presigned-url",
				Headers: map[string]string{
					"Authorization": "Bearer valid-jwt-token",
					"X-User-ID":     "user-456",
				},
				PathParameters: map[string]string{
					"id": "other-user-bot",
				},
				Body: `{"botId": "other-user-bot", "fileName": "test.pdf", "contentType": "application/pdf", "fileSize": 1024000}`,
			}

			response, err := handler.HandleDocumentRequest(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusForbidden, response.StatusCode)

			var errorResponse map[string]string
			err = json.Unmarshal([]byte(response.Body), &errorResponse)
			assert.NoError(t, err)
			assert.Contains(t, errorResponse["error"], "Access denied")

			mockBotService.AssertExpectations(t)
			// Document service should not be called if access is denied
			mockDocService.AssertNotCalled(t, "GeneratePresignedUploadURL")
		})
	})

	t.Run("HandleDocumentRequest_Validation", func(t *testing.T) {
		mockDocService := new(MockDocumentService)
		mockBotService := new(MockBotService)
		handler := handlers.NewDocumentHandler(mockDocService, mockBotService, "test-bucket")

		t.Run("Invalid_JSON_Body", func(t *testing.T) {
			// Mock bot access check
			mockBotService.On("GetBot", mock.Anything, "bot-123", "user-456", []string{}, false).Return(&models.Bot{
				ID:          "bot-123",
				OwnerUserID: "user-456",
			}, nil)

			request := events.APIGatewayProxyRequest{
				HTTPMethod: "POST",
				Path:       "/bots/bot-123/documents/presigned-url",
				Headers: map[string]string{
					"Content-Type":  "application/json",
					"Authorization": "Bearer valid-jwt-token",
					"X-User-ID":     "user-456",
				},
				Body: `{"invalid": "json" missing closing brace`,
				PathParameters: map[string]string{
					"id": "bot-123",
				},
			}

			response, err := handler.HandleDocumentRequest(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, response.StatusCode)

			var errorResponse map[string]string
			err = json.Unmarshal([]byte(response.Body), &errorResponse)
			assert.NoError(t, err)
			assert.Contains(t, errorResponse["error"], "Invalid JSON")

			mockBotService.AssertExpectations(t)
			// Document service should not be called for invalid JSON
			mockDocService.AssertNotCalled(t, "GeneratePresignedUploadURL")
		})

		t.Run("Missing_Required_Fields", func(t *testing.T) {
			// Mock bot access check
			mockBotService.On("GetBot", mock.Anything, "bot-123", "user-456", []string{}, false).Return(&models.Bot{
				ID:          "bot-123",
				OwnerUserID: "user-456",
			}, nil)

			request := events.APIGatewayProxyRequest{
				HTTPMethod: "POST",
				Path:       "/bots/bot-123/documents/presigned-url",
				Headers: map[string]string{
					"Content-Type":  "application/json",
					"Authorization": "Bearer valid-jwt-token",
					"X-User-ID":     "user-456",
				},
				Body: `{"botId": "bot-123", "fileName": "", "contentType": "application/pdf", "fileSize": 1024000}`, // Empty fileName
				PathParameters: map[string]string{
					"id": "bot-123",
				},
			}

			response, err := handler.HandleDocumentRequest(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, response.StatusCode)

			var errorResponse map[string]string
			err = json.Unmarshal([]byte(response.Body), &errorResponse)
			assert.NoError(t, err)
			assert.Contains(t, errorResponse["error"], "validation failed")

			mockBotService.AssertExpectations(t)
			mockDocService.AssertNotCalled(t, "GeneratePresignedUploadURL")
		})

		t.Run("File_Size_Too_Large", func(t *testing.T) {
			// Mock bot access check
			mockBotService.On("GetBot", mock.Anything, "bot-123", "user-456", []string{}, false).Return(&models.Bot{
				ID:          "bot-123",
				OwnerUserID: "user-456",
			}, nil)

			// Mock document service returning file size error
			mockDocService.On("GeneratePresignedUploadURL", mock.Anything, "test-bucket", mock.Anything).Return(nil, errors.New("file size exceeds maximum allowed (10MB)"))

			request := events.APIGatewayProxyRequest{
				HTTPMethod: "POST",
				Path:       "/bots/bot-123/documents/presigned-url",
				Headers: map[string]string{
					"Content-Type":  "application/json",
					"Authorization": "Bearer valid-jwt-token",
					"X-User-ID":     "user-456",
				},
				Body: `{"botId": "bot-123", "fileName": "huge-file.pdf", "contentType": "application/pdf", "fileSize": 15728640}`, // 15MB
				PathParameters: map[string]string{
					"id": "bot-123",
				},
			}

			response, err := handler.HandleDocumentRequest(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, response.StatusCode)

			var errorResponse map[string]string
			err = json.Unmarshal([]byte(response.Body), &errorResponse)
			assert.NoError(t, err)
			assert.Contains(t, errorResponse["error"], "file size exceeds maximum")

			mockBotService.AssertExpectations(t)
			mockDocService.AssertExpectations(t)
		})
	})

	t.Run("HandleDocumentRequest_Service_Errors", func(t *testing.T) {
		mockDocService := new(MockDocumentService)
		mockBotService := new(MockBotService)
		handler := handlers.NewDocumentHandler(mockDocService, mockBotService, "test-bucket")

		t.Run("S3_Service_Error", func(t *testing.T) {
			// Mock bot access check
			mockBotService.On("GetBot", mock.Anything, "bot-123", "user-456", []string{}, false).Return(&models.Bot{
				ID:          "bot-123",
				OwnerUserID: "user-456",
			}, nil)

			// Mock S3 service error
			mockDocService.On("GeneratePresignedUploadURL", mock.Anything, "test-bucket", mock.Anything).Return(nil, errors.New("S3 service temporarily unavailable"))

			request := events.APIGatewayProxyRequest{
				HTTPMethod: "POST",
				Path:       "/bots/bot-123/documents/presigned-url",
				Headers: map[string]string{
					"Content-Type":  "application/json",
					"Authorization": "Bearer valid-jwt-token",
					"X-User-ID":     "user-456",
				},
				Body: `{"botId": "bot-123", "fileName": "test.pdf", "contentType": "application/pdf", "fileSize": 1024000}`,
				PathParameters: map[string]string{
					"id": "bot-123",
				},
			}

			response, err := handler.HandleDocumentRequest(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusInternalServerError, response.StatusCode)

			var errorResponse map[string]string
			err = json.Unmarshal([]byte(response.Body), &errorResponse)
			assert.NoError(t, err)
			assert.Contains(t, errorResponse["error"], "Failed to generate presigned URL")

			mockBotService.AssertExpectations(t)
			mockDocService.AssertExpectations(t)
		})

		t.Run("Document_List_Service_Error", func(t *testing.T) {
			// Mock bot access check
			mockBotService.On("GetBot", mock.Anything, "bot-456", "user-789", []string{}, false).Return(&models.Bot{
				ID:          "bot-456",
				OwnerUserID: "user-789",
			}, nil)

			// Mock document service error
			mockDocService.On("ListDocuments", mock.Anything, "test-bucket", "bot-456").Return(nil, errors.New("S3 bucket access denied"))

			request := events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/bots/bot-456/documents",
				Headers: map[string]string{
					"Authorization": "Bearer valid-jwt-token",
					"X-User-ID":     "user-789",
				},
				PathParameters: map[string]string{
					"id": "bot-456",
				},
			}

			response, err := handler.HandleDocumentRequest(context.Background(), request)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusInternalServerError, response.StatusCode)

			var errorResponse map[string]string
			err = json.Unmarshal([]byte(response.Body), &errorResponse)
			assert.NoError(t, err)
			assert.Contains(t, errorResponse["error"], "Failed to list documents")

			mockBotService.AssertExpectations(t)
			mockDocService.AssertExpectations(t)
		})
	})

	t.Run("HandleDocumentRequest_Unsupported_Method", func(t *testing.T) {
		mockDocService := new(MockDocumentService)
		mockBotService := new(MockBotService)
		handler := handlers.NewDocumentHandler(mockDocService, mockBotService, "test-bucket")

		request := events.APIGatewayProxyRequest{
			HTTPMethod: "PATCH", // Unsupported method
			Path:       "/bots/bot-123/documents",
			Headers: map[string]string{
				"Authorization": "Bearer valid-jwt-token",
				"X-User-ID":     "user-456",
			},
		}

		response, err := handler.HandleDocumentRequest(context.Background(), request)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)

		var errorResponse map[string]string
		err = json.Unmarshal([]byte(response.Body), &errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse["error"], "Method not allowed")

		// No service calls should be made for unsupported methods
		mockDocService.AssertNotCalled(t, "GeneratePresignedUploadURL")
		mockDocService.AssertNotCalled(t, "ListDocuments")
		mockDocService.AssertNotCalled(t, "DeleteDocument")
		mockBotService.AssertNotCalled(t, "GetBot")
	})
}

// Test Document Handler Helper Functions
func TestDocumentHandlerHelpers(t *testing.T) {
	mockDocService := new(MockDocumentService)
	mockBotService := new(MockBotService)
	handler := handlers.NewDocumentHandler(mockDocService, mockBotService, "test-bucket")

	t.Run("ExtractBotIDFromPath", func(t *testing.T) {
		tests := []struct {
			path     string
			expected string
		}{
			{"/bots/bot-123/documents/presigned-url", "bot-123"},
			{"/bots/my-special-bot/documents", "my-special-bot"},
			{"/bots/bot-456/documents/documents%2Fbot-456%2Ffile.pdf", "bot-456"},
		}

		for _, test := range tests {
			result := handler.ExtractBotIDFromPath(test.path)
			assert.Equal(t, test.expected, result, "Path: %s", test.path)
		}
	})

	t.Run("ExtractS3KeyFromPath", func(t *testing.T) {
		tests := []struct {
			path     string
			expected string
		}{
			{"/bots/bot-123/documents/documents%2Fbot-123%2Ffile.pdf", "documents/bot-123/file.pdf"},
			{"/bots/bot-456/documents/documents%2Fbot-456%2Ffolder%2Fdoc.docx", "documents/bot-456/folder/doc.docx"},
		}

		for _, test := range tests {
			result := handler.ExtractS3KeyFromPath(test.path)
			assert.Equal(t, test.expected, result, "Path: %s", test.path)
		}
	})

	t.Run("ValidateDocumentAccess", func(t *testing.T) {
		// Test owner access
		ownerBot := &models.Bot{
			ID:          "bot-123",
			OwnerUserID: "user-owner",
			SharedScope: models.SharedScopePrivate,
		}

		assert.True(t, handler.ValidateDocumentAccess(ownerBot, "user-owner", []string{}))

		// Test non-owner access to private bot
		assert.False(t, handler.ValidateDocumentAccess(ownerBot, "user-other", []string{}))

		// Test public bot access
		publicBot := &models.Bot{
			ID:          "bot-456",
			OwnerUserID: "user-owner",
			SharedScope: models.SharedScopePublic,
		}

		assert.True(t, handler.ValidateDocumentAccess(publicBot, "user-other", []string{}))

		// Test partial sharing with group access
		partialBot := &models.Bot{
			ID:            "bot-789",
			OwnerUserID:   "user-owner",
			SharedScope:   models.SharedScopePartial,
			AllowedGroups: []string{"team-alpha", "team-beta"},
		}

		assert.True(t, handler.ValidateDocumentAccess(partialBot, "user-member", []string{"team-alpha", "other-group"}))
		assert.False(t, handler.ValidateDocumentAccess(partialBot, "user-outsider", []string{"different-group"}))
	})

	t.Run("BuildErrorResponse", func(t *testing.T) {
		response := handler.BuildErrorResponse(http.StatusBadRequest, "Invalid request parameters")

		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		assert.Equal(t, "application/json", response.Headers["Content-Type"])

		var errorData map[string]string
		err := json.Unmarshal([]byte(response.Body), &errorData)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid request parameters", errorData["error"])
	})

	t.Run("BuildSuccessResponse", func(t *testing.T) {
		data := map[string]interface{}{
			"message": "Operation successful",
			"id":      123,
			"active":  true,
		}

		response := handler.BuildSuccessResponse(data)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "application/json", response.Headers["Content-Type"])

		var responseData map[string]interface{}
		err := json.Unmarshal([]byte(response.Body), &responseData)
		assert.NoError(t, err)
		assert.Equal(t, "Operation successful", responseData["message"])
		assert.Equal(t, float64(123), responseData["id"]) // JSON numbers become float64
		assert.Equal(t, true, responseData["active"])
	})
}

// Test Path Parameter Extraction
func TestDocumentHandlerPathParameters(t *testing.T) {
	mockDocService := new(MockDocumentService)
	mockBotService := new(MockBotService)
	handler := handlers.NewDocumentHandler(mockDocService, mockBotService, "test-bucket")

	t.Run("Path_Parameter_Handling", func(t *testing.T) {
		// Test with path parameters in request
		request := events.APIGatewayProxyRequest{
			Path: "/bots/test-bot-id/documents/presigned-url",
			PathParameters: map[string]string{
				"id": "test-bot-id",
			},
		}

		botID := handler.GetBotIDFromRequest(request)
		assert.Equal(t, "test-bot-id", botID)

		// Test fallback to path parsing when path parameters are missing
		requestWithoutParams := events.APIGatewayProxyRequest{
			Path:           "/bots/fallback-bot-id/documents",
			PathParameters: nil,
		}

		botIDFallback := handler.GetBotIDFromRequest(requestWithoutParams)
		assert.Equal(t, "fallback-bot-id", botIDFallback)
	})

	t.Run("S3_Key_URL_Decoding", func(t *testing.T) {
		// Test URL-encoded S3 key extraction
		encodedPath := "/bots/bot-123/documents/documents%2Fbot-123%2Fmy%20file%20with%20spaces.pdf"
		
		s3Key := handler.ExtractS3KeyFromPath(encodedPath)
		expected := "documents/bot-123/my file with spaces.pdf"
		
		assert.Equal(t, expected, s3Key)
	})
}