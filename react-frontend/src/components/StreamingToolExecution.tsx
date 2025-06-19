// 🌊 STREAMING TOOL EXECUTION COMPONENT
// Real-time progress indicators for tool execution

import { useState, useEffect, useRef } from 'react';
import { 
  PlayIcon, 
  CheckCircleIcon, 
  ExclamationTriangleIcon,
  ClockIcon,
  ArrowPathIcon,
  SparklesIcon,
  ChevronDownIcon,
  ChevronUpIcon
} from '@heroicons/react/24/outline';
import { toolExecutor } from '../services/toolExecutor';
import type { UniversalTool, ToolProgress, ToolResult, ExecutionContext } from '../types/tools';

interface StreamingToolExecutionProps {
  tool: UniversalTool;
  capability: string;
  input: any;
  context: ExecutionContext;
  onComplete: (result: ToolResult) => void;
  onCancel: () => void;
  autoStart?: boolean;
  className?: string;
}

interface ProgressStep extends ToolProgress {
  id: string;
  timestamp: Date;
  duration?: number;
}

export function StreamingToolExecution({
  tool,
  capability,
  input,
  context,
  onComplete,
  onCancel,
  autoStart = true,
  className = ''
}: StreamingToolExecutionProps) {
  const [isExecuting, setIsExecuting] = useState(false);
  const [progressSteps, setProgressSteps] = useState<ProgressStep[]>([]);
  const [currentStep, setCurrentStep] = useState<ProgressStep | null>(null);
  const [result, setResult] = useState<ToolResult | null>(null);
  const [error, setError] = useState<string>('');
  const [isExpanded, setIsExpanded] = useState(true);
  const [executionStartTime, setExecutionStartTime] = useState<Date | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);

  useEffect(() => {
    if (autoStart) {
      startExecution();
    }
    
    return () => {
      // Cleanup on unmount
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
    };
  }, [autoStart]);

  const startExecution = async () => {
    if (isExecuting) return;

    setIsExecuting(true);
    setError('');
    setProgressSteps([]);
    setCurrentStep(null);
    setResult(null);
    setExecutionStartTime(new Date());

    // Create abort controller for cancellation
    abortControllerRef.current = new AbortController();

    try {
      const request = {
        toolId: tool.id,
        capability,
        input,
        context,
        options: { stream: true }
      };

      console.log(`🌊 Starting streaming execution: ${tool.name} -> ${capability}`);

      // Execute streaming tool
      const progressStream = toolExecutor.executeStream(request);
      
      for await (const progress of progressStream) {
        if (abortControllerRef.current?.signal.aborted) {
          break;
        }

        const progressStep: ProgressStep = {
          ...progress,
          id: `step_${Date.now()}_${Math.random().toString(36).substr(2, 6)}`,
          timestamp: new Date()
        };

        setCurrentStep(progressStep);
        
        // Add to progress steps if it's a meaningful update
        if (progress.stage !== 'progress' || progress.progress % 10 === 0) {
          setProgressSteps(prev => {
            const updated = [...prev];
            
            // Update duration for previous step
            if (updated.length > 0) {
              const lastStep = updated[updated.length - 1];
              if (!lastStep.duration) {
                lastStep.duration = progressStep.timestamp.getTime() - lastStep.timestamp.getTime();
              }
            }
            
            updated.push(progressStep);
            return updated;
          });
        }

        // Handle completion
        if (progress.stage === 'complete') {
          const finalResult: ToolResult = {
            success: true,
            data: progress.data,
            metadata: {
              executionTime: executionStartTime ? Date.now() - executionStartTime.getTime() : undefined,
              progressSteps: progressSteps.length + 1
            }
          };
          
          setResult(finalResult);
          onComplete(finalResult);
          break;
        }

        // Handle errors
        if (progress.stage === 'error') {
          const errorResult: ToolResult = {
            success: false,
            error: {
              code: 'EXECUTION_ERROR',
              message: progress.message
            }
          };
          
          setResult(errorResult);
          setError(progress.message);
          onComplete(errorResult);
          break;
        }
      }

    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Tool execution failed';
      setError(errorMsg);
      
      const errorResult: ToolResult = {
        success: false,
        error: {
          code: 'EXECUTION_ERROR',
          message: errorMsg
        }
      };
      
      setResult(errorResult);
      onComplete(errorResult);
    } finally {
      setIsExecuting(false);
      abortControllerRef.current = null;
    }
  };

  const cancelExecution = () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
    setIsExecuting(false);
    onCancel();
  };

  const getProgressBarWidth = () => {
    if (!currentStep) return 0;
    return Math.min(currentStep.progress, 100);
  };

  const getStatusIcon = (step: ProgressStep) => {
    switch (step.stage) {
      case 'complete':
        return <CheckCircleIcon className="w-4 h-4 text-green-500" />;
      case 'error':
        return <ExclamationTriangleIcon className="w-4 h-4 text-red-500" />;
      case 'processing':
        return <ArrowPathIcon className="w-4 h-4 text-blue-500 animate-spin" />;
      default:
        return <div className="w-2 h-2 bg-blue-500 rounded-full animate-pulse" />;
    }
  };

  const formatInputDisplay = () => {
    if (typeof input === 'string') return input;
    if (input.query) return `"${input.query}"`;
    if (input.topic) return `"${input.topic}"`;
    return JSON.stringify(input);
  };

  const getElapsedTime = () => {
    if (!executionStartTime) return '0s';
    const elapsed = Math.floor((Date.now() - executionStartTime.getTime()) / 1000);
    if (elapsed < 60) return `${elapsed}s`;
    return `${Math.floor(elapsed / 60)}m ${elapsed % 60}s`;
  };

  return (
    <div className={`bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden ${className}`}>
      {/* Header */}
      <div className="bg-gradient-to-r from-purple-50 to-blue-50 dark:from-purple-900/20 dark:to-blue-900/20 p-4 border-b border-gray-200 dark:border-gray-700">
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="w-full flex items-center justify-between text-left"
        >
          <div className="flex items-center space-x-3">
            <span className="text-2xl">{tool.icon || '🔧'}</span>
            <div>
              <h3 className="font-semibold text-gray-900 dark:text-white flex items-center space-x-2">
                <span>🌊 {tool.name}</span>
                {isExecuting && <SparklesIcon className="w-4 h-4 text-purple-500 animate-pulse" />}
              </h3>
              <div className="flex items-center space-x-2 mt-1">
                <span className="text-sm text-gray-600 dark:text-gray-300">
                  {capability} • {formatInputDisplay()}
                </span>
                <span className="text-xs px-2 py-1 bg-purple-100 dark:bg-purple-900 text-purple-800 dark:text-purple-200 rounded">
                  Streaming
                </span>
              </div>
            </div>
          </div>
          <div className="flex items-center space-x-3">
            {isExecuting && (
              <div className="text-right text-xs text-gray-500 dark:text-gray-400">
                <div className="flex items-center space-x-1">
                  <ClockIcon className="w-3 h-3" />
                  <span>{getElapsedTime()}</span>
                </div>
                {currentStep && (
                  <div className="mt-1">{Math.round(currentStep.progress)}%</div>
                )}
              </div>
            )}
            {isExpanded ? (
              <ChevronUpIcon className="w-5 h-5 text-gray-400" />
            ) : (
              <ChevronDownIcon className="w-5 h-5 text-gray-400" />
            )}
          </div>
        </button>

        {/* Progress Bar */}
        {isExecuting && (
          <div className="mt-3">
            <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
              <div 
                className="bg-gradient-to-r from-purple-500 to-blue-500 h-2 rounded-full transition-all duration-300 ease-out"
                style={{ width: `${getProgressBarWidth()}%` }}
              />
            </div>
            {currentStep && (
              <div className="flex justify-between mt-1 text-xs text-gray-500 dark:text-gray-400">
                <span>{currentStep.message}</span>
                <span>{Math.round(currentStep.progress)}%</span>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Content */}
      {isExpanded && (
        <div className="p-4">
          {/* Control Buttons */}
          {!isExecuting && !result && (
            <div className="mb-4">
              <button
                onClick={startExecution}
                className="flex items-center space-x-2 px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 font-medium"
              >
                <PlayIcon className="w-4 h-4" />
                <span>Start Execution</span>
              </button>
            </div>
          )}

          {isExecuting && (
            <div className="mb-4">
              <button
                onClick={cancelExecution}
                className="flex items-center space-x-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 font-medium"
              >
                <span>Cancel</span>
              </button>
            </div>
          )}

          {/* Progress Steps */}
          {progressSteps.length > 0 && (
            <div className="mb-4">
              <h4 className="font-medium text-gray-900 dark:text-white mb-3">Execution Progress</h4>
              <div className="space-y-3">
                {progressSteps.map((step, index) => (
                  <div key={step.id} className="flex items-start space-x-3 p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                    <div className="flex-shrink-0 mt-0.5">
                      {getStatusIcon(step)}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center space-x-2">
                        <span className="text-sm font-medium text-gray-900 dark:text-white">
                          Step {index + 1}: {step.stage}
                        </span>
                        <span className="text-xs text-gray-500 dark:text-gray-400">
                          {Math.round(step.progress)}%
                        </span>
                        {step.duration && (
                          <span className="text-xs text-gray-500 dark:text-gray-400">
                            ({step.duration}ms)
                          </span>
                        )}
                      </div>
                      <p className="text-sm text-gray-600 dark:text-gray-300 mt-1">
                        {step.message}
                      </p>
                      <span className="text-xs text-gray-400">
                        {step.timestamp.toLocaleTimeString()}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Final Result */}
          {result && !isExecuting && (
            <div className={`p-4 rounded-lg ${
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
                  <h4 className={`font-medium mb-2 ${
                    result.success ? 'text-green-800 dark:text-green-200' : 'text-red-800 dark:text-red-200'
                  }`}>
                    {result.success ? 'Execution Completed Successfully' : 'Execution Failed'}
                  </h4>
                  
                  {result.success ? (
                    <div className="text-sm text-green-700 dark:text-green-300">
                      Tool executed successfully with {progressSteps.length} steps completed.
                    </div>
                  ) : (
                    <div className="text-sm text-red-700 dark:text-red-300">
                      {result.error?.message || 'Unknown error occurred'}
                    </div>
                  )}

                  {/* Execution Metadata */}
                  {result.metadata && (
                    <div className="mt-3 flex items-center space-x-4 text-xs text-gray-600 dark:text-gray-400">
                      {result.metadata.executionTime && (
                        <span className="flex items-center space-x-1">
                          <ClockIcon className="w-3 h-3" />
                          <span>{result.metadata.executionTime}ms</span>
                        </span>
                      )}
                      {result.metadata.progressSteps && (
                        <span>{result.metadata.progressSteps} steps</span>
                      )}
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* Error Display */}
          {error && !result && (
            <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-3">
              <p className="text-sm text-red-700 dark:text-red-300">
                {error}
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}