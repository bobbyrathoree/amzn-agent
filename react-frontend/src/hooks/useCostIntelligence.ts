// 💰 COST INTELLIGENCE HOOK - AWS POWERED
// Advanced cost analysis, forecasting, and optimization recommendations

import { useState, useEffect, useCallback, useRef } from 'react';
import { ApiClient } from '../lib/api';

export interface CostAnalysis {
  currentMonthSpend: number;
  projectedMonthlySpend: number;
  costTrend: 'increasing' | 'decreasing' | 'stable' | 'insufficient_data';
  topCostServices: Array<{
    serviceId: string;
    cost: number;
    rank: number;
  }>;
  serviceCostBreakdown: Record<string, number>;
  costPerDay: number;
  budgetUtilization?: number;
  potentialSavings?: number;
  optimizationRecommendations: CostOptimization[];
}

export interface CostOptimization {
  type: 'service_substitution' | 'usage_optimization' | 'tier_change' | 'caching';
  title: string;
  description: string;
  potentialSavings: number;
  effort: 'low' | 'medium' | 'high';
  impact: 'low' | 'medium' | 'high';
  priority: number;
  actionable: boolean;
}

export interface BudgetAlert {
  type: 'approaching_limit' | 'exceeded_limit' | 'unusual_spike';
  severity: 'info' | 'warning' | 'critical';
  title: string;
  message: string;
  threshold: number;
  currentValue: number;
  recommendation?: string;
}

interface UseCostIntelligenceReturn {
  costAnalysis: CostAnalysis | null;
  costTrend: string | null;
  projectedSpend: number | null;
  budgetAlerts: BudgetAlert[];
  optimizations: CostOptimization[];
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
  setBudget: (amount: number) => Promise<void>;
  userBudget: number | null;
}

// Advanced cost calculation utilities
const CostCalculator = {
  // Calculate cost trend with statistical analysis
  calculateTrend: (dailyCosts: number[]): 'increasing' | 'decreasing' | 'stable' => {
    if (dailyCosts.length < 7) return 'stable';
    
    const midpoint = Math.floor(dailyCosts.length / 2);
    const firstHalf = dailyCosts.slice(0, midpoint);
    const secondHalf = dailyCosts.slice(midpoint);
    
    const firstAvg = firstHalf.reduce((sum, cost) => sum + cost, 0) / firstHalf.length;
    const secondAvg = secondHalf.reduce((sum, cost) => sum + cost, 0) / secondHalf.length;
    
    const changePercent = ((secondAvg - firstAvg) / firstAvg) * 100;
    
    if (changePercent > 15) return 'increasing';
    if (changePercent < -15) return 'decreasing';
    return 'stable';
  },

  // Project future costs using linear regression
  projectMonthlyCost: (dailyCosts: number[], daysInMonth: number = 30): number => {
    if (dailyCosts.length === 0) return 0;
    
    // Simple moving average with trend adjustment
    const recentDays = Math.min(7, dailyCosts.length);
    const recentCosts = dailyCosts.slice(-recentDays);
    const avgDailyCost = recentCosts.reduce((sum, cost) => sum + cost, 0) / recentCosts.length;
    
    // Trend factor
    const trend = CostCalculator.calculateTrend(dailyCosts);
    let trendMultiplier = 1.0;
    
    if (trend === 'increasing') trendMultiplier = 1.15;
    else if (trend === 'decreasing') trendMultiplier = 0.90;
    
    return avgDailyCost * daysInMonth * trendMultiplier;
  },

  // Generate optimization recommendations
  generateOptimizations: (costAnalysis: CostAnalysis): CostOptimization[] => {
    const optimizations: CostOptimization[] = [];
    
    // Check for expensive services that could be optimized
    const expensiveServices = costAnalysis.topCostServices.filter(s => s.cost > 10); // $10+
    
    for (const service of expensiveServices) {
      if (service.serviceId === 'openai' || service.serviceId === 'anthropic') {
        optimizations.push({
          type: 'usage_optimization',
          title: `Optimize ${service.serviceId.toUpperCase()} usage`,
          description: 'Consider caching responses, reducing token usage, or using smaller models for simple tasks',
          potentialSavings: service.cost * 0.3, // 30% potential savings
          effort: 'medium',
          impact: 'high',
          priority: 1,
          actionable: true
        });
      }
      
      if (service.serviceId === 'google-search' && service.cost > 5) {
        optimizations.push({
          type: 'service_substitution',
          title: 'Consider DuckDuckGo for basic searches',
          description: 'Use free DuckDuckGo for non-critical searches to reduce Google API costs',
          potentialSavings: service.cost * 0.5,
          effort: 'low',
          impact: 'medium',
          priority: 2,
          actionable: true
        });
      }
    }
    
    // General caching recommendation
    if (costAnalysis.currentMonthSpend > 50) {
      optimizations.push({
        type: 'caching',
        title: 'Implement response caching',
        description: 'Cache API responses to reduce redundant calls and costs',
        potentialSavings: costAnalysis.currentMonthSpend * 0.25,
        effort: 'medium',
        impact: 'high',
        priority: 1,
        actionable: true
      });
    }
    
    return optimizations.sort((a, b) => a.priority - b.priority);
  }
};

