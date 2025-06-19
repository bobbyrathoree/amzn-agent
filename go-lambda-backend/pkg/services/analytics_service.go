// 📊 ANALYTICS SERVICE
// Enterprise-grade usage analytics and intelligence system

package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
)

// 🎯 CORE DATA STRUCTURES

type APIUsageEvent struct {
	// Identity & Context
	ID        string `json:"id" dynamodb:"id"`
	UserID    string `json:"userId" dynamodb:"userId"`
	ServiceID string `json:"serviceId" dynamodb:"serviceId"`
	ToolID    string `json:"toolId,omitempty" dynamodb:"toolId,omitempty"`
	SessionID string `json:"sessionId,omitempty" dynamodb:"sessionId,omitempty"`

	// Timing
	Timestamp    time.Time `json:"timestamp" dynamodb:"timestamp"`
	ResponseTime int64     `json:"responseTime" dynamodb:"responseTime"` // milliseconds

	// Request Details
	RequestMethod string `json:"requestMethod" dynamodb:"requestMethod"`
	RequestSize   int64  `json:"requestSize,omitempty" dynamodb:"requestSize,omitempty"`
	RequestPath   string `json:"requestPath,omitempty" dynamodb:"requestPath,omitempty"`

	// Response Details
	Success      bool   `json:"success" dynamodb:"success"`
	StatusCode   int    `json:"statusCode" dynamodb:"statusCode"`
	ResponseSize int64  `json:"responseSize,omitempty" dynamodb:"responseSize,omitempty"`
	ErrorCode    string `json:"errorCode,omitempty" dynamodb:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty" dynamodb:"errorMessage,omitempty"`

	// Cost & Usage
	EstimatedCost   float64 `json:"estimatedCost" dynamodb:"estimatedCost"`     // USD cents
	CreditsUsed     int     `json:"creditsUsed,omitempty" dynamodb:"creditsUsed,omitempty"`
	TokensUsed      int     `json:"tokensUsed,omitempty" dynamodb:"tokensUsed,omitempty"`
	ResultsReturned int     `json:"resultsReturned,omitempty" dynamodb:"resultsReturned,omitempty"`

	// Performance Metrics
	CacheHit      bool   `json:"cacheHit,omitempty" dynamodb:"cacheHit,omitempty"`
	FallbackUsed  bool   `json:"fallbackUsed,omitempty" dynamodb:"fallbackUsed,omitempty"`
	FallbackLevel int    `json:"fallbackLevel,omitempty" dynamodb:"fallbackLevel,omitempty"`
	ServiceMode   string `json:"serviceMode" dynamodb:"serviceMode"` // premium | free

	// Metadata
	UserAgent     string            `json:"userAgent,omitempty" dynamodb:"userAgent,omitempty"`
	IPAddressHash string            `json:"ipAddressHash,omitempty" dynamodb:"ipAddressHash,omitempty"`
	Geolocation   map[string]string `json:"geolocation,omitempty" dynamodb:"geolocation,omitempty"`

	// Analytics Enrichment
	ProcessedAt time.Time `json:"processedAt" dynamodb:"processedAt"`
	Version     string    `json:"version" dynamodb:"version"`
}

type DailyUsageSummary struct {
	// Identity & Time
	ID        string `json:"id" dynamodb:"id"`         // userId_serviceId_date
	UserID    string `json:"userId" dynamodb:"userId"`
	ServiceID string `json:"serviceId" dynamodb:"serviceId"`
	Date      string `json:"date" dynamodb:"date"` // YYYY-MM-DD

	// Volume Metrics
	TotalRequests     int     `json:"totalRequests" dynamodb:"totalRequests"`
	SuccessfulRequests int    `json:"successfulRequests" dynamodb:"successfulRequests"`
	FailedRequests    int     `json:"failedRequests" dynamodb:"failedRequests"`
	SuccessRate       float64 `json:"successRate" dynamodb:"successRate"`

	// Performance Metrics
	AvgResponseTime float64 `json:"avgResponseTime" dynamodb:"avgResponseTime"`
	P95ResponseTime float64 `json:"p95ResponseTime" dynamodb:"p95ResponseTime"`
	MaxResponseTime int64   `json:"maxResponseTime" dynamodb:"maxResponseTime"`
	MinResponseTime int64   `json:"minResponseTime" dynamodb:"minResponseTime"`

	// Cost Metrics
	TotalCost         float64 `json:"totalCost" dynamodb:"totalCost"`
	AvgCostPerRequest float64 `json:"avgCostPerRequest" dynamodb:"avgCostPerRequest"`
	TotalCreditsUsed  int     `json:"totalCreditsUsed" dynamodb:"totalCreditsUsed"`
	TotalTokensUsed   int     `json:"totalTokensUsed" dynamodb:"totalTokensUsed"`

	// Usage Patterns
	PeakHour         int `json:"peakHour" dynamodb:"peakHour"`                   // 0-23
	PeakRequestCount int `json:"peakRequestCount" dynamodb:"peakRequestCount"`
	UniqueToolsUsed  int `json:"uniqueToolsUsed" dynamodb:"uniqueToolsUsed"`

	// Service Mode Distribution
	PremiumRequests  int `json:"premiumRequests" dynamodb:"premiumRequests"`
	FreeRequests     int `json:"freeRequests" dynamodb:"freeRequests"`
	FallbackRequests int `json:"fallbackRequests" dynamodb:"fallbackRequests"`

	// Data Quality
	LastUpdated time.Time `json:"lastUpdated" dynamodb:"lastUpdated"`
	RecordCount int       `json:"recordCount" dynamodb:"recordCount"`
}

