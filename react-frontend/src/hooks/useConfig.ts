import { useState, useEffect, useRef } from 'react';
import type { Config } from '../types';

export function useConfig() {
  const [config, setConfig] = useState<Config | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Use local cancelled flag to handle React Strict Mode
    let cancelled = false;

    const loadConfig = async () => {
      console.log('[useConfig] Starting loadConfig');

      try {
        setLoading(true);
        setError(null);

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
        console.log('[useConfig] Config data loaded:', configData);

        // Validate required fields
        if (!configData.userPoolId || !configData.userPoolClientId || !configData.apiEndpoint) {
          throw new Error('Invalid config: missing required fields');
        }

        if (!cancelled) {
          console.log('[useConfig] Setting config');
          setConfig(configData);
        }
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Unknown error loading config';
        console.error('Error loading config:', err);

        if (!cancelled) {
          setError(errorMessage);
        }

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

        if (fallbackConfig.userPoolId && !cancelled) {
          setConfig(fallbackConfig);
          setError(null);
        }
      } finally {
        if (!cancelled) {
          console.log('[useConfig] Setting loading to false');
          setLoading(false);
        }
      }
    };

    loadConfig();

    return () => {
      cancelled = true;
    };
  }, []);

  return { config, loading, error };
}