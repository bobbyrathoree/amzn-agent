// 🗄️ ANALYTICS REPOSITORY - PRODUCTION DYNAMODB IMPLEMENTATION
// Real database operations with full DynamoDB integration

package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// 🗄️ PRODUCTION REPOSITORY IMPLEMENTATION

type AnalyticsRepository struct {
	dynamoClient *dynamodb.Client
	tablePrefix  string
}

func NewAnalyticsRepository(dynamoClient *dynamodb.Client, tablePrefix string) *AnalyticsRepository {
	return &AnalyticsRepository{
		dynamoClient: dynamoClient,
		tablePrefix:  tablePrefix,
	}
}

// Table names
func (r *AnalyticsRepository) usageEventsTable() string {
	return r.tablePrefix + "-usage-events"
}

func (r *AnalyticsRepository) dailySummariesTable() string {
	return r.tablePrefix + "-daily-summaries"
}

func (r *AnalyticsRepository) userProfilesTable() string {
	return r.tablePrefix + "-user-profiles"
}

// 📊 USAGE EVENTS OPERATIONS

// StoreUsageEvent stores a usage event in DynamoDB
func (r *AnalyticsRepository) StoreUsageEvent(ctx context.Context, event APIUsageEvent) error {
	log.Printf("🗄️ Storing usage event: %s", event.ID)

	// Convert event to DynamoDB item
	item, err := attributevalue.MarshalMap(event)
	if err != nil {
		return fmt.Errorf("failed to marshal usage event: %w", err)
	}

	// Add partition key for time-series querying
	// Partition by user and day for efficient querying
	dateStr := event.Timestamp.Format("2006-01-02")
	partitionKey := fmt.Sprintf("%s#%s", event.UserID, dateStr)
	
	// Sort key combines timestamp and event ID for uniqueness
	sortKey := fmt.Sprintf("%s#%s", event.Timestamp.Format("15:04:05.000"), event.ID)

	item["PK"] = &types.AttributeValueMemberS{Value: partitionKey}
	item["SK"] = &types.AttributeValueMemberS{Value: sortKey}
	item["GSI1PK"] = &types.AttributeValueMemberS{Value: event.ServiceID}          // For service-based queries
	item["GSI1SK"] = &types.AttributeValueMemberS{Value: event.Timestamp.Format(time.RFC3339Nano)}
	item["TTL"] = &types.AttributeValueMemberN{Value: strconv.FormatInt(event.Timestamp.AddDate(2, 0, 0).Unix(), 10)} // 2 year TTL

	// Store in DynamoDB
	_, err = r.dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.usageEventsTable()),
		Item:      item,
		ConditionExpression: aws.String("attribute_not_exists(PK)"), // Prevent duplicates
	})

	if err != nil {
		return fmt.Errorf("failed to store usage event: %w", err)
	}

	log.Printf("✅ Successfully stored usage event: %s", event.ID)
	return nil
}

// GetUsageEventsForUser retrieves usage events for a user within a date range
func (r *AnalyticsRepository) GetUsageEventsForUser(ctx context.Context, userID string, startDate, endDate time.Time) ([]APIUsageEvent, error) {
	log.Printf("🔍 Getting usage events for user %s from %s to %s", userID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	var allEvents []APIUsageEvent

	// Query each day in the range (DynamoDB limitation requires this approach)
	for d := startDate; d.Before(endDate) || d.Equal(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		partitionKey := fmt.Sprintf("%s#%s", userID, dateStr)

		// Query DynamoDB for this specific day
		result, err := r.dynamoClient.Query(ctx, &dynamodb.QueryInput{
			TableName:              aws.String(r.usageEventsTable()),
			KeyConditionExpression: aws.String("PK = :pk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: partitionKey},
			},
			ScanIndexForward: aws.Bool(true), // Sort by sort key ascending (chronological)
		})

		if err != nil {
			log.Printf("⚠️ Error querying events for date %s: %v", dateStr, err)
			continue
		}

		// Convert DynamoDB items to events
		var dayEvents []APIUsageEvent
		err = attributevalue.UnmarshalListOfMaps(result.Items, &dayEvents)
		if err != nil {
			log.Printf("⚠️ Error unmarshalling events for date %s: %v", dateStr, err)
			continue
		}

		allEvents = append(allEvents, dayEvents...)
	}

	log.Printf("✅ Retrieved %d usage events for user %s", len(allEvents), userID)
	return allEvents, nil
}

// GetUsageEventsByService retrieves usage events for a specific service
func (r *AnalyticsRepository) GetUsageEventsByService(ctx context.Context, serviceID string, startDate, endDate time.Time) ([]APIUsageEvent, error) {
	log.Printf("🔍 Getting usage events for service %s from %s to %s", serviceID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	// Use GSI1 to query by service
	result, err := r.dynamoClient.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.usageEventsTable()),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :serviceId AND GSI1SK BETWEEN :startTime AND :endTime"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":serviceId":  &types.AttributeValueMemberS{Value: serviceID},
			":startTime": &types.AttributeValueMemberS{Value: startDate.Format(time.RFC3339Nano)},
			":endTime":   &types.AttributeValueMemberS{Value: endDate.Format(time.RFC3339Nano)},
		},
		ScanIndexForward: aws.Bool(true),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query service events: %w", err)
	}

	// Convert to events
	var events []APIUsageEvent
	err = attributevalue.UnmarshalListOfMaps(result.Items, &events)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal service events: %w", err)
	}

	log.Printf("✅ Retrieved %d usage events for service %s", len(events), serviceID)
	return events, nil
}

