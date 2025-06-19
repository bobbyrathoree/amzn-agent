// 🤖 USE BOT TOOLS HOOK
// Hook for managing bot tools integration

import { useState, useEffect, useCallback } from 'react';
import { initializeTools, getTool } from '../tools';
import { toolExecutor } from '../services/toolExecutor';
import { APIKeyVaultService } from '../services/vaultService';
import type { Bot, User } from '../types';
import type { UniversalTool, ToolResult } from '../types/tools';

interface UseBotToolsProps {
  bot?: Bot;
  user?: User;
  apiClient?: any;
}

export function useBotTools({ bot, user, apiClient }: UseBotToolsProps) {
  const [vaultService, setVaultService] = useState<APIKeyVaultService | null>(null);
  const [isVaultUnlocked, setIsVaultUnlocked] = useState(false);
  const [availableTools, setAvailableTools] = useState<UniversalTool[]>([]);
  const [isExecuting, setIsExecuting] = useState(false);

  // Initialize tools framework and vault (only once)
  useEffect(() => {
    if (apiClient && user) {
      // Initialize tools framework (safe to call multiple times)
      initializeTools();
      
      // Setup vault service
      const vault = new APIKeyVaultService(apiClient);
      setVaultService(vault);
      toolExecutor.setVaultService(vault);
    }
  }, []); // Empty dependency array to run only once

  // Get bot's available tools
  useEffect(() => {
    if (bot?.agentTools) {
      const tools = bot.agentTools
        .map(agentTool => getTool(agentTool.name))
        .filter(Boolean) as UniversalTool[];
      setAvailableTools(tools);
    }
  }, [bot?.agentTools]);

  const unlockVault = useCallback(async (password: string, vaultPin: string): Promise<boolean> => {
    if (!vaultService) return false;
    const success = await vaultService.unlockVault(password, vaultPin);
    setIsVaultUnlocked(success);
    return success;
  }, [vaultService]);

  const lockVault = useCallback(() => {
    vaultService?.lockVault();
    setIsVaultUnlocked(false);
  }, [vaultService]);

  const executeTool = useCallback(async (
    tool: UniversalTool, 
    capability: string, 
    input: any
  ): Promise<ToolResult> => {
    if (!user) throw new Error('User not authenticated');
    
    setIsExecuting(true);
    
    try {
      const result = await toolExecutor.execute({
        toolId: tool.id,
        capability,
        input,
        context: {
          userId: user.userId,
          botId: bot?.id,
          permissions: ['internet-access:read', 'external-api:read'],
          requestId: `req_${Date.now()}`,
          timestamp: new Date()
        }
      });
      
      return result;
    } finally {
      setIsExecuting(false);
    }
  }, [user, bot?.id]);

  return {
    vaultService,
    isVaultUnlocked,
    availableTools,
    isExecuting,
    unlockVault,
    lockVault,
    executeTool
  };
}