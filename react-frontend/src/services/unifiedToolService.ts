// 🛠️ UNIFIED TOOL SERVICE - Replaces separate tool execution systems
// Provides single interface for all tool operations with chat integration

import type { ApiClient } from '../lib/api';
import type { 
  UniversalTool, 
  ToolResult, 
  ToolProgress,
  ExecutionContext
} from '../types/tools';
import type { MessageContent } from '../types';

// New unified types for backend integration
export interface BackendToolExecutionRequest {
  toolId: string;
  capability: string;
  input: Record<string, any>;
  conversationId?: string;
  context: ExecutionContext;
  stream?: boolean;
}

export interface BackendToolResult {
  success: boolean;
  data?: any;
  error?: string;
  errorCode?: string;
  executionTime: number;
  tokensUsed?: number;
  costIncurred?: number;
  metadata?: Record<string, any>;
  toolVersion?: string;
  executedAt: string;
  apiKeysUsed?: string[];
}

export interface BackendToolExecutionResponse {
  success: boolean;
  result: BackendToolResult;
  executionTime: number;
  toolId: string;
  capability: string;
}

export interface ExecuteInChatRequest {
  conversationId: string;
  toolId: string;
  capability: string;
  input: Record<string, any>;
  botId?: string;
}

export interface ExecuteInChatResponse {
  success: boolean;
  result: BackendToolResult;
  conversationId: string;
  toolId: string;
  capability: string;
  executionTime: number;
  message: string;
}

export interface ToolSuggestion {
  toolId: string;
  capability: string;
  reason: string;
  confidence: number;
  priority: number;
  input?: Record<string, any>;
}

// Unified Tool Service - Replaces both toolExecutor and streaming services
export class UnifiedToolService {
  private apiClient: ApiClient;
  private abortControllers: Map<string, AbortController> = new Map();

  constructor(apiClient: ApiClient) {
    this.apiClient = apiClient;
  }

  // ✨ NEW: Execute tool with full backend integration
  async executeTool(request: BackendToolExecutionRequest): Promise<BackendToolResult> {
    try {
      const response = await this.apiClient.post('tools/execute', request);
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || 'Tool execution failed');
      }

      const data: BackendToolExecutionResponse = await response.json();
      
      if (!data.success) {
        throw new Error(data.result.error || 'Tool execution failed');
      }

