package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
)

// Test Advanced Agent Tools System
func TestAgentTool(t *testing.T) {
	t.Run("AgentTool_Creation", func(t *testing.T) {
		t.Run("Valid_Plain_Tool", func(t *testing.T) {
			tool := models.AgentTool{
				Type:        "plain",
				Name:        "calculator",
				Description: "Perform mathematical calculations",
				Config:      map[string]interface{}{},
			}

			assert.Equal(t, "plain", tool.Type)
			assert.Equal(t, "calculator", tool.Name)
			assert.Equal(t, "Perform mathematical calculations", tool.Description)
		})

		t.Run("Valid_Internet_Tool", func(t *testing.T) {
			tool := models.AgentTool{
				Type:        "internet",
				Name:        "web_search",
				Description: "Search the internet for information",
				Config: map[string]interface{}{
					"searchEngine": "tavily",
					"maxResults":   10,
					"apiKey":       "secret-key",
				},
			}

			assert.True(t, tool.IsValid())
			assert.Equal(t, "internet", tool.Type)
			assert.Equal(t, "web_search", tool.Name)
		})

		t.Run("Valid_Bedrock_Agent_Tool", func(t *testing.T) {
			tool := models.AgentTool{
				Type:        "bedrock_agent",
				Name:        "aws_service",
				Description: "AWS Bedrock agent integration",
				Config: map[string]interface{}{
					"agentId":    "AGENT123456",
					"agentAlias": "production",
					"region":     "us-east-1",
				},
			}

			assert.True(t, tool.IsValid())
			assert.Equal(t, "bedrock_agent", tool.Type)
			assert.Equal(t, "aws_service", tool.Name)
		})

		t.Run("Invalid_Tool_Type", func(t *testing.T) {
			tool := models.AgentTool{
				Type:        "invalid_type",
				Name:        "test_tool",
				Description: "Test description",
				Config:      map[string]interface{}{},
			}

			assert.False(t, tool.IsValid())
		})

		t.Run("Missing_Required_Fields", func(t *testing.T) {
			invalidTools := []models.AgentTool{
				{Type: "", Name: "calculator", Description: "Math tool"},                    // Missing type
				{Type: "plain", Name: "", Description: "Math tool"},                         // Missing name
				{Type: "plain", Name: "calculator", Description: ""},                        // Missing description
				{Type: "plain", Name: "calculator", Description: "Math tool", Config: nil}, // Nil config (should default to empty map)
			}

			for i, tool := range invalidTools[:3] { // First 3 are truly invalid
				assert.False(t, tool.IsValid(), "Tool %d should be invalid", i)
			}

			// Test nil config (should be valid - defaults to empty map)
			assert.True(t, invalidTools[3].IsValid(), "Tool with nil config should be valid")
		})
	})

	t.Run("AgentTool_JSON_Marshaling", func(t *testing.T) {
		tool := models.AgentTool{
			Type:        "internet",
			Name:        "web_search",
			Description: "Search the web for information",
			Config: map[string]interface{}{
				"searchEngine": "tavily",
				"maxResults":   20,
				"apiKey":       "secret-api-key",
			},
		}

		// Test JSON marshaling
		jsonData, err := json.Marshal(tool)
		assert.NoError(t, err)

		// Test JSON unmarshaling
		var unmarshaled models.AgentTool
		err = json.Unmarshal(jsonData, &unmarshaled)
		assert.NoError(t, err)

		assert.Equal(t, tool.Type, unmarshaled.Type)
		assert.Equal(t, tool.Name, unmarshaled.Name)
		assert.Equal(t, tool.Description, unmarshaled.Description)
		assert.Equal(t, "tavily", unmarshaled.Config["searchEngine"])
		assert.Equal(t, float64(20), unmarshaled.Config["maxResults"]) // JSON numbers become float64
		assert.Equal(t, "secret-api-key", unmarshaled.Config["apiKey"])
	})

	t.Run("AgentTool_Config_Type_Checking", func(t *testing.T) {
		t.Run("Internet_Tool_Config", func(t *testing.T) {
			tool := models.AgentTool{
				Type:        "internet",
				Name:        "web_search",
				Description: "Web search tool",
				Config: map[string]interface{}{
					"searchEngine": "duckduckgo",
					"maxResults":   25,
					"apiKey":       "optional-key",
				},
			}

			// Test config extraction
			searchEngine, ok := tool.Config["searchEngine"].(string)
			assert.True(t, ok)
			assert.Equal(t, "duckduckgo", searchEngine)

			maxResults, ok := tool.Config["maxResults"].(int)
			assert.True(t, ok)
			assert.Equal(t, 25, maxResults)

			apiKey, ok := tool.Config["apiKey"].(string)
			assert.True(t, ok)
			assert.Equal(t, "optional-key", apiKey)
		})

		t.Run("Bedrock_Agent_Config", func(t *testing.T) {
			tool := models.AgentTool{
				Type:        "bedrock_agent",
				Name:        "aws_agent",
				Description: "AWS Bedrock agent tool",
				Config: map[string]interface{}{
					"agentId":    "AGENT789",
					"agentAlias": "staging",
					"region":     "us-west-2",
				},
			}

			agentId, ok := tool.Config["agentId"].(string)
			assert.True(t, ok)
			assert.Equal(t, "AGENT789", agentId)

			agentAlias, ok := tool.Config["agentAlias"].(string)
			assert.True(t, ok)
			assert.Equal(t, "staging", agentAlias)

			region, ok := tool.Config["region"].(string)
			assert.True(t, ok)
			assert.Equal(t, "us-west-2", region)
		})
	})
}

