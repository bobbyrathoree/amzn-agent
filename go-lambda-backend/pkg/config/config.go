package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// Config holds all configuration for the application
type Config struct {
	// AWS related configuration
	AWSRegion       string
	DynamoDBTable   string
	S3BucketName    string
	SQSQueueURL     string
	SecretsName     string
	LogLevel        string

	// API related configuration
	APIVersion      string
	CORSOrigins     []string

	// Application specific configuration
	DefaultPageSize int
	MaxPageSize     int
}

// DatabaseConfig holds database connection info (retrieved from Secrets Manager)
type DatabaseConfig struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	Port     int    `json:"port"`
}

// NewConfig creates a new configuration instance from environment variables
func NewConfig() *Config {
	return &Config{
		AWSRegion:       getEnv("AWS_REGION", "us-west-2"),
		DynamoDBTable:   getEnv("DYNAMODB_TABLE", ""),
		S3BucketName:    getEnv("S3_BUCKET_NAME", ""),
		SQSQueueURL:     getEnv("SQS_QUEUE_URL", ""),
		SecretsName:     getEnv("SECRETS_NAME", ""),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		APIVersion:      getEnv("API_VERSION", "v1"),
		CORSOrigins:     getCORSOrigins(),
		DefaultPageSize: getEnvAsInt("DEFAULT_PAGE_SIZE", 20),
		MaxPageSize:     getEnvAsInt("MAX_PAGE_SIZE", 100),
	}
}

// GetDatabaseConfig retrieves database configuration from AWS Secrets Manager
func (c *Config) GetDatabaseConfig(ctx context.Context) (*DatabaseConfig, error) {
	// Create AWS SDK config
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(c.AWSRegion))
	if err != nil {
		return nil, err
	}

	// Create Secrets Manager client
	client := secretsmanager.NewFromConfig(cfg)

	// Get the secret
	result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(c.SecretsName),
	})
	if err != nil {
		return nil, err
	}

	// Parse the secret JSON
	var dbConfig DatabaseConfig
	err = json.Unmarshal([]byte(*result.SecretString), &dbConfig)
	if err != nil {
		return nil, err
	}

	return &dbConfig, nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt retrieves an environment variable as an integer
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value := 0
	_, err := fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		return defaultValue
	}

	return value
}

// getCORSOrigins parses the CORS_ORIGINS environment variable into a slice
func getCORSOrigins() []string {
	origins := getEnv("CORS_ORIGINS", "*")
	if origins == "*" {
		return []string{"*"}
	}
	return strings.Split(origins, ",")
}