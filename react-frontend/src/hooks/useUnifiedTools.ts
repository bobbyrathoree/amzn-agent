// 🔧 USE UNIFIED TOOLS HOOK
// Replaces useBotTools and useStreamingExecution with unified interface

import { useState, useEffect, useCallback, useMemo } from 'react';
import { useMountedRef } from './useMountedRef';
import { UnifiedToolService, createUnifiedToolService } from '../services/unifiedToolService';
import type { 
  BackendToolResult, 
  ExecuteInChatRequest, 
  ToolSuggestion,
  BackendToolExecutionRequest 
} from '../services/unifiedToolService';
import type { ApiClient } from '../lib/api';
import type { UniversalTool, ToolProgress } from '../types/tools';
import type { Bot, User } from '../types';

interface UseUnifiedToolsProps {
  apiClient: ApiClient;
  user: User;
  bot?: Bot;
}

interface UseUnifiedToolsReturn {
  // Service instance
  toolService: UnifiedToolService;
  
  // Available tools
  availableTools: UniversalTool[];
  isLoadingTools: boolean;
  
  // Execution state
  isExecuting: boolean;
  currentExecution: string | null;
  executionProgress: ToolProgress | null;
  
  // Main execution functions
  executeTool: (request: BackendToolExecutionRequest) => Promise<BackendToolResult>;
  executeToolInChat: (request: ExecuteInChatRequest) => Promise<any>;
  executeToolWithProgress: (
    toolId: string,
    capability: string,
    input: Record<string, any>,
    options?: {
      conversationId?: string;
      onProgress?: (progress: ToolProgress) => void;
    }
  ) => Promise<BackendToolResult>;
  
  // Suggestions
  getSuggestions: (content: string, conversationId?: string) => Promise<ToolSuggestion[]>;
  
  // Tool information
  getTool: (toolId: string) => Promise<UniversalTool | null>;
  refreshTools: () => Promise<void>;
  
  // Execution control
  cancelExecution: (requestId?: string) => boolean;
  
  // Legacy compatibility
  executeLegacyTool: (
    tool: UniversalTool,
    capability: string,
    input: any
  ) => Promise<any>;
}

