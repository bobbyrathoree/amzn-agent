package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
)

// Mock S3 Client
type MockS3Client struct {
	mock.Mock
}

func (m *MockS3Client) PutObjectPresign(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*s3.PresignedPutObjectRequest, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.PresignedPutObjectRequest), args.Error(1)
}

func (m *MockS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.ListObjectsV2Options)) (*s3.ListObjectsV2Output, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.ListObjectsV2Output), args.Error(1)
}

func (m *MockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.DeleteObjectOptions)) (*s3.DeleteObjectOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.DeleteObjectOutput), args.Error(1)
}

func (m *MockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.HeadObjectOptions)) (*s3.HeadObjectOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.HeadObjectOutput), args.Error(1)
}

// Test Document Service Core Functionality
func TestDocumentService(t *testing.T) {
	t.Run("GeneratePresignedUploadURL", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			mockS3 := new(MockS3Client)
			service := services.NewDocumentService(mockS3, "us-east-1")

			// Mock successful presigned URL generation
			expectedURL := "https://test-bucket.s3.amazonaws.com/documents/bot-123/test-file.pdf?presigned=params"
			mockS3.On("PutObjectPresign", mock.Anything, mock.MatchedBy(func(params *s3.PutObjectInput) bool {
				return *params.Bucket == "test-bucket" &&
					*params.Key == "documents/bot-123/test-file.pdf" &&
					*params.ContentType == "application/pdf"
			})).Return(&s3.PresignedPutObjectRequest{
				URL: expectedURL,
			}, nil)

			// Test request
			req := services.PresignedUploadRequest{
				BotID:       "bot-123",
				FileName:    "test-file.pdf",
				ContentType: "application/pdf",
				FileSize:    1024000,
			}

			result, err := service.GeneratePresignedUploadURL(context.Background(), "test-bucket", req)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, expectedURL, result.UploadURL)
			assert.Equal(t, "documents/bot-123/test-file.pdf", result.S3Key)
			assert.Contains(t, result.S3Url, "test-bucket")
			assert.Contains(t, result.S3Url, "documents/bot-123/test-file.pdf")
			assert.True(t, result.ExpiresAt > time.Now().Unix())

			mockS3.AssertExpectations(t)
		})

		t.Run("S3_Error", func(t *testing.T) {
			mockS3 := new(MockS3Client)
			service := services.NewDocumentService(mockS3, "us-east-1")

			// Mock S3 error
			mockS3.On("PutObjectPresign", mock.Anything, mock.Anything).Return(nil, errors.New("S3 access denied"))

			req := services.PresignedUploadRequest{
				BotID:       "bot-123",
				FileName:    "test-file.pdf",
				ContentType: "application/pdf",
				FileSize:    1024000,
			}

			result, err := service.GeneratePresignedUploadURL(context.Background(), "test-bucket", req)

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "failed to generate presigned URL")

			mockS3.AssertExpectations(t)
		})

		t.Run("File_Size_Validation", func(t *testing.T) {
			mockS3 := new(MockS3Client)
			service := services.NewDocumentService(mockS3, "us-east-1")

			// Test maximum file size validation (10MB limit)
			req := services.PresignedUploadRequest{
				BotID:       "bot-123",
				FileName:    "large-file.pdf",
				ContentType: "application/pdf",
				FileSize:    11 * 1024 * 1024, // 11MB - exceeds limit
			}

			result, err := service.GeneratePresignedUploadURL(context.Background(), "test-bucket", req)

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "file size exceeds maximum")

			// No S3 calls should be made for validation failures
			mockS3.AssertNotCalled(t, "PutObjectPresign")
		})

		t.Run("Content_Type_Validation", func(t *testing.T) {
			mockS3 := new(MockS3Client)
			service := services.NewDocumentService(mockS3, "us-east-1")

			// Test unsupported content type
			req := services.PresignedUploadRequest{
				BotID:       "bot-123",
				FileName:    "image.jpg",
				ContentType: "image/jpeg", // Not in allowed types
				FileSize:    1024000,
			}

			result, err := service.GeneratePresignedUploadURL(context.Background(), "test-bucket", req)

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "unsupported content type")

			mockS3.AssertNotCalled(t, "PutObjectPresign")
		})
	})

	t.Run("ListDocuments", func(t *testing.T) {
		t.Run("Success_With_Documents", func(t *testing.T) {
			mockS3 := new(MockS3Client)
			service := services.NewDocumentService(mockS3, "us-east-1")

			// Mock successful S3 list response
			mockTime := time.Now()
			size1 := int64(1024000)
			size2 := int64(2048000)

			mockS3.On("ListObjectsV2", mock.Anything, mock.MatchedBy(func(params *s3.ListObjectsV2Input) bool {
				return *params.Bucket == "test-bucket" &&
					*params.Prefix == "documents/bot-123/"
			})).Return(&s3.ListObjectsV2Output{
				Contents: []s3types.Object{
					{
						Key:          aws.String("documents/bot-123/file1.pdf"),
						Size:         &size1,
						LastModified: &mockTime,
					},
					{
						Key:          aws.String("documents/bot-123/file2.docx"),
						Size:         &size2,
						LastModified: &mockTime,
					},
				},
			}, nil)

			// Mock HeadObject calls for content type
			mockS3.On("HeadObject", mock.Anything, mock.MatchedBy(func(params *s3.HeadObjectInput) bool {
				return *params.Key == "documents/bot-123/file1.pdf"
			})).Return(&s3.HeadObjectOutput{
				ContentType: aws.String("application/pdf"),
			}, nil)

			mockS3.On("HeadObject", mock.Anything, mock.MatchedBy(func(params *s3.HeadObjectInput) bool {
				return *params.Key == "documents/bot-123/file2.docx"
			})).Return(&s3.HeadObjectOutput{
				ContentType: aws.String("application/vnd.openxmlformats-officedocument.wordprocessingml.document"),
			}, nil)

			result, err := service.ListDocuments(context.Background(), "test-bucket", "bot-123")

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, 2, result.Count)
			assert.Len(t, result.Documents, 2)

			// Verify first document
			doc1 := result.Documents[0]
			assert.Equal(t, "documents/bot-123/file1.pdf", doc1.S3Key)
			assert.Equal(t, "file1.pdf", doc1.FileName)
			assert.Equal(t, "application/pdf", doc1.ContentType)
			assert.Equal(t, size1, doc1.Size)
			assert.Equal(t, mockTime.UTC(), doc1.LastModified.UTC())

			// Verify second document
			doc2 := result.Documents[1]
			assert.Equal(t, "documents/bot-123/file2.docx", doc2.S3Key)
			assert.Equal(t, "file2.docx", doc2.FileName)
			assert.Equal(t, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", doc2.ContentType)
			assert.Equal(t, size2, doc2.Size)

			mockS3.AssertExpectations(t)
		})

		t.Run("Success_Empty_Bucket", func(t *testing.T) {
			mockS3 := new(MockS3Client)
			service := services.NewDocumentService(mockS3, "us-east-1")

			// Mock empty S3 response
			mockS3.On("ListObjectsV2", mock.Anything, mock.Anything).Return(&s3.ListObjectsV2Output{
				Contents: []s3types.Object{},
			}, nil)

			result, err := service.ListDocuments(context.Background(), "test-bucket", "bot-123")

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, 0, result.Count)
			assert.Len(t, result.Documents, 0)

			mockS3.AssertExpectations(t)
		})

		t.Run("S3_List_Error", func(t *testing.T) {
			mockS3 := new(MockS3Client)
			service := services.NewDocumentService(mockS3, "us-east-1")

			// Mock S3 error
			mockS3.On("ListObjectsV2", mock.Anything, mock.Anything).Return(nil, errors.New("S3 bucket not found"))

			result, err := service.ListDocuments(context.Background(), "test-bucket", "bot-123")

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "failed to list documents")

			mockS3.AssertExpectations(t)
		})
	})

	t.Run("DeleteDocument", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			mockS3 := new(MockS3Client)
			service := services.NewDocumentService(mockS3, "us-east-1")

			// Mock successful delete
			mockS3.On("DeleteObject", mock.Anything, mock.MatchedBy(func(params *s3.DeleteObjectInput) bool {
				return *params.Bucket == "test-bucket" &&
					*params.Key == "documents/bot-123/test-file.pdf"
			})).Return(&s3.DeleteObjectOutput{}, nil)

			err := service.DeleteDocument(context.Background(), "test-bucket", "documents/bot-123/test-file.pdf")

			assert.NoError(t, err)
			mockS3.AssertExpectations(t)
		})

		t.Run("S3_Delete_Error", func(t *testing.T) {
			mockS3 := new(MockS3Client)
			service := services.NewDocumentService(mockS3, "us-east-1")

			// Mock S3 error
			mockS3.On("DeleteObject", mock.Anything, mock.Anything).Return(nil, errors.New("S3 access denied"))

			err := service.DeleteDocument(context.Background(), "test-bucket", "documents/bot-123/test-file.pdf")

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to delete document")

			mockS3.AssertExpectations(t)
		})

		t.Run("Invalid_S3_Key", func(t *testing.T) {
			mockS3 := new(MockS3Client)
			service := services.NewDocumentService(mockS3, "us-east-1")

			// Test with invalid S3 key (not in documents/ prefix)
			err := service.DeleteDocument(context.Background(), "test-bucket", "invalid/path/file.pdf")

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid S3 key")

			// No S3 calls should be made for validation failures
			mockS3.AssertNotCalled(t, "DeleteObject")
		})
	})
}

