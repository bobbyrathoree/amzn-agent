// 💰 SERVICE COST CALCULATORS
// Intelligent cost calculation for each API service

package services

import (
	"log"
	"math"
)

// 🤖 OPENAI COST CALCULATOR
type OpenAICostCalculator struct{}

func (c *OpenAICostCalculator) GetServiceID() string {
	return "openai"
}

func (c *OpenAICostCalculator) CalculateCost(usage UsageMetrics) CostBreakdown {
	// OpenAI pricing (as of Dec 2024):
	// GPT-4: $0.03/1K input tokens, $0.06/1K output tokens
	// GPT-3.5-turbo: $0.001/1K input tokens, $0.002/1K output tokens
	
	var baseCost float64 = 0.5 // Base cost per request (cents)
	var tokenCost float64 = 0
	
	if usage.Tokens > 0 {
		// Assume average mix of input/output tokens and GPT-3.5-turbo
		avgCostPerToken := 0.0015 / 1000 // $0.0015 per 1K tokens average
		tokenCost = float64(usage.Tokens) * avgCostPerToken * 100 // Convert to cents
	}
	
	totalCost := baseCost + tokenCost
	
	return CostBreakdown{
		BaseCost:   baseCost,
		VolumeCost: tokenCost,
		TotalCost:  totalCost,
		Confidence: 0.85, // High confidence for well-documented pricing
	}
}

// 🧠 ANTHROPIC COST CALCULATOR  
type AnthropicCostCalculator struct{}

func (c *AnthropicCostCalculator) GetServiceID() string {
	return "anthropic"
}

func (c *AnthropicCostCalculator) CalculateCost(usage UsageMetrics) CostBreakdown {
	// Anthropic Claude pricing:
	// Claude-3 Haiku: $0.00025/1K input, $0.00125/1K output
	// Claude-3 Sonnet: $0.003/1K input, $0.015/1K output
	
	var baseCost float64 = 0.3 // Base cost per request (cents)
	var tokenCost float64 = 0
	
	if usage.Tokens > 0 {
		// Assume average mix and Claude-3 Haiku pricing
		avgCostPerToken := 0.000675 / 1000 // Average of input/output costs
		tokenCost = float64(usage.Tokens) * avgCostPerToken * 100 // Convert to cents
	}
	
	totalCost := baseCost + tokenCost
	
	return CostBreakdown{
		BaseCost:   baseCost,
		VolumeCost: tokenCost,
		TotalCost:  totalCost,
		Confidence: 0.85,
	}
}

// 🔍 GOOGLE SEARCH COST CALCULATOR
type GoogleSearchCostCalculator struct{}

func (c *GoogleSearchCostCalculator) GetServiceID() string {
	return "google-search"
}

func (c *GoogleSearchCostCalculator) CalculateCost(usage UsageMetrics) CostBreakdown {
	// Google Custom Search API:
	// Free: 100 queries/day
	// Paid: $5 per 1000 queries after free tier
	
	var baseCost float64 = 0.5 // 0.5 cents per query ($5/1000)
	var volumeCost float64 = 0
	
	// Add small volume discount for bulk usage
	if usage.Results > 10 {
		volumeCost = 0.1 // Small premium for more results
	}
	
	totalCost := baseCost + volumeCost
	
	return CostBreakdown{
		BaseCost:   baseCost,
		VolumeCost: volumeCost,
		TotalCost:  totalCost,
		Confidence: 0.9, // Very high confidence - well documented
	}
}

// 🌤️ WEATHER API COST CALCULATOR
type WeatherAPICostCalculator struct{}

func (c *WeatherAPICostCalculator) GetServiceID() string {
	return "weather-api"
}

func (c *WeatherAPICostCalculator) CalculateCost(usage UsageMetrics) CostBreakdown {
	// OpenWeatherMap pricing:
	// Free: 1000 calls/day
	// Paid plans start at $40/month for 100K calls
	
	var baseCost float64 = 0.04 // Roughly $40/100K = 0.04 cents per call
	
	return CostBreakdown{
		BaseCost:   baseCost,
		VolumeCost: 0,
		TotalCost:  baseCost,
		Confidence: 0.8,
	}
}

// 📰 NEWS API COST CALCULATOR
type NewsAPICostCalculator struct{}

func (c *NewsAPICostCalculator) GetServiceID() string {
	return "news-api"
}

func (c *NewsAPICostCalculator) CalculateCost(usage UsageMetrics) CostBreakdown {
	// News API pricing:
	// Free: 1000 requests/month
	// Developer: $449/month for 250K requests
	
	var baseCost float64 = 0.18 // $449/250K = ~0.18 cents per request
	
	return CostBreakdown{
		BaseCost:   baseCost,
		VolumeCost: 0,
		TotalCost:  baseCost,
		Confidence: 0.75,
	}
}