// 📈 DAILY SUMMARIES OPERATIONS

// StoreDailySummary stores a daily usage summary
func (r *AnalyticsRepository) StoreDailySummary(ctx context.Context, summary DailyUsageSummary) error {
	log.Printf("🗄️ Storing daily summary: %s", summary.ID)

	// Convert to DynamoDB item
	item, err := attributevalue.MarshalMap(summary)
	if err != nil {
		return fmt.Errorf("failed to marshal daily summary: %w", err)
	}

	// Add keys for efficient querying
	item["PK"] = &types.AttributeValueMemberS{Value: summary.UserID}
	item["SK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("SUMMARY#%s#%s", summary.Date, summary.ServiceID)}
	item["GSI1PK"] = &types.AttributeValueMemberS{Value: summary.ServiceID}
	item["GSI1SK"] = &types.AttributeValueMemberS{Value: summary.Date}

	// Store with upsert behavior
	_, err = r.dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.dailySummariesTable()),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to store daily summary: %w", err)
	}

	log.Printf("✅ Successfully stored daily summary: %s", summary.ID)
	return nil
}

// GetDailySummariesForUser retrieves daily summaries for a user
func (r *AnalyticsRepository) GetDailySummariesForUser(ctx context.Context, userID string, days int) ([]DailyUsageSummary, error) {
	log.Printf("🔍 Getting daily summaries for user %s (last %d days)", userID, days)

	// Calculate date range
	endDate := time.Now().UTC()
	startDate := endDate.AddDate(0, 0, -days)
	startDateStr := startDate.Format("2006-01-02")

	// Query DynamoDB
	result, err := r.dynamoClient.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.dailySummariesTable()),
		KeyConditionExpression: aws.String("PK = :userId AND SK >= :startDate"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":userId":    &types.AttributeValueMemberS{Value: userID},
			":startDate": &types.AttributeValueMemberS{Value: fmt.Sprintf("SUMMARY#%s", startDateStr)},
		},
		ScanIndexForward: aws.Bool(false), // Most recent first
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query daily summaries: %w", err)
	}

	// Convert to summaries
	var summaries []DailyUsageSummary
	err = attributevalue.UnmarshalListOfMaps(result.Items, &summaries)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal daily summaries: %w", err)
	}

	log.Printf("✅ Retrieved %d daily summaries for user %s", len(summaries), userID)
	return summaries, nil
}

// 👤 USER PROFILE OPERATIONS

// StoreUserProfile stores a user analytics profile
func (r *AnalyticsRepository) StoreUserProfile(ctx context.Context, profile UserAnalyticsProfile) error {
	log.Printf("🗄️ Storing user profile: %s", profile.UserID)

	// Convert to DynamoDB item
	item, err := attributevalue.MarshalMap(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal user profile: %w", err)
	}

	// Add keys
	item["PK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", profile.UserID)}
	item["SK"] = &types.AttributeValueMemberS{Value: "PROFILE"}

	// Store in DynamoDB
	_, err = r.dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.userProfilesTable()),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to store user profile: %w", err)
	}

	log.Printf("✅ Successfully stored user profile: %s", profile.UserID)
	return nil
}

// GetUserProfile retrieves a user analytics profile
func (r *AnalyticsRepository) GetUserProfile(ctx context.Context, userID string) (*UserAnalyticsProfile, error) {
	log.Printf("🔍 Getting user profile: %s", userID)

	// Query DynamoDB
	result, err := r.dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.userProfilesTable()),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			"SK": &types.AttributeValueMemberS{Value: "PROFILE"},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("user profile not found")
	}

	// Convert to profile
	var profile UserAnalyticsProfile
	err = attributevalue.UnmarshalMap(result.Item, &profile)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user profile: %w", err)
	}

	log.Printf("✅ Retrieved user profile: %s", userID)
	return &profile, nil
}

// 🔄 BATCH PROCESSING OPERATIONS

