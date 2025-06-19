// 📊 STAT CARD COMPONENT - NETFLIX QUALITY
// Beautiful metric cards with animations and real-time updates

import React from 'react';
import { 
  ArrowTrendingUpIcon as TrendingUpIcon, 
  ArrowTrendingDownIcon as TrendingDownIcon, 
  MinusIcon 
} from '@heroicons/react/24/outline';

interface StatCardProps {
  title: string;
  value: string | number;
  change?: string;
  changeType?: 'positive' | 'negative' | 'neutral';
  icon: React.ComponentType<{ className?: string }>;
  color?: 'blue' | 'green' | 'purple' | 'red' | 'yellow' | 'indigo' | 'pink' | 'emerald';
  isLoading?: boolean;
  onClick?: () => void;
  subtitle?: string;
}

const colorClasses = {
  blue: {
    bg: 'bg-blue-500',
    text: 'text-blue-600',
    lightBg: 'bg-blue-50 dark:bg-blue-900/20',
    ring: 'ring-blue-500/20'
  },
  green: {
    bg: 'bg-green-500',
    text: 'text-green-600',
    lightBg: 'bg-green-50 dark:bg-green-900/20',
    ring: 'ring-green-500/20'
  },
  purple: {
    bg: 'bg-purple-500',
    text: 'text-purple-600',
    lightBg: 'bg-purple-50 dark:bg-purple-900/20',
    ring: 'ring-purple-500/20'
  },
  red: {
    bg: 'bg-red-500',
    text: 'text-red-600',
    lightBg: 'bg-red-50 dark:bg-red-900/20',
    ring: 'ring-red-500/20'
  },
  yellow: {
    bg: 'bg-yellow-500',
    text: 'text-yellow-600',
    lightBg: 'bg-yellow-50 dark:bg-yellow-900/20',
    ring: 'ring-yellow-500/20'
  },
  indigo: {
    bg: 'bg-indigo-500',
    text: 'text-indigo-600',
    lightBg: 'bg-indigo-50 dark:bg-indigo-900/20',
    ring: 'ring-indigo-500/20'
  },
  pink: {
    bg: 'bg-pink-500',
    text: 'text-pink-600',
    lightBg: 'bg-pink-50 dark:bg-pink-900/20',
    ring: 'ring-pink-500/20'
  },
  emerald: {
    bg: 'bg-emerald-500',
    text: 'text-emerald-600',
    lightBg: 'bg-emerald-50 dark:bg-emerald-900/20',
    ring: 'ring-emerald-500/20'
  }
};

const changeTypeClasses = {
  positive: 'text-green-600 dark:text-green-400',
  negative: 'text-red-600 dark:text-red-400',
  neutral: 'text-gray-600 dark:text-gray-400'
};