// 🐱 GITHUB COST CALCULATOR
type GitHubCostCalculator struct{}

func (c *GitHubCostCalculator) GetServiceID() string {
	return "github"
}

func (c *GitHubCostCalculator) CalculateCost(usage UsageMetrics) CostBreakdown {
	// GitHub API is generally free with rate limits
	// Enterprise pricing exists but most usage is free
	
	var baseCost float64 = 0.01 // Minimal cost for tracking
	
	return CostBreakdown{
		BaseCost:   baseCost,
		VolumeCost: 0,
		TotalCost:  baseCost,
		Confidence: 0.5, // Lower confidence due to mostly free usage
	}
}

// 💬 SLACK COST CALCULATOR
type SlackCostCalculator struct{}

func (c *SlackCostCalculator) GetServiceID() string {
	return "slack"
}

func (c *SlackCostCalculator) CalculateCost(usage UsageMetrics) CostBreakdown {
	// Slack API calls are generally free
	// Cost is usually in the Slack subscription itself
	
	var baseCost float64 = 0.02 // Minimal tracking cost
	
	return CostBreakdown{
		BaseCost:   baseCost,
		VolumeCost: 0,
		TotalCost:  baseCost,
		Confidence: 0.4, // Low confidence - mostly free
	}
}

// 🏢 HUBSPOT COST CALCULATOR
type HubSpotCostCalculator struct{}

func (c *HubSpotCostCalculator) GetServiceID() string {
	return "hubspot"
}

func (c *HubSpotCostCalculator) CalculateCost(usage UsageMetrics) CostBreakdown {
	// HubSpot API limits based on subscription tier
	// Free tier: 100 requests per 10 seconds
	// Paid tiers: Higher limits
	
	var baseCost float64 = 0.05 // Estimated cost per API call
	
	return CostBreakdown{
		BaseCost:   baseCost,
		VolumeCost: 0,
		TotalCost:  baseCost,
		Confidence: 0.6,
	}
}

// 🔎 TAVILY COST CALCULATOR
type TavilyCostCalculator struct{}

func (c *TavilyCostCalculator) GetServiceID() string {
	return "tavily"
}

func (c *TavilyCostCalculator) CalculateCost(usage UsageMetrics) CostBreakdown {
	// Tavily pricing:
	// Free: 1000 requests/month
	// Pro: $20/month for 10K requests
	
	var baseCost float64 = 0.2 // $20/10K = 0.2 cents per request
	var volumeCost float64 = 0
	
	// Add premium for more results
	if usage.Results > 5 {
		volumeCost = 0.05 * float64(usage.Results-5)
	}
	
	totalCost := baseCost + volumeCost
	
	return CostBreakdown{
		BaseCost:   baseCost,
		VolumeCost: volumeCost,
		TotalCost:  totalCost,
		Confidence: 0.8,
	}
}

// 🐍 SERPAPI COST CALCULATOR
type SerpAPICostCalculator struct{}

func (c *SerpAPICostCalculator) GetServiceID() string {
	return "serpapi"
}

func (c *SerpAPICostCalculator) CalculateCost(usage UsageMetrics) CostBreakdown {
	// SerpAPI pricing:
	// Free: 100 searches/month
	// Paid plans start at $50/month for 5K searches
	
	var baseCost float64 = 1.0 // $50/5K = 1 cent per search
	var volumeCost float64 = 0
	
	// Volume discount for bulk usage
	if usage.Requests > 100 {
		volumeCost = -0.1 // Small discount
	}
	
	totalCost := math.Max(baseCost + volumeCost, 0.5) // Minimum 0.5 cents
	
	return CostBreakdown{
		BaseCost:   baseCost,
		VolumeCost: volumeCost,
		TotalCost:  totalCost,
		Confidence: 0.85,
	}
}

// 🧮 COST CALCULATION UTILITIES

// GetAllCostCalculators returns all available cost calculators
func GetAllCostCalculators() map[string]ServiceCostCalculator {
	calculators := make(map[string]ServiceCostCalculator)
	
	calculators["openai"] = &OpenAICostCalculator{}
	calculators["anthropic"] = &AnthropicCostCalculator{}
	calculators["google-search"] = &GoogleSearchCostCalculator{}
	calculators["weather-api"] = &WeatherAPICostCalculator{}
	calculators["news-api"] = &NewsAPICostCalculator{}
	calculators["github"] = &GitHubCostCalculator{}
	calculators["slack"] = &SlackCostCalculator{}
	calculators["hubspot"] = &HubSpotCostCalculator{}
	calculators["tavily"] = &TavilyCostCalculator{}
	calculators["serpapi"] = &SerpAPICostCalculator{}
	
	log.Printf("💰 Initialized %d cost calculators", len(calculators))
	return calculators
}

