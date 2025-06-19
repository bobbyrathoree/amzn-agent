// 🔥 STREAMING PROGRESS INDICATOR
// Compact real-time progress display for inline tool execution

import { ArrowPathIcon, CheckCircleIcon, ExclamationTriangleIcon, ClockIcon } from '@heroicons/react/24/outline';
import type { ToolProgress } from '../types/tools';

interface StreamingProgressIndicatorProps {
  isExecuting: boolean;
  currentStep?: ToolProgress | null;
  progress?: number;
  message?: string;
  elapsedTime?: string;
  error?: string;
  success?: boolean;
  className?: string;
  size?: 'sm' | 'md' | 'lg';
  showMessage?: boolean;
  showElapsedTime?: boolean;
}

export function StreamingProgressIndicator({
  isExecuting,
  currentStep,
  progress,
  message,
  elapsedTime,
  error,
  success,
  className = '',
  size = 'md',
  showMessage = true,
  showElapsedTime = true
}: StreamingProgressIndicatorProps) {
  const sizeClasses = {
    sm: 'text-xs',
    md: 'text-sm',
    lg: 'text-base'
  };

  const iconSizes = {
    sm: 'w-3 h-3',
    md: 'w-4 h-4',
    lg: 'w-5 h-5'
  };

  const progressBarHeights = {
    sm: 'h-1',
    md: 'h-2',
    lg: 'h-3'
  };

  const getStatusIcon = () => {
    if (error) {
      return <ExclamationTriangleIcon className={`${iconSizes[size]} text-red-500`} />;
    }
    
    if (success) {
      return <CheckCircleIcon className={`${iconSizes[size]} text-green-500`} />;
    }
    
    if (isExecuting) {
      return <ArrowPathIcon className={`${iconSizes[size]} text-blue-500 animate-spin`} />;
    }
    
    return null;
  };

  const getProgressValue = () => {
    if (currentStep) return currentStep.progress;
    if (progress !== undefined) return progress;
    return 0;
  };

  const getCurrentMessage = () => {
    if (error) return error;
    if (currentStep?.message) return currentStep.message;
    if (message) return message;
    if (success) return 'Completed successfully';
    if (isExecuting) return 'Executing...';
    return '';
  };

  const progressValue = Math.min(getProgressValue(), 100);
  const currentMessage = getCurrentMessage();

  if (!isExecuting && !error && !success) {
    return null;
  }

  return (
    <div className={`inline-flex items-center space-x-2 ${sizeClasses[size]} ${className}`}>
      {/* Status Icon */}
      <div className="flex-shrink-0">
        {getStatusIcon()}
      </div>

      {/* Progress Bar */}
      {isExecuting && (
        <div className="flex-1 min-w-0">
          <div className={`w-full bg-gray-200 dark:bg-gray-700 rounded-full ${progressBarHeights[size]}`}>
            <div 
              className={`bg-gradient-to-r from-blue-500 to-purple-500 ${progressBarHeights[size]} rounded-full transition-all duration-300 ease-out`}
              style={{ width: `${progressValue}%` }}
            />
          </div>
        </div>
      )}

      {/* Message and Progress */}
      <div className="flex items-center space-x-2 text-gray-600 dark:text-gray-300">
        {showMessage && currentMessage && (
          <span className="truncate max-w-48">
            {currentMessage}
          </span>
        )}
        
        {isExecuting && (
          <span className="text-gray-500 dark:text-gray-400 font-mono">
            {Math.round(progressValue)}%
          </span>
        )}
        
        {showElapsedTime && elapsedTime && (
          <span className="flex items-center space-x-1 text-gray-400">
            <ClockIcon className="w-3 h-3" />
            <span className="font-mono">{elapsedTime}</span>
          </span>
        )}
      </div>
    </div>
  );
}

// Compact version for chat bubbles
export function CompactStreamingIndicator({
  isExecuting,
  progress = 0,
  message,
  success,
  error
}: {
  isExecuting: boolean;
  progress?: number;
  message?: string;
  success?: boolean;
  error?: string;
}) {
  if (!isExecuting && !success && !error) return null;

  return (
    <div className="inline-flex items-center space-x-2 text-xs text-gray-500 dark:text-gray-400">
      {isExecuting && (
        <>
          <ArrowPathIcon className="w-3 h-3 text-blue-500 animate-spin" />
          <div className="w-16 bg-gray-200 dark:bg-gray-700 rounded-full h-1">
            <div 
              className="bg-blue-500 h-1 rounded-full transition-all duration-300"
              style={{ width: `${Math.min(progress, 100)}%` }}
            />
          </div>
          <span className="font-mono">{Math.round(progress)}%</span>
        </>
      )}
      
      {success && (
        <>
          <CheckCircleIcon className="w-3 h-3 text-green-500" />
          <span>Complete</span>
        </>
      )}
      
      {error && (
        <>
          <ExclamationTriangleIcon className="w-3 h-3 text-red-500" />
          <span>Failed</span>
        </>
      )}
      
      {message && <span className="truncate max-w-32">{message}</span>}
    </div>
  );
}

// Progress dots animation for minimal space
export function ProgressDots({
  isActive = false,
  className = ''
}: {
  isActive?: boolean;
  className?: string;
}) {
  return (
    <div className={`inline-flex items-center space-x-1 ${className}`}>
      <div className={`w-1 h-1 rounded-full ${isActive ? 'bg-blue-500 animate-pulse' : 'bg-gray-300'}`} />
      <div className={`w-1 h-1 rounded-full ${isActive ? 'bg-blue-500 animate-pulse animation-delay-200' : 'bg-gray-300'}`} />
      <div className={`w-1 h-1 rounded-full ${isActive ? 'bg-blue-500 animate-pulse animation-delay-400' : 'bg-gray-300'}`} />
    </div>
  );
}