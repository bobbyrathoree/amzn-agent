// 🔑 KEY RECOMMENDATIONS
// Non-blocking suggestions for API key upgrades

import { useState } from 'react';
import { KeyIcon, ChevronDownIcon, ChevronUpIcon, ArrowTopRightOnSquareIcon, SparklesIcon } from '@heroicons/react/24/outline';
import type { KeyRecommendation } from '../types';

interface KeyRecommendationsProps {
  recommendations: KeyRecommendation[];
  onAddKey?: (serviceId: string) => void;
  className?: string;
}

export function KeyRecommendations({ recommendations, onAddKey, className = '' }: KeyRecommendationsProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  if (recommendations.length === 0) {
    return null;
  }

  // Sort by priority: high -> medium -> low
  const sortedRecommendations = [...recommendations].sort((a, b) => {
    const priorityOrder = { high: 3, medium: 2, low: 1 };
    return priorityOrder[b.priority] - priorityOrder[a.priority];
  });

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'high': return 'text-orange-600 dark:text-orange-400 bg-orange-100 dark:bg-orange-900';
      case 'medium': return 'text-blue-600 dark:text-blue-400 bg-blue-100 dark:bg-blue-900';
      case 'low': return 'text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-700';
      default: return 'text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-700';
    }
  };

  return (
    <div className={`bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 border border-blue-200 dark:border-blue-800 rounded-lg ${className}`}>
      {/* Header */}
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full flex items-center justify-between p-4 text-left hover:bg-blue-100/50 dark:hover:bg-blue-800/20 rounded-lg transition-colors"
      >
        <div className="flex items-center space-x-3">
          <SparklesIcon className="w-5 h-5 text-blue-600 dark:text-blue-400" />
          <div>
            <h3 className="font-medium text-blue-900 dark:text-blue-100">
              ✨ Unlock Premium Features
            </h3>
            <p className="text-sm text-blue-700 dark:text-blue-300">
              {recommendations.length} enhancement{recommendations.length !== 1 ? 's' : ''} available
            </p>
          </div>
        </div>
        {isExpanded ? (
          <ChevronUpIcon className="w-5 h-5 text-blue-500" />
        ) : (
          <ChevronDownIcon className="w-5 h-5 text-blue-500" />
        )}
      </button>

      {/* Recommendations List */}
      {isExpanded && (
        <div className="px-4 pb-4 space-y-3">
          {sortedRecommendations.map((rec, index) => (
            <div
              key={`${rec.serviceId}-${index}`}
              className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-lg p-4"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center space-x-2 mb-2">
                    <KeyIcon className="w-4 h-4 text-gray-500" />
                    <h4 className="font-medium text-gray-900 dark:text-white">
                      {rec.serviceName}
                    </h4>
                    <span className={`text-xs px-2 py-1 rounded-full ${getPriorityColor(rec.priority)}`}>
                      {rec.priority} priority
                    </span>
                  </div>
                  
                  <p className="text-sm text-gray-600 dark:text-gray-300 mb-3">
                    {rec.description}
                  </p>

                  {/* Benefits */}
                  {rec.benefits.length > 0 && (
                    <div className="mb-3">
                      <p className="text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                        Benefits:
                      </p>
                      <ul className="text-xs text-gray-600 dark:text-gray-400 space-y-1">
                        {rec.benefits.map((benefit, idx) => (
                          <li key={idx} className="flex items-start space-x-2">
                            <span className="text-green-500 mt-0.5">•</span>
                            <span>{benefit}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}

                  {/* Pricing */}
                  {rec.pricingInfo && (
                    <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">
                      💰 {rec.pricingInfo}
                    </p>
                  )}
                </div>
              </div>

              {/* Actions */}
              <div className="flex items-center space-x-3 mt-3 pt-3 border-t border-gray-200 dark:border-gray-600">
                {onAddKey && (
                  <button
                    onClick={() => onAddKey(rec.serviceId)}
                    className="flex items-center space-x-2 px-3 py-2 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700 transition-colors"
                  >
                    <KeyIcon className="w-4 h-4" />
                    <span>Add API Key</span>
                  </button>
                )}
                
                {rec.signupUrl && (
                  <a
                    href={rec.signupUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center space-x-2 px-3 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 text-sm rounded-md hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                  >
                    <ArrowTopRightOnSquareIcon className="w-4 h-4" />
                    <span>Sign Up</span>
                  </a>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}