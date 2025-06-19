// 🌊 STREAMING EXECUTION HOOK
// Reusable hook for streaming tool execution with progress tracking

import { useState, useCallback, useRef } from 'react';
import { useMountedRef } from './useMountedRef';
import { toolExecutor } from '../services/toolExecutor';
import type { UniversalTool, ToolProgress, ToolResult, ExecutionContext } from '../types/tools';

interface ProgressStep extends ToolProgress {
  id: string;
  timestamp: Date;
  duration?: number;
}

interface UseStreamingExecutionOptions {
  onComplete?: (result: ToolResult) => void;
  onError?: (error: string) => void;
  onProgress?: (progress: ToolProgress) => void;
  autoStart?: boolean;
}

interface UseStreamingExecutionReturn {
  // State
  isExecuting: boolean;
  progressSteps: ProgressStep[];
  currentStep: ProgressStep | null;
  result: ToolResult | null;
  error: string;
  executionStartTime: Date | null;
  
  // Actions
  startExecution: (tool: UniversalTool, capability: string, input: any, context: ExecutionContext) => Promise<void>;
  cancelExecution: () => void;
  reset: () => void;
  
  // Utilities
  getElapsedTime: () => string;
  getProgressPercentage: () => number;
  getTotalSteps: () => number;
}

export function useStreamingExecution(options: UseStreamingExecutionOptions = {}): UseStreamingExecutionReturn {
  const { onComplete, onError, onProgress } = options;
  
  const [isExecuting, setIsExecuting] = useState(false);
  const [progressSteps, setProgressSteps] = useState<ProgressStep[]>([]);
  const [currentStep, setCurrentStep] = useState<ProgressStep | null>(null);
  const [result, setResult] = useState<ToolResult | null>(null);
  const [error, setError] = useState<string>('');
  const [executionStartTime, setExecutionStartTime] = useState<Date | null>(null);
  
  // Track component mount state to prevent memory leaks
  const mountedRef = useMountedRef();
  const abortControllerRef = useRef<AbortController | null>(null);

  const reset = useCallback(() => {
    if (mountedRef.current) {
      setIsExecuting(false);
      setProgressSteps([]);
      setCurrentStep(null);
      setResult(null);
      setError('');
      setExecutionStartTime(null);
    }
    
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
  }, [mountedRef]);

  const startExecution = useCallback(async (
    tool: UniversalTool,
    capability: string,
    input: any,
    context: ExecutionContext
  ) => {
    if (isExecuting || !mountedRef.current) return;

    // Reset state
    reset();
    
    if (mountedRef.current) {
      setIsExecuting(true);
      setExecutionStartTime(new Date());
    }

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
        // Check if execution was cancelled or component unmounted
        if (abortControllerRef.current?.signal.aborted || !mountedRef.current) {
          console.log('🛑 Execution cancelled by user or component unmounted');
          break;
        }

        const progressStep: ProgressStep = {
          ...progress,
          id: `step_${Date.now()}_${Math.random().toString(36).substr(2, 6)}`,
          timestamp: new Date()
        };

        if (mountedRef.current) {
          setCurrentStep(progressStep);
        }
        
        // Call progress callback
        if (onProgress) {
          onProgress(progress);
        }
        
        // Add to progress steps for significant updates
        if ((progress.stage !== 'progress' || progress.progress % 10 === 0) && mountedRef.current) {
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
              progressSteps: progressSteps.length + 1,
              totalSteps: progressSteps.length + 1
            }
          };
          
          if (mountedRef.current) {
            setResult(finalResult);
          }
          
          if (onComplete) {
            onComplete(finalResult);
          }
          
          console.log(`✅ Streaming execution completed: ${tool.name}`);
          break;
        }

        // Handle errors
        if (progress.stage === 'error') {
          const errorMsg = progress.message || 'Tool execution failed';
          const errorResult: ToolResult = {
            success: false,
            error: {
              code: 'EXECUTION_ERROR',
              message: errorMsg
            },
            metadata: {
              executionTime: executionStartTime ? Date.now() - executionStartTime.getTime() : undefined,
              failedAtStep: progressSteps.length + 1
            }
          };
          
          if (mountedRef.current) {
            setResult(errorResult);
            setError(errorMsg);
          }
          
          if (onError) {
            onError(errorMsg);
          }
          
          if (onComplete) {
            onComplete(errorResult);
          }
          
          console.error(`❌ Streaming execution failed: ${errorMsg}`);
          break;
        }
      }

    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Tool execution failed';
      
      const errorResult: ToolResult = {
        success: false,
        error: {
          code: 'EXECUTION_ERROR',
          message: errorMsg
        },
        metadata: {
          executionTime: executionStartTime ? Date.now() - executionStartTime.getTime() : undefined
        }
      };
      
      if (mountedRef.current) {
        setResult(errorResult);
        setError(errorMsg);
      }
      
      if (onError) {
        onError(errorMsg);
      }
      
      if (onComplete) {
        onComplete(errorResult);
      }
      
      console.error(`❌ Streaming execution error:`, err);
      
    } finally {
      if (mountedRef.current) {
        setIsExecuting(false);
      }
      abortControllerRef.current = null;
    }
  }, [isExecuting, executionStartTime, progressSteps.length, onComplete, onError, onProgress, reset, mountedRef]);

  const cancelExecution = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      console.log('🛑 Cancelling streaming execution');
    }
    if (mountedRef.current) {
      setIsExecuting(false);
    }
  }, [mountedRef]);

  const getElapsedTime = useCallback(() => {
    if (!executionStartTime) return '0s';
    const elapsed = Math.floor((Date.now() - executionStartTime.getTime()) / 1000);
    if (elapsed < 60) return `${elapsed}s`;
    const minutes = Math.floor(elapsed / 60);
    const seconds = elapsed % 60;
    return `${minutes}m ${seconds}s`;
  }, [executionStartTime]);

  const getProgressPercentage = useCallback(() => {
    return currentStep ? Math.min(currentStep.progress, 100) : 0;
  }, [currentStep]);

  const getTotalSteps = useCallback(() => {
    return progressSteps.length;
  }, [progressSteps.length]);

  return {
    // State
    isExecuting,
    progressSteps,
    currentStep,
    result,
    error,
    executionStartTime,
    
    // Actions
    startExecution,
    cancelExecution,
    reset,
    
    // Utilities
    getElapsedTime,
    getProgressPercentage,
    getTotalSteps
  };
}