// Test Helper Functions
func TestDocumentServiceHelpers(t *testing.T) {
	t.Run("BuildS3Key", func(t *testing.T) {
		service := services.NewDocumentService(nil, "us-east-1")

		tests := []struct {
			botID    string
			fileName string
			expected string
		}{
			{"bot-123", "document.pdf", "documents/bot-123/document.pdf"},
			{"bot-456", "my file.docx", "documents/bot-456/my file.docx"},
			{"test-bot", "README.md", "documents/test-bot/README.md"},
		}

		for _, test := range tests {
			result := service.BuildS3Key(test.botID, test.fileName)
			assert.Equal(t, test.expected, result)
		}
	})

	t.Run("ExtractFileName", func(t *testing.T) {
		service := services.NewDocumentService(nil, "us-east-1")

		tests := []struct {
			s3Key    string
			expected string
		}{
			{"documents/bot-123/document.pdf", "document.pdf"},
			{"documents/bot-456/my file.docx", "my file.docx"},
			{"documents/test-bot/README.md", "README.md"},
			{"documents/bot-789/nested/folder/file.txt", "file.txt"},
		}

		for _, test := range tests {
			result := service.ExtractFileName(test.s3Key)
			assert.Equal(t, test.expected, result)
		}
	})

	t.Run("IsValidContentType", func(t *testing.T) {
		service := services.NewDocumentService(nil, "us-east-1")

		// Valid content types
		validTypes := []string{
			"application/pdf",
			"application/msword",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"text/plain",
			"text/markdown",
		}

		for _, contentType := range validTypes {
			assert.True(t, service.IsValidContentType(contentType), "Should accept: %s", contentType)
		}

		// Invalid content types
		invalidTypes := []string{
			"image/jpeg",
			"image/png",
			"video/mp4",
			"application/zip",
			"text/html",
		}

		for _, contentType := range invalidTypes {
			assert.False(t, service.IsValidContentType(contentType), "Should reject: %s", contentType)
		}
	})

	t.Run("BuildS3URL", func(t *testing.T) {
		service := services.NewDocumentService(nil, "us-east-1")

		result := service.BuildS3URL("test-bucket", "documents/bot-123/file.pdf")
		expected := "https://test-bucket.s3.us-east-1.amazonaws.com/documents/bot-123/file.pdf"

		assert.Equal(t, expected, result)
	})
}

