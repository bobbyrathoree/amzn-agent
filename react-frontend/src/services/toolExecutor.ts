// ⚡ TOOL EXECUTION ENGINE
// Secure, scalable, and observable tool execution

import type { 
  UniversalTool, 
  ToolExecutionRequest, 
  ToolResult, 
  ToolProgress,
  ExecutionContext,
  ToolExecutionLog,
  RateLimitResult
} from '../types/tools';
import { toolRegistry } from './toolRegistry';

export class ToolExecutor {
  private executionLogs: Map<string, ToolExecutionLog> = new Map();
  private activeExecutions: Set<string> = new Set();
  private rateLimits: Map<string, { count: number; resetTime: Date }> = new Map();
  private vaultService: any; // APIKeyVaultService instance

  constructor(vaultService?: any) {
    this.vaultService = vaultService;
  }

  setVaultService(vaultService: any): void {
    this.vaultService = vaultService;
  }

  // ⚡ MAIN EXECUTION METHOD
  async execute(request: ToolExecutionRequest): Promise<ToolResult> {
    const startTime = new Date();
    const executionId = this.generateExecutionId();
    
    try {
      // Get tool
      const tool = toolRegistry.getTool(request.toolId);
      if (!tool) {
        throw new Error(`Tool ${request.toolId} not found`);
      }

      // Check API key requirements BEFORE execution
      const apiKeyCheck = await this.checkAPIKeyRequirements(tool, request.context);
      if (!apiKeyCheck.satisfied) {
        return {
          success: false,
          error: {
            code: 'API_KEY_REQUIRED',
            message: `This tool requires API keys that are not configured`,
            details: {
              toolId: request.toolId,
              missingServices: apiKeyCheck.missingServices,
              requiredServices: apiKeyCheck.requiredServices,
              setupInstructions: apiKeyCheck.setupInstructions
            }
          }
        };
      }

      // Validate permissions
      await this.validatePermissions(tool, request.context);
      
      // Check rate limits
      const rateLimitResult = await this.checkRateLimit(request.context.userId, tool);
      if (!rateLimitResult.allowed) {
        throw new Error(`Rate limit exceeded. Try again in ${rateLimitResult.retryAfter} seconds`);
      }

      // Find capability
      const capability = tool.capabilities.find(cap => cap.name === request.capability);
      if (!capability) {
        throw new Error(`Capability ${request.capability} not found in tool ${request.toolId}`);
      }

      // Log execution start
      const log: ToolExecutionLog = {
        id: executionId,
        toolId: request.toolId,
        capability: request.capability,
        userId: request.context.userId,
        startTime,
        success: false,
        inputSize: JSON.stringify(request.input).length,
        metadata: {
          requestId: request.context.requestId,
          conversationId: request.context.conversationId,
          botId: request.context.botId
        }
      };
      
      this.executionLogs.set(executionId, log);
      this.activeExecutions.add(executionId);

      console.log(`🚀 Executing tool: ${tool.name} -> ${capability.name}`);

      // Execute tool with timeout
      const timeout = request.options?.timeout || tool.config.timeout || 30000;
      const result = await this.executeWithTimeout(tool, request.input, request.context, timeout);

      // Log successful execution
      const endTime = new Date();
      log.endTime = endTime;
      log.duration = endTime.getTime() - startTime.getTime();
      log.success = true;
      log.outputSize = JSON.stringify(result).length;

      console.log(`✅ Tool execution completed in ${log.duration}ms`);

      return result;

    } catch (error) {
      // Log failed execution
      const endTime = new Date();
      const log = this.executionLogs.get(executionId);
      if (log) {
        log.endTime = endTime;
        log.duration = endTime.getTime() - startTime.getTime();
        log.error = error instanceof Error ? error.message : String(error);
      }

      console.error(`❌ Tool execution failed:`, error);

      return {
        success: false,
        error: {
          code: 'EXECUTION_ERROR',
          message: error instanceof Error ? error.message : String(error),
          details: { executionId, toolId: request.toolId, capability: request.capability }
        }
      };
    } finally {
      this.activeExecutions.delete(executionId);
    }
  }