export function useCostIntelligence(
  userId: string,
  options: {
    budget?: number;
    alertThresholds?: {
      warningPercent: number;
      criticalPercent: number;
    };
  } = {}
): UseCostIntelligenceReturn {
  const { budget, alertThresholds = { warningPercent: 80, criticalPercent: 95 } } = options;

  const [costAnalysis, setCostAnalysis] = useState<CostAnalysis | null>(null);
  const [budgetAlerts, setBudgetAlerts] = useState<BudgetAlert[]>([]);
  const [optimizations, setOptimizations] = useState<CostOptimization[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [userBudget, setUserBudget] = useState<number | null>(budget || null);
  
  const abortControllerRef = useRef<AbortController | null>(null);

  // Fetch cost analysis from API
  const fetchCostAnalysis = useCallback(async (): Promise<void> => {
    try {
      console.log(`💰 Fetching cost analysis for user ${userId}`);
      
      // Abort previous request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
      
      abortControllerRef.current = new AbortController();
      setIsLoading(true);
      
      const apiClient = new ApiClient(
        async () => null,
        () => userId
      );
      const response = await apiClient.get('analytics/cost-analysis');
      
      if (!response.ok) {
        throw new Error(`Cost analysis API error: ${response.status}`);
      }
      
      const data: CostAnalysis = await response.json();
      
      // Generate optimizations
      const generatedOptimizations = CostCalculator.generateOptimizations(data);
      
      // Generate budget alerts
      const alerts = generateBudgetAlerts(data, userBudget, alertThresholds);
      
      setCostAnalysis(data);
      setOptimizations(generatedOptimizations);
      setBudgetAlerts(alerts);
      setError(null);
      
      console.log(`✅ Cost analysis updated: $${(data.currentMonthSpend / 100).toFixed(2)} current spend`);
      
    } catch (err) {
      if (err instanceof Error && err.name !== 'AbortError') {
        console.error('❌ Cost analysis error:', err.message);
        setError(err.message);
      }
    } finally {
      setIsLoading(false);
      abortControllerRef.current = null;
    }
  }, [userId, userBudget, alertThresholds]);

  // Generate budget alerts based on spending and limits
  const generateBudgetAlerts = useCallback((
    analysis: CostAnalysis, 
    budget: number | null, 
    thresholds: { warningPercent: number; criticalPercent: number }
  ): BudgetAlert[] => {
    const alerts: BudgetAlert[] = [];
    
    if (!budget) return alerts;
    
    const currentSpendPercent = (analysis.currentMonthSpend / budget) * 100;
    const projectedSpendPercent = (analysis.projectedMonthlySpend / budget) * 100;
    
    // Current spending alerts
    if (currentSpendPercent >= thresholds.criticalPercent) {
      alerts.push({
        type: 'exceeded_limit',
        severity: 'critical',
        title: 'Budget Exceeded',
        message: `You've spent ${currentSpendPercent.toFixed(1)}% of your monthly budget`,
        threshold: budget * (thresholds.criticalPercent / 100),
        currentValue: analysis.currentMonthSpend,
        recommendation: 'Consider pausing non-essential API usage or increasing your budget'
      });
    } else if (currentSpendPercent >= thresholds.warningPercent) {
      alerts.push({
        type: 'approaching_limit',
        severity: 'warning',
        title: 'Approaching Budget Limit',
        message: `You've spent ${currentSpendPercent.toFixed(1)}% of your monthly budget`,
        threshold: budget * (thresholds.warningPercent / 100),
        currentValue: analysis.currentMonthSpend,
        recommendation: 'Monitor usage closely to avoid exceeding your budget'
      });
    }
    
    // Projected spending alerts
    if (projectedSpendPercent >= thresholds.criticalPercent && analysis.costTrend === 'increasing') {
      alerts.push({
        type: 'unusual_spike',
        severity: 'warning',
        title: 'Projected Overspend',
        message: `At current rate, you'll spend ${projectedSpendPercent.toFixed(1)}% of budget this month`,
        threshold: budget,
        currentValue: analysis.projectedMonthlySpend,
        recommendation: 'Review recent usage patterns and consider optimization'
      });
    }
    
    return alerts;
  }, []);

  // Set user budget
  const setBudget = useCallback(async (amount: number): Promise<void> => {
    try {
      // Save budget to backend (optional)
      const apiClient = new ApiClient(
        async () => null,
        () => userId
      );
      await apiClient.post('analytics/budget', { 
        budget: amount * 100 // Convert to cents
      });
      
      setUserBudget(amount);
      
      // Regenerate alerts with new budget
      if (costAnalysis) {
        const alerts = generateBudgetAlerts(costAnalysis, amount, alertThresholds);
        setBudgetAlerts(alerts);
      }
      
      console.log(`💰 Budget set to $${amount}`);
    } catch (err) {
      console.error('❌ Error setting budget:', err);
      throw new Error('Failed to set budget');
    }
  }, [costAnalysis, alertThresholds, generateBudgetAlerts]);

  // Refetch function
  const refetch = useCallback(async (): Promise<void> => {
    await fetchCostAnalysis();
  }, [fetchCostAnalysis]);

  // Initial fetch
  useEffect(() => {
    fetchCostAnalysis();
  }, [fetchCostAnalysis]);

  // Cleanup
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
    };
  }, []);

  // Derived values
  const costTrend = costAnalysis?.costTrend || null;
  const projectedSpend = costAnalysis?.projectedMonthlySpend ? costAnalysis.projectedMonthlySpend / 100 : null;

  return {
    costAnalysis,
    costTrend,
    projectedSpend,
    budgetAlerts,
    optimizations,
    isLoading,
    error,
    refetch,
    setBudget,
    userBudget
  };
}

