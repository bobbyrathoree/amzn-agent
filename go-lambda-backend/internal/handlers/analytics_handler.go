// 📊 ANALYTICS API HANDLER
// REST endpoints for usage analytics and intelligence

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
)

type AnalyticsHandler struct {
	analyticsService *services.AnalyticsService
}

func NewAnalyticsHandler(analyticsService *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
	}
}

// 📊 ANALYTICS ENDPOINTS

// TrackUsage records a new API usage event
func (h *AnalyticsHandler) TrackUsage(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("📊 Tracking usage event from user: %s", getUserID(request))

	// Parse request body
	var event services.APIUsageEvent
	if err := json.Unmarshal([]byte(request.Body), &event); err != nil {
		log.Printf("❌ Error parsing usage event: %v", err)
		return errorResponse(http.StatusBadRequest, "Invalid request body"), nil
	}

	// Set user ID from authentication context
	event.UserID = getUserID(request)
	event.Timestamp = time.Now().UTC()

	// Extract additional context from request
	if event.UserAgent == "" {
		event.UserAgent = hashString(request.Headers["User-Agent"])
	}
	if event.IPAddressHash == "" {
		event.IPAddressHash = hashString(getClientIP(request))
	}

	// Track the usage
	err := h.analyticsService.TrackAPIUsage(ctx, event)
	if err != nil {
		log.Printf("❌ Error tracking usage: %v", err)
		return errorResponse(http.StatusInternalServerError, "Failed to track usage"), nil
	}

	response := map[string]interface{}{
		"success":   true,
		"message":   "Usage tracked successfully",
		"eventId":   event.ID,
		"timestamp": event.Timestamp,
	}

	return successResponse(response), nil
}

// GetUsageSummary retrieves user usage summary
func (h *AnalyticsHandler) GetUsageSummary(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userID := getUserID(request)
	log.Printf("📈 Getting usage summary for user: %s", userID)

	// Parse query parameters
	daysParam := request.QueryStringParameters["days"]
	days := 30 // Default to 30 days
	if daysParam != "" {
		if parsedDays, err := strconv.Atoi(daysParam); err == nil && parsedDays > 0 {
			days = parsedDays
		}
	}

	// Get usage summary
	summaries, err := h.analyticsService.GetUserUsageSummary(ctx, userID, days)
	if err != nil {
		log.Printf("❌ Error getting usage summary: %v", err)
		return errorResponse(http.StatusInternalServerError, "Failed to get usage summary"), nil
	}

	// Calculate aggregate metrics
	aggregateMetrics := calculateAggregateMetrics(summaries)

	response := map[string]interface{}{
		"summaries":    summaries,
		"aggregates":   aggregateMetrics,
		"periodDays":   days,
		"generatedAt":  time.Now().UTC(),
	}

	return successResponse(response), nil
}

// GetUserProfile retrieves comprehensive user analytics profile
func (h *AnalyticsHandler) GetUserProfile(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userID := getUserID(request)
	log.Printf("👤 Getting analytics profile for user: %s", userID)

	// Get user profile
	profile, err := h.analyticsService.GetUserAnalyticsProfile(ctx, userID)
	if err != nil {
		log.Printf("❌ Error getting user profile: %v", err)
		return errorResponse(http.StatusInternalServerError, "Failed to get user profile"), nil
	}

	return successResponse(profile), nil
}

// GetCostAnalysis provides detailed cost analysis and forecasting
func (h *AnalyticsHandler) GetCostAnalysis(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userID := getUserID(request)
	log.Printf("💰 Getting cost analysis for user: %s", userID)

	// Get usage data for cost analysis
	summaries, err := h.analyticsService.GetUserUsageSummary(ctx, userID, 60) // Last 60 days
	if err != nil {
		log.Printf("❌ Error getting usage data: %v", err)
		return errorResponse(http.StatusInternalServerError, "Failed to get cost data"), nil
	}

	// Calculate cost analysis
	costAnalysis := calculateCostAnalysis(summaries)

	return successResponse(costAnalysis), nil
}

// GetServicePerformance provides service performance metrics
func (h *AnalyticsHandler) GetServicePerformance(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userID := getUserID(request)
	log.Printf("⚡ Getting service performance for user: %s", userID)

	// Get usage data
	summaries, err := h.analyticsService.GetUserUsageSummary(ctx, userID, 30)
	if err != nil {
		log.Printf("❌ Error getting performance data: %v", err)
		return errorResponse(http.StatusInternalServerError, "Failed to get performance data"), nil
	}

	// Calculate service rankings
	rankings := services.GetServiceCostRanking(summaries)

	// Group by service for detailed metrics
	serviceMetrics := groupServiceMetrics(summaries)

	response := map[string]interface{}{
		"rankings":        rankings,
		"serviceMetrics":  serviceMetrics,
		"generatedAt":     time.Now().UTC(),
	}

	return successResponse(response), nil
}

