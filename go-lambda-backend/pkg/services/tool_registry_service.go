package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
)

// ToolRegistryService manages the registry of available tools
type ToolRegistryService struct {
	tools     map[string]*models.Tool
	entries   map[string]*models.ToolRegistryEntry
	mutex     sync.RWMutex
	lastSync  time.Time
}

// NewToolRegistryService creates a new tool registry service
func NewToolRegistryService() *ToolRegistryService {
	service := &ToolRegistryService{
		tools:   make(map[string]*models.Tool),
		entries: make(map[string]*models.ToolRegistryEntry),
	}
	
	// Initialize with built-in tools
	service.initializeBuiltinTools()
	
	return service
}

// GetTool retrieves a tool by ID
func (s *ToolRegistryService) GetTool(toolID string) (*models.Tool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	entry, exists := s.entries[toolID]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", toolID)
	}
	
	if !entry.Enabled {
		return nil, fmt.Errorf("tool %s is disabled", toolID)
	}
	
	return &entry.Tool, nil
}

// GetAllTools returns all available tools
func (s *ToolRegistryService) GetAllTools() ([]*models.Tool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	var tools []*models.Tool
	for _, entry := range s.entries {
		if entry.Enabled {
			tools = append(tools, &entry.Tool)
		}
	}
	
	return tools, nil
}

// GetToolsByCategory returns tools filtered by category
func (s *ToolRegistryService) GetToolsByCategory(category string) ([]*models.Tool, error) {
	allTools, err := s.GetAllTools()
	if err != nil {
		return nil, err
	}
	
	var filtered []*models.Tool
	for _, tool := range allTools {
		if tool.Category == category {
			filtered = append(filtered, tool)
		}
	}
	
	return filtered, nil
}

// RegisterTool adds a new tool to the registry
func (s *ToolRegistryService) RegisterTool(tool *models.Tool) error {
	if err := tool.Validate(); err != nil {
		return fmt.Errorf("invalid tool: %w", err)
	}
	
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Check if tool already exists
	if _, exists := s.entries[tool.ID]; exists {
		return fmt.Errorf("tool %s already registered", tool.ID)
	}
	
	// Create registry entry
	entry := &models.ToolRegistryEntry{
		Tool:       *tool,
		Enabled:    true,
		UsageCount: 0,
	}
	
	s.tools[tool.ID] = tool
	s.entries[tool.ID] = entry
	
	return nil
}

// UpdateToolUsage increments usage count for a tool
func (s *ToolRegistryService) UpdateToolUsage(toolID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if entry, exists := s.entries[toolID]; exists {
		entry.UsageCount++
		entry.LastUsed = time.Now()
	}
}

// EnableTool enables a tool
func (s *ToolRegistryService) EnableTool(toolID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	entry, exists := s.entries[toolID]
	if !exists {
		return fmt.Errorf("tool %s not found", toolID)
	}
	
	entry.Enabled = true
	return nil
}

// DisableTool disables a tool
func (s *ToolRegistryService) DisableTool(toolID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	entry, exists := s.entries[toolID]
	if !exists {
		return fmt.Errorf("tool %s not found", toolID)
	}
	
	entry.Enabled = false
	return nil
}

// GetToolStats returns usage statistics for all tools
func (s *ToolRegistryService) GetToolStats() map[string]models.ToolRegistryEntry {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	stats := make(map[string]models.ToolRegistryEntry)
	for id, entry := range s.entries {
		stats[id] = *entry
	}
	
	return stats
}