  // 🌊 STREAMING EXECUTION
  async *executeStream(request: ToolExecutionRequest): AsyncGenerator<ToolProgress> {
    const tool = toolRegistry.getTool(request.toolId);
    if (!tool) {
      throw new Error(`Tool ${request.toolId} not found`);
    }

    // Validate permissions
    await this.validatePermissions(tool, request.context);

    try {
      console.log(`🌊 Starting streaming execution: ${tool.name}`);
      
      const result = tool.handler(request.input, request.context);
      
      if (Symbol.asyncIterator in Object(result)) {
        // Handler returns async generator
        for await (const progress of result as AsyncGenerator<ToolProgress>) {
          yield progress;
        }
      } else {
        // Handler returns promise - convert to single progress event
        const finalResult = await result as ToolResult;
        yield {
          stage: 'complete',
          progress: 100,
          message: 'Execution completed',
          data: finalResult.data
        };
      }
    } catch (error) {
      yield {
        stage: 'error',
        progress: 0,
        message: error instanceof Error ? error.message : String(error),
        data: { error: true }
      };
    }
  }

  // ⏱️ EXECUTION WITH TIMEOUT
  private async executeWithTimeout(
    tool: UniversalTool, 
    input: any, 
    context: ExecutionContext, 
    timeoutMs: number
  ): Promise<ToolResult> {
    return new Promise(async (resolve, reject) => {
      // Set timeout
      const timeoutId = setTimeout(() => {
        reject(new Error(`Tool execution timed out after ${timeoutMs}ms`));
      }, timeoutMs);

      try {
        const result = await tool.handler(input, context);
        clearTimeout(timeoutId);
        
        // Handle different return types
        if (Symbol.asyncIterator in Object(result)) {
          // Async generator - collect all results
          const progress: ToolProgress[] = [];
          for await (const item of result as AsyncGenerator<ToolProgress>) {
            progress.push(item);
          }
          
          const lastProgress = progress[progress.length - 1];
          resolve({
            success: true,
            data: lastProgress?.data,
            metadata: {
              executionTime: Date.now(),
              progressSteps: progress.length
            }
          });
        } else {
          // Regular promise result
          resolve(result as ToolResult);
        }
      } catch (error) {
        clearTimeout(timeoutId);
        reject(error);
      }
    });
  }

  // 🛡️ PERMISSION VALIDATION
  private async validatePermissions(tool: UniversalTool, context: ExecutionContext): Promise<void> {
    for (const permission of tool.permissions) {
      const hasPermission = context.permissions.includes(`${permission.type}:${permission.level}`);
      
      if (!hasPermission) {
        throw new Error(
          `Missing permission: ${permission.type}:${permission.level}. ` +
          `Tool ${tool.name} requires this permission to execute.`
        );
      }
    }
  }

  // 🔑 API KEY REQUIREMENTS CHECK
  private async checkAPIKeyRequirements(tool: UniversalTool, _context: ExecutionContext): Promise<{
    satisfied: boolean;
    missingServices: string[];
    requiredServices: string[];
    setupInstructions: Record<string, any>;
  }> {
    // If tool doesn't have API requirements, it's satisfied
    if (!tool.apiRequirements) {
      return {
        satisfied: true,
        missingServices: [],
        requiredServices: [],
        setupInstructions: {}
      };
    }

    const missingServices: string[] = [];
    const requiredServices: string[] = [];
    const setupInstructions: Record<string, any> = {};

    // Check each API requirement
    for (const [serviceId, requirement] of Object.entries(tool.apiRequirements)) {
      requiredServices.push(serviceId);
      setupInstructions[serviceId] = requirement;

      // Skip if not required (tool can work without it)
      if (!requirement.required) {
        continue;
      }

      // Check if we have the vault service
      if (!this.vaultService || !this.vaultService.isVaultUnlocked()) {
        missingServices.push(serviceId);
        continue;
      }

      // Check if API key exists for this service
      try {
        await this.vaultService.getKey(serviceId);
        // Key exists and is accessible
      } catch (error) {
        // Key doesn't exist or is inaccessible
        missingServices.push(serviceId);
      }
    }

    return {
      satisfied: missingServices.length === 0,
      missingServices,
      requiredServices,
      setupInstructions
    };
  }