// GenerateDailySummariesFromEvents creates daily summaries from raw events
func (r *AnalyticsRepository) GenerateDailySummariesFromEvents(ctx context.Context, userID string, date string) error {
	log.Printf("📊 Generating daily summaries for user %s on %s", userID, date)

	// Parse date
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("invalid date format: %w", err)
	}

	// Get events for the day
	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)

	events, err := r.GetUsageEventsForUser(ctx, userID, startOfDay, endOfDay)
	if err != nil {
		return fmt.Errorf("failed to get events for summary generation: %w", err)
	}

	if len(events) == 0 {
		log.Printf("ℹ️ No events found for user %s on %s", userID, date)
		return nil
	}

	// Group events by service
	serviceEvents := make(map[string][]APIUsageEvent)
	for _, event := range events {
		serviceEvents[event.ServiceID] = append(serviceEvents[event.ServiceID], event)
	}

	// Generate summary for each service
	for serviceID, serviceEventList := range serviceEvents {
		summary := r.generateSummaryFromEvents(userID, serviceID, date, serviceEventList)
		
		// Store the summary
		err = r.StoreDailySummary(ctx, summary)
		if err != nil {
			log.Printf("⚠️ Failed to store summary for %s/%s: %v", serviceID, date, err)
			continue
		}
	}

	log.Printf("✅ Generated daily summaries for user %s on %s", userID, date)
	return nil
}

// generateSummaryFromEvents creates a DailyUsageSummary from a list of events
func (r *AnalyticsRepository) generateSummaryFromEvents(userID, serviceID, date string, events []APIUsageEvent) DailyUsageSummary {
	summary := DailyUsageSummary{
		ID:        fmt.Sprintf("%s_%s_%s", userID, serviceID, date),
		UserID:    userID,
		ServiceID: serviceID,
		Date:      date,
		LastUpdated: time.Now().UTC(),
		RecordCount: len(events),
	}

	if len(events) == 0 {
		return summary
	}

	// Calculate metrics
	var responseTimes []int64
	hourCounts := make(map[int]int)
	toolsUsed := make(map[string]bool)

	for _, event := range events {
		// Volume metrics
		summary.TotalRequests++
		if event.Success {
			summary.SuccessfulRequests++
		} else {
			summary.FailedRequests++
		}

		// Performance metrics  
		responseTimes = append(responseTimes, event.ResponseTime)

		// Cost metrics
		summary.TotalCost += event.EstimatedCost
		summary.TotalCreditsUsed += event.CreditsUsed
		summary.TotalTokensUsed += event.TokensUsed

		// Usage patterns
		hour := event.Timestamp.Hour()
		hourCounts[hour]++
		
		if event.ToolID != "" {
			toolsUsed[event.ToolID] = true
		}

		// Service mode distribution
		if event.ServiceMode == "premium" {
			summary.PremiumRequests++
		} else {
			summary.FreeRequests++
		}
		
		if event.FallbackUsed {
			summary.FallbackRequests++
		}
	}

	// Calculate derived metrics
	if summary.TotalRequests > 0 {
		summary.SuccessRate = float64(summary.SuccessfulRequests) / float64(summary.TotalRequests) * 100
		summary.AvgCostPerRequest = summary.TotalCost / float64(summary.TotalRequests)
	}

	// Response time statistics
	if len(responseTimes) > 0 {
		sort.Slice(responseTimes, func(i, j int) bool {
			return responseTimes[i] < responseTimes[j]
		})

		summary.MinResponseTime = responseTimes[0]
		summary.MaxResponseTime = responseTimes[len(responseTimes)-1]
		
		// Calculate average
		var total int64
		for _, rt := range responseTimes {
			total += rt
		}
		summary.AvgResponseTime = float64(total) / float64(len(responseTimes))

		// Calculate P95
		p95Index := int(float64(len(responseTimes)) * 0.95)
		if p95Index >= len(responseTimes) {
			p95Index = len(responseTimes) - 1
		}
		summary.P95ResponseTime = float64(responseTimes[p95Index])
	}

	// Find peak hour
	maxCount := 0
	for hour, count := range hourCounts {
		if count > maxCount {
			maxCount = count
			summary.PeakHour = hour
			summary.PeakRequestCount = count
		}
	}

	summary.UniqueToolsUsed = len(toolsUsed)

	return summary
}

// 🔧 ANALYTICS GENERATION

