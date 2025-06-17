import { useMemo } from 'react';

// Custom chat utilities to replace what AI SDK would have provided
export interface StreamConfig {
  onStart?: () => void;
  onToken?: (token: string) => void;
  onFinish?: (response: string) => void;
  onError?: (error: Error) => void;
}

export class ChatStreamHandler {
  private decoder = new TextDecoder();
  
  async handleStream(response: Response, config: StreamConfig = {}): Promise<string> {
    if (!response.body) {
      throw new Error('Response body is empty');
    }

    config.onStart?.();
    
    const reader = response.body.getReader();
    let result = '';
    
    try {
      while (true) {
        const { done, value } = await reader.read();
        
        if (done) break;
        
        const chunk = this.decoder.decode(value, { stream: true });
        const lines = chunk.split('\n');
        
        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = line.slice(6);
              if (data === '[DONE]') {
                config.onFinish?.(result);
                return result;
              }
              
              const parsed = JSON.parse(data);
              if (parsed.token) {
                result += parsed.token;
                config.onToken?.(parsed.token);
              }
            } catch (e) {
              // Skip malformed JSON
              continue;
            }
          }
        }
      }
      
      config.onFinish?.(result);
      return result;
    } catch (error) {
      config.onError?.(error instanceof Error ? error : new Error(String(error)));
      throw error;
    } finally {
      reader.releaseLock();
    }
  }
}

export class ApiClient {
  private getAccessToken: () => Promise<string | null>;
  private getUserId: () => string;
  private streamHandler: ChatStreamHandler;

  constructor(
    getAccessToken: () => Promise<string | null>,
    getUserId: () => string
  ) {
    this.getAccessToken = getAccessToken;
    this.getUserId = getUserId;
    this.streamHandler = new ChatStreamHandler();
  }

  private async getHeaders(): Promise<Record<string, string>> {
    const token = await this.getAccessToken();
    return {
      'Content-Type': 'application/json',
      'Authorization': token ? `Bearer ${token}` : '',
      'X-User-ID': this.getUserId(),
    };
  }

  private getUrl(path: string): string {
    // Use relative URLs - CloudFront will proxy /prod/* to API Gateway
    const cleanPath = path.replace(/^\/+/, ''); // Remove leading slashes
    return `/prod/${cleanPath}`;
  }

  async get(path: string): Promise<Response> {
    const headers = await this.getHeaders();
    const url = this.getUrl(path);
    console.log('GET request to:', url);
    return fetch(url, {
      method: 'GET',
      headers,
    });
  }

  async post(path: string, body?: any): Promise<Response> {
    const headers = await this.getHeaders();
    const url = this.getUrl(path);
    console.log('POST request to:', url, 'Body:', body);
    return fetch(url, {
      method: 'POST',
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  async put(path: string, body?: any): Promise<Response> {
    const headers = await this.getHeaders();
    return fetch(this.getUrl(path), {
      method: 'PUT',
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  async delete(path: string): Promise<Response> {
    const headers = await this.getHeaders();
    return fetch(this.getUrl(path), {
      method: 'DELETE',
      headers,
    });
  }

  async patch(path: string, body?: any): Promise<Response> {
    const headers = await this.getHeaders();
    return fetch(this.getUrl(path), {
      method: 'PATCH',
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  // Helper method for file uploads (without JSON Content-Type)
  async upload(url: string, file: File, contentType: string): Promise<Response> {
    return fetch(url, {
      method: 'PUT',
      headers: {
        'Content-Type': contentType,
      },
      body: file,
    });
  }

  // Enhanced chat methods that replace AI SDK functionality
  async chatStream(path: string, body: any, config: StreamConfig = {}): Promise<string> {
    const headers = await this.getHeaders();
    const url = this.getUrl(path);
    
    console.log('Streaming chat request to:', url, 'Body:', body);
    
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        ...headers,
        'Accept': 'text/event-stream',
      },
      body: JSON.stringify({ ...body, stream: true }),
    });

    if (!response.ok) {
      throw new Error(`Chat stream failed: ${response.status}`);
    }

    return this.streamHandler.handleStream(response, config);
  }

  // Retry mechanism for enhanced reliability
  async postWithRetry(path: string, body?: any, maxRetries: number = 2): Promise<Response> {
    let lastError: Error | null = null;
    
    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      try {
        const response = await this.post(path, body);
        
        // If we get a 5xx error, retry (server errors)
        if (attempt < maxRetries && response.status >= 500 && response.status < 600) {
          lastError = new Error(`Server error: ${response.status}`);
          await this.delay(Math.pow(2, attempt) * 1000); // Exponential backoff
          continue;
        }
        
        return response;
      } catch (error) {
        lastError = error instanceof Error ? error : new Error(String(error)); 
        if (attempt < maxRetries) {
          await this.delay(Math.pow(2, attempt) * 1000); // Exponential backoff
          continue;
        }
      }
    }
    
    throw lastError || new Error('Max retries exceeded');
  }

  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  // Enhanced error handling with better user messages
  static getErrorMessage(error: any): string {
    if (error.message?.includes('HTML instead of JSON')) {
      return 'Service temporarily unavailable. Please try again in a moment.';
    }
    
    if (error.message?.includes('NetworkError') || error.message?.includes('Failed to fetch')) {
      return 'Connection error. Please check your internet connection and try again.';
    }
    
    if (error.message?.includes('401') || error.message?.includes('Unauthorized')) {
      return 'Your session has expired. Please sign in again.';
    }
    
    if (error.message?.includes('403') || error.message?.includes('Forbidden')) {
      return 'You don\'t have permission to access this resource.';
    }
    
    if (error.message?.includes('404') || error.message?.includes('Not Found')) {
      return 'The requested resource was not found.';
    }
    
    if (error.message?.includes('429') || error.message?.includes('Too Many Requests')) {
      return 'Too many requests. Please wait a moment and try again.';
    }
    
    return error.message || 'An unexpected error occurred. Please try again.';
  }
}

// Hook to get API client instance

export function useApiClient(
  getAccessToken: () => Promise<string | null>,
  getUserId: () => string | null
): ApiClient | null {
  return useMemo(() => {
    if (!getUserId()) {
      return null;
    }
    
    return new ApiClient(getAccessToken, () => getUserId() || 'unknown');
  }, [getAccessToken, getUserId]);
}