export function StatCard({
  title,
  value,
  change,
  changeType = 'neutral',
  icon: Icon,
  color = 'blue',
  isLoading = false,
  onClick,
  subtitle
}: StatCardProps) {
  const colors = colorClasses[color];
  const changeColors = changeTypeClasses[changeType];

  const formatValue = (val: string | number): string => {
    if (typeof val === 'number') {
      // Format large numbers with commas
      return val.toLocaleString();
    }
    return val;
  };

  const getTrendIcon = () => {
    switch (changeType) {
      case 'positive':
        return <TrendingUpIcon className="w-4 h-4" />;
      case 'negative':
        return <TrendingDownIcon className="w-4 h-4" />;
      default:
        return <MinusIcon className="w-4 h-4" />;
    }
  };

  return (
    <div
      className={`
        relative overflow-hidden rounded-xl border border-gray-200 dark:border-gray-700
        bg-white dark:bg-gray-800 p-6 transition-all duration-200
        ${onClick ? 'cursor-pointer hover:shadow-lg hover:scale-[1.02] active:scale-[0.98]' : ''}
        ${colors.ring} hover:ring-4 hover:ring-opacity-20
      `}
      onClick={onClick}
    >
      {/* Loading overlay */}
      {isLoading && (
        <div className="absolute inset-0 bg-white/50 dark:bg-gray-800/50 backdrop-blur-sm flex items-center justify-center z-10">
          <div className="animate-spin rounded-full h-6 w-6 border-2 border-gray-300 border-t-blue-600"></div>
        </div>
      )}

      {/* Background decoration */}
      <div className={`absolute top-0 right-0 w-20 h-20 ${colors.lightBg} rounded-full -translate-y-4 translate-x-4 opacity-50`} />
      
      <div className="relative z-10">
        {/* Header */}
        <div className="flex items-center justify-between mb-3">
          <div className={`p-2 ${colors.lightBg} rounded-lg`}>
            <Icon className={`w-6 h-6 ${colors.text}`} />
          </div>
          
          {change && (
            <div className={`flex items-center space-x-1 ${changeColors}`}>
              {getTrendIcon()}
              <span className="text-sm font-medium">{change}</span>
            </div>
          )}
        </div>

        {/* Title */}
        <h3 className="text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">
          {title}
        </h3>

        {/* Value */}
        <div className="flex items-baseline space-x-2">
          <p className="text-2xl font-bold text-gray-900 dark:text-white">
            {formatValue(value)}
          </p>
          
          {subtitle && (
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {subtitle}
            </p>
          )}
        </div>

        {/* Animated pulse for real-time updates */}
        {!isLoading && (
          <div className={`absolute bottom-0 left-0 h-1 ${colors.bg} rounded-full animate-pulse`} 
               style={{ width: '60%' }} />
        )}
      </div>
    </div>
  );
}

// Specialized stat cards for common metrics
export function RequestsStatCard({ 
  requests, 
  change, 
  isLoading = false 
}: { 
  requests: number; 
  change?: string; 
  isLoading?: boolean; 
}) {
  return (
    <StatCard
      title="API Requests"
      value={requests}
      change={change}
      changeType={change?.includes('+') ? 'positive' : change?.includes('-') ? 'negative' : 'neutral'}
      icon={({ className }) => (
        <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
        </svg>
      )}
      color="blue"
      isLoading={isLoading}
    />
  );
}

export function CostStatCard({ 
  cost, 
  change, 
  isLoading = false 
}: { 
  cost: string; 
  change?: string; 
  isLoading?: boolean; 
}) {
  return (
    <StatCard
      title="Total Cost"
      value={cost}
      change={change}
      changeType={change?.includes('↑') ? 'negative' : change?.includes('↓') ? 'positive' : 'neutral'}
      icon={({ className }) => (
        <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1" />
        </svg>
      )}
      color="green"
      isLoading={isLoading}
    />
  );
}

export function SuccessRateStatCard({ 
  successRate, 
  change, 
  isLoading = false 
}: { 
  successRate: number; 
  change?: string; 
  isLoading?: boolean; 
}) {
  const getColor = (rate: number) => {
    if (rate >= 95) return 'green';
    if (rate >= 85) return 'yellow';
    return 'red';
  };

  return (
    <StatCard
      title="Success Rate"
      value={`${successRate.toFixed(1)}%`}
      change={change}
      changeType={successRate >= 95 ? 'positive' : successRate >= 85 ? 'neutral' : 'negative'}
      icon={({ className }) => (
        <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      )}
      color={getColor(successRate)}
      isLoading={isLoading}
    />
  );
}

export function ResponseTimeStatCard({ 
  responseTime, 
  change, 
  isLoading = false 
}: { 
  responseTime: number; 
  change?: string; 
  isLoading?: boolean; 
}) {
  const getColor = (time: number) => {
    if (time < 1000) return 'green';
    if (time < 3000) return 'yellow';
    return 'red';
  };

  return (
    <StatCard
      title="Avg Response Time"
      value={`${responseTime.toFixed(0)}ms`}
      change={change}
      changeType={responseTime < 1000 ? 'positive' : responseTime < 3000 ? 'neutral' : 'negative'}
      icon={({ className }) => (
        <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
        </svg>
      )}
      color={getColor(responseTime)}
      isLoading={isLoading}
    />
  );
}