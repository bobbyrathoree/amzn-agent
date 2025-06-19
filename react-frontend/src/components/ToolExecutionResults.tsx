// 🛠️ TOOL EXECUTION RESULTS
// Display tool execution status and results

import { CheckCircleIcon, ExclamationTriangleIcon, ClockIcon, KeyIcon } from '@heroicons/react/24/outline';
import type { ToolExecution, ToolSkipped } from '../types';

interface ToolExecutionResultsProps {
  toolsExecuted: ToolExecution[];
  toolsSkipped: ToolSkipped[];
  className?: string;
}

export function ToolExecutionResults({ toolsExecuted, toolsSkipped, className = '' }: ToolExecutionResultsProps) {
  if (toolsExecuted.length === 0 && toolsSkipped.length === 0) {
    return null;
  }

  return (
    <div className={`space-y-3 ${className}`}>
      {/* Successfully Executed Tools */}
      {toolsExecuted.map((execution, index) => (
        <div
          key={`executed-${index}`}
          className="flex items-center space-x-3 p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg"
        >
          <CheckCircleIcon className="w-5 h-5 text-green-600 dark:text-green-400 flex-shrink-0" />
          <div className="flex-1 min-w-0">
            <div className="flex items-center space-x-2">
              <h4 className="text-sm font-medium text-green-800 dark:text-green-200">
                🛠️ {execution.toolName}
              </h4>
              {execution.usedApiKey && (
                <span className="inline-flex items-center space-x-1 px-2 py-1 bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 text-xs rounded-full">
                  <KeyIcon className="w-3 h-3" />
                  <span>Premium</span>
                </span>
              )}
              {execution.engine && (
                <span className="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 rounded">
                  {execution.engine}
                </span>
              )}
            </div>
            <div className="flex items-center space-x-4 mt-1 text-xs text-green-600 dark:text-green-400">
              <div className="flex items-center space-x-1">
                <ClockIcon className="w-3 h-3" />
                <span>{execution.executionTime}ms</span>
              </div>
              {execution.resultCount && (
                <span>{execution.resultCount} results</span>
              )}
              <span className="capitalize">{execution.capability}</span>
            </div>
          </div>
        </div>
      ))}

      {/* Skipped Tools */}
      {toolsSkipped.map((skipped, index) => (
        <div
          key={`skipped-${index}`}
          className="flex items-center space-x-3 p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg"
        >
          <ExclamationTriangleIcon className="w-5 h-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0" />
          <div className="flex-1 min-w-0">
            <h4 className="text-sm font-medium text-yellow-800 dark:text-yellow-200">
              ⏸️ {skipped.toolName} (Skipped)
            </h4>
            <p className="text-xs text-yellow-600 dark:text-yellow-400 mt-1">
              {skipped.reason}
              {skipped.fallbackUsed && ' • Used fallback'}
            </p>
            {skipped.requiredService && (
              <p className="text-xs text-yellow-500 dark:text-yellow-300 mt-1">
                Requires: {skipped.requiredService}
              </p>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}