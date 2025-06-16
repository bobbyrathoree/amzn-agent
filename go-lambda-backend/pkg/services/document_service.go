package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// DocumentService handles document upload and management for Knowledge Bases
type DocumentService struct {
	s3Client *s3.Client
	region   string
}

// NewDocumentService creates a new document service
func NewDocumentService(s3Client *s3.Client, region string) *DocumentService {
	return &DocumentService{
		s3Client: s3Client,
		region:   region,
	}
}

// PresignedUploadRequest represents a request for presigned upload URL
type PresignedUploadRequest struct {
	BotID       string `json:"botId" validate:"required"`
	FileName    string `json:"fileName" validate:"required"`
	ContentType string `json:"contentType" validate:"required"`
	FileSize    int64  `json:"fileSize" validate:"required,min=1,max=10485760"` // 10MB max
}

// PresignedUploadResponse contains the presigned URL and metadata
type PresignedUploadResponse struct {
	UploadURL string `json:"uploadUrl"`
	S3Key     string `json:"s3Key"`
	S3URL     string `json:"s3Url"`
	ExpiresAt int64  `json:"expiresAt"`
}

// PresignedDownloadRequest represents a request for presigned download URL
type PresignedDownloadRequest struct {
	BotID string `json:"botId" validate:"required"`
	S3Key string `json:"s3Key" validate:"required"`
}

// PresignedDownloadResponse contains the presigned download URL
type PresignedDownloadResponse struct {
	DownloadURL string `json:"downloadUrl"`
	ExpiresAt   int64  `json:"expiresAt"`
}

// DocumentListResponse represents a list of documents for a bot
type DocumentListResponse struct {
	Documents []DocumentInfo `json:"documents"`
	Count     int            `json:"count"`
}

// DocumentInfo represents metadata about a document
type DocumentInfo struct {
	S3Key        string    `json:"s3Key"`
	FileName     string    `json:"fileName"`
	ContentType  string    `json:"contentType"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
	S3URL        string    `json:"s3Url"`
}

// GeneratePresignedUploadURL creates a presigned URL for uploading documents
func (s *DocumentService) GeneratePresignedUploadURL(ctx context.Context, bucketName string, req PresignedUploadRequest) (*PresignedUploadResponse, error) {
	// Generate S3 key with bot ID and timestamp for uniqueness
	timestamp := time.Now().Unix()
	s3Key := fmt.Sprintf("bots/%s/documents/%d_%s", req.BotID, timestamp, req.FileName)
	
	// Create presigned upload URL
	presignClient := s3.NewPresignClient(s.s3Client)
	request, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(s3Key),
		ContentType: aws.String(req.ContentType),
		Metadata: map[string]string{
			"bot-id":     req.BotID,
			"file-name":  req.FileName,
			"file-size":  fmt.Sprintf("%d", req.FileSize),
			"upload-time": time.Now().Format(time.RFC3339),
		},
	}, func(opts *s3.PresignOptions) {
		opts.Expires = 15 * time.Minute // URL expires in 15 minutes
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}
	
	// Generate the final S3 URL for the uploaded file
	s3URL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, s.region, s3Key)
	
	return &PresignedUploadResponse{
		UploadURL: request.URL,
		S3Key:     s3Key,
		S3URL:     s3URL,
		ExpiresAt: time.Now().Add(15 * time.Minute).Unix(),
	}, nil
}

// GeneratePresignedDownloadURL creates a presigned URL for downloading documents
func (s *DocumentService) GeneratePresignedDownloadURL(ctx context.Context, bucketName string, req PresignedDownloadRequest) (*PresignedDownloadResponse, error) {
	// Validate that the document belongs to the bot (security check)
	if err := s.validateBotOwnership(ctx, bucketName, req.S3Key, req.BotID); err != nil {
		return nil, fmt.Errorf("unauthorized access to document: %w", err)
	}
	
	// Create presigned download URL
	presignClient := s3.NewPresignClient(s.s3Client)
	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(req.S3Key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = 1 * time.Hour // URL expires in 1 hour
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned download URL: %w", err)
	}
	
	return &PresignedDownloadResponse{
		DownloadURL: request.URL,
		ExpiresAt:   time.Now().Add(1 * time.Hour).Unix(),
	}, nil
}

// ListDocuments lists all documents for a specific bot
func (s *DocumentService) ListDocuments(ctx context.Context, bucketName, botID string) (*DocumentListResponse, error) {
	prefix := fmt.Sprintf("bots/%s/documents/", botID)
	
	// List objects with the bot prefix
	result, err := s.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(prefix),
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	
	documents := make([]DocumentInfo, 0, len(result.Contents))
	for _, obj := range result.Contents {
		// Get object metadata
		headResult, err := s.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(bucketName),
			Key:    obj.Key,
		})
		
		fileName := *obj.Key
		contentType := "application/octet-stream"
		
		if err == nil {
			// Extract metadata if available
			if headResult.Metadata != nil {
				if name, ok := headResult.Metadata["file-name"]; ok {
					fileName = name
				}
			}
			if headResult.ContentType != nil {
				contentType = *headResult.ContentType
			}
		}
		
		s3URL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, s.region, *obj.Key)
		
		documents = append(documents, DocumentInfo{
			S3Key:        *obj.Key,
			FileName:     fileName,
			ContentType:  contentType,
			Size:         *obj.Size,
			LastModified: *obj.LastModified,
			S3URL:        s3URL,
		})
	}
	
	return &DocumentListResponse{
		Documents: documents,
		Count:     len(documents),
	}, nil
}

// DeleteDocument deletes a specific document
func (s *DocumentService) DeleteDocument(ctx context.Context, bucketName, botID, s3Key string) error {
	// Validate that the document belongs to the bot (security check)
	if err := s.validateBotOwnership(ctx, bucketName, s3Key, botID); err != nil {
		return fmt.Errorf("unauthorized access to document: %w", err)
	}
	
	_, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(s3Key),
	})
	
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	
	return nil
}

// validateBotOwnership validates that an S3 key belongs to the specified bot
func (s *DocumentService) validateBotOwnership(ctx context.Context, bucketName, s3Key, botID string) error {
	expectedPrefix := fmt.Sprintf("bots/%s/documents/", botID)
	if !contains(s3Key, expectedPrefix) {
		return fmt.Errorf("document does not belong to bot %s", botID)
	}
	
	// Additional check: verify object exists and get metadata
	_, err := s.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(s3Key),
	})
	
	if err != nil {
		var nsk *types.NoSuchKey
		if err.(*types.NoSuchKey) != nil || nsk != nil {
			return fmt.Errorf("document not found")
		}
		return fmt.Errorf("failed to validate document: %w", err)
	}
	
	return nil
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}