// Test Specific Tool Type Models
func TestPlainTool(t *testing.T) {
	t.Run("PlainTool_Creation", func(t *testing.T) {
		tool := models.PlainTool{
			Name:        "calculator",
			Description: "Basic mathematical operations",
		}

		assert.Equal(t, "calculator", tool.Name)
		assert.Equal(t, "Basic mathematical operations", tool.Description)
	})

	t.Run("PlainTool_ToAgentTool", func(t *testing.T) {
		plainTool := models.PlainTool{
			Name:        "code_interpreter",
			Description: "Execute and analyze code",
		}

		agentTool := plainTool.ToAgentTool()

		assert.Equal(t, "plain", agentTool.Type)
		assert.Equal(t, plainTool.Name, agentTool.Name)
		assert.Equal(t, plainTool.Description, agentTool.Description)
		assert.NotNil(t, agentTool.Config)
		assert.Empty(t, agentTool.Config)
	})

	t.Run("PlainTool_JSON_Marshaling", func(t *testing.T) {
		tool := models.PlainTool{
			Name:        "file_analysis",
			Description: "Analyze file contents and structure",
		}

		jsonData, err := json.Marshal(tool)
		assert.NoError(t, err)

		var unmarshaled models.PlainTool
		err = json.Unmarshal(jsonData, &unmarshaled)
		assert.NoError(t, err)

		assert.Equal(t, tool.Name, unmarshaled.Name)
		assert.Equal(t, tool.Description, unmarshaled.Description)
	})
}

func TestInternetTool(t *testing.T) {
	t.Run("InternetTool_Creation", func(t *testing.T) {
		tool := models.InternetTool{
			Name:         "web_search",
			Description:  "Search the internet for information",
			SearchEngine: "tavily",
			APIKey:       "secret-key",
			MaxResults:   15,
		}

		assert.Equal(t, "web_search", tool.Name)
		assert.Equal(t, "tavily", tool.SearchEngine)
		assert.Equal(t, 15, tool.MaxResults)
	})

	t.Run("InternetTool_Validation", func(t *testing.T) {
		t.Run("Valid_Search_Engines", func(t *testing.T) {
			validEngines := []string{"tavily", "duckduckgo", "google"}

			for _, engine := range validEngines {
				tool := models.InternetTool{
					Name:         "web_search",
					Description:  "Web search",
					SearchEngine: engine,
					MaxResults:   10,
				}

				assert.True(t, tool.IsValid(), "Engine %s should be valid", engine)
			}
		})

		t.Run("Invalid_Search_Engine", func(t *testing.T) {
			tool := models.InternetTool{
				Name:         "web_search",
				Description:  "Web search",
				SearchEngine: "bing", // Not in allowed list
				MaxResults:   10,
			}

			assert.False(t, tool.IsValid())
		})

		t.Run("MaxResults_Validation", func(t *testing.T) {
			// Valid range: 1-100
			validTool := models.InternetTool{
				Name:         "web_search",
				Description:  "Web search",
				SearchEngine: "tavily",
				MaxResults:   50,
			}
			assert.True(t, validTool.IsValid())

			// Invalid: too low
			invalidTool1 := models.InternetTool{
				Name:         "web_search",
				Description:  "Web search",
				SearchEngine: "tavily",
				MaxResults:   0,
			}
			assert.False(t, invalidTool1.IsValid())

			// Invalid: too high
			invalidTool2 := models.InternetTool{
				Name:         "web_search",
				Description:  "Web search",
				SearchEngine: "tavily",
				MaxResults:   150,
			}
			assert.False(t, invalidTool2.IsValid())
		})
	})

	t.Run("InternetTool_ToAgentTool", func(t *testing.T) {
		internetTool := models.InternetTool{
			Name:         "web_search",
			Description:  "Advanced web search capabilities",
			SearchEngine: "google",
			APIKey:       "google-api-key",
			MaxResults:   25,
		}

		agentTool := internetTool.ToAgentTool()

		assert.Equal(t, "internet", agentTool.Type)
		assert.Equal(t, internetTool.Name, agentTool.Name)
		assert.Equal(t, internetTool.Description, agentTool.Description)

		// Verify config mapping
		assert.Equal(t, "google", agentTool.Config["searchEngine"])
		assert.Equal(t, "google-api-key", agentTool.Config["apiKey"])
		assert.Equal(t, 25, agentTool.Config["maxResults"])
	})
}

