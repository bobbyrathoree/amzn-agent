import { useState, useEffect } from 'react';
import { useMountedRef } from './useMountedRef';
import type { Config } from '../types';

export function useConfig() {
  const [config, setConfig] = useState<Config | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  
  // Track component mount state to prevent memory leaks
  const mountedRef = useMountedRef();

  useEffect(() => {
    const loadConfig = async () => {
      if (!mountedRef.current) return;
      
      try {
        if (mountedRef.current) {
          setLoading(true);
          setError(null);
        }

        // Try to load config.json from public directory (deployed by CDK)
        const response = await fetch('/config.json', {
          cache: 'no-cache',
          headers: {
            'Cache-Control': 'no-cache'
          }
        });

        if (!response.ok) {
          throw new Error(`Failed to load config: ${response.status} ${response.statusText}`);
        }

        const configData = await response.json() as Config;
        
        // Validate required fields
        if (!configData.userPoolId || !configData.userPoolClientId || !configData.apiEndpoint) {
          throw new Error('Invalid config: missing required fields');
        }

        if (mountedRef.current) {
          setConfig(configData);
        }
        // Config loaded successfully
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Unknown error loading config';
        
        if (mountedRef.current) {
          setError(errorMessage);
        }
        console.error('Error loading config:', err);
        
        // Fallback to environment variables for local development
        console.log('Attempting fallback to environment variables...');
        const fallbackConfig: Config = {
          environment: 'development',
          userPoolId: import.meta.env.VITE_USER_POOL_ID || '',
          userPoolClientId: import.meta.env.VITE_USER_POOL_CLIENT_ID || '',
          apiEndpoint: import.meta.env.VITE_API_ENDPOINT || 'https://c9wu2knteb.execute-api.us-east-1.amazonaws.com/prod',
          websocketEndpoint: import.meta.env.VITE_WEBSOCKET_ENDPOINT || 'wss://bh9bgcvljl.execute-api.us-east-1.amazonaws.com/prod',
          region: import.meta.env.VITE_AWS_REGION || 'us-east-1'
        };

        if (fallbackConfig.userPoolId && mountedRef.current) {
          setConfig(fallbackConfig);
          setError(null);
          // Using fallback config from environment variables
        }
      } finally {
        if (mountedRef.current) {
          setLoading(false);
        }
      }
    };

    loadConfig();
  }, []);

  return { config, loading, error };
}