// Utility functions for cost formatting and analysis
export const CostUtils = {
  formatCurrency: (cents: number): string => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2
    }).format(cents / 100);
  },

  formatTrend: (trend: string): { label: string; color: string; icon: string } => {
    switch (trend) {
      case 'increasing':
        return { label: 'Increasing', color: 'text-red-600', icon: '📈' };
      case 'decreasing':
        return { label: 'Decreasing', color: 'text-green-600', icon: '📉' };
      case 'stable':
        return { label: 'Stable', color: 'text-blue-600', icon: '➡️' };
      default:
        return { label: 'Unknown', color: 'text-gray-600', icon: '❓' };
    }
  },

  getCostEfficiencyScore: (costPerRequest: number, successRate: number): {
    score: number;
    grade: 'A' | 'B' | 'C' | 'D' | 'F';
    color: string;
  } => {
    // Calculate efficiency based on cost per successful request
    const effectiveCost = costPerRequest / (successRate / 100);
    
    let score = 100;
    if (effectiveCost > 0.10) score -= 20; // > 10 cents per request
    if (effectiveCost > 0.05) score -= 15; // > 5 cents per request
    if (effectiveCost > 0.01) score -= 10; // > 1 cent per request
    if (successRate < 95) score -= (95 - successRate); // Penalty for low success rate
    
    score = Math.max(0, Math.min(100, score));
    
    let grade: 'A' | 'B' | 'C' | 'D' | 'F';
    let color: string;
    
    if (score >= 90) { grade = 'A'; color = 'text-green-600'; }
    else if (score >= 80) { grade = 'B'; color = 'text-blue-600'; }
    else if (score >= 70) { grade = 'C'; color = 'text-yellow-600'; }
    else if (score >= 60) { grade = 'D'; color = 'text-orange-600'; }
    else { grade = 'F'; color = 'text-red-600'; }
    
    return { score, grade, color };
  }
};