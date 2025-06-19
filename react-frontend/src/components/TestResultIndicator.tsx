// 🧪 TEST RESULT INDICATOR
// Beautiful display component for API key test results

import { CheckCircleIcon, ClockIcon, XCircleIcon } from '@heroicons/react/24/outline';
import type { KeyTestResponse } from '../services/vaultService';

interface TestResultIndicatorProps {
  result: KeyTestResponse;
  isLoading?: boolean;
  className?: string;
}

export function TestResultIndicator({ result, isLoading = false, className = '' }: TestResultIndicatorProps) {
  if (isLoading) {
    return (
      <div className={`flex items-center space-x-2 ${className}`}>
        <div className="w-5 h-5 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
        <span className="text-sm text-blue-600 dark:text-blue-400">Testing API key...</span>
      </div>
    );
  }

  const getStatusDisplay = () => {
    if (result.valid) {
      return {
        icon: CheckCircleIcon,
        color: 'text-green-500',
        bgColor: 'bg-green-50 dark:bg-green-900/50',
        borderColor: 'border-green-200 dark:border-green-800',
        textColor: 'text-green-700 dark:text-green-300'
      };
    } else {
      return {
        icon: XCircleIcon,
        color: 'text-red-500',
        bgColor: 'bg-red-50 dark:bg-red-900/50',
        borderColor: 'border-red-200 dark:border-red-800',
        textColor: 'text-red-700 dark:text-red-300'
      };
    }
  };

  const status = getStatusDisplay();
  const StatusIcon = status.icon;

  return (
    <div className={`${status.bgColor} ${status.borderColor} border rounded-lg p-4 ${className}`}>
      <div className="flex items-start space-x-3">
        <div className="flex-shrink-0 mt-0.5">
          <StatusIcon className={`w-5 h-5 ${status.color}`} />
        </div>
        
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between mb-2">
            <h4 className={`font-medium ${status.textColor}`}>
              {result.valid ? 'API Key Valid' : 'API Key Invalid'}
            </h4>
            {result.details?.responseTime && (
              <div className="flex items-center space-x-1 text-xs text-gray-500 dark:text-gray-400">
                <ClockIcon className="w-3 h-3" />
                <span>{result.details.responseTime}ms</span>
              </div>
            )}
          </div>
          
          <p className={`text-sm ${status.textColor} mb-3`}>
            {result.message}
          </p>

          {/* Test Details */}
          {result.details && (
            <div className="space-y-2">
              {/* Backend Validation */}
              {result.details.backendValidation && (
                <div className="flex items-center space-x-2 text-xs">
                  <span className="font-medium text-gray-600 dark:text-gray-300">Backend:</span>
                  <span className={`px-2 py-1 rounded ${ 
                    result.details.backendValidation === 'passed' 
                      ? 'bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300' 
                      : 'bg-red-100 dark:bg-red-900 text-red-700 dark:text-red-300'
                  }`}>
                    {result.details.backendValidation}
                  </span>
                </div>
              )}

              {/* Frontend Test Status */}
              {result.details.frontendTest && (
                <div className="flex items-center space-x-2 text-xs">
                  <span className="font-medium text-gray-600 dark:text-gray-300">Service Test:</span>
                  <span className={`px-2 py-1 rounded ${
                    result.details.frontendTest === 'completed' 
                      ? 'bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300'
                      : result.details.frontendTest === 'failed'
                      ? 'bg-yellow-100 dark:bg-yellow-900 text-yellow-700 dark:text-yellow-300'
                      : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300'
                  }`}>
                    {result.details.frontendTest}
                  </span>
                  {result.details.reason && (
                    <span className="text-gray-500 dark:text-gray-400">
                      ({result.details.reason})
                    </span>
                  )}
                </div>
              )}

              {/* Additional Details */}
              {result.details.testDetails && (
                <div className="mt-2 pt-2 border-t border-gray-200 dark:border-gray-600">
                  <div className="grid grid-cols-2 gap-2 text-xs">
                    {Object.entries(result.details.testDetails).map(([key, value]) => (
                      <div key={key} className="flex justify-between">
                        <span className="text-gray-500 dark:text-gray-400 capitalize">
                          {key.replace(/([A-Z])/g, ' $1').trim()}:
                        </span>
                        <span className="text-gray-700 dark:text-gray-300 font-medium">
                          {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Error Details */}
              {result.details.error && (
                <div className="mt-2 p-2 bg-red-100 dark:bg-red-900/50 border border-red-200 dark:border-red-800 rounded">
                  <p className="text-xs text-red-700 dark:text-red-300">
                    <span className="font-medium">Error:</span> {result.details.error}
                  </p>
                </div>
              )}
            </div>
          )}

          {/* Test Timestamp */}
          <div className="mt-3 pt-2 border-t border-gray-200 dark:border-gray-600">
            <p className="text-xs text-gray-500 dark:text-gray-400">
              Tested: {result.testedAt.toLocaleString()}
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

// Quick status indicator for inline use
interface QuickTestStatusProps {
  isValid: boolean;
  isLoading?: boolean;
  className?: string;
}

export function QuickTestStatus({ isValid, isLoading = false, className = '' }: QuickTestStatusProps) {
  if (isLoading) {
    return (
      <div className={`flex items-center space-x-1 ${className}`}>
        <div className="w-3 h-3 border border-blue-500 border-t-transparent rounded-full animate-spin" />
        <span className="text-xs text-blue-600 dark:text-blue-400">Testing...</span>
      </div>
    );
  }

  return (
    <div className={`flex items-center space-x-1 ${className}`}>
      {isValid ? (
        <>
          <CheckCircleIcon className="w-3 h-3 text-green-500" />
          <span className="text-xs text-green-600 dark:text-green-400">Valid</span>
        </>
      ) : (
        <>
          <XCircleIcon className="w-3 h-3 text-red-500" />
          <span className="text-xs text-red-600 dark:text-red-400">Invalid</span>
        </>
      )}
    </div>
  );
}