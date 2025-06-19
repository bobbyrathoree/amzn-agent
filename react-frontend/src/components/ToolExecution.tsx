// ⚡ TOOL EXECUTION COMPONENT
// Seamless tool execution within chat conversations

import { useState, useEffect } from 'react';
import { CheckCircleIcon, ExclamationTriangleIcon, KeyIcon, PlayIcon } from '@heroicons/react/24/outline';
import { toolExecutor } from '../services/toolExecutor';
import { APIKeySetupModal } from './APIKeySetupModal';
import { ServiceModeIndicator } from './ServiceModeIndicator';
import type { UniversalTool, ToolResult, ToolProgress } from '../types/tools';
import type { User } from '../types';

interface ToolExecutionProps {
  tool: UniversalTool;
  capability: string;
  input: any;
  user: User;
  vaultService: any;
  onComplete: (result: ToolResult) => void;
  onCancel: () => void;
}

export function ToolExecution({ 
  tool, 
  capability, 
  input, 
  user, 
  vaultService, 
  onComplete, 
  onCancel 
}: ToolExecutionProps) {
  const [isExecuting, setIsExecuting] = useState(false);
  const [progress, setProgress] = useState<ToolProgress[]>([]);
  const [result, setResult] = useState<ToolResult | null>(null);
  const [showKeySetup, setShowKeySetup] = useState(false);
  const [missingService, setMissingService] = useState<string>('');
  const [error, setError] = useState<string>('');

  useEffect(() => {
    executeToolAutomatically();
  }, []);

  const executeToolAutomatically = async () => {
    setIsExecuting(true);
    setError('');
    setProgress([]);

    try {
      const executionResult = await toolExecutor.execute({
        toolId: tool.id,
        capability,
        input,
        context: {
          userId: user.userId,
          permissions: ['internet-access:read', 'external-api:read'],
          requestId: `req_${Date.now()}`,
          timestamp: new Date()
        }
      });

      setResult(executionResult);

      // Handle API key required error
      if (!executionResult.success && executionResult.error?.code === 'API_KEY_REQUIRED') {
        const missing = executionResult.error.details?.missingServices?.[0];
        if (missing) {
          setMissingService(missing);
          setShowKeySetup(true);
        }
      } else {
        // Tool executed successfully or with different error
        onComplete(executionResult);
      }

    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Tool execution failed';
      setError(errorMsg);
      setResult({
        success: false,
        error: {
          code: 'EXECUTION_ERROR',
          message: errorMsg
        }
      });
    } finally {
      setIsExecuting(false);
    }
  };

  const handleRetryWithKeys = async () => {
    setShowKeySetup(false);
    // Retry execution after API key setup
    await executeToolAutomatically();
  };

  const getCapabilityInfo = () => {
    return tool.capabilities.find(cap => cap.name === capability);
  };

  const formatInputDisplay = () => {
    if (typeof input === 'string') return input;
    if (input.query) return `"${input.query}"`;
    if (input.topic) return `"${input.topic}"`;
    return JSON.stringify(input);
  };

  const capabilityInfo = getCapabilityInfo();
  const serviceName = tool.apiRequirements?.[missingService]?.keyName || missingService;

  return (
    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4 my-4">
      {/* Tool Header */}
      <div className="flex items-center space-x-3 mb-4">
        <div className="flex-shrink-0">
          <span className="text-2xl">{tool.icon}</span>
        </div>
        <div className="flex-1">
          <div className="flex items-center space-x-2">
            <h3 className="font-medium text-gray-900 dark:text-white">
              {tool.name}
            </h3>
            <span className="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 rounded">
              {capability}
            </span>
            {tool.apiRequirements && (
              <span className="text-xs px-2 py-1 bg-purple-100 dark:bg-purple-900 text-purple-800 dark:text-purple-200 rounded">
                Premium Features
              </span>
            )}
          </div>
          <p className="text-sm text-gray-600 dark:text-gray-300">
            {formatInputDisplay()}
          </p>
        </div>
        <div className="flex items-center space-x-2">
          {capabilityInfo?.timeEstimate && (
            <span className="text-xs text-gray-500">
              ~{Math.round(capabilityInfo.timeEstimate / 1000)}s
            </span>
          )}
          <button
            onClick={onCancel}
            className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
          >
            ×
          </button>
        </div>
      </div>

      {/* Execution Status */}
      {isExecuting && (
        <div className="mb-4">
          <div className="flex items-center space-x-2 mb-2">
            <PlayIcon className="w-4 h-4 text-blue-500 animate-pulse" />
            <span className="text-sm text-blue-600 dark:text-blue-400 font-medium">
              Executing tool...
            </span>
          </div>
          
          {progress.length > 0 && (
            <div className="space-y-2">
              {progress.map((p, index) => (
                <div key={index} className="flex items-center space-x-3">
                  <div className="w-2 h-2 bg-blue-500 rounded-full animate-pulse" />
                  <span className="text-sm text-gray-600 dark:text-gray-300">
                    {p.message}
                  </span>
                  <span className="text-xs text-gray-500">
                    {p.progress}%
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Results */}
      {result && !isExecuting && (
        <div className={`rounded-lg p-4 ${
          result.success 
            ? 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800'
            : 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800'
        }`}>
          <div className="flex items-start space-x-3">
            <div className="flex-shrink-0 mt-0.5">
              {result.success ? (
                <CheckCircleIcon className="w-5 h-5 text-green-500" />
              ) : (
                <ExclamationTriangleIcon className="w-5 h-5 text-red-500" />
              )}
            </div>
            <div className="flex-1">
              {result.success ? (
                <div>
                  <h4 className="font-medium text-green-800 dark:text-green-200 mb-2">
                    Tool executed successfully!
                  </h4>
                  
                  {/* Web Search Results */}
                  {tool.id === 'web-search' && result.data?.results && (
                    <div className="space-y-3">
                      <p className="text-sm text-green-700 dark:text-green-300">
                        Found {result.data.results.length} results in {result.data.searchTime}ms using {result.data.engine}
                      </p>
                      <div className="space-y-2">
                        {result.data.results.slice(0, 3).map((item: any, index: number) => (
                          <div key={index} className="bg-white dark:bg-gray-800 p-3 rounded border">
                            <a 
                              href={item.url} 
                              target="_blank" 
                              rel="noopener noreferrer"
                              className="font-medium text-blue-600 hover:text-blue-700 dark:text-blue-400 block mb-1"
                            >
                              {item.title}
                            </a>
                            <p className="text-sm text-gray-600 dark:text-gray-300 mb-1">
                              {item.snippet}
                            </p>
                            <span className="text-xs text-gray-500">
                              {item.displayUrl} • Rank {item.rank}
                            </span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Research Results */}
                  {tool.id === 'research-assistant' && result.data?.keyFindings && (
                    <div className="space-y-3">
                      <p className="text-sm text-green-700 dark:text-green-300">
                        Research completed with {result.data.totalSources} sources ({Math.round(result.data.confidence * 100)}% confidence)
                      </p>
                      <div>
                        <h5 className="font-medium text-green-800 dark:text-green-200 mb-2">Key Findings:</h5>
                        <ul className="list-disc list-inside space-y-1 text-sm text-green-700 dark:text-green-300">
                          {result.data.keyFindings.slice(0, 3).map((finding: string, index: number) => (
                            <li key={index}>{finding}</li>
                          ))}
                        </ul>
                      </div>
                    </div>
                  )}

                  <div className="mt-3 space-y-2">
                    {/* Service Mode Indicator */}
                    <ServiceModeIndicator 
                      toolResult={result} 
                      showUpgradePrompt={true}
                      onUpgradeClick={() => setShowKeySetup(true)}
                    />
                    
                    {/* Execution Metadata */}
                    <div className="flex items-center space-x-4 text-xs text-green-600 dark:text-green-400">
                      {result.metadata?.executionTime && (
                        <span>⏱️ {result.metadata.executionTime}ms</span>
                      )}
                      {result.metadata?.cost && (
                        <span>💎 {result.metadata.cost} credits</span>
                      )}
                      {result.citations && (
                        <span>📚 {result.citations.length} sources</span>
                      )}
                    </div>
                  </div>
                </div>
              ) : (
                <div>
                  <h4 className="font-medium text-red-800 dark:text-red-200 mb-2">
                    {result.error?.code === 'API_KEY_REQUIRED' ? (
                      <>
                        <KeyIcon className="w-4 h-4 inline mr-1" />
                        Premium Features Available
                      </>
                    ) : (
                      'Tool execution failed'
                    )}
                  </h4>
                  
                  {result.error?.code === 'API_KEY_REQUIRED' ? (
                    <div>
                      <p className="text-sm text-red-700 dark:text-red-300 mb-3">
                        This tool can provide enhanced results with premium API access. 
                        Add your {serviceName} to unlock advanced features.
                      </p>
                      <button
                        onClick={() => setShowKeySetup(true)}
                        className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 font-medium text-sm"
                      >
                        Setup {serviceName}
                      </button>
                    </div>
                  ) : (
                    <p className="text-sm text-red-700 dark:text-red-300">
                      {result.error?.message}
                    </p>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Error Display */}
      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-3">
          <p className="text-sm text-red-700 dark:text-red-300">
            {error}
          </p>
        </div>
      )}

      {/* API Key Setup Modal */}
      {showKeySetup && tool.apiRequirements && (
        <APIKeySetupModal
          isOpen={showKeySetup}
          onClose={() => setShowKeySetup(false)}
          serviceId={missingService}
          service={{
            id: missingService,
            name: tool.apiRequirements[missingService]?.keyName || missingService,
            category: tool.category,
            description: tool.apiRequirements[missingService]?.description || '',
            website: tool.apiRequirements[missingService]?.signupUrl || '',
            pricing: tool.apiRequirements[missingService]?.pricingInfo || '',
            icon: tool.icon || '🔧',
            color: tool.color || '#6366f1',
            isActive: true,
            setupSteps: tool.apiRequirements[missingService]?.setupInstructions || []
          }}
          onKeyAdded={handleRetryWithKeys}
          vaultService={vaultService}
        />
      )}
    </div>
  );
}