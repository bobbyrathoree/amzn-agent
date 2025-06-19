// ⚡ REAL-TIME METRICS HOOK - AWS POWERED
// Live dashboard updates with WebSocket integration and intelligent polling

import { useState, useEffect, useCallback, useRef } from 'react';
import { ApiClient } from '../lib/api';

export interface RealTimeMetrics {
  todayRequests: number;
  todayCost: number;
  todaySuccessRate: number;
  todayAvgResponse: number;
  peakHour: number;
  isActive: boolean;
  lastUpdated: string;
  currentHourRequests?: number;
  activeServices?: string[];
  liveEvents?: LiveEvent[];
}

export interface LiveEvent {
  id: string;
  timestamp: string;
  serviceId: string;
  success: boolean;
  responseTime: number;
  cost: number;
}

interface UseRealTimeMetricsReturn {
  realTimeMetrics: RealTimeMetrics | null;
  isLoading: boolean;
  error: string | null;
  isConnected: boolean;
  lastUpdated: Date | null;
  connectionQuality: 'excellent' | 'good' | 'poor' | 'disconnected';
}

// WebSocket connection manager for real-time updates
class WebSocketManager {
  private ws: WebSocket | null = null;
  private url: string;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private heartbeatInterval: any = null;
  private subscribers = new Set<(data: RealTimeMetrics) => void>();
  private onConnectionChange: ((connected: boolean) => void) | null = null;

  constructor(url: string) {
    this.url = url;
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.url);
        
        this.ws.onopen = () => {
          console.log('🔌 WebSocket connected for real-time metrics');
          this.reconnectAttempts = 0;
          this.startHeartbeat();
          this.onConnectionChange?.(true);
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const data: RealTimeMetrics = JSON.parse(event.data);
            this.notifySubscribers(data);
          } catch (err) {
            console.error('❌ Error parsing WebSocket message:', err);
          }
        };

        this.ws.onclose = () => {
          console.log('🔌 WebSocket disconnected');
          this.stopHeartbeat();
          this.onConnectionChange?.(false);
          this.scheduleReconnect();
        };

        this.ws.onerror = (error) => {
          console.error('❌ WebSocket error:', error);
          this.onConnectionChange?.(false);
          reject(error);
        };

      } catch (error) {
        reject(error);
      }
    });
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.stopHeartbeat();
  }

  subscribe(callback: (data: RealTimeMetrics) => void): () => void {
    this.subscribers.add(callback);
    return () => this.subscribers.delete(callback);
  }

  setConnectionChangeHandler(handler: (connected: boolean) => void): void {
    this.onConnectionChange = handler;
  }

  private notifySubscribers(data: RealTimeMetrics): void {
    this.subscribers.forEach(callback => callback(data));
  }

  private startHeartbeat(): void {
    this.heartbeatInterval = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: 'ping' }));
      }
    }, 30000); // 30 second heartbeat
  }

  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts);
      console.log(`🔄 Scheduling WebSocket reconnect in ${delay}ms (attempt ${this.reconnectAttempts + 1})`);
      
      setTimeout(() => {
        this.reconnectAttempts++;
        this.connect().catch(err => {
          console.error('❌ WebSocket reconnect failed:', err);
        });
      }, delay);
    } else {
      console.log('❌ Max WebSocket reconnect attempts reached');
    }
  }

  get isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

// Global WebSocket manager instance
let wsManager: WebSocketManager | null = null;

