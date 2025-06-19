// 🎯 TOOLS SUGGESTION COMPONENT
// Smart tool suggestions integrated into chat input

import { useState, useEffect, useMemo } from 'react';
import { WrenchScrewdriverIcon, SparklesIcon, XMarkIcon } from '@heroicons/react/24/outline';
import { findToolsForIntent, getAllTools } from '../tools';
import type { UniversalTool, ToolRoutingPlan } from '../types/tools';

interface ToolsSuggestionProps {
  input: string;
  onToolSelect: (tool: UniversalTool, capability: string, suggestedInput?: any) => void;
  className?: string;
}

export function ToolsSuggestion({ input, onToolSelect, className = '' }: ToolsSuggestionProps) {
  const [toolPlan, setToolPlan] = useState<ToolRoutingPlan | null>(null);
  const [isVisible, setIsVisible] = useState(false);
  const [availableTools] = useState(() => getAllTools());

  // Keywords that trigger tool suggestions
  const toolKeywords = useMemo(() => ({
    search: ['search', 'find', 'look up', 'google', 'web', 'internet'],
    research: ['research', 'analyze', 'study', 'investigate', 'comprehensive', 'deep dive'],
    data: ['data', 'statistics', 'numbers', 'metrics', 'analytics'],
    news: ['news', 'latest', 'recent', 'current', 'today', 'breaking'],
    academic: ['academic', 'paper', 'journal', 'study', 'scientific', 'peer reviewed']
  }), []);

  useEffect(() => {
    const checkForToolSuggestions = async () => {
      if (!input.trim() || input.length < 10) {
        setIsVisible(false);
        return;
      }

      const inputLower = input.toLowerCase();
      
      // Check if input contains tool-triggering keywords
      const hasToolKeywords = Object.values(toolKeywords).some(keywords =>
        keywords.some(keyword => inputLower.includes(keyword))
      );

      if (!hasToolKeywords) {
        setIsVisible(false);
        return;
      }

      try {
        const plan = await findToolsForIntent(input, {
          userId: 'demo',
          permissions: ['internet-access:read', 'external-api:read'],
          requestId: `req_${Date.now()}`,
          timestamp: new Date()
        });

        if (plan && plan.confidence > 0.3) {
          setToolPlan(plan);
          setIsVisible(true);
        } else {
          setIsVisible(false);
        }
      } catch (error) {
        console.error('Tool suggestion error:', error);
        setIsVisible(false);
      }
    };

    const debounceTimeout = setTimeout(checkForToolSuggestions, 500);
    return () => clearTimeout(debounceTimeout);
  }, [input, toolKeywords]);

  const handleToolSelect = (step: any) => {
    const tool = availableTools.find(t => t.id === step.toolId);
    if (!tool) return;

    // Generate suggested input based on the user's message
    let suggestedInput: any = {};
    
    if (tool.id === 'web-search') {
      // Extract search query from input
      const searchQuery = input.replace(/(?:search|find|look up|google)\s+(?:for\s+)?/i, '').trim();
      suggestedInput = {
        query: searchQuery || input,
        engine: 'duckduckgo',
        limit: 10
      };
    } else if (tool.id === 'research-assistant') {
      // Extract research topic
      const topic = input.replace(/(?:research|analyze|study)\s+(?:about\s+)?/i, '').trim();
      suggestedInput = {
        topic: topic || input,
        depth: 'detailed',
        includeAcademic: true,
        includeNews: true
      };
    }

    onToolSelect(tool, step.capability, suggestedInput);
    setIsVisible(false);
  };

  if (!isVisible || !toolPlan) return null;

  return (
    <div className={`bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4 ${className}`}>
      <div className="flex items-start justify-between">
        <div className="flex items-center space-x-2 mb-3">
          <SparklesIcon className="w-5 h-5 text-blue-600 dark:text-blue-400" />
          <span className="text-sm font-medium text-blue-800 dark:text-blue-200">
            AI Tool Suggestion
          </span>
          <span className="text-xs px-2 py-1 bg-blue-100 dark:bg-blue-800 text-blue-700 dark:text-blue-300 rounded-full">
            {Math.round(toolPlan.confidence * 100)}% confident
          </span>
        </div>
        <button
          onClick={() => setIsVisible(false)}
          className="text-blue-400 hover:text-blue-600 dark:text-blue-500 dark:hover:text-blue-300"
        >
          <XMarkIcon className="w-4 h-4" />
        </button>
      </div>

      <p className="text-sm text-blue-700 dark:text-blue-300 mb-3">
        I can help you get better results using specialized tools:
      </p>

      <div className="space-y-2">
        {toolPlan.steps.slice(0, 2).map((step, index) => {
          const tool = availableTools.find(t => t.id === step.toolId);
          if (!tool) return null;

          return (
            <button
              key={index}
              onClick={() => handleToolSelect(step)}
              className="w-full flex items-center space-x-3 p-3 bg-white dark:bg-gray-800 border border-blue-200 dark:border-blue-700 rounded-lg hover:bg-blue-50 dark:hover:bg-blue-900/30 transition-colors text-left"
            >
              <div className="flex-shrink-0">
                <span className="text-lg">{tool.icon}</span>
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center space-x-2">
                  <span className="font-medium text-gray-900 dark:text-white">
                    {tool.name}
                  </span>
                  {tool.apiRequirements && (
                    <span className="text-xs px-2 py-1 bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 rounded">
                      Premium
                    </span>
                  )}
                </div>
                <p className="text-sm text-gray-600 dark:text-gray-300 truncate">
                  {step.reasoning}
                </p>
                <div className="flex items-center space-x-2 mt-1">
                  <WrenchScrewdriverIcon className="w-3 h-3 text-gray-400" />
                  <span className="text-xs text-gray-500">
                    Capability: {step.capability}
                  </span>
                </div>
              </div>
              <div className="flex-shrink-0">
                <span className="text-xs text-blue-600 dark:text-blue-400 font-medium">
                  Use Tool →
                </span>
              </div>
            </button>
          );
        })}
      </div>

      {toolPlan.alternatives && toolPlan.alternatives.length > 0 && (
        <div className="mt-3 pt-3 border-t border-blue-200 dark:border-blue-700">
          <p className="text-xs text-blue-600 dark:text-blue-400 mb-2">
            Alternative suggestions:
          </p>
          <div className="flex flex-wrap gap-2">
            {toolPlan.alternatives.slice(0, 2).map((alt, index) => {
              const tool = availableTools.find(t => t.id === alt.steps[0]?.toolId);
              if (!tool) return null;

              return (
                <button
                  key={index}
                  onClick={() => handleToolSelect(alt.steps[0])}
                  className="text-xs px-3 py-1 bg-blue-100 dark:bg-blue-800 text-blue-700 dark:text-blue-300 rounded-full hover:bg-blue-200 dark:hover:bg-blue-700"
                >
                  {tool.icon} {tool.name}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}