export function useUnifiedTools({ 
  apiClient, 
  user, 
  bot 
}: UseUnifiedToolsProps): UseUnifiedToolsReturn {
  
  // Create service instance
  const toolService = useMemo(() => {
    return createUnifiedToolService(apiClient);
  }, [apiClient]);

  // State
  const [availableTools, setAvailableTools] = useState<UniversalTool[]>([]);
  const [isLoadingTools, setIsLoadingTools] = useState(false);
  const [isExecuting, setIsExecuting] = useState(false);
  const [currentExecution, setCurrentExecution] = useState<string | null>(null);
  const [executionProgress, setExecutionProgress] = useState<ToolProgress | null>(null);

  // Track if component is mounted to prevent state updates after unmount
  const mountedRef = useMountedRef();

  // Load available tools - memoized to prevent infinite loops
  const refreshTools = useCallback(async () => {
    if (!apiClient || !mountedRef.current) return;
    
    if (mountedRef.current) {
      setIsLoadingTools(true);
    }
    
    try {
      const tools = await toolService.getAvailableTools();
      if (mountedRef.current) {
        setAvailableTools(tools);
      }
    } catch (error) {
      console.error('Failed to load tools:', error);
      if (mountedRef.current) {
        setAvailableTools([]);
      }
    } finally {
      if (mountedRef.current) {
        setIsLoadingTools(false);
      }
    }
  }, [toolService, apiClient, mountedRef]);

  // Load tools on mount - fixed dependency array to prevent infinite loops
  useEffect(() => {
    refreshTools();
  }, [apiClient]); // Only depend on apiClient, not refreshTools

  // Execute tool (basic)
  const executeTool = useCallback(async (request: BackendToolExecutionRequest): Promise<BackendToolResult> => {
    if (mountedRef.current) {
      setIsExecuting(true);
      setCurrentExecution(request.toolId);
      setExecutionProgress(null);
    }
    
    try {
      const result = await toolService.executeTool(request);
      return result;
    } finally {
      if (mountedRef.current) {
        setIsExecuting(false);
        setCurrentExecution(null);
        setExecutionProgress(null);
      }
    }
  }, [toolService, mountedRef]);

  // Execute tool in chat context
  const executeToolInChat = useCallback(async (request: ExecuteInChatRequest) => {
    if (mountedRef.current) {
      setIsExecuting(true);
      setCurrentExecution(request.toolId);
    }
    
    try {
      const result = await toolService.executeToolInConversation(request);
      return result;
    } finally {
      if (mountedRef.current) {
        setIsExecuting(false);
        setCurrentExecution(null);
      }
    }
  }, [toolService, mountedRef]);

  // Execute tool with progress tracking
  const executeToolWithProgress = useCallback(async (
    toolId: string,
    capability: string,
    input: Record<string, any>,
    options?: {
      conversationId?: string;
      onProgress?: (progress: ToolProgress) => void;
    }
  ): Promise<BackendToolResult> => {
    if (mountedRef.current) {
      setIsExecuting(true);
      setCurrentExecution(toolId);
      setExecutionProgress(null);
    }
    
    try {
      const result = await toolService.executeToolWithProgress(
        toolId,
        capability,
        input,
        {
          conversationId: options?.conversationId,
          botId: bot?.id,
          userId: user.userId,
          onProgress: (progress) => {
            if (mountedRef.current) {
              setExecutionProgress(progress);
            }
            options?.onProgress?.(progress);
          },
        }
      );
      return result;
    } finally {
      if (mountedRef.current) {
        setIsExecuting(false);
        setCurrentExecution(null);
        setExecutionProgress(null);
      }
    }
  }, [toolService, bot?.id, user.userId, mountedRef]);

  // Get tool suggestions
  const getSuggestions = useCallback(async (
    content: string, 
    conversationId?: string
  ): Promise<ToolSuggestion[]> => {
    try {
      return await toolService.suggestTools(content, conversationId, bot?.id);
    } catch (error) {
      console.error('Failed to get tool suggestions:', error);
      return [];
    }
  }, [toolService, bot?.id]);

  // Get specific tool
  const getTool = useCallback(async (toolId: string): Promise<UniversalTool | null> => {
    try {
      return await toolService.getTool(toolId);
    } catch (error) {
      console.error('Failed to get tool:', error);
      return null;
    }
  }, [toolService]);

  // Cancel execution
  const cancelExecution = useCallback((requestId?: string): boolean => {
    const id = requestId || currentExecution;
    if (id) {
      const cancelled = toolService.cancelExecution(id);
      if (cancelled) {
        setIsExecuting(false);
        setCurrentExecution(null);
        setExecutionProgress(null);
      }
      return cancelled;
    }
    return false;
  }, [toolService, currentExecution]);

  // Legacy compatibility for existing code
  const executeLegacyTool = useCallback(async (
    tool: UniversalTool,
    capability: string,
    input: any
  ) => {
    const context = {
      userId: user.userId,
      botId: bot?.id,
      permissions: ['internet-access:read', 'external-api:read'] as string[],
      requestId: `legacy_${Date.now()}`,
      timestamp: new Date(),
    };

    return await toolService.executeLegacyTool(tool, capability, input, context);
  }, [toolService, user.userId, bot?.id]);

  return {
    // Service
    toolService,
    
    // Tools
    availableTools,
    isLoadingTools,
    
    // Execution state
    isExecuting,
    currentExecution,
    executionProgress,
    
    // Functions
    executeTool,
    executeToolInChat,
    executeToolWithProgress,
    getSuggestions,
    getTool,
    refreshTools,
    cancelExecution,
    executeLegacyTool,
  };
}

// Convenience hook for bot-specific tools
export function useBotTools(bot: Bot, user: User, apiClient: ApiClient) {
  const unified = useUnifiedTools({ apiClient, user, bot });
  
  // Filter tools to only those available to the bot
  const botTools = useMemo(() => {
    if (!bot.agentTools || !unified.availableTools) return [];
    
    const botToolIds = bot.agentTools.map(t => t.name);
    return unified.availableTools.filter(tool => botToolIds.includes(tool.id));
  }, [bot.agentTools, unified.availableTools]);

  // Execute tool in context of current bot
  const executeInBotContext = useCallback(async (
    toolId: string,
    capability: string,
    input: Record<string, any>,
    conversationId?: string
  ) => {
    return unified.executeToolWithProgress(toolId, capability, input, {
      conversationId,
    });
  }, [unified]);

  return {
    ...unified,
    botTools,
    executeInBotContext,
  };
}