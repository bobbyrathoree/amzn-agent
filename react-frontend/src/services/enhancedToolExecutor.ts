// 🚀 ENHANCED TOOL EXECUTOR
// Netflix-like tool execution with intelligent fallbacks

import { ServiceFallbackManager } from './serviceFallbackManager';
import type { 
  UniversalTool, 
  ToolResult, 
  ExecutionContext, 
  ToolExecutionRequest,
  ServiceFallback
} from '../types/tools';

export interface ExecutionPlan {
  tool: UniversalTool;
  capability: string;
  selectedService: ServiceFallback;
  fallbackLevel: number;
  reasoning: string;
  estimatedTime: number;
  estimatedCost: number;
}

/**
 * 🚀 Enhanced Tool Executor
 * Executes tools with intelligent fallback system
 */
export class EnhancedToolExecutor {
  private fallbackManager: ServiceFallbackManager;
  private vaultService: any;

  constructor(vaultService: any) {
    this.vaultService = vaultService;
    this.fallbackManager = new ServiceFallbackManager(vaultService);
  }

  /**
   * Execute tool with intelligent fallback system
   */
  async execute(request: ToolExecutionRequest): Promise<ToolResult> {
    console.log(`🚀 Enhanced execution for tool: ${request.toolId}`);

    try {
      // Get the tool
      const tool = await this.getTool(request.toolId);
      if (!tool) {
        throw new Error(`Tool not found: ${request.toolId}`);
      }

      // Get the capability
      const capability = tool.capabilities.find(cap => cap.name === request.capability);
      if (!capability) {
        throw new Error(`Capability not found: ${request.capability}`);
      }

      // Plan execution with fallback system
      const plan = await this.planExecution(tool, capability, request.context);
      
      console.log(`📋 Execution plan:`, plan);

      // Execute with selected service
      const result = await this.executeWithService(
        plan.tool,
        plan.capability,
        plan.selectedService,
        request.input,
        request.context
      );

      // Enhance result with service info
      return this.enhanceResultWithServiceInfo(result, plan);

    } catch (error) {
      console.error(`❌ Enhanced execution failed:`, error);
      return {
        success: false,
        error: {
          code: 'ENHANCED_EXECUTION_ERROR',
          message: error instanceof Error ? error.message : 'Execution failed'
        }
      };
    }
  }

  /**
   * Plan execution with fallback system
   */
  private async planExecution(
    tool: UniversalTool,
    capability: any,
    _context: ExecutionContext
  ): Promise<ExecutionPlan> {
    console.log(`📋 Planning execution for ${tool.name}:${capability.name}`);

    // If no fallback hierarchy, use direct execution
    if (!capability.fallbackHierarchy || capability.fallbackHierarchy.length === 0) {
      return {
        tool,
        capability,
        selectedService: {
          serviceId: 'direct',
          priority: 1,
          mode: 'free'
        },
        fallbackLevel: 0,
        reasoning: 'Direct execution (no fallback hierarchy)',
        estimatedTime: capability.timeEstimate || 5000,
        estimatedCost: capability.costEstimate?.credits || 0
      };
    }

    // Use fallback manager to select best service
    const selection = await this.fallbackManager.selectBestService(capability, _context);

    return {
      tool,
      capability,
      selectedService: selection.selectedService,
      fallbackLevel: selection.fallbackLevel,
      reasoning: selection.reasoning,
      estimatedTime: this.calculateEstimatedTime(capability, selection.selectedService),
      estimatedCost: this.calculateEstimatedCost(capability, selection.selectedService)
    };
  }

  /**
   * Execute tool with specific service
   */
  private async executeWithService(
    tool: UniversalTool,
    _capability: string,
    service: ServiceFallback,
    input: any,
    context: ExecutionContext
  ): Promise<ToolResult> {
    console.log(`⚡ Executing with service: ${service.serviceId}`);

    // Create enhanced context with service info
    const enhancedContext = {
      ...context,
      selectedService: service,
      vaultService: this.vaultService
    };

    try {
      // Execute the tool with enhanced context
      const result = await tool.handler(input, enhancedContext);
      
      // If result is async generator (streaming), handle it
      if (this.isAsyncGenerator(result)) {
        return await this.handleStreamingExecution(result);
      }

      // If result is array of progress updates, return final result
      if (Array.isArray(result)) {
        const finalUpdate = result[result.length - 1];
        return {
          success: true,
          data: finalUpdate.data,
          metadata: finalUpdate.metadata
        };
      }

      // Direct result
      return result as ToolResult;

    } catch (error) {
      console.error(`❌ Service execution failed:`, error);
      
      // If it's an API key error, provide upgrade suggestions
      if (error instanceof Error && error.message.includes('API_KEY_REQUIRED')) {
        return {
          success: false,
          error: {
            code: 'API_KEY_REQUIRED',
            message: `${service.serviceId} API key required for premium features`,
            details: { serviceId: service.serviceId }
          }
        };
      }

      throw error;
    }
  }

