package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

var (
	jwkSet    jwk.Set
	userPoolID string
	awsRegion  string
)

// init function runs once during cold start to fetch JWKS
func init() {
	userPoolID = os.Getenv("USER_POOL_ID")
	awsRegion = os.Getenv("AWS_REGION")

	if userPoolID == "" {
		log.Fatal("USER_POOL_ID environment variable is required")
	}
	if awsRegion == "" {
		log.Fatal("AWS_REGION environment variable is required")
	}

	// Fetch JWKS from Cognito
	jwksURL := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", awsRegion, userPoolID)
	log.Printf("Fetching JWKS from: %s", jwksURL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	jwkSet, err = jwk.Fetch(ctx, jwksURL)
	if err != nil {
		log.Fatalf("Failed to fetch JWKS: %v", err)
	}
	log.Printf("Successfully fetched JWKS with %d keys", jwkSet.Len())
}

// Handler processes WebSocket authorization requests
// Security: This authorizer validates Cognito JWT tokens for WebSocket connections
// Reference: AppSec vulnerability ticket V1796596356
func Handler(ctx context.Context, request events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	log.Printf("Authorization request for methodArn: %s", request.MethodArn)

	// Extract token from query string parameter
	token, ok := request.QueryStringParameters["token"]
	if !ok || token == "" {
		log.Println("Authorization denied: token not found in query string")
		return generatePolicy("", "Deny", request.MethodArn, nil), nil
	}

	// Validate and parse the JWT token
	claims, err := validateToken(token)
	if err != nil {
		log.Printf("Authorization denied: token validation failed: %v", err)
		return generatePolicy("", "Deny", request.MethodArn, nil), nil
	}

	// Extract user ID (sub claim)
	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		log.Println("Authorization denied: sub claim not found in token")
		return generatePolicy("", "Deny", request.MethodArn, nil), nil
	}

	log.Printf("Authorization granted for user: %s", userID)

	// Return Allow policy with userId in context
	context := map[string]interface{}{
		"userId": userID,
	}

	return generatePolicy(userID, "Allow", request.MethodArn, context), nil
}

// validateToken validates the JWT token against Cognito JWKS
func validateToken(tokenString string) (jwt.MapClaims, error) {
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing algorithm
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get the key ID from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid header not found")
		}

		// Find the key in JWKS
		key, found := jwkSet.LookupKeyID(kid)
		if !found {
			return nil, fmt.Errorf("key with kid %s not found in JWKS", kid)
		}

		// Convert JWK to RSA public key
		var rawKey interface{}
		if err := key.Raw(&rawKey); err != nil {
			return nil, fmt.Errorf("failed to get raw key: %w", err)
		}

		return rawKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Validate token claims
	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to parse claims")
	}

	// Verify token_use claim (should be "id" for ID tokens or "access" for access tokens)
	tokenUse, ok := claims["token_use"].(string)
	if !ok || (tokenUse != "id" && tokenUse != "access") {
		return nil, fmt.Errorf("invalid token_use claim: %v", tokenUse)
	}

	// Verify issuer
	expectedIssuer := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", awsRegion, userPoolID)
	iss, ok := claims["iss"].(string)
	if !ok || iss != expectedIssuer {
		return nil, fmt.Errorf("invalid issuer: %v", iss)
	}

	// Validate expiration (exp claim)
	// Security: Prevent expired token reuse
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return nil, fmt.Errorf("token has expired")
		}
	} else {
		return nil, fmt.Errorf("exp claim not found")
	}

	// Validate audience (for ID tokens) or client_id (for access tokens)
	// Security: Prevent token misuse across different applications
	expectedClientID := os.Getenv("USER_POOL_CLIENT_ID")
	if expectedClientID == "" {
		log.Println("Warning: USER_POOL_CLIENT_ID not set, skipping audience validation")
	} else {
		// Check 'aud' first (ID tokens), fall back to 'client_id' (access tokens)
		if aud, ok := claims["aud"].(string); ok {
			if aud != expectedClientID {
				return nil, fmt.Errorf("invalid audience: %v", aud)
			}
		} else if clientID, ok := claims["client_id"].(string); ok {
			if clientID != expectedClientID {
				return nil, fmt.Errorf("invalid client_id: %v", clientID)
			}
		} else {
			return nil, fmt.Errorf("neither aud nor client_id found in token")
		}
	}

	// Validate issued-at time (not in future)
	// Security: Prevent time-manipulation attacks
	if iat, ok := claims["iat"].(float64); ok {
		if int64(iat) > time.Now().Unix() {
			return nil, fmt.Errorf("token issued in the future")
		}
	}

	return claims, nil
}

// generatePolicy creates an IAM policy for API Gateway
func generatePolicy(principalID, effect, resource string, context map[string]interface{}) events.APIGatewayCustomAuthorizerResponse {
	// For WebSocket APIs, we need to allow all routes under the connection
	// Extract the base ARN and allow all routes
	resourceParts := strings.Split(resource, "/")
	if len(resourceParts) >= 2 {
		// Convert specific route ARN to wildcard
		// Example: arn:aws:execute-api:region:account:api-id/stage/route -> arn:aws:execute-api:region:account:api-id/stage/*
		baseArn := strings.Join(resourceParts[:len(resourceParts)-1], "/")
		resource = baseArn + "/*"
	}

	authResponse := events.APIGatewayCustomAuthorizerResponse{
		PrincipalID: principalID,
		PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
			Statement: []events.IAMPolicyStatement{
				{
					Action:   []string{"execute-api:Invoke"},
					Effect:   effect,
					Resource: []string{resource},
				},
			},
		},
	}

	// Add context if provided and effect is Allow
	if effect == "Allow" && context != nil {
		// Convert context to string map (API Gateway requirement)
		contextStr := make(map[string]interface{})
		for k, v := range context {
			// Convert to JSON string if not already a string
			if str, ok := v.(string); ok {
				contextStr[k] = str
			} else {
				jsonBytes, _ := json.Marshal(v)
				contextStr[k] = string(jsonBytes)
			}
		}
		authResponse.Context = contextStr
	}

	return authResponse
}

func main() {
	lambda.Start(Handler)
}
