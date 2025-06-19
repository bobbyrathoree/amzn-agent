// 📊 ANALYTICS DATA HOOK - AWS POWERED
// Real-time analytics data with intelligent caching and error handling

import { useState, useEffect, useCallback, useRef } from 'react';
import { ApiClient } from '../lib/api';

export interface DailyUsageSummary {
  id: string;
  userId: string;
  serviceId: string;
  date: string;
  totalRequests: number;
  successfulRequests: number;
  failedRequests: number;
  successRate: number;
  avgResponseTime: number;
  p95ResponseTime: number;
  maxResponseTime: number;
  minResponseTime: number;
  totalCost: number;
  avgCostPerRequest: number;
  totalCreditsUsed: number;
  totalTokensUsed: number;
  peakHour: number;
  peakRequestCount: number;
  uniqueToolsUsed: number;
  premiumRequests: number;
  freeRequests: number;
  fallbackRequests: number;
  lastUpdated: string;
  recordCount: number;
}

export interface AggregateMetrics {
  totalRequests: number;
  totalCost: number;
  avgResponseTime: number;
  overallSuccessRate: number;
  activeServices: number;
  costPerRequest: number;
}

export interface AnalyticsResponse {
  summaries: DailyUsageSummary[];
  aggregates: AggregateMetrics;
  periodDays: number;
  generatedAt: string;
}

interface UseAnalyticsDataReturn {
  summaries: DailyUsageSummary[] | null;
  aggregates: AggregateMetrics | null;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
  lastFetched: Date | null;
}

interface CacheEntry {
  data: AnalyticsResponse;
  timestamp: number;
  expiry: number;
}

// Smart caching with TTL
const globalAnalyticsCache = new Map<string, CacheEntry>();
const CACHE_TTL = 5 * 60 * 1000; // 5 minutes
const STALE_WHILE_REVALIDATE_TTL = 30 * 60 * 1000; // 30 minutes

