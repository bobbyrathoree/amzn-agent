import { useState } from 'react';
import { useAuth } from '../components/AuthProvider';
import { useApiClient } from '../lib/api';

export function AuthDebugger() {
  const { user, getAccessToken, environment } = useAuth();
  const apiClient = useApiClient(getAccessToken, () => user?.userId || user?.username || null, environment);
  const [debugOutput, setDebugOutput] = useState<string>('');

  const log = (message: string, data?: any) => {
    const timestamp = new Date().toLocaleTimeString();
    const logEntry = `[${timestamp}] ${message}${data ? '\n' + JSON.stringify(data, null, 2) : ''}`;
    setDebugOutput(prev => prev + '\n' + logEntry);
    console.log(message, data);
  };

  const testAuth = async () => {
    setDebugOutput('Starting authentication debug...');
    
    try {
      // Test 1: Check user state
      log('Current user state:', user);
      
      // Test 2: Get access token
      const token = await getAccessToken();
      log('Access token received:', token ? `${token.substring(0, 20)}...` : 'null');
      
      // Test 3: Decode JWT (basic check)
      if (token) {
        try {
          const parts = token.split('.');
          const payload = JSON.parse(atob(parts[1]));
          log('JWT payload:', payload);
        } catch (e) {
          log('Failed to decode JWT:', e);
        }
      }
      
      // Test 4: Test API call to bots endpoint
      log('Testing API call via API client...');
      if (!apiClient) {
        log('API client not available');
        return;
      }
      
      const response = await apiClient.get('bots');
      
      log('API Response Status:', response.status);
      log('API Response Headers:', Object.fromEntries(response.headers.entries()));
      
      const responseText = await response.text();
      log('API Response Body:', responseText);
      
      // Try to parse as JSON
      try {
        const responseJson = JSON.parse(responseText);
        log('Parsed JSON Response:', responseJson);
      } catch (e) {
        log('Response is not JSON');
      }
      
    } catch (error) {
      log('Debug test failed:', error);
    }
  };

  const testDirect = async () => {
    setDebugOutput('Testing direct API call...');
    
    try {
      const token = await getAccessToken();
      log('Testing direct call to production API...');
      
      const response = await fetch('https://c9wu2knteb.execute-api.us-east-1.amazonaws.com/prod/bots', {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${token}`,
          'X-User-ID': user?.userId || user?.username || 'unknown',
          'Content-Type': 'application/json'
        }
      });
      
      log('Direct API Response Status:', response.status);
      log('Direct API Response Headers:', Object.fromEntries(response.headers.entries()));
      
      const responseText = await response.text();
      log('Direct API Response Body:', responseText);
      
    } catch (error) {
      log('Direct API test failed:', error);
    }
  };

  return (
    <div className="p-4 bg-gray-100 rounded-lg">
      <h3 className="text-lg font-bold mb-4">Authentication Debugger</h3>
      
      <div className="space-x-2 mb-4">
        <button
          onClick={testAuth}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
        >
          Test Auth & Proxy
        </button>
        <button
          onClick={testDirect}
          className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700"
        >
          Test Direct API
        </button>
        <button
          onClick={() => setDebugOutput('')}
          className="px-4 py-2 bg-gray-600 text-white rounded hover:bg-gray-700"
        >
          Clear
        </button>
      </div>
      
      <div className="bg-black text-green-400 p-4 rounded font-mono text-sm max-h-96 overflow-y-auto">
        <pre>{debugOutput || 'Click "Test Auth & Proxy" to start debugging...'}</pre>
      </div>
    </div>
  );
}