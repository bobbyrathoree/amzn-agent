// 🧪 API KEY TESTER SERVICE
// Frontend service for testing API keys with actual service endpoints

export interface TestResult {
  success: boolean;
  valid: boolean;
  message: string;
  details?: Record<string, any>;
  error?: string;
  responseTime?: number;
}

/**
 * API Key Tester - Tests API keys against actual service endpoints
 */
export class APIKeyTester {
  private static readonly TIMEOUT = 10000; // 10 seconds
  
  /**
   * Test an API key for a specific service
   */
  static async testAPIKey(serviceId: string, apiKey: string): Promise<TestResult> {
    const startTime = Date.now();
    
    try {
      console.log(`🧪 Testing API key for service: ${serviceId}`);
      
      let result: TestResult;
      
      switch (serviceId) {
        case 'openai':
          result = await this.testOpenAI(apiKey);
          break;
        case 'anthropic':
          result = await this.testAnthropic(apiKey);
          break;
        case 'google-search':
          result = await this.testGoogleSearch(apiKey);
          break;
        case 'weather-api':
          result = await this.testWeatherAPI(apiKey);
          break;
        case 'news-api':
          result = await this.testNewsAPI(apiKey);
          break;
        case 'github':
          result = await this.testGitHub(apiKey);
          break;
        case 'slack':
          result = await this.testSlack(apiKey);
          break;
        case 'hubspot':
          result = await this.testHubSpot(apiKey);
          break;
        case 'tavily':
          result = await this.testTavily(apiKey);
          break;
        case 'serpapi':
          result = await this.testSerpAPI(apiKey);
          break;
        default:
          result = {
            success: false,
            valid: false,
            message: `Testing not implemented for service: ${serviceId}`,
            error: 'Service testing not available'
          };
      }
      
      result.responseTime = Date.now() - startTime;
      console.log(`✅ API key test completed for ${serviceId}:`, result);
      
      return result;
    } catch (error) {
      const responseTime = Date.now() - startTime;
      console.error(`❌ API key test failed for ${serviceId}:`, error);
      
      return {
        success: false,
        valid: false,
        message: `Test failed: ${error instanceof Error ? error.message : 'Unknown error'}`,
        error: error instanceof Error ? error.message : 'Unknown error',
        responseTime
      };
    }
  }