func TestBedrockAgentTool(t *testing.T) {
	t.Run("BedrockAgentTool_Creation", func(t *testing.T) {
		tool := models.BedrockAgentTool{
			Name:        "aws_service",
			Description: "AWS Bedrock agent integration",
			AgentID:     "AGENT123456789",
			AgentAlias:  "production",
			Region:      "us-east-1",
		}

		assert.Equal(t, "aws_service", tool.Name)
		assert.Equal(t, "AGENT123456789", tool.AgentID)
		assert.Equal(t, "production", tool.AgentAlias)
		assert.Equal(t, "us-east-1", tool.Region)
	})

	t.Run("BedrockAgentTool_Validation", func(t *testing.T) {
		t.Run("Valid_Tool", func(t *testing.T) {
			tool := models.BedrockAgentTool{
				Name:        "bedrock_agent",
				Description: "AWS Bedrock agent tool",
				AgentID:     "AGENT123",
				AgentAlias:  "staging",
				Region:      "us-west-2",
			}

			assert.True(t, tool.IsValid())
		})

		t.Run("Missing_Required_Fields", func(t *testing.T) {
			invalidTools := []models.BedrockAgentTool{
				{Name: "agent", Description: "desc", AgentID: "", AgentAlias: "alias", Region: "us-east-1"},    // Missing AgentID
				{Name: "agent", Description: "desc", AgentID: "AGENT123", AgentAlias: "", Region: "us-east-1"}, // Missing AgentAlias
				{Name: "agent", Description: "desc", AgentID: "AGENT123", AgentAlias: "alias", Region: ""},     // Missing Region
			}

			for i, tool := range invalidTools {
				assert.False(t, tool.IsValid(), "Tool %d should be invalid", i)
			}
		})

		t.Run("Valid_AWS_Regions", func(t *testing.T) {
			validRegions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}

			for _, region := range validRegions {
				tool := models.BedrockAgentTool{
					Name:        "bedrock_agent",
					Description: "AWS agent",
					AgentID:     "AGENT123",
					AgentAlias:  "production",
					Region:      region,
				}

				assert.True(t, tool.IsValid(), "Region %s should be valid", region)
			}
		})
	})

	t.Run("BedrockAgentTool_ToAgentTool", func(t *testing.T) {
		bedrockTool := models.BedrockAgentTool{
			Name:        "data_processor",
			Description: "Process data using AWS Bedrock agent",
			AgentID:     "AGENT987654321",
			AgentAlias:  "v2",
			Region:      "eu-central-1",
		}

		agentTool := bedrockTool.ToAgentTool()

		assert.Equal(t, "bedrock_agent", agentTool.Type)
		assert.Equal(t, bedrockTool.Name, agentTool.Name)
		assert.Equal(t, bedrockTool.Description, agentTool.Description)

		// Verify config mapping
		assert.Equal(t, "AGENT987654321", agentTool.Config["agentId"])
		assert.Equal(t, "v2", agentTool.Config["agentAlias"])
		assert.Equal(t, "eu-central-1", agentTool.Config["region"])
	})
}