// GetRealTimeMetrics provides current usage statistics
func (h *AnalyticsHandler) GetRealTimeMetrics(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userID := getUserID(request)
	log.Printf("⚡ Getting real-time metrics for user: %s", userID)

	// Get today's usage
	today := time.Now().UTC().Format("2006-01-02")
	summaries, err := h.analyticsService.GetUserUsageSummary(ctx, userID, 1)
	if err != nil {
		log.Printf("❌ Error getting real-time metrics: %v", err)
		return errorResponse(http.StatusInternalServerError, "Failed to get real-time metrics"), nil
	}

	// Calculate real-time stats
	realTimeStats := calculateRealTimeStats(summaries, today)

	return successResponse(realTimeStats), nil
}

// 🧮 ANALYTICS CALCULATIONS

func calculateAggregateMetrics(summaries []services.DailyUsageSummary) map[string]interface{} {
	if len(summaries) == 0 {
		return map[string]interface{}{
			"totalRequests":     0,
			"totalCost":         0,
			"avgResponseTime":   0,
			"overallSuccessRate": 0,
			"activeServices":    0,
		}
	}

	totalRequests := 0
	totalCost := 0.0
	totalResponseTime := 0.0
	totalSuccessful := 0
	activeServices := make(map[string]bool)

	for _, summary := range summaries {
		totalRequests += summary.TotalRequests
		totalCost += summary.TotalCost
		totalResponseTime += summary.AvgResponseTime * float64(summary.TotalRequests)
		totalSuccessful += summary.SuccessfulRequests
		activeServices[summary.ServiceID] = true
	}

	avgResponseTime := 0.0
	if totalRequests > 0 {
		avgResponseTime = totalResponseTime / float64(totalRequests)
	}

	successRate := 0.0
	if totalRequests > 0 {
		successRate = float64(totalSuccessful) / float64(totalRequests) * 100
	}

	return map[string]interface{}{
		"totalRequests":      totalRequests,
		"totalCost":          totalCost,
		"avgResponseTime":    avgResponseTime,
		"overallSuccessRate": successRate,
		"activeServices":     len(activeServices),
		"costPerRequest":     totalCost / float64(totalRequests),
	}
}

func calculateCostAnalysis(summaries []services.DailyUsageSummary) map[string]interface{} {
	if len(summaries) == 0 {
		return map[string]interface{}{
			"currentMonthSpend":    0,
			"projectedMonthlySpend": 0,
			"costTrend":           "stable",
			"topCostServices":     []interface{}{},
		}
	}

	// Calculate current month spend
	currentMonth := time.Now().Format("2006-01")
	currentMonthSpend := 0.0
	
	// Calculate service costs
	serviceCosts := make(map[string]float64)
	
	for _, summary := range summaries {
		if summary.Date[:7] == currentMonth {
			currentMonthSpend += summary.TotalCost
		}
		serviceCosts[summary.ServiceID] += summary.TotalCost
	}

	// Project monthly spend
	projectedSpend := services.EstimateMonthlySpend(summaries)

	// Calculate cost trend
	costTrend := calculateCostTrend(summaries)

	// Get top cost services
	topServices := getTopCostServices(serviceCosts, 5)

	return map[string]interface{}{
		"currentMonthSpend":    currentMonthSpend,
		"projectedMonthlySpend": projectedSpend,
		"costTrend":           costTrend,
		"topCostServices":     topServices,
		"serviceCostBreakdown": serviceCosts,
		"costPerDay":          projectedSpend / 30,
	}
}

func calculateCostTrend(summaries []services.DailyUsageSummary) string {
	if len(summaries) < 14 {
		return "insufficient_data"
	}

	// Compare first week vs last week
	midpoint := len(summaries) / 2
	firstHalfCost := 0.0
	secondHalfCost := 0.0

	for i, summary := range summaries {
		if i < midpoint {
			firstHalfCost += summary.TotalCost
		} else {
			secondHalfCost += summary.TotalCost
		}
	}

	if firstHalfCost == 0 {
		return "increasing"
	}

	ratio := secondHalfCost / firstHalfCost

	if ratio > 1.1 {
		return "increasing"
	} else if ratio < 0.9 {
		return "decreasing"
	}
	return "stable"
}

