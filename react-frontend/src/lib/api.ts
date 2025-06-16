import { useMemo } from 'react';

export class ApiClient {
  private getAccessToken: () => Promise<string | null>;
  private getUserId: () => string;

  constructor(
    getAccessToken: () => Promise<string | null>,
    getUserId: () => string
  ) {
    this.getAccessToken = getAccessToken;
    this.getUserId = getUserId;
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