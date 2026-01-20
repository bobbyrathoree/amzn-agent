import React, { useState, useEffect } from 'react';
import { ChevronDown } from 'lucide-react';
import type { KnowledgeSearchStage } from '../types';

interface KnowledgeSearchStagesProps {
  stages: KnowledgeSearchStage[];
  isLoading?: boolean;
  defaultCollapsed?: boolean;
}

// 🚀 INGENIOUS ENHANCEMENT: Progressive Knowledge Search Visualization
// This surpasses bedrock-chat by showing detailed search stages with timing and metadata
export const KnowledgeSearchStages: React.FC<KnowledgeSearchStagesProps> = ({
  stages,
  isLoading = false,
  defaultCollapsed = false
}) => {
  const [isCollapsed, setIsCollapsed] = useState(defaultCollapsed);

  // Auto-collapse when loading completes (stages are done and not loading)
  useEffect(() => {
    if (!isLoading && stages.length > 0 && defaultCollapsed) {
      setIsCollapsed(true);
    }
  }, [isLoading, stages.length, defaultCollapsed]);

  // Auto-expand when loading starts
  useEffect(() => {
    if (isLoading) {
      setIsCollapsed(false);
    }
  }, [isLoading]);

  if (stages.length === 0 && !isLoading) {
    return null;
  }

  const getStageIcon = (stage: string, success: boolean, metadata?: Record<string, unknown>) => {
    if (!success) {
      return (
        <div className="w-4 h-4 rounded-full bg-red-500 flex items-center justify-center">
          <svg className="w-2 h-2 text-white" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
          </svg>
        </div>
      );
    }

    switch (stage) {
      case 'intent_classification':
        // Show yellow/amber for skipped, lightning bolt icon
        return (
          <div className={`w-4 h-4 rounded-full ${metadata?.skipped ? 'bg-amber-500' : 'bg-blue-500'} flex items-center justify-center`}>
            <svg className="w-2 h-2 text-white" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M11.3 1.046A1 1 0 0112 2v5h4a1 1 0 01.82 1.573l-7 10A1 1 0 018 18v-5H4a1 1 0 01-.82-1.573l7-10a1 1 0 011.12-.38z" clipRule="evenodd" />
            </svg>
          </div>
        );
      case 'query_enhancement':
        return (
          <div className="w-4 h-4 rounded-full bg-blue-500 flex items-center justify-center">
            <svg className="w-2 h-2 text-white" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clipRule="evenodd" />
            </svg>
          </div>
        );
      case 'primary_search':
      case 'enhanced_search':
      case 'contextual_search':
        return (
          <div className="w-4 h-4 rounded-full bg-green-500 flex items-center justify-center">
            <svg className="w-2 h-2 text-white" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clipRule="evenodd" />
            </svg>
          </div>
        );
      case 'result_optimization':
        return (
          <div className="w-4 h-4 rounded-full bg-purple-500 flex items-center justify-center">
            <svg className="w-2 h-2 text-white" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M3 3a1 1 0 000 2v8a2 2 0 002 2h2.586l-1.293 1.293a1 1 0 101.414 1.414L10 15.414l2.293 2.293a1 1 0 001.414-1.414L12.414 15H15a2 2 0 002-2V5a1 1 0 100-2H3zm11 4a1 1 0 10-2 0v4a1 1 0 102 0V7zm-3 1a1 1 0 10-2 0v3a1 1 0 102 0V8zM8 9a1 1 0 00-2 0v2a1 1 0 102 0V9z" clipRule="evenodd" />
            </svg>
          </div>
        );
      default:
        return (
          <div className="w-4 h-4 rounded-full bg-gray-400 flex items-center justify-center">
            <svg className="w-2 h-2 text-white" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
            </svg>
          </div>
        );
    }
  };

  const getStageLabel = (stage: string, metadata?: Record<string, unknown>) => {
    switch (stage) {
      case 'intent_classification':
        return metadata?.skipped ? 'KB Search Skipped' : 'Intent Classification';
      case 'query_enhancement':
        return 'Analyzing Query';
      case 'primary_search':
        return 'Primary Search';
      case 'enhanced_search':
        return 'Enhanced Search';
      case 'contextual_search':
        return 'Contextual Search';
      case 'result_optimization':
        return 'Optimizing Results';
      default:
        return stage.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase());
    }
  };

  const formatDuration = (duration: string) => {
    // Parse duration string like "123.456ms" or "1.23s"
    const match = duration.match(/^(\d+(?:\.\d+)?)(ms|s|µs)/);
    if (match) {
      const value = parseFloat(match[1]);
      const unit = match[2];

      if (unit === 'ms' && value < 100) {
        return `${Math.round(value)}ms`;
      } else if (unit === 's') {
        return `${value.toFixed(2)}s`;
      } else if (unit === 'µs') {
        return `${Math.round(value)}µs`;
      } else {
        return `${Math.round(value)}ms`;
      }
    }
    return duration;
  };

  // Calculate total results
  const totalResults = stages.reduce((sum, stage) => sum + (stage.result_count || 0), 0);

  return (
    <div className="glass-card border border-border/30 rounded-lg p-4 mb-4">
      {/* Collapsible Header */}
      <button
        type="button"
        onClick={() => setIsCollapsed(!isCollapsed)}
        className="w-full flex items-center justify-between hover:bg-primary/5 rounded-lg transition-colors -m-1 p-1"
      >
        <div className="flex items-center">
          <svg className="w-4 h-4 text-primary mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
          </svg>
          <h3 className="text-sm font-medium text-foreground">Knowledge Base Search</h3>
          {!isLoading && stages.length > 0 && (
            <span className="ml-2 text-xs text-muted-foreground">
              ({totalResults} result{totalResults !== 1 ? 's' : ''})
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          {isLoading && (
            <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-primary"></div>
          )}
          <ChevronDown
            className={`w-4 h-4 text-muted-foreground transition-transform duration-300 ${
              isCollapsed ? '' : 'rotate-180'
            }`}
          />
        </div>
      </button>

      {/* Collapsible Content */}
      {!isCollapsed && (
        <div className="space-y-3 mt-3 pt-3 border-t border-border/20">
          {stages.map((stage, index) => (
            <div key={index} className="flex items-start space-x-3">
              <div className="flex-shrink-0 mt-0.5">
                {getStageIcon(stage.stage, stage.success, stage.metadata)}
              </div>

              <div className="flex-1 min-w-0">
                <div className="flex items-center justify-between">
                  <p className="text-sm font-medium text-foreground">
                    {getStageLabel(stage.stage, stage.metadata)}
                  </p>
                  <div className="flex items-center space-x-2 text-xs text-muted-foreground">
                    {stage.stage === 'intent_classification' && stage.metadata?.skipped ? (
                      <>
                        <span className="text-amber-600">Skipped</span>
                        <span>•</span>
                        <span>{formatDuration(stage.duration)}</span>
                      </>
                    ) : (
                      <>
                        <span>{stage.result_count} results</span>
                        <span>•</span>
                        <span>{formatDuration(stage.duration)}</span>
                      </>
                    )}
                  </div>
                </div>

                {stage.query && stage.query !== stage.stage && (
                  <p className="text-xs text-muted-foreground mt-1 truncate">
                    Query: {stage.query}
                  </p>
                )}

                {stage.metadata && (
                  <div className="mt-2">
                    {/* Intent classification metadata */}
                    {stage.metadata.skipped && stage.metadata.intent && (
                      <div className="text-xs text-muted-foreground">
                        Intent: <span className="text-amber-600 font-medium">{String(stage.metadata.intent)}</span>
                        {stage.metadata.confidence && (
                          <span className="ml-2">
                            (confidence: {(Number(stage.metadata.confidence) * 100).toFixed(0)}%)
                          </span>
                        )}
                      </div>
                    )}

                    {stage.metadata.reason && (
                      <div className="text-xs text-muted-foreground mt-1">
                        {String(stage.metadata.reason)}
                      </div>
                    )}

                    {stage.metadata.enhanced_queries && Array.isArray(stage.metadata.enhanced_queries) && (
                      <div className="text-xs text-muted-foreground">
                        Enhanced: {(stage.metadata.enhanced_queries as string[]).slice(0, 2).join(', ')}
                        {(stage.metadata.enhanced_queries as string[]).length > 2 && '...'}
                      </div>
                    )}

                    {stage.metadata.key_terms && Array.isArray(stage.metadata.key_terms) && (
                      <div className="text-xs text-muted-foreground">
                        Key terms: {(stage.metadata.key_terms as string[]).join(', ')}
                      </div>
                    )}

                    {stage.metadata.optimization && (
                      <div className="text-xs text-muted-foreground">
                        Optimization: {String(stage.metadata.optimization)}
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          ))}

          {isLoading && stages.length === 0 && (
            <div className="flex items-start space-x-3">
              <div className="flex-shrink-0 mt-0.5">
                <div className="w-4 h-4 rounded-full bg-blue-500 animate-pulse"></div>
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-foreground">Initializing search...</p>
                <p className="text-xs text-muted-foreground mt-1">Preparing to search knowledge base</p>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
};
