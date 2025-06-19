package handlers

import (
	"strings"
)

// isNotFoundError checks if an error indicates a resource was not found
func isNotFoundError(err error) bool {
	errStr := err.Error()
	return contains(errStr, "ResourceNotFound") ||
		contains(errStr, "NotFound") ||
		contains(errStr, "does not exist")
}

// isAccessDeniedError checks if an error indicates access was denied
func isAccessDeniedError(err error) bool {
	errStr := err.Error()
	return contains(errStr, "AccessDenied") ||
		contains(errStr, "UnauthorizedOperation") ||
		contains(errStr, "Forbidden") ||
		contains(errStr, "access denied")
}

// corsHeaders returns standard CORS headers for API responses
func corsHeaders() map[string]string {
	return map[string]string{
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization, X-Amz-Date, X-Api-Key, X-Amz-Security-Token, x-user-id",
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}