type UserAnalyticsProfile struct {
	// Identity
	UserID string `json:"userId" dynamodb:"userId"`

	// Usage Patterns
	TotalAPIKeys      int      `json:"totalApiKeys" dynamodb:"totalApiKeys"`
	ActiveServices    []string `json:"activeServices" dynamodb:"activeServices"`
	FavoriteService   string   `json:"favoriteService" dynamodb:"favoriteService"`
	UsageFrequency    string   `json:"usageFrequency" dynamodb:"usageFrequency"` // light | moderate | heavy
	
	// Cost & Spending
	TotalSpent       float64 `json:"totalSpent" dynamodb:"totalSpent"`
	MonthlySpend     float64 `json:"monthlySpend" dynamodb:"monthlySpend"`
	AvgMonthlySpend  float64 `json:"avgMonthlySpend" dynamodb:"avgMonthlySpend"`
	CostTrend        string  `json:"costTrend" dynamodb:"costTrend"` // increasing | stable | decreasing

	// Behavioral Insights
	PreferredUsageHours   []int   `json:"preferredUsageHours" dynamodb:"preferredUsageHours"`
	AvgSessionDuration    float64 `json:"avgSessionDuration" dynamodb:"avgSessionDuration"`
	ToolAdoptionRate      float64 `json:"toolAdoptionRate" dynamodb:"toolAdoptionRate"`
	PremiumFeatureUsage   float64 `json:"premiumFeatureUsage" dynamodb:"premiumFeatureUsage"`

	// Efficiency Metrics
	SuccessRate       float64 `json:"successRate" dynamodb:"successRate"`
	AvgResponseTime   float64 `json:"avgResponseTime" dynamodb:"avgResponseTime"`
	CacheUtilization  float64 `json:"cacheUtilization" dynamodb:"cacheUtilization"`

	// Lifecycle
	FirstUsage     time.Time `json:"firstUsage" dynamodb:"firstUsage"`
	LastUsage      time.Time `json:"lastUsage" dynamodb:"lastUsage"`
	UsageDays      int       `json:"usageDays" dynamodb:"usageDays"`
	RetentionScore float64   `json:"retentionScore" dynamodb:"retentionScore"`

	// Data Freshness
	LastUpdated  time.Time `json:"lastUpdated" dynamodb:"lastUpdated"`
	CalculatedAt time.Time `json:"calculatedAt" dynamodb:"calculatedAt"`
}

// 🔧 SERVICE COST CALCULATORS

type ServiceCostCalculator interface {
	CalculateCost(usage UsageMetrics) CostBreakdown
	GetServiceID() string
}

type UsageMetrics struct {
	Requests     int     `json:"requests"`
	Tokens       int     `json:"tokens,omitempty"`
	Results      int     `json:"results,omitempty"`
	ResponseTime int64   `json:"responseTime,omitempty"`
	DataSize     int64   `json:"dataSize,omitempty"`
}

type CostBreakdown struct {
	BaseCost    float64 `json:"baseCost"`    // Fixed cost per request
	VolumeCost  float64 `json:"volumeCost"`  // Cost based on volume
	PremiumCost float64 `json:"premiumCost"` // Premium feature costs
	TotalCost   float64 `json:"totalCost"`   // Total estimated cost
	Confidence  float64 `json:"confidence"`  // Confidence in estimation (0-1)
}

// 🏭 ANALYTICS SERVICE

type AnalyticsService struct {
	repository *AnalyticsRepository
	costCalcs  map[string]ServiceCostCalculator
}

func NewAnalyticsService(dynamoClient *dynamodb.Client, tablePrefix string) *AnalyticsService {
	service := &AnalyticsService{
		repository: NewAnalyticsRepository(dynamoClient, tablePrefix),
		costCalcs:  make(map[string]ServiceCostCalculator),
	}

	// Initialize cost calculators
	service.initializeCostCalculators()
	
	return service
}

