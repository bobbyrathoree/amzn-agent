// 🎨 UNIVERSAL TOOL RESULT RENDERER
// Intelligent routing to appropriate visualization components

import { WebSearchResults } from './WebSearchResults';
import { ResearchResults } from './ResearchResults';
import { GenericToolResult } from './GenericToolResult';

interface ToolResultData {
  // Web Search specific
  query?: string;
  engine?: string;
  results?: any[];
  totalResults?: number;
  searchTime?: number;
  engineCapabilities?: {
    realTime: boolean;
    premium: boolean;
    apiRequired: boolean;
  };
  vaultInfo?: {
    usedApiKey: boolean;
    requestedEngine: string;
    actualEngine: string;
    fallbackUsed: boolean;
  };
  
  // Research specific
  topic?: string;
  sources?: any[];
  analysis?: {
    keyFindings: string[];
    trends: string[];
    gaps: string[];
    recommendations: string[];
    confidence: number;
    methodology?: string;
    limitations?: string[];
  };
  depth?: 'surface' | 'detailed' | 'comprehensive';
  
  // Generic data
  [key: string]: any;
}

interface ToolResultRendererProps {
  toolName: string;
  toolId: string;
  capability: string;
  data: ToolResultData;
  executionTime: number;
  usedApiKey: boolean;
  success: boolean;
  errorMessage?: string;
  className?: string;
}

export function ToolResultRenderer({
  toolName,
  toolId,
  capability,
  data,
  executionTime,
  usedApiKey,
  success,
  errorMessage,
  className = ''
}: ToolResultRendererProps) {
  
  // Handle errors first
  if (!success) {
    return (
      <div className={`bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 ${className}`}>
        <div className="flex items-center space-x-2 mb-2">
          <span className="text-xl">❌</span>
          <h3 className="font-medium text-red-800 dark:text-red-200">
            {toolName} Failed
          </h3>
        </div>
        <p className="text-sm text-red-700 dark:text-red-300">
          {errorMessage || 'Tool execution failed'}
        </p>
      </div>
    );
  }

  // Route to appropriate visualization based on tool type
  switch (toolId) {
    case 'web-search':
      if (data.results && data.query && data.engine) {
        return (
          <WebSearchResults
            results={data.results}
            query={data.query}
            engine={data.vaultInfo?.actualEngine || data.engine}
            totalResults={data.totalResults || data.results.length}
            searchTime={data.searchTime || executionTime}
            usedApiKey={data.vaultInfo?.usedApiKey || usedApiKey}
            fallbackUsed={data.vaultInfo?.fallbackUsed || false}
            requestedEngine={data.vaultInfo?.requestedEngine}
            className={className}
          />
        );
      }
      break;

    case 'research-assistant':
      if (data.sources && data.analysis && data.topic) {
        return (
          <ResearchResults
            topic={data.topic}
            sources={data.sources}
            analysis={data.analysis}
            executionTime={executionTime}
            usedPremiumSources={usedApiKey}
            depth={data.depth || 'detailed'}
            className={className}
          />
        );
      }
      break;

    // Add more tool-specific visualizations here
    case 'file-processor':
    case 'data-analyzer':
    case 'code-executor':
    default:
      // Fall back to generic visualization
      break;
  }

  // Generic fallback for unknown tools or incomplete data
  return (
    <GenericToolResult
      toolName={toolName}
      toolId={toolId}
      capability={capability}
      data={data}
      executionTime={executionTime}
      usedApiKey={usedApiKey}
      className={className}
    />
  );
}