// initializeBuiltinTools registers the built-in tools
func (s *ToolRegistryService) initializeBuiltinTools() {
	// Web Search Tool
	webSearchTool := &models.Tool{
		ID:          "web-search",
		Name:        "Advanced Web Search",
		Description: "Search the internet for current information using multiple search engines",
		Category:    models.ToolCategoryInformation,
		Version:     "1.0.0",
		Author:      "Foundry Team",
		Icon:        "🔍",
		Featured:    true,
		Tags:        []string{"search", "web", "information", "research"},
		Keywords:    []string{"search", "google", "web", "internet", "find", "lookup"},
		Capabilities: []models.ToolCapability{
			{
				Name:        "search",
				Description: "Search the web for information",
				Parameters: []models.ParameterDef{
					{
						Name:        "query",
						Type:        "string",
						Required:    true,
						Description: "Search query to execute",
					},
					{
						Name:        "engine",
						Type:        "string",
						Required:    false,
						Description: "Search engine to use (auto, google, tavily, serp-api, duckduckgo)",
						Default:     "auto",
						Enum:        []string{"auto", "google", "tavily", "serp-api", "duckduckgo"},
					},
					{
						Name:        "limit",
						Type:        "number",
						Required:    false,
						Description: "Maximum number of results to return",
						Default:     10,
					},
				},
				Examples: []string{
					"latest AI news",
					"weather in San Francisco",
					"best restaurants in Tokyo",
				},
				TimeEstimate: intPtr(5000), // 5 seconds
				CostEstimate: &models.CostEstimate{
					Credits:     1,
					Description: "1 credit per search",
				},
			},
		},
		APIRequirements: map[string]models.APIRequirement{
			"google-search": {
				Required:    false,
				KeyName:     "Google Custom Search API Key",
				Description: "Enables high-quality Google search results",
				SignupURL:   "https://developers.google.com/custom-search/v1/introduction",
				PricingInfo: "Free: 100 queries/day, Paid: $5 per 1,000 queries",
				SetupInstructions: []string{
					"Go to Google Cloud Console",
					"Enable Custom Search API",
					"Create credentials (API Key)",
					"Copy the API key",
				},
			},
			"tavily": {
				Required:    false,
				KeyName:     "Tavily Search API Key",
				Description: "AI-optimized search engine for agents",
				SignupURL:   "https://tavily.com",
				PricingInfo: "Free: 1000/month, Pro: $20/month",
			},
			"serp-api": {
				Required:    false,
				KeyName:     "SerpAPI Key",
				Description: "Google/Bing search results API",
				SignupURL:   "https://serpapi.com",
				PricingInfo: "Free: 100/month, Paid: $50/month",
			},
		},
		Examples: []models.ToolExample{
			{
				Input:  `{"query": "latest AI trends 2024", "engine": "auto", "limit": 5}`,
				Output: "Returns 5 search results about latest AI trends",
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Research Assistant Tool
	researchTool := &models.Tool{
		ID:          "research-assistant",
		Name:        "AI Research Assistant",
		Description: "Conduct comprehensive research on any topic with analysis and synthesis",
		Category:    models.ToolCategoryResearch,
		Version:     "1.0.0",
		Author:      "Foundry Team",
		Icon:        "🔬",
		Featured:    true,
		Tags:        []string{"research", "analysis", "synthesis", "academic"},
		Keywords:    []string{"research", "analyze", "study", "investigate", "report"},
		Capabilities: []models.ToolCapability{
			{
				Name:        "comprehensive-research",
				Description: "Conduct multi-source research with analysis and synthesis",
				Parameters: []models.ParameterDef{
					{
						Name:        "topic",
						Type:        "string",
						Required:    true,
						Description: "Research topic or question",
					},
					{
						Name:        "depth",
						Type:        "string",
						Required:    false,
						Description: "Research depth level",
						Default:     "detailed",
						Enum:        []string{"surface", "detailed", "comprehensive"},
					},
					{
						Name:        "sources",
						Type:        "array",
						Required:    false,
						Description: "Preferred source types",
					},
				},
				Examples: []string{
					"machine learning applications in healthcare",
					"renewable energy trends 2024",
					"impact of remote work on productivity",
				},
				TimeEstimate: intPtr(30000), // 30 seconds
				CostEstimate: &models.CostEstimate{
					Credits:     5,
					Description: "5 credits per comprehensive research",
				},
			},
		},
		APIRequirements: map[string]models.APIRequirement{
			"tavily": {
				Required:    false,
				KeyName:     "Tavily Search API Key",
				Description: "Enhanced research capabilities with AI-optimized search",
				SignupURL:   "https://tavily.com",
			},
		},
		Examples: []models.ToolExample{
			{
				Input:  `{"topic": "quantum computing breakthroughs", "depth": "comprehensive"}`,
				Output: "Comprehensive research report on quantum computing with key findings, trends, and analysis",
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Register built-in tools
	s.RegisterTool(webSearchTool)
	s.RegisterTool(researchTool)
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}