func (s *AnalyticsService) initializeCostCalculators() {
	log.Println("🧮 Initializing cost calculators...")
	
	// Register cost calculators for each service
	s.costCalcs["openai"] = &OpenAICostCalculator{}
	s.costCalcs["anthropic"] = &AnthropicCostCalculator{}
	s.costCalcs["google-search"] = &GoogleSearchCostCalculator{}
	s.costCalcs["weather-api"] = &WeatherAPICostCalculator{}
	s.costCalcs["news-api"] = &NewsAPICostCalculator{}
	s.costCalcs["github"] = &GitHubCostCalculator{}
	s.costCalcs["slack"] = &SlackCostCalculator{}
	s.costCalcs["hubspot"] = &HubSpotCostCalculator{}
	s.costCalcs["tavily"] = &TavilyCostCalculator{}
	s.costCalcs["serpapi"] = &SerpAPICostCalculator{}

	log.Printf("✅ Initialized %d cost calculators", len(s.costCalcs))
}

// 📊 CORE ANALYTICS METHODS

// TrackAPIUsage records a new API usage event
func (s *AnalyticsService) TrackAPIUsage(ctx context.Context, event APIUsageEvent) error {
	log.Printf("📊 Tracking API usage for user %s, service %s", event.UserID, event.ServiceID)

	// Generate ID if not provided
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	// Set processing metadata
	event.ProcessedAt = time.Now().UTC()
	event.Version = "1.0"

	// Calculate cost if not provided
	if event.EstimatedCost == 0 {
		cost := s.calculateEventCost(event)
		event.EstimatedCost = cost.TotalCost
	}

	// Store using repository
	err := s.repository.StoreUsageEvent(ctx, event)
	if err != nil {
		log.Printf("❌ Error storing usage event: %v", err)
		return fmt.Errorf("failed to store usage event: %w", err)
	}

	log.Printf("✅ Successfully tracked API usage event: %s", event.ID)

	// Trigger real-time processing (async)
	go s.processEventRealTime(ctx, event)

	return nil
}

// GetUserUsageSummary retrieves usage summary for a user
func (s *AnalyticsService) GetUserUsageSummary(ctx context.Context, userID string, days int) ([]DailyUsageSummary, error) {
	log.Printf("📈 Getting usage summary for user %s (last %d days)", userID, days)

	// Use repository for real data retrieval
	summaries, err := s.repository.GetDailySummariesForUser(ctx, userID, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get user summaries: %w", err)
	}

	log.Printf("✅ Retrieved %d usage summaries for user %s", len(summaries), userID)
	return summaries, nil
}

// GetUserAnalyticsProfile retrieves comprehensive user analytics
func (s *AnalyticsService) GetUserAnalyticsProfile(ctx context.Context, userID string) (*UserAnalyticsProfile, error) {
	log.Printf("👤 Getting analytics profile for user %s", userID)

	// Try to get existing profile
	profile, err := s.repository.GetUserProfile(ctx, userID)
	if err != nil {
		log.Printf("ℹ️ No existing profile found, generating new one: %v", err)
		
		// Generate fresh profile using repository
		profile, err = s.repository.GenerateUserAnalyticsProfile(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to generate user profile: %w", err)
		}

		// Store the generated profile
		err = s.repository.StoreUserProfile(ctx, *profile)
		if err != nil {
			log.Printf("⚠️ Failed to store user profile: %v", err)
		}
	}

	log.Printf("✅ Retrieved analytics profile for user %s", userID)
	return profile, nil
}

// 💰 COST CALCULATION

func (s *AnalyticsService) calculateEventCost(event APIUsageEvent) CostBreakdown {
	calculator, exists := s.costCalcs[event.ServiceID]
	if !exists {
		log.Printf("⚠️ No cost calculator found for service: %s", event.ServiceID)
		return CostBreakdown{
			TotalCost:  0.01, // Default minimal cost
			Confidence: 0.1,  // Low confidence
		}
	}

	usage := UsageMetrics{
		Requests:     1,
		Tokens:       event.TokensUsed,
		Results:      event.ResultsReturned,
		ResponseTime: event.ResponseTime,
	}

	cost := calculator.CalculateCost(usage)
	log.Printf("💰 Calculated cost for %s: $%.4f (confidence: %.2f)", event.ServiceID, cost.TotalCost/100, cost.Confidence)
	
	return cost
}

// 🔄 REAL-TIME PROCESSING

func (s *AnalyticsService) processEventRealTime(ctx context.Context, event APIUsageEvent) {
	log.Printf("⚡ Processing event in real-time: %s", event.ID)

	// Update running totals (could use Redis for high performance)
	// Update user session data
	// Trigger alerts if thresholds exceeded
	// Update real-time dashboards via WebSocket

	// For now, just log the processing
	log.Printf("✅ Real-time processing completed for event: %s", event.ID)
}

// 📊 BATCH PROCESSING METHODS

// GenerateDailySummariesForUser generates daily summaries for a specific user and date
func (s *AnalyticsService) GenerateDailySummariesForUser(ctx context.Context, userID string, date string) error {
	return s.repository.GenerateDailySummariesFromEvents(ctx, userID, date)
}

// BatchGenerateSummaries processes daily summaries for all users for a given date
func (s *AnalyticsService) BatchGenerateSummaries(ctx context.Context, date string) error {
	return s.repository.BatchProcessDailySummaries(ctx, date)
}