  /**
   * Test OpenAI API Key
   */
  private static async testOpenAI(apiKey: string): Promise<TestResult> {
    try {
      const response = await fetch('https://api.openai.com/v1/models', {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${apiKey}`,
          'Content-Type': 'application/json'
        },
        signal: AbortSignal.timeout(this.TIMEOUT)
      });

      if (response.ok) {
        const data = await response.json();
        return {
          success: true,
          valid: true,
          message: `OpenAI API key is valid. Access to ${data.data?.length || 0} models.`,
          details: {
            modelsCount: data.data?.length || 0,
            status: response.status
          }
        };
      } else {
        const errorData = await response.json().catch(() => null);
        return {
          success: true,
          valid: false,
          message: `OpenAI API key is invalid: ${errorData?.error?.message || response.statusText}`,
          details: {
            status: response.status,
            error: errorData?.error
          }
        };
      }
    } catch (error) {
      return {
        success: false,
        valid: false,
        message: `OpenAI API test failed: ${error instanceof Error ? error.message : 'Network error'}`,
        error: error instanceof Error ? error.message : 'Network error'
      };
    }
  }

  /**
   * Test Anthropic API Key
   */
  private static async testAnthropic(apiKey: string): Promise<TestResult> {
    try {
      // Anthropic doesn't have a simple models endpoint, so we'll try a minimal message
      const response = await fetch('https://api.anthropic.com/v1/messages', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${apiKey}`,
          'Content-Type': 'application/json',
          'anthropic-version': '2023-06-01'
        },
        body: JSON.stringify({
          model: 'claude-3-haiku-20240307',
          max_tokens: 1,
          messages: [{ role: 'user', content: 'Hi' }]
        }),
        signal: AbortSignal.timeout(this.TIMEOUT)
      });

      if (response.ok) {
        return {
          success: true,
          valid: true,
          message: 'Anthropic API key is valid and working.',
          details: {
            status: response.status
          }
        };
      } else {
        const errorData = await response.json().catch(() => null);
        return {
          success: true,
          valid: false,
          message: `Anthropic API key is invalid: ${errorData?.error?.message || response.statusText}`,
          details: {
            status: response.status,
            error: errorData?.error
          }
        };
      }
    } catch (error) {
      return {
        success: false,
        valid: false,
        message: `Anthropic API test failed: ${error instanceof Error ? error.message : 'Network error'}`,
        error: error instanceof Error ? error.message : 'Network error'
      };
    }
  }

  /**
   * Test Google Custom Search API Key
   */
  private static async testGoogleSearch(apiKey: string): Promise<TestResult> {
    try {
      // We need a search engine ID for this test, so we'll test the API key format
      if (!apiKey.startsWith('AIza') || apiKey.length < 39) {
        return {
          success: true,
          valid: false,
          message: 'Google API key format appears invalid. Should start with "AIza" and be 39+ characters.',
          details: {
            format: 'invalid',
            length: apiKey.length
          }
        };
      }

      // Test with a simple custom search (this will fail without a search engine ID, but validates the key)
      const response = await fetch(`https://www.googleapis.com/customsearch/v1?key=${apiKey}&cx=test&q=test`, {
        method: 'GET',
        signal: AbortSignal.timeout(this.TIMEOUT)
      });

      if (response.status === 400) {
        // Expected error - invalid search engine ID, but key format is accepted
        return {
          success: true,
          valid: true,
          message: 'Google API key format is valid. You need to set up a Custom Search Engine ID.',
          details: {
            status: response.status,
            note: 'Key is valid but requires Custom Search Engine configuration'
          }
        };
      } else if (response.status === 403) {
        const errorData = await response.json().catch(() => null);
        return {
          success: true,
          valid: false,
          message: `Google API key is invalid or lacks permissions: ${errorData?.error?.message || 'Access denied'}`,
          details: {
            status: response.status,
            error: errorData?.error
          }
        };
      } else {
        return {
          success: true,
          valid: true,
          message: 'Google API key appears to be working.',
          details: {
            status: response.status
          }
        };
      }
    } catch (error) {
      return {
        success: false,
        valid: false,
        message: `Google Search API test failed: ${error instanceof Error ? error.message : 'Network error'}`,
        error: error instanceof Error ? error.message : 'Network error'
      };
    }
  }

  /**
   * Test Weather API Key
   */
  private static async testWeatherAPI(apiKey: string): Promise<TestResult> {
    try {
      const response = await fetch(`https://api.openweathermap.org/data/2.5/weather?q=London&appid=${apiKey}`, {
        method: 'GET',
        signal: AbortSignal.timeout(this.TIMEOUT)
      });

      if (response.ok) {
        const data = await response.json();
        return {
          success: true,
          valid: true,
          message: `Weather API key is valid. Test location: ${data.name}, ${data.sys?.country}`,
          details: {
            location: `${data.name}, ${data.sys?.country}`,
            temperature: data.main?.temp,
            status: response.status
          }
        };
      } else {
        const errorData = await response.json().catch(() => null);
        return {
          success: true,
          valid: false,
          message: `Weather API key is invalid: ${errorData?.message || response.statusText}`,
          details: {
            status: response.status,
            error: errorData
          }
        };
      }
    } catch (error) {
      return {
        success: false,
        valid: false,
        message: `Weather API test failed: ${error instanceof Error ? error.message : 'Network error'}`,
        error: error instanceof Error ? error.message : 'Network error'
      };
    }
  }

  /**
   * Test News API Key
   */
  private static async testNewsAPI(apiKey: string): Promise<TestResult> {
    try {
      const response = await fetch(`https://newsapi.org/v2/top-headlines?country=us&pageSize=1&apiKey=${apiKey}`, {
        method: 'GET',
        signal: AbortSignal.timeout(this.TIMEOUT)
      });

      if (response.ok) {
        const data = await response.json();
        return {
          success: true,
          valid: true,
          message: `News API key is valid. Found ${data.totalResults || 0} articles.`,
          details: {
            totalResults: data.totalResults,
            status: response.status
          }
        };
      } else {
        const errorData = await response.json().catch(() => null);
        return {
          success: true,
          valid: false,
          message: `News API key is invalid: ${errorData?.message || response.statusText}`,
          details: {
            status: response.status,
            error: errorData
          }
        };
      }
    } catch (error) {
      return {
        success: false,
        valid: false,
        message: `News API test failed: ${error instanceof Error ? error.message : 'Network error'}`,
        error: error instanceof Error ? error.message : 'Network error'
      };
    }
  }

  /**
   * Test GitHub API Key
   */
  private static async testGitHub(apiKey: string): Promise<TestResult> {
    try {
      const response = await fetch('https://api.github.com/user', {
        method: 'GET',
        headers: {
          'Authorization': `token ${apiKey}`,
          'Accept': 'application/vnd.github.v3+json'
        },
        signal: AbortSignal.timeout(this.TIMEOUT)
      });

      if (response.ok) {
        const data = await response.json();
        return {
          success: true,
          valid: true,
          message: `GitHub API key is valid. Authenticated as: ${data.login}`,
          details: {
            username: data.login,
            name: data.name,
            status: response.status
          }
        };
      } else {
        const errorData = await response.json().catch(() => null);
        return {
          success: true,
          valid: false,
          message: `GitHub API key is invalid: ${errorData?.message || response.statusText}`,
          details: {
            status: response.status,
            error: errorData
          }
        };
      }
    } catch (error) {
      return {
        success: false,
        valid: false,
        message: `GitHub API test failed: ${error instanceof Error ? error.message : 'Network error'}`,
        error: error instanceof Error ? error.message : 'Network error'
      };
    }
  }

  /**
   * Test Slack API Key
   */
  private static async testSlack(apiKey: string): Promise<TestResult> {
    try {
      const response = await fetch('https://slack.com/api/auth.test', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${apiKey}`,
          'Content-Type': 'application/x-www-form-urlencoded'
        },
        signal: AbortSignal.timeout(this.TIMEOUT)
      });

      if (response.ok) {
        const data = await response.json();
        if (data.ok) {
          return {
            success: true,
            valid: true,
            message: `Slack API key is valid. Team: ${data.team}, User: ${data.user}`,
            details: {
              team: data.team,
              user: data.user,
              status: response.status
            }
          };
        } else {
          return {
            success: true,
            valid: false,
            message: `Slack API key is invalid: ${data.error}`,
            details: {
              error: data.error,
              status: response.status
            }
          };
        }
      } else {
        return {
          success: true,
          valid: false,
          message: `Slack API key test failed: ${response.statusText}`,
          details: {
            status: response.status
          }
        };
      }
    } catch (error) {
      return {
        success: false,
        valid: false,
        message: `Slack API test failed: ${error instanceof Error ? error.message : 'Network error'}`,
        error: error instanceof Error ? error.message : 'Network error'
      };
    }
  }

  /**
   * Test HubSpot API Key
   */
  private static async testHubSpot(apiKey: string): Promise<TestResult> {
    try {
      const response = await fetch(`https://api.hubapi.com/contacts/v1/lists/all/contacts/all?hapikey=${apiKey}&count=1`, {
        method: 'GET',
        signal: AbortSignal.timeout(this.TIMEOUT)
      });

      if (response.ok) {
        return {
          success: true,
          valid: true,
          message: 'HubSpot API key is valid and working.',
          details: {
            status: response.status
          }
        };
      } else {
        const errorData = await response.json().catch(() => null);
        return {
          success: true,
          valid: false,
          message: `HubSpot API key is invalid: ${errorData?.message || response.statusText}`,
          details: {
            status: response.status,
            error: errorData
          }
        };
      }
    } catch (error) {
      return {
        success: false,
        valid: false,
        message: `HubSpot API test failed: ${error instanceof Error ? error.message : 'Network error'}`,
        error: error instanceof Error ? error.message : 'Network error'
      };
    }
  }

  /**
   * Test Tavily API Key
   */
  private static async testTavily(apiKey: string): Promise<TestResult> {
    try {
      const response = await fetch('https://api.tavily.com/search', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          api_key: apiKey,
          query: 'test',
          max_results: 1
        }),
        signal: AbortSignal.timeout(this.TIMEOUT)
      });

      if (response.ok) {
        const data = await response.json();
        return {
          success: true,
          valid: true,
          message: `Tavily API key is valid. Found ${data.results?.length || 0} results.`,
          details: {
            resultsCount: data.results?.length || 0,
            status: response.status
          }
        };
      } else {
        const errorData = await response.json().catch(() => null);
        return {
          success: true,
          valid: false,
          message: `Tavily API key is invalid: ${errorData?.detail || response.statusText}`,
          details: {
            status: response.status,
            error: errorData
          }
        };
      }
    } catch (error) {
      return {
        success: false,
        valid: false,
        message: `Tavily API test failed: ${error instanceof Error ? error.message : 'Network error'}`,
        error: error instanceof Error ? error.message : 'Network error'
      };
    }
  }

  /**
   * Test SerpAPI Key
   */
  private static async testSerpAPI(apiKey: string): Promise<TestResult> {
    try {
      const response = await fetch(`https://serpapi.com/search?q=test&engine=google&api_key=${apiKey}&num=1`, {
        method: 'GET',
        signal: AbortSignal.timeout(this.TIMEOUT)
      });

      if (response.ok) {
        const data = await response.json();
        if (data.error) {
          return {
            success: true,
            valid: false,
            message: `SerpAPI key is invalid: ${data.error}`,
            details: {
              error: data.error,
              status: response.status
            }
          };
        }
        return {
          success: true,
          valid: true,
          message: `SerpAPI key is valid. Found ${data.organic_results?.length || 0} results.`,
          details: {
            resultsCount: data.organic_results?.length || 0,
            status: response.status
          }
        };
      } else {
        return {
          success: true,
          valid: false,
          message: `SerpAPI key test failed: ${response.statusText}`,
          details: {
            status: response.status
          }
        };
      }
    } catch (error) {
      return {
        success: false,
        valid: false,
        message: `SerpAPI test failed: ${error instanceof Error ? error.message : 'Network error'}`,
        error: error instanceof Error ? error.message : 'Network error'
      };
    }
  }
}