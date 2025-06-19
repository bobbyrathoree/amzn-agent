// 🔧 SERVICE MODE INDICATOR
// Netflix-like mode indicator showing Premium vs Free mode

import { useState } from 'react';
import { 
  ShieldCheckIcon, 
  ExclamationTriangleIcon,
  ArrowUpIcon,
  SparklesIcon,
  LockClosedIcon,
  StarIcon
} from '@heroicons/react/24/outline';
import type { ToolResult } from '../types/tools';

interface ServiceModeIndicatorProps {
  toolResult?: ToolResult;
  className?: string;
  showUpgradePrompt?: boolean;
  onUpgradeClick?: () => void;
}

export function ServiceModeIndicator({ 
  toolResult, 
  className = '', 
  showUpgradePrompt = true,
  onUpgradeClick 
}: ServiceModeIndicatorProps) {
  const [showUpgradeModal, setShowUpgradeModal] = useState(false);

  if (!toolResult?.serviceInfo) {
    return null;
  }

  const { serviceInfo } = toolResult;
  const isPremium = serviceInfo.serviceMode === 'premium';
  const isFallback = serviceInfo.fallbackLevel > 0;

  const getModeDisplay = () => {
    if (isPremium && !isFallback) {
      return {
        icon: StarIcon,
        label: 'Premium Mode',
        color: 'text-yellow-600 dark:text-yellow-400',
        bgColor: 'bg-yellow-50 dark:bg-yellow-900/50',
        borderColor: 'border-yellow-200 dark:border-yellow-800',
        description: `Using premium ${serviceInfo.usedService} for best quality results`
      };
    } else if (isPremium && isFallback) {
      return {
        icon: ShieldCheckIcon,
        label: 'Premium Fallback',
        color: 'text-blue-600 dark:text-blue-400',
        bgColor: 'bg-blue-50 dark:bg-blue-900/50',
        borderColor: 'border-blue-200 dark:border-blue-800',
        description: `Using ${serviceInfo.usedService} (fallback level ${serviceInfo.fallbackLevel})`
      };
    } else {
      return {
        icon: ExclamationTriangleIcon,
        label: 'Free Mode',
        color: 'text-orange-600 dark:text-orange-400',
        bgColor: 'bg-orange-50 dark:bg-orange-900/50',
        borderColor: 'border-orange-200 dark:border-orange-800',
        description: `Using free ${serviceInfo.usedService} with limited features`
      };
    }
  };

  const mode = getModeDisplay();
  const ModeIcon = mode.icon;

  return (
    <>
      <div className={`inline-flex items-center space-x-2 px-3 py-2 rounded-lg border ${mode.bgColor} ${mode.borderColor} ${className}`}>
        <ModeIcon className={`w-4 h-4 ${mode.color}`} />
        <div className="flex flex-col">
          <span className={`text-sm font-medium ${mode.color}`}>
            {mode.label}
          </span>
          <span className="text-xs text-gray-600 dark:text-gray-300">
            {mode.description}
          </span>
        </div>

        {/* Quality Indicator */}
        {serviceInfo.limitations && (
          <div className="flex items-center space-x-1">
            <div className={`w-2 h-2 rounded-full ${
              serviceInfo.limitations.quality === 'high' 
                ? 'bg-green-500' 
                : serviceInfo.limitations.quality === 'medium'
                ? 'bg-yellow-500'
                : 'bg-red-500'
            }`} />
            <span className="text-xs text-gray-500 dark:text-gray-400 capitalize">
              {serviceInfo.limitations.quality} Quality
            </span>
          </div>
        )}

        {/* Upgrade Button */}
        {showUpgradePrompt && serviceInfo.availableUpgrades && serviceInfo.availableUpgrades.length > 0 && (
          <button
            onClick={() => setShowUpgradeModal(true)}
            className="ml-2 px-2 py-1 text-xs bg-blue-600 hover:bg-blue-700 text-white rounded-md flex items-center space-x-1 transition-colors"
          >
            <ArrowUpIcon className="w-3 h-3" />
            <span>Upgrade</span>
          </button>
        )}
      </div>

      {/* Service Limitations */}
      {serviceInfo.limitations && (
        <div className="mt-2 text-xs text-gray-500 dark:text-gray-400">
          {serviceInfo.limitations.maxResults && (
            <span className="mr-3">
              📊 Max {serviceInfo.limitations.maxResults} results
            </span>
          )}
          {serviceInfo.limitations.features && serviceInfo.limitations.features.length > 0 && (
            <span>
              ✨ {serviceInfo.limitations.features.slice(0, 2).join(', ')}
              {serviceInfo.limitations.features.length > 2 && ` +${serviceInfo.limitations.features.length - 2} more`}
            </span>
          )}
        </div>
      )}

      {/* Upgrade Modal */}
      {showUpgradeModal && serviceInfo.availableUpgrades && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md mx-4">
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-blue-100 dark:bg-blue-900 rounded-lg">
                <SparklesIcon className="w-6 h-6 text-blue-600 dark:text-blue-400" />
              </div>
              <div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                  Upgrade Available
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-300">
                  Get better results with premium services
                </p>
              </div>
            </div>

            <div className="space-y-3 mb-6">
              {serviceInfo.availableUpgrades.map((upgrade, index) => (
                <div key={index} className="border border-gray-200 dark:border-gray-700 rounded-lg p-3">
                  <div className="flex items-center justify-between mb-2">
                    <h4 className="font-medium text-gray-900 dark:text-white capitalize">
                      {upgrade.serviceId.replace('-', ' ')}
                    </h4>
                    {upgrade.setupRequired && (
                      <span className="text-xs px-2 py-1 bg-orange-100 dark:bg-orange-900 text-orange-700 dark:text-orange-300 rounded">
                        Setup Required
                      </span>
                    )}
                  </div>
                  <ul className="text-sm text-gray-600 dark:text-gray-300 space-y-1">
                    {upgrade.benefits.map((benefit, benefitIndex) => (
                      <li key={benefitIndex} className="flex items-center space-x-2">
                        <span className="text-green-500">✓</span>
                        <span>{benefit}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>

            <div className="flex space-x-3">
              <button
                onClick={() => setShowUpgradeModal(false)}
                className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700"
              >
                Maybe Later
              </button>
              <button
                onClick={() => {
                  setShowUpgradeModal(false);
                  onUpgradeClick?.();
                }}
                className="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg flex items-center justify-center space-x-2"
              >
                <StarIcon className="w-4 h-4" />
                <span>Upgrade Now</span>
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

// Quick inline service mode badge
interface ServiceModeBadgeProps {
  serviceMode: 'premium' | 'free';
  fallbackLevel?: number;
  className?: string;
}

export function ServiceModeBadge({ serviceMode, fallbackLevel = 0, className = '' }: ServiceModeBadgeProps) {
  const isPremium = serviceMode === 'premium';
  const isFallback = fallbackLevel > 0;

  if (isPremium && !isFallback) {
    return (
      <span className={`inline-flex items-center space-x-1 px-2 py-1 text-xs font-medium bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 rounded-full ${className}`}>
        <StarIcon className="w-3 h-3" />
        <span>Premium</span>
      </span>
    );
  } else if (isPremium && isFallback) {
    return (
      <span className={`inline-flex items-center space-x-1 px-2 py-1 text-xs font-medium bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 rounded-full ${className}`}>
        <ShieldCheckIcon className="w-3 h-3" />
        <span>Premium Fallback</span>
      </span>
    );
  } else {
    return (
      <span className={`inline-flex items-center space-x-1 px-2 py-1 text-xs font-medium bg-orange-100 dark:bg-orange-900 text-orange-800 dark:text-orange-200 rounded-full ${className}`}>
        <LockClosedIcon className="w-3 h-3" />
        <span>Free</span>
      </span>
    );
  }
}