export function useRealTimeMetrics(
  userId: string,
  pollingInterval: number = 30000, // 30 seconds
  options: {
    enableWebSocket?: boolean;
    fallbackToPolling?: boolean;
    maxRetries?: number;
  } = {}
): UseRealTimeMetricsReturn {
  const {
    enableWebSocket = true,
    fallbackToPolling = true,
    maxRetries = 3
  } = options;

  const [realTimeMetrics, setRealTimeMetrics] = useState<RealTimeMetrics | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Connection state is managed by wsManager
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [connectionQuality, setConnectionQuality] = useState<'excellent' | 'good' | 'poor' | 'disconnected'>('disconnected');

  const pollingIntervalRef = useRef<any>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const retryCountRef = useRef(0);
  const lastFetchTimeRef = useRef<number>(0);

  // Fetch metrics via HTTP polling
  const fetchMetrics = useCallback(async (): Promise<void> => {
    try {
      const startTime = Date.now();
      
      // Abort previous request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
      
      abortControllerRef.current = new AbortController();
      
      const apiClient = new ApiClient(
        async () => null,
        () => userId
      );
      const response = await apiClient.get('analytics/real-time-metrics');

      if (!response.ok) {
        throw new Error(`Real-time metrics API error: ${response.status}`);
      }

      const data: RealTimeMetrics = await response.json();
      const fetchTime = Date.now() - startTime;
      
      // Update connection quality based on response time
      if (fetchTime < 500) setConnectionQuality('excellent');
      else if (fetchTime < 1500) setConnectionQuality('good');
      else setConnectionQuality('poor');

      setRealTimeMetrics(data);
      setLastUpdated(new Date());
      setError(null);
      retryCountRef.current = 0;
      lastFetchTimeRef.current = Date.now();

      console.log(`⚡ Real-time metrics updated in ${fetchTime}ms`);

    } catch (err) {
      if (err instanceof Error && err.name !== 'AbortError') {
        console.error('❌ Real-time metrics error:', err.message);
        setError(err.message);
        setConnectionQuality('disconnected');
        
        // Implement exponential backoff for retries
        if (retryCountRef.current < maxRetries) {
          retryCountRef.current++;
          const retryDelay = Math.min(1000 * Math.pow(2, retryCountRef.current), 10000);
          
          setTimeout(() => {
            console.log(`🔄 Retrying real-time metrics (attempt ${retryCountRef.current})`);
            fetchMetrics();
          }, retryDelay);
        }
      }
    } finally {
      setIsLoading(false);
      abortControllerRef.current = null;
    }
  }, [userId, maxRetries]);

  // Initialize WebSocket connection
  const initializeWebSocket = useCallback(async (): Promise<void> => {
    if (!enableWebSocket) return;

    try {
      // Create WebSocket URL (replace with your actual WebSocket endpoint)
      const wsUrl = `wss://api.example.com/ws/analytics/${userId}`;
      
      if (!wsManager) {
        wsManager = new WebSocketManager(wsUrl);
      }

      // Set up connection change handler
      wsManager.setConnectionChangeHandler((connected) => {
        // Connection state managed by wsManager
        if (connected) {
          setConnectionQuality('excellent');
          setError(null);
        } else {
          setConnectionQuality('disconnected');
          // Fallback to polling if WebSocket fails
          if (fallbackToPolling) {
            console.log('🔄 Falling back to HTTP polling');
            startPolling();
          }
        }
      });

      // Subscribe to real-time updates
      wsManager.subscribe((data: RealTimeMetrics) => {
        setRealTimeMetrics(data);
        setLastUpdated(new Date());
        setError(null);
        setConnectionQuality('excellent');
      });

      // Connect to WebSocket
      await wsManager.connect();

    } catch (error) {
      console.error('❌ WebSocket initialization failed:', error);
      if (fallbackToPolling) {
        console.log('🔄 Falling back to HTTP polling');
        startPolling();
      }
    }
  }, [userId, enableWebSocket, fallbackToPolling]);

  // Start HTTP polling
  const startPolling = useCallback((): void => {
    if (pollingIntervalRef.current) {
      clearInterval(pollingIntervalRef.current);
    }

    // Initial fetch
    fetchMetrics();

    // Set up polling interval
    pollingIntervalRef.current = setInterval(() => {
      // Only poll if WebSocket is not connected
      if (!wsManager?.isConnected) {
        fetchMetrics();
      }
    }, pollingInterval);

    console.log(`⏰ Started polling real-time metrics every ${pollingInterval}ms`);
  }, [fetchMetrics, pollingInterval]);

  // Stop polling
  const stopPolling = useCallback((): void => {
    if (pollingIntervalRef.current) {
      clearInterval(pollingIntervalRef.current);
      pollingIntervalRef.current = null;
    }
  }, []);

  // Initialize real-time metrics
  useEffect(() => {
    const initialize = async () => {
      if (enableWebSocket) {
        await initializeWebSocket();
      } else {
        startPolling();
      }
    };

    initialize();

    return () => {
      stopPolling();
      if (wsManager && enableWebSocket) {
        wsManager.disconnect();
      }
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
    };
  }, [enableWebSocket, initializeWebSocket, startPolling, stopPolling]);

  // Health check - ensure we're getting updates
  useEffect(() => {
    const healthCheckInterval = setInterval(() => {
      const now = Date.now();
      const timeSinceLastUpdate = now - lastFetchTimeRef.current;
      
      // If we haven't received updates in 2x the polling interval, something's wrong
      if (timeSinceLastUpdate > pollingInterval * 2 && !isLoading) {
        console.log('⚠️ Real-time metrics health check failed, restarting connection');
        
        if (wsManager?.isConnected) {
          wsManager.disconnect();
          setTimeout(() => initializeWebSocket(), 1000);
        } else {
          startPolling();
        }
      }
    }, pollingInterval);

    return () => clearInterval(healthCheckInterval);
  }, [pollingInterval, isLoading, initializeWebSocket, startPolling]);

  return {
    realTimeMetrics,
    isLoading,
    error,
    isConnected: wsManager?.isConnected || false,
    lastUpdated,
    connectionQuality
  };
}

// Utility functions for real-time metrics
export const RealTimeUtils = {
  // Format real-time status
  formatConnectionStatus: (quality: string, isConnected: boolean): {
    label: string;
    color: string;
    icon: string;
  } => {
    if (!isConnected) {
      return { label: 'Disconnected', color: 'text-red-500', icon: '🔴' };
    }

    switch (quality) {
      case 'excellent':
        return { label: 'Live', color: 'text-green-500', icon: '🟢' };
      case 'good':
        return { label: 'Good', color: 'text-blue-500', icon: '🔵' };
      case 'poor':
        return { label: 'Slow', color: 'text-yellow-500', icon: '🟡' };
      default:
        return { label: 'Unknown', color: 'text-gray-500', icon: '⚪' };
    }
  },

  // Check if metrics are stale
  areMetricsStale: (lastUpdated: Date | null, maxAgeMinutes: number = 5): boolean => {
    if (!lastUpdated) return true;
    
    const now = new Date();
    const ageMinutes = (now.getTime() - lastUpdated.getTime()) / (1000 * 60);
    
    return ageMinutes > maxAgeMinutes;
  },

  // Calculate activity level
  getActivityLevel: (todayRequests: number): {
    level: 'low' | 'medium' | 'high' | 'very_high';
    label: string;
    color: string;
  } => {
    if (todayRequests < 10) {
      return { level: 'low', label: 'Low Activity', color: 'text-gray-500' };
    } else if (todayRequests < 50) {
      return { level: 'medium', label: 'Moderate Activity', color: 'text-blue-500' };
    } else if (todayRequests < 200) {
      return { level: 'high', label: 'High Activity', color: 'text-green-500' };
    } else {
      return { level: 'very_high', label: 'Very High Activity', color: 'text-purple-500' };
    }
  }
};