// Test Bot Model Integration with Agent Tools
func TestBotAgentToolsIntegration(t *testing.T) {
	t.Run("Bot_With_Multiple_Agent_Tools", func(t *testing.T) {
		bot := &models.Bot{
			ID:          "bot-123",
			Title:       "Multi-Tool Bot",
			Description: "Bot with multiple agent tools",
			AgentTools: []models.AgentTool{
				{
					Type:        "plain",
					Name:        "calculator",
					Description: "Mathematical calculations",
					Config:      map[string]interface{}{},
				},
				{
					Type:        "internet",
					Name:        "web_search",
					Description: "Web search capabilities",
					Config: map[string]interface{}{
						"searchEngine": "tavily",
						"maxResults":   20,
					},
				},
				{
					Type:        "bedrock_agent",
					Name:        "aws_processor",
					Description: "AWS data processing",
					Config: map[string]interface{}{
						"agentId":    "AGENT123",
						"agentAlias": "production",
						"region":     "us-east-1",
					},
				},
			},
		}

		assert.Len(t, bot.AgentTools, 3)

		// Test individual tools
		calculatorTool := bot.AgentTools[0]
		assert.Equal(t, "plain", calculatorTool.Type)
		assert.Equal(t, "calculator", calculatorTool.Name)

		webSearchTool := bot.AgentTools[1]
		assert.Equal(t, "internet", webSearchTool.Type)
		assert.Equal(t, "tavily", webSearchTool.Config["searchEngine"])

		awsTool := bot.AgentTools[2]
		assert.Equal(t, "bedrock_agent", awsTool.Type)
		assert.Equal(t, "AGENT123", awsTool.Config["agentId"])
	})

	t.Run("Bot_AgentTools_JSON_Marshaling", func(t *testing.T) {
		bot := &models.Bot{
			ID:    "bot-456",
			Title: "Test Bot",
			AgentTools: []models.AgentTool{
				{
					Type:        "plain",
					Name:        "text_processor",
					Description: "Process text content",
					Config:      map[string]interface{}{},
				},
				{
					Type:        "internet",
					Name:        "search_tool",
					Description: "Internet search tool",
					Config: map[string]interface{}{
						"searchEngine": "duckduckgo",
						"maxResults":   15,
					},
				},
			},
		}

		// Test JSON marshaling
		jsonData, err := json.Marshal(bot)
		assert.NoError(t, err)

		// Test JSON unmarshaling
		var unmarshaled models.Bot
		err = json.Unmarshal(jsonData, &unmarshaled)
		assert.NoError(t, err)

		assert.Equal(t, bot.ID, unmarshaled.ID)
		assert.Equal(t, bot.Title, unmarshaled.Title)
		assert.Len(t, unmarshaled.AgentTools, 2)

		// Verify first tool
		tool1 := unmarshaled.AgentTools[0]
		assert.Equal(t, "plain", tool1.Type)
		assert.Equal(t, "text_processor", tool1.Name)

		// Verify second tool
		tool2 := unmarshaled.AgentTools[1]
		assert.Equal(t, "internet", tool2.Type)
		assert.Equal(t, "duckduckgo", tool2.Config["searchEngine"])
		assert.Equal(t, float64(15), tool2.Config["maxResults"]) // JSON numbers become float64
	})

	t.Run("Bot_HasAgentTool", func(t *testing.T) {
		bot := &models.Bot{
			AgentTools: []models.AgentTool{
				{Type: "plain", Name: "calculator", Description: "Math tool"},
				{Type: "internet", Name: "web_search", Description: "Search tool"},
			},
		}

		assert.True(t, bot.HasAgentTool("calculator"))
		assert.True(t, bot.HasAgentTool("web_search"))
		assert.False(t, bot.HasAgentTool("nonexistent_tool"))
	})

	t.Run("Bot_GetAgentToolByName", func(t *testing.T) {
		calculatorTool := models.AgentTool{
			Type:        "plain",
			Name:        "calculator",
			Description: "Mathematical calculations",
		}

		bot := &models.Bot{
			AgentTools: []models.AgentTool{calculatorTool},
		}

		foundTool := bot.GetAgentToolByName("calculator")
		assert.NotNil(t, foundTool)
		assert.Equal(t, "calculator", foundTool.Name)
		assert.Equal(t, "plain", foundTool.Type)

		notFoundTool := bot.GetAgentToolByName("nonexistent")
		assert.Nil(t, notFoundTool)
	})

	t.Run("Bot_AgentTools_Validation", func(t *testing.T) {
		// Test bot with valid agent tools
		validBot := &models.Bot{
			ID:    "bot-valid",
			Title: "Valid Bot",
			AgentTools: []models.AgentTool{
				{Type: "plain", Name: "calculator", Description: "Math calculations"},
				{Type: "internet", Name: "search", Description: "Web search", Config: map[string]interface{}{"searchEngine": "tavily", "maxResults": 10}},
			},
		}

		assert.True(t, validBot.AreAgentToolsValid())

		// Test bot with invalid agent tools
		invalidBot := &models.Bot{
			ID:    "bot-invalid",
			Title: "Invalid Bot",
			AgentTools: []models.AgentTool{
				{Type: "plain", Name: "calculator", Description: "Math calculations"},
				{Type: "invalid_type", Name: "bad_tool", Description: "Invalid tool"},
			},
		}

		assert.False(t, invalidBot.AreAgentToolsValid())
	})
}