  // 🚦 RATE LIMITING
  private async checkRateLimit(userId: string, tool: UniversalTool): Promise<RateLimitResult> {
    const rateLimit = tool.config.rateLimit;
    if (!rateLimit) {
      return { allowed: true, remaining: Infinity, resetTime: new Date(Date.now() + 60000) };
    }

    const key = `${userId}:${tool.id}`;
    const now = new Date();
    const limit = this.rateLimits.get(key);

    if (!limit || now >= limit.resetTime) {
      // Reset or initialize rate limit
      this.rateLimits.set(key, {
        count: 1,
        resetTime: new Date(now.getTime() + (rateLimit.window * 1000))
      });
      
      return {
        allowed: true,
        remaining: rateLimit.requests - 1,
        resetTime: new Date(now.getTime() + (rateLimit.window * 1000))
      };
    }

    if (limit.count >= rateLimit.requests) {
      return {
        allowed: false,
        remaining: 0,
        resetTime: limit.resetTime,
        retryAfter: Math.ceil((limit.resetTime.getTime() - now.getTime()) / 1000)
      };
    }

    // Increment counter
    limit.count++;
    
    return {
      allowed: true,
      remaining: rateLimit.requests - limit.count,
      resetTime: limit.resetTime
    };
  }

  // 🔧 UTILITY METHODS
  private generateExecutionId(): string {
    return `exec_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  // 📊 MONITORING & ANALYTICS
  getActiveExecutions(): ToolExecutionLog[] {
    return Array.from(this.activeExecutions)
      .map(id => this.executionLogs.get(id))
      .filter(Boolean) as ToolExecutionLog[];
  }

  getExecutionHistory(limit: number = 100): ToolExecutionLog[] {
    return Array.from(this.executionLogs.values())
      .sort((a, b) => b.startTime.getTime() - a.startTime.getTime())
      .slice(0, limit);
  }

  getExecutionStats(): {
    totalExecutions: number;
    successRate: number;
    averageExecutionTime: number;
    activeExecutions: number;
  } {
    const logs = Array.from(this.executionLogs.values());
    const successfulLogs = logs.filter(log => log.success);
    const completedLogs = logs.filter(log => log.duration !== undefined);
    
    return {
      totalExecutions: logs.length,
      successRate: logs.length > 0 ? successfulLogs.length / logs.length : 0,
      averageExecutionTime: completedLogs.length > 0 
        ? completedLogs.reduce((sum, log) => sum + (log.duration || 0), 0) / completedLogs.length 
        : 0,
      activeExecutions: this.activeExecutions.size
    };
  }

  // 🧹 CLEANUP
  cleanup(maxAge: number = 24 * 60 * 60 * 1000): void {
    const cutoff = new Date(Date.now() - maxAge);
    const idsToRemove: string[] = [];
    
    this.executionLogs.forEach((log, id) => {
      if (log.startTime < cutoff && !this.activeExecutions.has(id)) {
        idsToRemove.push(id);
      }
    });
    
    idsToRemove.forEach(id => this.executionLogs.delete(id));
    
    console.log(`🧹 Cleaned up ${idsToRemove.length} old execution logs`);
  }
}

// 🌟 SINGLETON INSTANCE
export const toolExecutor = new ToolExecutor();

// Auto-cleanup every hour
setInterval(() => {
  toolExecutor.cleanup();
}, 60 * 60 * 1000);