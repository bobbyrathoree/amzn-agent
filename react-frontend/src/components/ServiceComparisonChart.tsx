// 📊 SERVICE COMPARISON CHART
// Netflix-like service tier comparison

import { CheckIcon, XMarkIcon, StarIcon } from '@heroicons/react/24/outline';
import type { ServiceFallback } from '../types/tools';

interface ServiceComparisonChartProps {
  fallbackHierarchy: ServiceFallback[];
  currentService?: string;
  onServiceSelect?: (serviceId: string) => void;
  className?: string;
}

interface ServiceFeature {
  name: string;
  description: string;
  category: 'quality' | 'limits' | 'features' | 'speed';
}

const FEATURE_DEFINITIONS: Record<string, ServiceFeature> = {
  'high-quality': {
    name: 'High Quality Results',
    description: 'Superior accuracy and relevance',
    category: 'quality'
  },
  'instant-answers': {
    name: 'Instant Answers',
    description: 'Direct answers and knowledge cards',
    category: 'features'
  },
  'knowledge-graph': {
    name: 'Knowledge Graph',
    description: 'Rich entity information and relationships',
    category: 'features'
  },
  'image-results': {
    name: 'Image Results',
    description: 'Visual search results and thumbnails',
    category: 'features'
  },
  'ai-optimized': {
    name: 'AI Optimized',
    description: 'Results curated specifically for AI agents',
    category: 'quality'
  },
  'structured-data': {
    name: 'Structured Data',
    description: 'Rich metadata and structured information',
    category: 'features'
  },
  'real-time-results': {
    name: 'Real-time Results',
    description: 'Up-to-the-minute fresh information',
    category: 'speed'
  },
  'privacy-focused': {
    name: 'Privacy Focused',
    description: 'No tracking or personal data collection',
    category: 'features'
  },
  'no-tracking': {
    name: 'No Tracking',
    description: 'Anonymous searches without data retention',
    category: 'features'
  }
};

