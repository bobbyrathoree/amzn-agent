'use client';

import { createContext, useContext, useEffect, useState } from 'react';

export interface Config {
  environment: string;
  apiEndpoint: string;
  websocketEndpoint: string;
  userPoolId: string;
  userPoolClientId: string;
  region: string;
}

interface ConfigContextType {
  config: Config | null;
  loading: boolean;
}

export const ConfigContext = createContext<ConfigContextType>({
  config: null,
  loading: true,
});

export function ConfigProvider({ children }: { children: React.ReactNode }) {
  const [config, setConfig] = useState<Config | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function loadConfig() {
      try {
        // Use local config during development
        const isDevelopment = process.env.NODE_ENV === 'development';
        const configUrl = isDevelopment ? '/config.local.json' : '/config.json';
        
        const response = await fetch(configUrl);
        if (response.ok) {
          const configData = await response.json();
          setConfig(configData);
          console.log(`Loaded config from ${configUrl}:`, configData);
        } else {
          console.error(`Failed to load config from ${configUrl}`);
          // Fallback to the other config file
          const fallbackUrl = isDevelopment ? '/config.json' : '/config.local.json';
          const fallbackResponse = await fetch(fallbackUrl);
          if (fallbackResponse.ok) {
            const configData = await fallbackResponse.json();
            setConfig(configData);
            console.log(`Loaded fallback config from ${fallbackUrl}:`, configData);
          }
        }
      } catch (error) {
        console.error('Error loading config:', error);
      } finally {
        setLoading(false);
      }
    }

    loadConfig();
  }, []);

  return (
    <ConfigContext.Provider value={{ config, loading }}>
      {children}
    </ConfigContext.Provider>
  );
}

export function useConfig() {
  const context = useContext(ConfigContext);
  if (context === undefined) {
    throw new Error('useConfig must be used within a ConfigProvider');
  }
  return context;
}