      return data.result;
    } catch (error) {
      console.error('Tool execution failed:', error);
      throw error;
    }
  }

  // ⭐ NEW: Execute tool within a conversation context
  async executeToolInConversation(request: ExecuteInChatRequest): Promise<ExecuteInChatResponse> {
    try {
      const response = await this.apiClient.post('tools/execute-in-chat', request);
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || 'Tool execution in chat failed');
      }

      const data: ExecuteInChatResponse = await response.json();
      return data;
    } catch (error) {
      console.error('Tool execution in chat failed:', error);
      throw error;
    }
  }

  // 🔄 Stream tool execution (for long-running tools)
  async streamTool(
    request: BackendToolExecutionRequest,
    onProgress: (progress: ToolProgress) => void
  ): Promise<BackendToolResult> {
    // For now, fall back to regular execution
    // TODO: Implement SSE streaming when backend supports it
    const result = await this.executeTool(request);
    
    // Simulate progress callback for compatibility
    onProgress({
      stage: 'complete',
      progress: 100,
      message: 'Tool execution completed',
    });

    return result;
  }

  // 🎯 Get AI-powered tool suggestions based on content
  async suggestTools(content: string, conversationId?: string, botId?: string): Promise<ToolSuggestion[]> {
    try {
      const response = await this.apiClient.post('tools/suggest', {
        content,
        conversationId,
        botId,
      });

      if (!response.ok) {
        console.error('Failed to get tool suggestions');
        return [];
      }

      const data = await response.json();
      return data.suggestions || [];
    } catch (error) {
      console.error('Failed to get tool suggestions:', error);
      return [];
    }
  }

  // 📋 Get available tools from backend
  async getAvailableTools(category?: string): Promise<UniversalTool[]> {
    try {
      const url = category ? `tools?category=${category}` : 'tools';
      const response = await this.apiClient.get(url);

      if (!response.ok) {
        throw new Error('Failed to get available tools');
      }

      const data = await response.json();
      return data.tools || [];
    } catch (error) {
      console.error('Failed to get available tools:', error);
      return [];
    }
  }

  // 🔍 Get specific tool details
  async getTool(toolId: string): Promise<UniversalTool | null> {
    try {
      const response = await this.apiClient.get(`tools/${toolId}`);

      if (!response.ok) {
        if (response.status === 404) {
          return null;
        }
        throw new Error('Failed to get tool details');
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get tool details:', error);
      return null;
    }
  }

  // 🛡️ Check tool health and availability
  async checkToolHealth(): Promise<Record<string, any>> {
    try {
      const response = await this.apiClient.get('tools/health');

      if (!response.ok) {
        throw new Error('Failed to check tool health');
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to check tool health:', error);
      return { status: 'error', error: error instanceof Error ? error.message : 'Unknown error' };
    }
  }

  // 🔧 Legacy compatibility - execute tool using old interface
  async executeLegacyTool(
    tool: UniversalTool, 
    capability: string, 
    input: any,
    context: ExecutionContext
  ): Promise<ToolResult> {
    const request: BackendToolExecutionRequest = {
      toolId: tool.id,
      capability,
      input,
      context,
    };

    try {
      const backendResult = await this.executeTool(request);
      
      // Convert backend result to legacy format
      return {
        success: backendResult.success,
        data: backendResult.data,
        error: backendResult.error ? { code: 'EXECUTION_ERROR', message: backendResult.error, details: backendResult } : undefined,
        metadata: { 
          ...backendResult.metadata,
          executionTime: backendResult.executionTime
        },
      };
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? { code: 'EXECUTION_ERROR', message: error.message } : { code: 'UNKNOWN_ERROR', message: 'Unknown error' },
        metadata: { executionTime: 0 },
      };
    }
  }

  // 🎬 Execute tool with progress tracking and conversation integration
  async executeToolWithProgress(
    toolId: string,
    capability: string,
    input: Record<string, any>,
    options: {
      conversationId?: string;
      botId?: string;
      userId: string;
      onProgress?: (progress: ToolProgress) => void;
      abortSignal?: AbortSignal;
    }
  ): Promise<BackendToolResult> {
    const requestId = `${toolId}_${Date.now()}`;
    
    // Setup abort controller if not provided
    if (!options.abortSignal) {
      const controller = new AbortController();
      this.abortControllers.set(requestId, controller);
      options.abortSignal = controller.signal;
    }

    // Ensure cleanup happens regardless of execution path
    const cleanup = () => {
      this.abortControllers.delete(requestId);
    };

    try {
      const request: BackendToolExecutionRequest = {
        toolId,
        capability,
        input,
        conversationId: options.conversationId,
        context: {
          userId: options.userId,
          botId: options.botId,
          permissions: ['internet-access:read', 'external-api:read'],
          requestId,
          timestamp: new Date(),
        },
      };

      // Use streaming if progress callback is provided
      if (options.onProgress) {
        const result = await this.streamTool(request, options.onProgress);
        cleanup();
        return result;
      } else {
        const result = await this.executeTool(request);
        cleanup();
        return result;
      }
    } catch (error) {
      cleanup();
      throw error;
    }
  }

  // 🛑 Cancel tool execution
  cancelExecution(requestId: string): boolean {
    const controller = this.abortControllers.get(requestId);
    if (controller) {
      controller.abort();
      this.abortControllers.delete(requestId);
      return true;
    }
    return false;
  }

  // 🔗 Helper: Convert tool execution to message content
  static createToolExecutionMessages(
    toolId: string,
    _capability: string,
    input: Record<string, any>,
    result: BackendToolResult
  ): MessageContent[] {
    const messages: MessageContent[] = [];

    // Tool use message
    messages.push({
      type: 'tool_use',
      toolUse: {
        toolUseId: `tool_${Date.now()}`,
        name: toolId,
        input,
      },
    });

    // Tool result message
    messages.push({
      type: 'tool_result',
      toolResult: {
        toolUseId: `tool_${Date.now()}`,
        content: result.success ? result.data : result.error,
        isError: !result.success,
      },
    });

    return messages;
  }

  // 📊 Get execution statistics
  getExecutionStats(): Record<string, any> {
    return {
      activeExecutions: this.abortControllers.size,
      timestamp: new Date(),
    };
  }
}

// Export singleton instance creation function
export function createUnifiedToolService(apiClient: ApiClient): UnifiedToolService {
  return new UnifiedToolService(apiClient);
}

// Types are already exported as interfaces above