// 🧪 KEY TEST BUTTON
// Standalone button component for testing API keys

import { useState } from 'react';
import { BeakerIcon } from '@heroicons/react/24/outline';
import { TestResultIndicator } from './TestResultIndicator';
import type { KeyTestResponse } from '../services/vaultService';

interface KeyTestButtonProps {
  serviceId: string;
  vaultService: any;
  disabled?: boolean;
  className?: string;
  variant?: 'primary' | 'secondary' | 'ghost';
  size?: 'sm' | 'md' | 'lg';
}

export function KeyTestButton({ 
  serviceId, 
  vaultService, 
  disabled = false, 
  className = '',
  variant = 'secondary',
  size = 'md'
}: KeyTestButtonProps) {
  const [isTesting, setIsTesting] = useState(false);
  const [testResult, setTestResult] = useState<KeyTestResponse | null>(null);
  const [showResult, setShowResult] = useState(false);

  const handleTest = async () => {
    setIsTesting(true);
    setTestResult(null);
    setShowResult(false);

    try {
      const result = await vaultService.testKey(serviceId);
      setTestResult(result);
      setShowResult(true);
    } catch (error) {
      setTestResult({
        serviceId,
        valid: false,
        message: `Test failed: ${error instanceof Error ? error.message : 'Unknown error'}`,
        testedAt: new Date()
      });
      setShowResult(true);
    } finally {
      setIsTesting(false);
    }
  };

  const getButtonClasses = () => {
    const baseClasses = 'inline-flex items-center justify-center font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed';
    
    const sizeClasses = {
      sm: 'px-2 py-1 text-xs',
      md: 'px-3 py-2 text-sm',
      lg: 'px-4 py-3 text-base'
    };

    const variantClasses = {
      primary: 'bg-blue-600 hover:bg-blue-700 text-white',
      secondary: 'bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-900 dark:text-white border border-gray-300 dark:border-gray-600',
      ghost: 'hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300'
    };

    return `${baseClasses} ${sizeClasses[size]} ${variantClasses[variant]} ${className}`;
  };

  const iconSize = size === 'sm' ? 'w-3 h-3' : size === 'lg' ? 'w-5 h-5' : 'w-4 h-4';

  return (
    <>
      <button
        onClick={handleTest}
        disabled={disabled || isTesting}
        className={getButtonClasses()}
        title={`Test ${serviceId} API key`}
      >
        {isTesting ? (
          <>
            <div className={`${iconSize} border-2 border-current border-t-transparent rounded-full animate-spin mr-2`} />
            Testing...
          </>
        ) : (
          <>
            <BeakerIcon className={`${iconSize} mr-2`} />
            Test Key
          </>
        )}
      </button>

      {/* Floating Test Result */}
      {showResult && testResult && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" onClick={() => setShowResult(false)}>
          <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md mx-4 max-h-[80vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
            <div className="flex justify-between items-center mb-4">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                API Key Test Results
              </h3>
              <button
                onClick={() => setShowResult(false)}
                className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              >
                ×
              </button>
            </div>
            
            <TestResultIndicator result={testResult} />
            
            <div className="mt-4 flex justify-end">
              <button
                onClick={() => setShowResult(false)}
                className="px-4 py-2 text-gray-600 dark:text-gray-300 hover:text-gray-800 dark:hover:text-white"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}