// 🛠️ GENERIC TOOL RESULT DISPLAY
// Beautiful fallback for any tool result

import { useState } from 'react';
import { 
  ChevronDownIcon,
  ChevronUpIcon,
  ClockIcon,
  KeyIcon,
  CheckCircleIcon,
  DocumentTextIcon
} from '@heroicons/react/24/outline';

interface GenericToolResultProps {
  toolName: string;
  toolId: string;
  capability: string;
  data: any;
  executionTime: number;
  usedApiKey: boolean;
  className?: string;
}

export function GenericToolResult({
  toolName,
  toolId,
  capability,
  data,
  executionTime,
  usedApiKey,
  className = ''
}: GenericToolResultProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [showRawData, setShowRawData] = useState(false);

  const getToolIcon = (toolId: string) => {
    switch (toolId) {
      case 'file-processor': return '📁';
      case 'data-analyzer': return '📊';
      case 'code-executor': return '💻';
      case 'image-generator': return '🎨';
      case 'translator': return '🌐';
      default: return '🛠️';
    }
  };

  const formatDataForDisplay = (data: any): { summary: string; details: any[] } => {
    if (!data || typeof data !== 'object') {
      return {
        summary: String(data || 'No data returned'),
        details: []
      };
    }

    // Extract meaningful information from the data
    const summary = data.summary || data.result || data.message || 'Tool execution completed';
    const details: any[] = [];

    // Add key-value pairs that might be interesting
    Object.entries(data).forEach(([key, value]) => {
      if (key !== 'summary' && key !== 'result' && key !== 'message') {
        details.push({ key, value });
      }
    });

    return { summary, details };
  };

  const { summary, details } = formatDataForDisplay(data);

  return (
    <div className={`bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden ${className}`}>
      {/* Header */}
      <div className="bg-gradient-to-r from-gray-50 to-blue-50 dark:from-gray-900/50 dark:to-blue-900/20 p-4 border-b border-gray-200 dark:border-gray-700">
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="w-full flex items-center justify-between text-left"
        >
          <div className="flex items-center space-x-3">
            <span className="text-2xl">{getToolIcon(toolId)}</span>
            <div>
              <h3 className="font-semibold text-gray-900 dark:text-white">
                {toolName}
              </h3>
              <div className="flex items-center space-x-2 mt-1">
                <span className="text-sm text-gray-600 dark:text-gray-300 capitalize">
                  {capability}
                </span>
                {usedApiKey && (
                  <span className="inline-flex items-center space-x-1 px-2 py-1 bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 text-xs rounded-full">
                    <KeyIcon className="w-3 h-3" />
                    <span>Premium</span>
                  </span>
                )}
                <span className="inline-flex items-center space-x-1 px-2 py-1 bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200 text-xs rounded-full">
                  <CheckCircleIcon className="w-3 h-3" />
                  <span>Success</span>
                </span>
              </div>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <div className="text-right text-xs text-gray-500 dark:text-gray-400">
              <div className="flex items-center space-x-1">
                <ClockIcon className="w-3 h-3" />
                <span>{executionTime}ms</span>
              </div>
            </div>
            {isExpanded ? (
              <ChevronUpIcon className="w-5 h-5 text-gray-400" />
            ) : (
              <ChevronDownIcon className="w-5 h-5 text-gray-400" />
            )}
          </div>
        </button>
      </div>

      {/* Content */}
      {isExpanded && (
        <div className="p-4">
          {/* Summary */}
          <div className="mb-4">
            <h4 className="font-medium text-gray-900 dark:text-white mb-2">Result</h4>
            <div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-3">
              <p className="text-sm text-gray-700 dark:text-gray-300">
                {summary}
              </p>
            </div>
          </div>

          {/* Details */}
          {details.length > 0 && (
            <div className="mb-4">
              <h4 className="font-medium text-gray-900 dark:text-white mb-2">Details</h4>
              <div className="space-y-2">
                {details.slice(0, 5).map(({ key, value }, index) => (
                  <div key={index} className="flex items-start justify-between py-2 border-b border-gray-200 dark:border-gray-600 last:border-b-0">
                    <span className="text-sm font-medium text-gray-600 dark:text-gray-400 capitalize">
                      {key.replace(/([A-Z])/g, ' $1').replace(/^./, (str: string) => str.toUpperCase())}:
                    </span>
                    <span className="text-sm text-gray-900 dark:text-white ml-4 flex-1 text-right">
                      {typeof value === 'object' ? 
                        JSON.stringify(value).substring(0, 100) + (JSON.stringify(value).length > 100 ? '...' : '') :
                        String(value)
                      }
                    </span>
                  </div>
                ))}
                {details.length > 5 && (
                  <button
                    onClick={() => setShowRawData(!showRawData)}
                    className="text-sm text-blue-600 dark:text-blue-400 hover:underline"
                  >
                    {showRawData ? 'Hide' : 'Show'} all data ({details.length} items)
                  </button>
                )}
              </div>
            </div>
          )}

          {/* Raw Data (if requested) */}
          {showRawData && (
            <div>
              <div className="flex items-center space-x-2 mb-2">
                <DocumentTextIcon className="w-4 h-4 text-gray-500" />
                <h4 className="font-medium text-gray-900 dark:text-white">Raw Data</h4>
              </div>
              <div className="bg-gray-900 dark:bg-gray-800 rounded-lg p-3 overflow-x-auto">
                <pre className="text-xs text-green-400 font-mono">
                  {JSON.stringify(data, null, 2)}
                </pre>
              </div>
            </div>
          )}

          {/* Footer */}
          <div className="mt-4 pt-3 border-t border-gray-200 dark:border-gray-600">
            <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
              <span>Tool ID: {toolId}</span>
              <span>Executed at {new Date().toLocaleTimeString()}</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}