// Test Request/Response Model Validation
func TestDocumentModels(t *testing.T) {
	t.Run("PresignedUploadRequest_Validation", func(t *testing.T) {
		// Test valid request
		validRequest := services.PresignedUploadRequest{
			BotID:       "bot-123",
			FileName:    "document.pdf",
			ContentType: "application/pdf",
			FileSize:    1024000,
		}

		assert.True(t, validRequest.IsValid())

		// Test invalid requests
		invalidRequests := []services.PresignedUploadRequest{
			{BotID: "", FileName: "document.pdf", ContentType: "application/pdf", FileSize: 1024000}, // Empty BotID
			{BotID: "bot-123", FileName: "", ContentType: "application/pdf", FileSize: 1024000},     // Empty FileName
			{BotID: "bot-123", FileName: "document.pdf", ContentType: "", FileSize: 1024000},        // Empty ContentType
			{BotID: "bot-123", FileName: "document.pdf", ContentType: "application/pdf", FileSize: 0}, // Zero FileSize
		}

		for i, req := range invalidRequests {
			assert.False(t, req.IsValid(), "Request %d should be invalid", i)
		}
	})

	t.Run("DocumentInfo_JSON_Marshaling", func(t *testing.T) {
		doc := services.DocumentInfo{
			S3Key:        "documents/bot-123/test.pdf",
			FileName:     "test.pdf",
			ContentType:  "application/pdf",
			Size:         1024000,
			LastModified: time.Date(2025, 6, 11, 12, 0, 0, 0, time.UTC),
			S3Url:        "https://bucket.s3.amazonaws.com/documents/bot-123/test.pdf",
		}

		// Test JSON marshaling
		jsonData, err := json.Marshal(doc)
		assert.NoError(t, err)

		// Test JSON unmarshaling
		var unmarshaled services.DocumentInfo
		err = json.Unmarshal(jsonData, &unmarshaled)
		assert.NoError(t, err)

		assert.Equal(t, doc.S3Key, unmarshaled.S3Key)
		assert.Equal(t, doc.FileName, unmarshaled.FileName)
		assert.Equal(t, doc.ContentType, unmarshaled.ContentType)
		assert.Equal(t, doc.Size, unmarshaled.Size)
		assert.Equal(t, doc.S3Url, unmarshaled.S3Url)
		assert.True(t, doc.LastModified.Equal(unmarshaled.LastModified))
	})
}