export function ServiceComparisonChart({ 
  fallbackHierarchy, 
  currentService,
  onServiceSelect,
  className = '' 
}: ServiceComparisonChartProps) {
  if (!fallbackHierarchy || fallbackHierarchy.length === 0) return null;

  const sortedServices = fallbackHierarchy.sort((a, b) => a.priority - b.priority);
  const allFeatures = extractAllFeatures(fallbackHierarchy);

  return (
    <div className={`bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden ${className}`}>
      <div className="p-4 bg-gray-50 dark:bg-gray-750 border-b border-gray-200 dark:border-gray-700">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
          Service Comparison
        </h3>
        <p className="text-sm text-gray-600 dark:text-gray-300 mt-1">
          Compare features and limitations across different service tiers
        </p>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full">
          {/* Header */}
          <thead className="bg-gray-50 dark:bg-gray-750">
            <tr>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-900 dark:text-white">
                Feature
              </th>
              {sortedServices.map((service) => (
                <th key={service.serviceId} className="px-4 py-3 text-center min-w-[120px]">
                  <div className="flex flex-col items-center space-y-2">
                    <div className={`flex items-center space-x-2 ${
                      currentService === service.serviceId 
                        ? 'text-blue-600 dark:text-blue-400' 
                        : 'text-gray-900 dark:text-white'
                    }`}>
                      {service.mode === 'premium' && (
                        <StarIcon className="w-4 h-4 text-yellow-500" />
                      )}
                      <span className="font-medium capitalize">
                        {service.serviceId.replace('-', ' ')}
                      </span>
                    </div>
                    
                    <div className="flex items-center space-x-2">
                      <span className={`px-2 py-1 text-xs rounded-full ${
                        service.mode === 'premium'
                          ? 'bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200'
                          : 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200'
                      }`}>
                        {service.mode === 'premium' ? 'Premium' : 'Free'}
                      </span>
                      
                      {service.requirements?.apiKey && (
                        <span className="px-2 py-1 text-xs bg-orange-100 dark:bg-orange-900 text-orange-800 dark:text-orange-200 rounded-full">
                          API Key
                        </span>
                      )}
                    </div>

                    {currentService === service.serviceId && (
                      <span className="px-2 py-1 text-xs bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 rounded-full">
                        Current
                      </span>
                    )}

                    {currentService !== service.serviceId && onServiceSelect && (
                      <button
                        onClick={() => onServiceSelect(service.serviceId)}
                        className="px-3 py-1 text-xs bg-blue-600 hover:bg-blue-700 text-white rounded-md transition-colors"
                      >
                        Select
                      </button>
                    )}
                  </div>
                </th>
              ))}
            </tr>
          </thead>

          <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
            {/* Quality Row */}
            <tr className="bg-white dark:bg-gray-800">
              <td className="px-4 py-3 text-sm font-medium text-gray-900 dark:text-white">
                Result Quality
              </td>
              {sortedServices.map((service) => (
                <td key={service.serviceId} className="px-4 py-3 text-center">
                  <QualityIndicator quality={service.limitations?.quality || 'medium'} />
                </td>
              ))}
            </tr>

            {/* Max Results Row */}
            <tr className="bg-gray-50 dark:bg-gray-750">
              <td className="px-4 py-3 text-sm font-medium text-gray-900 dark:text-white">
                Max Results
              </td>
              {sortedServices.map((service) => (
                <td key={service.serviceId} className="px-4 py-3 text-center text-sm">
                  <span className="font-medium text-gray-900 dark:text-white">
                    {service.limitations?.maxResults || '∞'}
                  </span>
                </td>
              ))}
            </tr>

            {/* Features */}
            {allFeatures.map((featureKey) => {
              const feature = FEATURE_DEFINITIONS[featureKey];
              if (!feature) return null;

              return (
                <tr key={featureKey} className="bg-white dark:bg-gray-800">
                  <td className="px-4 py-3">
                    <div className="flex flex-col">
                      <span className="text-sm font-medium text-gray-900 dark:text-white">
                        {feature.name}
                      </span>
                      <span className="text-xs text-gray-500 dark:text-gray-400">
                        {feature.description}
                      </span>
                    </div>
                  </td>
                  {sortedServices.map((service) => (
                    <td key={service.serviceId} className="px-4 py-3 text-center">
                      <FeatureSupport 
                        supported={service.limitations?.features?.includes(featureKey) || false}
                      />
                    </td>
                  ))}
                </tr>
              );
            })}

            {/* Cost Row */}
            <tr className="bg-gray-50 dark:bg-gray-750">
              <td className="px-4 py-3 text-sm font-medium text-gray-900 dark:text-white">
                Cost per Query
              </td>
              {sortedServices.map((service) => (
                <td key={service.serviceId} className="px-4 py-3 text-center text-sm">
                  <span className="font-medium text-gray-900 dark:text-white">
                    {service.mode === 'free' ? 'Free' : `${service.requirements?.credits || 1} credit${(service.requirements?.credits || 1) > 1 ? 's' : ''}`}
                  </span>
                </td>
              ))}
            </tr>
          </tbody>
        </table>
      </div>

      {/* Footer Note */}
      <div className="p-4 bg-gray-50 dark:bg-gray-750 border-t border-gray-200 dark:border-gray-700">
        <p className="text-xs text-gray-600 dark:text-gray-300">
          💡 Premium services automatically fallback to free alternatives when API keys are unavailable
        </p>
      </div>
    </div>
  );
}

function QualityIndicator({ quality }: { quality: 'high' | 'medium' | 'low' }) {
  const configs = {
    high: { 
      color: 'text-green-600 dark:text-green-400', 
      bgColor: 'bg-green-100 dark:bg-green-900',
      label: 'High' 
    },
    medium: { 
      color: 'text-yellow-600 dark:text-yellow-400', 
      bgColor: 'bg-yellow-100 dark:bg-yellow-900',
      label: 'Medium' 
    },
    low: { 
      color: 'text-red-600 dark:text-red-400', 
      bgColor: 'bg-red-100 dark:bg-red-900',
      label: 'Low' 
    }
  };

  const config = configs[quality];

  return (
    <span className={`inline-flex items-center px-2 py-1 text-xs font-medium rounded-full ${config.bgColor} ${config.color}`}>
      {config.label}
    </span>
  );
}

function FeatureSupport({ supported }: { supported: boolean }) {
  return (
    <div className="flex justify-center">
      {supported ? (
        <CheckIcon className="w-5 h-5 text-green-500" />
      ) : (
        <XMarkIcon className="w-5 h-5 text-gray-300 dark:text-gray-600" />
      )}
    </div>
  );
}

function extractAllFeatures(hierarchy: ServiceFallback[]): string[] {
  const allFeatures = new Set<string>();
  
  hierarchy.forEach(service => {
    if (service.limitations?.features) {
      service.limitations.features.forEach(feature => {
        allFeatures.add(feature);
      });
    }
  });

  return Array.from(allFeatures).sort();
}