// 🤖 BOT TOOLS PANEL
// Shows available tools for the current bot in chat

import { useState, useEffect, useMemo } from 'react';
import { useMountedRef } from '../hooks/useMountedRef';
import {
  WrenchScrewdriverIcon,
  ChevronDownIcon,
  ChevronUpIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  KeyIcon
} from '@heroicons/react/24/outline';
import { getTool } from '../tools';
import type { Bot } from '../types';
import type { UniversalTool } from '../types/tools';
import { useApiClient } from '../lib/api';
import { useAuth } from './AuthProvider';

interface BotToolsPanelProps {
  bot: Bot;
  onToolExecute: (tool: UniversalTool, capability: string, input: any) => void;
  onStreamingExecute?: (toolId: string, capability: string, input: any) => Promise<void>; // New streaming execution callback
  className?: string;
  getAccessToken: () => Promise<string | null>;
  getUserId: () => string | null;
  onUpgradeRequest?: (toolId: string) => void; // Callback for upgrade requests
}

export function BotToolsPanel({ bot, onToolExecute, onStreamingExecute, className = '', getAccessToken, getUserId, onUpgradeRequest }: BotToolsPanelProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [selectedTool, setSelectedTool] = useState<UniversalTool | null>(null);
  const [apiKeyStatuses, setApiKeyStatuses] = useState<Record<string, 'available' | 'missing' | 'unknown'>>({});
  const [checkingKeys, setCheckingKeys] = useState(false);

  // Track component mount state to prevent memory leaks
  const mountedRef = useMountedRef();

  const { environment } = useAuth();
  const apiClient = useApiClient(getAccessToken, getUserId, environment);

  if (!bot.agentTools || bot.agentTools.length === 0) {
    return null;
  }

  // Memoize botTools to prevent unnecessary re-renders
  const botTools = useMemo(() => {
    return bot.agentTools
      .map(agentTool => getTool(agentTool.name))
      .filter(Boolean) as UniversalTool[];
  }, [bot.agentTools]);

  // Check API key availability from vault with proper cleanup
  useEffect(() => {
    let isMounted = true;
    const abortController = new AbortController();
    
    const checkApiKeyAvailability = async () => {
      if (!apiClient || botTools.length === 0 || !mountedRef.current) return;
      
      if (isMounted) {
        setCheckingKeys(true);
      }
      
      const statuses: Record<string, 'available' | 'missing' | 'unknown'> = {};
      
      try {
        // Check each tool's API requirements
        for (const tool of botTools) {
          // Check if request was aborted or component unmounted
          if (abortController.signal.aborted || !isMounted || !mountedRef.current) {
            return;
          }
          
          if (tool.apiRequirements) {
            // Get required services for this tool
            const requiredServices = Object.keys(tool.apiRequirements);
            let hasAnyKey = false;
            
            // Check if any required service has an API key in vault
            for (const serviceId of requiredServices) {
              try {
                const response = await apiClient.get(`vault/keys/${serviceId}`);
                if (response.ok) {
                  hasAnyKey = true;
                  break;
                }
              } catch (error) {
                // Handle aborted requests gracefully
                if (abortController.signal.aborted) {
                  return;
                }
                // Continue checking other services
              }
            }
            
            statuses[tool.id] = hasAnyKey ? 'available' : 'missing';
          } else {
            // Tool doesn't require API keys
            statuses[tool.id] = 'available';
          }
        }
        
        // Only update state if component is still mounted
        if (isMounted && mountedRef.current) {
          setApiKeyStatuses(statuses);
        }
      } catch (error) {
        // Handle aborted requests gracefully
        if (abortController.signal.aborted) {
          return;
        }
        
        // Set all as unknown on error
        botTools.forEach(tool => {
          statuses[tool.id] = 'unknown';
        });
        
        if (isMounted && mountedRef.current) {
          setApiKeyStatuses(statuses);
        }
      } finally {
        if (isMounted && mountedRef.current) {
          setCheckingKeys(false);
        }
      }
    };

    checkApiKeyAvailability();
    
    // Cleanup function
    return () => {
      isMounted = false;
      abortController.abort();
    };
  }, [apiClient, bot.id, botTools, mountedRef]); // Include necessary dependencies

  // Helper function to get status icon and text
  const getToolStatusInfo = (tool: UniversalTool) => {
    const status = apiKeyStatuses[tool.id] || 'unknown';
    
    if (checkingKeys) {
      return {
        icon: <div className="w-4 h-4 border-2 border-gray-300 border-t-blue-600 rounded-full animate-spin" />,
        text: 'Checking...',
        color: 'text-gray-500',
        bgColor: 'bg-gray-100'
      };
    }
    
    switch (status) {
      case 'available':
        return tool.apiRequirements ? {
          icon: <CheckCircleIcon className="w-4 h-4" />,
          text: 'Premium Ready',
          color: 'text-green-600',
          bgColor: 'bg-green-100'
        } : {
          icon: <CheckCircleIcon className="w-4 h-4" />,
          text: 'Ready',
          color: 'text-green-600',
          bgColor: 'bg-green-100'
        };
      case 'missing':
        return {
          icon: <KeyIcon className="w-4 h-4" />,
          text: 'Free Mode',
          color: 'text-yellow-600',
          bgColor: 'bg-yellow-100'
        };
      case 'unknown':
      default:
        return {
          icon: <ExclamationTriangleIcon className="w-4 h-4" />,
          text: 'Unknown',
          color: 'text-gray-500',
          bgColor: 'bg-gray-100'
        };
    }
  };

  const handleToolSelect = (tool: UniversalTool) => {
    if (selectedTool?.id === tool.id) {
      setSelectedTool(null);
    } else {
      setSelectedTool(tool);
    }
  };

  const handleCapabilityExecute = (capability: string) => {
    if (!selectedTool) return;
    
    // Generate default input based on tool type
    let defaultInput: any = {};
    
    if (selectedTool.id === 'web-search') {
      defaultInput = { query: '', engine: 'duckduckgo', limit: 10 };
    } else if (selectedTool.id === 'research-assistant') {
      defaultInput = { topic: '', depth: 'detailed' };
    }
    
    onToolExecute(selectedTool, capability, defaultInput);
    setSelectedTool(null);
  };

  const handleStreamingExecute = async (capability: string) => {
    if (!selectedTool || !onStreamingExecute) return;
    
    // Generate default input based on tool type
    let defaultInput: any = {};
    
    if (selectedTool.id === 'web-search') {
      defaultInput = { query: 'artificial intelligence trends 2024', engine: 'duckduckgo', limit: 10 };
    } else if (selectedTool.id === 'research-assistant') {
      defaultInput = { topic: 'machine learning applications in healthcare', depth: 'detailed' };
    }
    
    try {
      await onStreamingExecute(selectedTool.id, capability, defaultInput);
      setSelectedTool(null);
    } catch (error) {
      console.error('Streaming execution failed:', error);
    }
  };

  return (
    <div className={`bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg ${className}`}>
      {/* Header */}
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full flex items-center justify-between p-4 text-left hover:bg-gray-50 dark:hover:bg-gray-700/50 rounded-lg transition-colors"
      >
        <div className="flex items-center space-x-3">
          <WrenchScrewdriverIcon className="w-5 h-5 text-gray-500 dark:text-gray-400" />
          <div>
            <h3 className="font-medium text-gray-900 dark:text-white">
              Bot Tools ({botTools.length})
            </h3>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {bot.title} can use these tools
            </p>
          </div>
        </div>
        {isExpanded ? (
          <ChevronUpIcon className="w-5 h-5 text-gray-400" />
        ) : (
          <ChevronDownIcon className="w-5 h-5 text-gray-400" />
        )}
      </button>

      {/* Tools List */}
      {isExpanded && (
        <div className="px-4 pb-4 space-y-2">
          {botTools.map(tool => (
            <div key={tool.id} className="border border-gray-200 dark:border-gray-600 rounded-lg">
              <button
                onClick={() => handleToolSelect(tool)}
                className={`w-full flex items-center space-x-3 p-3 text-left transition-colors rounded-lg ${
                  selectedTool?.id === tool.id
                    ? 'bg-blue-50 dark:bg-blue-900/20'
                    : 'hover:bg-gray-50 dark:hover:bg-gray-700/50'
                }`}
              >
                <span className="text-xl">{tool.icon}</span>
                <div className="flex-1">
                  <div className="flex items-center space-x-2">
                    <h4 className="font-medium text-gray-900 dark:text-white">
                      {tool.name}
                    </h4>
                    {(() => {
                      const statusInfo = getToolStatusInfo(tool);
                      return (
                        <span className={`inline-flex items-center space-x-1 text-xs px-2 py-1 rounded-full ${statusInfo.bgColor} ${statusInfo.color}`}>
                          {statusInfo.icon}
                          <span>{statusInfo.text}</span>
                        </span>
                      );
                    })()}
                    {/* Upgrade button for tools in free mode */}
                    {apiKeyStatuses[tool.id] === 'missing' && tool.apiRequirements && onUpgradeRequest && (
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          onUpgradeRequest(tool.id);
                        }}
                        className="text-xs px-2 py-1 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
                        title="Upgrade to Premium"
                      >
                        ✨ Upgrade
                      </button>
                    )}
                  </div>
                  <p className="text-sm text-gray-600 dark:text-gray-300">
                    {tool.description}
                  </p>
                </div>
                <span className="text-xs text-gray-500">
                  {tool.capabilities.length} capabilities
                </span>
              </button>

              {/* Capabilities */}
              {selectedTool?.id === tool.id && (
                <div className="border-t border-gray-200 dark:border-gray-600 p-3 space-y-2">
                  <h5 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    Available Capabilities:
                  </h5>
                  {tool.capabilities.map(capability => (
                    <div key={capability.name} className="space-y-2">
                      <div className="flex items-center justify-between p-2 bg-gray-50 dark:bg-gray-700 rounded">
                        <div className="flex-1">
                          <span className="text-sm font-medium text-gray-900 dark:text-white">
                            {capability.name}
                          </span>
                          <p className="text-xs text-gray-600 dark:text-gray-300">
                            {capability.description}
                          </p>
                          <div className="text-xs text-gray-500 flex items-center space-x-2 mt-1">
                            {capability.timeEstimate && (
                              <span>~{Math.round(capability.timeEstimate / 1000)}s</span>
                            )}
                            {capability.costEstimate && (
                              <span>{capability.costEstimate.credits} credits</span>
                            )}
                          </div>
                        </div>
                        <div className="flex space-x-2">
                          <button
                            onClick={() => handleCapabilityExecute(capability.name)}
                            className="px-3 py-1 text-xs bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
                          >
                            Execute
                          </button>
                          {onStreamingExecute && (
                            <button
                              onClick={() => handleStreamingExecute(capability.name)}
                              className="px-3 py-1 text-xs bg-purple-600 text-white rounded hover:bg-purple-700 transition-colors flex items-center space-x-1"
                            >
                              <span>🌊</span>
                              <span>Stream</span>
                            </button>
                          )}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}