// EstimateMonthlySpend calculates projected monthly spending based on usage patterns
func EstimateMonthlySpend(dailyUsage []DailyUsageSummary) float64 {
	if len(dailyUsage) == 0 {
		return 0
	}
	
	// Calculate average daily spend
	totalSpend := 0.0
	for _, day := range dailyUsage {
		totalSpend += day.TotalCost
	}
	
	avgDailySpend := totalSpend / float64(len(dailyUsage))
	
	// Project to monthly (30 days)
	monthlyEstimate := avgDailySpend * 30
	
	// Add growth factor for trending usage
	growthFactor := calculateGrowthFactor(dailyUsage)
	adjustedEstimate := monthlyEstimate * growthFactor
	
	log.Printf("📊 Monthly spend estimate: $%.2f (growth factor: %.2f)", adjustedEstimate/100, growthFactor)
	
	return adjustedEstimate
}

// calculateGrowthFactor analyzes usage trend and returns a growth multiplier
func calculateGrowthFactor(dailyUsage []DailyUsageSummary) float64 {
	if len(dailyUsage) < 7 {
		return 1.0 // Not enough data for trend analysis
	}
	
	// Compare first week vs last week
	firstWeekSpend := 0.0
	lastWeekSpend := 0.0
	
	weekSize := len(dailyUsage) / 2
	
	for i := 0; i < weekSize; i++ {
		firstWeekSpend += dailyUsage[i].TotalCost
	}
	
	for i := len(dailyUsage) - weekSize; i < len(dailyUsage); i++ {
		lastWeekSpend += dailyUsage[i].TotalCost
	}
	
	if firstWeekSpend == 0 {
		return 1.2 // Slight growth assumption for new users
	}
	
	growthRatio := lastWeekSpend / firstWeekSpend
	
	// Cap growth factor between 0.5x and 2.0x
	growthFactor := math.Max(0.5, math.Min(2.0, growthRatio))
	
	return growthFactor
}

// CalculateCostEfficiency compares actual vs estimated costs
func CalculateCostEfficiency(actualCost, estimatedCost float64) float64 {
	if estimatedCost == 0 {
		return 1.0
	}
	
	accuracy := 1.0 - math.Abs(actualCost-estimatedCost)/estimatedCost
	return math.Max(0.0, math.Min(1.0, accuracy))
}

// GetServiceCostRanking ranks services by cost-effectiveness 
func GetServiceCostRanking(summaries []DailyUsageSummary) []ServiceRanking {
	serviceStats := make(map[string]*ServiceStats)
	
	// Aggregate stats by service
	for _, summary := range summaries {
		stats, exists := serviceStats[summary.ServiceID]
		if !exists {
			stats = &ServiceStats{
				ServiceID: summary.ServiceID,
			}
			serviceStats[summary.ServiceID] = stats
		}
		
		stats.TotalRequests += summary.TotalRequests
		stats.TotalCost += summary.TotalCost
		stats.TotalResponseTime += summary.AvgResponseTime * float64(summary.TotalRequests)
		stats.SuccessfulRequests += summary.SuccessfulRequests
	}
	
	// Calculate rankings
	var rankings []ServiceRanking
	for _, stats := range serviceStats {
		if stats.TotalRequests == 0 {
			continue
		}
		
		avgCostPerRequest := stats.TotalCost / float64(stats.TotalRequests)
		avgResponseTime := stats.TotalResponseTime / float64(stats.TotalRequests)
		successRate := float64(stats.SuccessfulRequests) / float64(stats.TotalRequests)
		
		// Calculate efficiency score (lower cost + faster response + higher success = better)
		efficiencyScore := (successRate * 100) - (avgCostPerRequest * 10) - (avgResponseTime / 100)
		
		rankings = append(rankings, ServiceRanking{
			ServiceID:         stats.ServiceID,
			EfficiencyScore:   efficiencyScore,
			AvgCostPerRequest: avgCostPerRequest,
			AvgResponseTime:   avgResponseTime,
			SuccessRate:       successRate,
			TotalRequests:     stats.TotalRequests,
		})
	}
	
	// Sort by efficiency score (descending)
	for i := 0; i < len(rankings)-1; i++ {
		for j := i + 1; j < len(rankings); j++ {
			if rankings[i].EfficiencyScore < rankings[j].EfficiencyScore {
				rankings[i], rankings[j] = rankings[j], rankings[i]
			}
		}
	}
	
	return rankings
}

type ServiceStats struct {
	ServiceID          string
	TotalRequests      int
	SuccessfulRequests int
	TotalCost          float64
	TotalResponseTime  float64
}

type ServiceRanking struct {
	ServiceID         string  `json:"serviceId"`
	EfficiencyScore   float64 `json:"efficiencyScore"`
	AvgCostPerRequest float64 `json:"avgCostPerRequest"`
	AvgResponseTime   float64 `json:"avgResponseTime"`
	SuccessRate       float64 `json:"successRate"`
	TotalRequests     int     `json:"totalRequests"`
}