func getTopCostServices(serviceCosts map[string]float64, limit int) []map[string]interface{} {
	// Convert to slice for sorting
	type serviceEntry struct {
		ServiceID string
		Cost      float64
	}

	var entries []serviceEntry
	for serviceID, cost := range serviceCosts {
		entries = append(entries, serviceEntry{
			ServiceID: serviceID,
			Cost:      cost,
		})
	}

	// Sort by cost (descending)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].Cost < entries[j].Cost {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Convert to response format
	var result []map[string]interface{}
	maxEntries := limit
	if len(entries) < maxEntries {
		maxEntries = len(entries)
	}

	for i := 0; i < maxEntries; i++ {
		result = append(result, map[string]interface{}{
			"serviceId": entries[i].ServiceID,
			"cost":      entries[i].Cost,
			"rank":      i + 1,
		})
	}

	return result
}

func groupServiceMetrics(summaries []services.DailyUsageSummary) map[string]interface{} {
	serviceGroups := make(map[string][]services.DailyUsageSummary)
	
	// Group summaries by service
	for _, summary := range summaries {
		serviceGroups[summary.ServiceID] = append(serviceGroups[summary.ServiceID], summary)
	}

	// Calculate metrics for each service
	result := make(map[string]interface{})
	for serviceID, serviceSummaries := range serviceGroups {
		metrics := calculateServiceMetrics(serviceSummaries)
		result[serviceID] = metrics
	}

	return result
}

func calculateServiceMetrics(summaries []services.DailyUsageSummary) map[string]interface{} {
	if len(summaries) == 0 {
		return map[string]interface{}{}
	}

	totalRequests := 0
	totalCost := 0.0
	totalResponseTime := 0.0
	totalSuccessful := 0

	for _, summary := range summaries {
		totalRequests += summary.TotalRequests
		totalCost += summary.TotalCost
		totalResponseTime += summary.AvgResponseTime * float64(summary.TotalRequests)
		totalSuccessful += summary.SuccessfulRequests
	}

	avgResponseTime := 0.0
	if totalRequests > 0 {
		avgResponseTime = totalResponseTime / float64(totalRequests)
	}

	successRate := 0.0
	if totalRequests > 0 {
		successRate = float64(totalSuccessful) / float64(totalRequests) * 100
	}

	return map[string]interface{}{
		"totalRequests":   totalRequests,
		"totalCost":       totalCost,
		"avgResponseTime": avgResponseTime,
		"successRate":     successRate,
		"costPerRequest":  totalCost / float64(totalRequests),
		"activeDays":      len(summaries),
	}
}

func calculateRealTimeStats(summaries []services.DailyUsageSummary, today string) map[string]interface{} {
	// Find today's data
	var todaysSummary *services.DailyUsageSummary
	for _, summary := range summaries {
		if summary.Date == today {
			todaysSummary = &summary
			break
		}
	}

	if todaysSummary == nil {
		return map[string]interface{}{
			"todayRequests": 0,
			"todayCost":     0,
			"todayServices": 0,
			"isActive":      false,
		}
	}

	return map[string]interface{}{
		"todayRequests":     todaysSummary.TotalRequests,
		"todayCost":         todaysSummary.TotalCost,
		"todaySuccessRate":  todaysSummary.SuccessRate,
		"todayAvgResponse":  todaysSummary.AvgResponseTime,
		"peakHour":          todaysSummary.PeakHour,
		"isActive":          todaysSummary.TotalRequests > 0,
		"lastUpdated":       todaysSummary.LastUpdated,
	}
}

// 🛠️ HELPER FUNCTIONS

func getUserID(request events.APIGatewayProxyRequest) string {
	// Extract user ID from JWT token or authentication context
	// This would be implemented based on your authentication system
	if userID, exists := request.RequestContext.Authorizer["userId"]; exists {
		if str, ok := userID.(string); ok {
			return str
		}
	}
	return "anonymous" // Fallback for development
}

func getClientIP(request events.APIGatewayProxyRequest) string {
	// Try various headers for client IP
	if ip := request.Headers["X-Forwarded-For"]; ip != "" {
		return ip
	}
	if ip := request.Headers["X-Real-IP"]; ip != "" {
		return ip
	}
	return request.RequestContext.Identity.SourceIP
}

func hashString(input string) string {
	// Simple hash for privacy (in production, use proper crypto hashing)
	if input == "" {
		return ""
	}
	return fmt.Sprintf("hash_%x", len(input)*17+42) // Placeholder hash
}

func successResponse(data interface{}) events.APIGatewayProxyResponse {
	jsonData, _ := json.Marshal(data)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(jsonData),
	}
}

func errorResponse(statusCode int, message string) events.APIGatewayProxyResponse {
	errorData := map[string]interface{}{
		"error":   true,
		"message": message,
	}
	jsonData, _ := json.Marshal(errorData)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(jsonData),
	}
}