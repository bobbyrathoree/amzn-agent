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
    return fetch(this.getUrl(path), {
      method: 'GET',
      headers,
    });
  }

  async post(path: string, body?: any): Promise<Response> {
    const headers = await this.getHeaders();
    return fetch(this.getUrl(path), {
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
}

// Hook to get API client instance
export function useApiClient(
  getAccessToken: () => Promise<string | null>,
  getUserId: () => string | null
): ApiClient | null {
  if (!getUserId()) {
    return null;
  }
  
  return new ApiClient(getAccessToken, () => getUserId() || 'unknown');
}