  /**
   * Handle streaming execution
   */
  private async handleStreamingExecution(generator: AsyncGenerator<any>): Promise<ToolResult> {
    let finalResult: ToolResult = { success: false };

    try {
      for await (const progress of generator) {
        if (progress.stage === 'complete') {
          finalResult = {
            success: true,
            data: progress.data,
            metadata: progress.metadata || {}
          };
          break;
        }
      }
    } catch (error) {
      finalResult = {
        success: false,
        error: {
          code: 'STREAMING_ERROR',
          message: error instanceof Error ? error.message : 'Streaming execution failed'
        }
      };
    }

    return finalResult;
  }

  /**
   * Enhance result with service information
   */
  private enhanceResultWithServiceInfo(
    result: ToolResult,
    plan: ExecutionPlan
  ): ToolResult {
    const serviceInfo = {
      usedService: plan.selectedService.serviceId,
      serviceMode: plan.selectedService.mode,
      fallbackLevel: plan.fallbackLevel,
      limitations: plan.selectedService.limitations ? {
        quality: plan.selectedService.limitations.quality || 'medium',
        features: plan.selectedService.limitations.features || [],
        maxResults: plan.selectedService.limitations.maxResults
      } : undefined,
      availableUpgrades: plan.fallbackLevel > 0 ? 
        this.getUpgradeOptions(plan) : undefined
    };

    return {
      ...result,
      serviceInfo,
      metadata: {
        ...result.metadata,
        executionPlan: {
          reasoning: plan.reasoning,
          estimatedTime: plan.estimatedTime,
          estimatedCost: plan.estimatedCost
        }
      }
    };
  }

  /**
   * Get upgrade options for current execution
   */
  private getUpgradeOptions(plan: ExecutionPlan) {
    const capability = plan.tool.capabilities.find(c => c.name === plan.capability);
    if (!capability?.fallbackHierarchy) return [];

    const higherPriorityServices = capability.fallbackHierarchy
      .filter(s => s.priority < plan.selectedService.priority)
      .sort((a, b) => a.priority - b.priority);

    return higherPriorityServices.map(service => ({
      serviceId: service.serviceId,
      benefits: this.getServiceBenefits(service, plan.selectedService),
      setupRequired: service.requirements?.apiKey === true
    }));
  }

  /**
   * Get benefits of upgrading to a service
   */
  private getServiceBenefits(targetService: ServiceFallback, currentService: ServiceFallback): string[] {
    const benefits = [];

    if (targetService.mode === 'premium' && currentService.mode === 'free') {
      benefits.push('Premium quality results');
    }

    if (targetService.limitations?.quality === 'high' && 
        currentService.limitations?.quality !== 'high') {
      benefits.push('Higher quality results');
    }

    return benefits.length > 0 ? benefits : ['Enhanced service features'];
  }

  /**
   * Calculate estimated execution time
   */
  private calculateEstimatedTime(capability: any, service: ServiceFallback): number {
    let baseTime = capability.timeEstimate || 5000;

    // Adjust based on service mode
    if (service.mode === 'free') {
      baseTime *= 1.5; // Free services might be slower
    }

    // Adjust based on fallback level
    if (service.priority > 1) {
      baseTime *= 1.2; // Fallback services might be slightly slower
    }

    return Math.round(baseTime);
  }

  /**
   * Calculate estimated cost
   */
  private calculateEstimatedCost(capability: any, service: ServiceFallback): number {
    if (service.mode === 'free') return 0;
    
    return capability.costEstimate?.credits || 1;
  }

  /**
   * Get tool by ID (placeholder - would integrate with tool registry)
   */
  private async getTool(_toolId: string): Promise<UniversalTool | null> {
    // This would typically integrate with a tool registry
    // For now, return null to trigger fallback to existing system
    return null;
  }

  /**
   * Check if result is async generator
   */
  private isAsyncGenerator(obj: any): obj is AsyncGenerator<any> {
    return obj && typeof obj[Symbol.asyncIterator] === 'function';
  }
}