export function useAnalyticsData(
  userId: string, 
  days: number = 30,
  options: {
    refreshInterval?: number;
    enableRealTime?: boolean;
    cacheStrategy?: 'fresh' | 'cache-first' | 'stale-while-revalidate';
  } = {}
): UseAnalyticsDataReturn {
  const {
    refreshInterval = 0,
    enableRealTime = false,
    cacheStrategy = 'stale-while-revalidate'
  } = options;

  const [summaries, setSummaries] = useState<DailyUsageSummary[] | null>(null);
  const [aggregates, setAggregates] = useState<AggregateMetrics | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastFetched, setLastFetched] = useState<Date | null>(null);
  
  const abortControllerRef = useRef<AbortController | null>(null);
  const intervalRef = useRef<any>(null);

  // Generate cache key
  const cacheKey = `analytics_${userId}_${days}`;

  // Optimized fetch function with caching
  const fetchAnalyticsData = useCallback(async (force: boolean = false): Promise<void> => {
    try {
      console.log(`📊 Fetching analytics data for user ${userId} (${days} days)`);

      // Check cache first (unless force refresh)
      if (!force && cacheStrategy !== 'fresh') {
        const cached = globalAnalyticsCache.get(cacheKey);
        const now = Date.now();

        if (cached) {
          if (now < cached.expiry) {
            // Cache hit - use fresh data
            console.log('📦 Using cached analytics data');
            setSummaries(cached.data.summaries);
            setAggregates(cached.data.aggregates);
            setLastFetched(new Date(cached.timestamp));
            setIsLoading(false);
            setError(null);
            return;
          } else if (cacheStrategy === 'stale-while-revalidate' && now < cached.timestamp + STALE_WHILE_REVALIDATE_TTL) {
            // Serve stale data while revalidating in background
            console.log('🔄 Serving stale data while revalidating');
            setSummaries(cached.data.summaries);
            setAggregates(cached.data.aggregates);
            setLastFetched(new Date(cached.timestamp));
            setIsLoading(false);
            setError(null);
            // Continue to fetch fresh data below
          }
        }
      }

      // Abort previous request if still pending
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      // Create new abort controller
      abortControllerRef.current = new AbortController();
      
      const apiClient = new ApiClient(
        async () => null, // Mock access token getter
        () => userId // Use userId from parameter
      );

      // Set loading state (only if no cached data)
      if (!summaries || force) {
        setIsLoading(true);
      }

      // Fetch from API
      const startTime = Date.now();
      const response = await apiClient.get(`analytics/usage-summary?days=${days}`);

      const fetchTime = Date.now() - startTime;
      console.log(`✅ Analytics data fetched in ${fetchTime}ms`);

      if (!response.ok) {
        throw new Error(`API Error: ${response.status} - ${response.statusText}`);
      }

      const data: AnalyticsResponse = await response.json();

      // Validate response structure
      if (!data.summaries || !Array.isArray(data.summaries)) {
        throw new Error('Invalid analytics response structure');
      }

      // Cache the response
      const now = Date.now();
      globalAnalyticsCache.set(cacheKey, {
        data,
        timestamp: now,
        expiry: now + CACHE_TTL
      });

      // Update state
      setSummaries(data.summaries);
      setAggregates(data.aggregates);
      setLastFetched(new Date());
      setError(null);

      console.log(`📊 Analytics data updated: ${data.summaries.length} summaries, ${Object.keys(data.aggregates).length} metrics`);

    } catch (err) {
      // Handle different error types
      if (err instanceof Error) {
        if (err.name === 'AbortError') {
          console.log('📊 Analytics request aborted');
          return; // Don't set error for aborted requests
        }
        
        console.error('❌ Analytics fetch error:', err.message);
        setError(err.message);
      } else {
        console.error('❌ Unknown analytics error:', err);
        setError('Failed to fetch analytics data');
      }

      // On error, keep existing data if available
      if (!summaries) {
        setError('Failed to load analytics data');
      }
    } finally {
      setIsLoading(false);
      abortControllerRef.current = null;
    }
  }, [userId, days, cacheKey, cacheStrategy, summaries]);

  // Refetch function for manual refresh
  const refetch = useCallback(async (): Promise<void> => {
    console.log('🔄 Manual refresh triggered');
    await fetchAnalyticsData(true);
  }, [fetchAnalyticsData]);

  // Initial data fetch
  useEffect(() => {
    fetchAnalyticsData();
  }, [fetchAnalyticsData]);

  // Set up auto-refresh interval
  useEffect(() => {
    if (refreshInterval > 0 && enableRealTime) {
      console.log(`⏰ Setting up auto-refresh every ${refreshInterval}ms`);
      
      intervalRef.current = setInterval(() => {
        console.log('🔄 Auto-refresh triggered');
        fetchAnalyticsData();
      }, refreshInterval);

      return () => {
        if (intervalRef.current) {
          clearInterval(intervalRef.current);
          intervalRef.current = null;
        }
      };
    }
  }, [refreshInterval, enableRealTime, fetchAnalyticsData]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, []);

  return {
    summaries,
    aggregates,
    isLoading,
    error,
    refetch,
    lastFetched
  };
}

// Cache management utilities
export const AnalyticsCacheManager = {
  clear: () => {
    globalAnalyticsCache.clear();
    console.log('🗑️ Analytics cache cleared');
  },
  
  getCacheSize: () => globalAnalyticsCache.size,
  
  getCacheInfo: () => {
    const entries = Array.from(globalAnalyticsCache.entries()).map(([key, entry]) => ({
      key,
      timestamp: new Date(entry.timestamp),
      expiry: new Date(entry.expiry),
      isExpired: Date.now() > entry.expiry,
      size: JSON.stringify(entry.data).length
    }));
    
    return {
      totalEntries: entries.length,
      totalSize: entries.reduce((sum, entry) => sum + entry.size, 0),
      entries
    };
  },

  cleanupExpired: () => {
    const now = Date.now();
    let cleanedCount = 0;
    
    for (const [key, entry] of globalAnalyticsCache.entries()) {
      if (now > entry.timestamp + STALE_WHILE_REVALIDATE_TTL) {
        globalAnalyticsCache.delete(key);
        cleanedCount++;
      }
    }
    
    if (cleanedCount > 0) {
      console.log(`🧹 Cleaned up ${cleanedCount} expired cache entries`);
    }
    
    return cleanedCount;
  }
};

// Auto cleanup expired entries every 10 minutes
setInterval(() => {
  AnalyticsCacheManager.cleanupExpired();
}, 10 * 60 * 1000);