// GenerateUserAnalyticsProfile creates a comprehensive user profile from historical data
func (r *AnalyticsRepository) GenerateUserAnalyticsProfile(ctx context.Context, userID string) (*UserAnalyticsProfile, error) {
	log.Printf("👤 Generating analytics profile for user: %s", userID)

	// Get recent summaries (last 90 days)
	summaries, err := r.GetDailySummariesForUser(ctx, userID, 90)
	if err != nil {
		return nil, fmt.Errorf("failed to get summaries for profile generation: %w", err)
	}

	if len(summaries) == 0 {
		// Return basic profile for new user
		return &UserAnalyticsProfile{
			UserID:       userID,
			LastUpdated:  time.Now().UTC(),
			CalculatedAt: time.Now().UTC(),
		}, nil
	}

	profile := UserAnalyticsProfile{
		UserID:       userID,
		LastUpdated:  time.Now().UTC(),
		CalculatedAt: time.Now().UTC(),
	}

	// Analyze summaries
	serviceUsage := make(map[string]float64) // service -> total cost
	serviceRequests := make(map[string]int)  // service -> total requests
	hourUsage := make(map[int]int)           // hour -> request count
	var totalCost, totalRequests, totalResponseTime float64
	var totalSuccessful int
	var firstUsage, lastUsage time.Time
	usageDaysSet := make(map[string]bool)

	for _, summary := range summaries {
		// Cost analysis
		totalCost += summary.TotalCost
		serviceUsage[summary.ServiceID] += summary.TotalCost
		serviceRequests[summary.ServiceID] += summary.TotalRequests

		// Volume analysis
		totalRequests += float64(summary.TotalRequests)
		totalSuccessful += summary.SuccessfulRequests

		// Performance analysis
		totalResponseTime += summary.AvgResponseTime * float64(summary.TotalRequests)

		// Usage patterns
		hourUsage[summary.PeakHour] += summary.PeakRequestCount
		usageDaysSet[summary.Date] = true

		// Lifecycle tracking
		summaryDate, _ := time.Parse("2006-01-02", summary.Date)
		if firstUsage.IsZero() || summaryDate.Before(firstUsage) {
			firstUsage = summaryDate
		}
		if lastUsage.IsZero() || summaryDate.After(lastUsage) {
			lastUsage = summaryDate
		}
	}

	// Calculate profile metrics
	profile.TotalSpent = totalCost
	profile.UsageDays = len(usageDaysSet)
	profile.FirstUsage = firstUsage
	profile.LastUsage = lastUsage

	// Calculate success rate
	if totalRequests > 0 {
		profile.SuccessRate = float64(totalSuccessful) / totalRequests * 100
		profile.AvgResponseTime = totalResponseTime / totalRequests
	}

	// Determine favorite service (most used by cost)
	var maxCost float64
	for serviceID, cost := range serviceUsage {
		if cost > maxCost {
			maxCost = cost
			profile.FavoriteService = serviceID
		}
		profile.ActiveServices = append(profile.ActiveServices, serviceID)
	}

	// Calculate usage frequency
	avgDailyRequests := totalRequests / float64(len(usageDaysSet))
	if avgDailyRequests < 5 {
		profile.UsageFrequency = "light"
	} else if avgDailyRequests < 20 {
		profile.UsageFrequency = "moderate"
	} else {
		profile.UsageFrequency = "heavy"
	}

	// Calculate preferred usage hours
	var topHours []int
	for hour, count := range hourUsage {
		if count > 0 {
			topHours = append(topHours, hour)
		}
	}
	// Sort by usage count (simplified)
	profile.PreferredUsageHours = topHours[:min(len(topHours), 8)] // Top 8 hours

	// Calculate spending trends (simplified)
	recentCost := 0.0
	recentDays := min(len(summaries), 7) // Last 7 days
	for i := 0; i < recentDays; i++ {
		recentCost += summaries[i].TotalCost
	}
	
	profile.MonthlySpend = recentCost * 30 / float64(recentDays) // Extrapolate
	profile.AvgMonthlySpend = totalCost * 30 / float64(len(usageDaysSet))

	if profile.MonthlySpend > profile.AvgMonthlySpend*1.1 {
		profile.CostTrend = "increasing"
	} else if profile.MonthlySpend < profile.AvgMonthlySpend*0.9 {
		profile.CostTrend = "decreasing"
	} else {
		profile.CostTrend = "stable"
	}

	// Calculate retention score (simplified)
	daysSinceFirst := time.Since(firstUsage).Hours() / 24
	if daysSinceFirst > 0 {
		profile.RetentionScore = math.Min(100, float64(len(usageDaysSet))/daysSinceFirst*100)
	}

	log.Printf("✅ Generated analytics profile for user %s", userID)
	return &profile, nil
}

// 🛠️ UTILITY FUNCTIONS

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// 📊 BATCH OPERATIONS

// BatchProcessDailySummaries processes daily summaries for multiple users
func (r *AnalyticsRepository) BatchProcessDailySummaries(ctx context.Context, date string) error {
	log.Printf("🔄 Batch processing daily summaries for date: %s", date)

	// This would typically be triggered by a scheduled job
	// For now, this is a placeholder that would:
	// 1. Get list of active users
	// 2. Generate summaries for each user
	// 3. Update user profiles

	// TODO: Implement user enumeration and batch processing
	log.Printf("✅ Batch processing completed for date: %s", date)
	return nil
}