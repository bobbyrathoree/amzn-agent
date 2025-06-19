// 🛠️ UNIVERSAL TOOLS TYPE SYSTEM
// State-of-the-art type definitions for the bot tools framework

export type ToolCategory = 
  | 'information'
  | 'creative'
  | 'business'
  | 'development'
  | 'research'
  | 'automation'
  | 'communication';

export type PermissionType = 
  | 'internet-access'
  | 'database-access'
  | 'file-access'
  | 'external-api'
  | 'system-access'
  | 'user-data';

export type PermissionLevel = 
  | 'read'
  | 'write'
  | 'execute'
  | 'admin';

export interface ParameterDefinition {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'object' | 'array' | 'enum';
  description: string;
  required?: boolean;
  default?: any;
  enum?: string[]; // For enum type
  items?: string; // For array type
  properties?: Record<string, ParameterDefinition>; // For object type
  examples?: any[];
}

export interface ToolCapability {
  name: string;
  description: string;
  parameters: ParameterDefinition[];
  examples: string[];
  costEstimate?: {
    credits?: number;
    description: string;
  };
  timeEstimate?: number; // milliseconds
  
  // 🔧 FALLBACK SYSTEM
  fallbackHierarchy?: ServiceFallback[];
  serviceMode?: 'premium' | 'free' | 'hybrid';
}

export interface ServiceFallback {
  serviceId: string;
  priority: number; // 1 = highest priority
  mode: 'premium' | 'free';
  requirements?: {
    apiKey?: boolean;
    subscription?: boolean;
    credits?: number;
  };
  limitations?: {
    maxRequests?: number;
    maxResults?: number;
    features?: string[];
    quality?: 'high' | 'medium' | 'low';
  };
  fallbackReason?: string;
}

export interface PermissionRequirement {
  type: PermissionType;
  level: PermissionLevel;
  scope?: string[];
  description?: string;
}

export interface ToolExample {
  name: string;
  description: string;
  input: any;
  expectedOutput: any;
}

export interface ToolConfig {
  timeout?: number; // milliseconds
  maxMemory?: string; // e.g., '256MB'
  maxCpu?: string; // e.g., '0.5'
  retries?: number;
  rateLimit?: {
    requests: number;
    window: number; // seconds
  };
  cache?: {
    enabled: boolean;
    ttl: number; // seconds
  };
}

export interface ExecutionContext {
  userId: string;
  conversationId?: string;
  botId?: string;
  permissions: string[];
  requestId: string;
  timestamp: Date;
  vaultService?: any; // API Key Vault service for premium features
  metadata?: Record<string, any>;
}

export interface ToolProgress {
  stage: string;
  progress: number; // 0-100
  message: string;
  data?: any;
  metadata?: Record<string, any>;
}

export interface ToolResult {
  success: boolean;
  data?: any;
  error?: {
    code: string;
    message: string;
    details?: any;
  };
  metadata?: {
    executionTime?: number;
    tokensUsed?: number;
    cost?: number;
    [key: string]: any;
  };
  citations?: Array<{
    source: string;
    url?: string;
    title?: string;
    excerpt?: string;
  }>;
  
  // 🔧 FALLBACK SYSTEM
  serviceInfo?: {
    usedService: string;
    serviceMode: 'premium' | 'free';
    fallbackLevel: number; // 0 = primary, 1+ = fallback levels
    availableUpgrades?: Array<{
      serviceId: string;
      benefits: string[];
      setupRequired: boolean;
    }>;
    limitations?: {
      quality?: 'high' | 'medium' | 'low';
      features?: string[];
      maxResults?: number;
    };
  };
}

export type ToolHandler = (
  input: any, 
  context: ExecutionContext
) => Promise<ToolResult> | Promise<ToolProgress[]> | AsyncGenerator<ToolProgress>;

// 🔧 UNIVERSAL TOOL INTERFACE
export interface UniversalTool {
  // Meta information
  id: string;
  name: string;
  description: string;
  category: ToolCategory;
  version: string;
  author: string;
  
  // Capabilities
  capabilities: ToolCapability[];
  
  // Runtime
  handler: ToolHandler;
  permissions: PermissionRequirement[];
  config: ToolConfig;
  
  // API Key requirements
  apiRequirements?: {
    [serviceName: string]: {
      required: boolean;
      keyName: string;
      description: string;
      signupUrl?: string;
      testEndpoint?: string;
      pricingInfo?: string;
      setupInstructions: string[];
    };
  };
  
  // Discovery
  tags: string[];
  keywords: string[];
  examples: ToolExample[];
  
  // UI
  icon?: string;
  color?: string;
  featured?: boolean;
}

// 🔍 TOOL EXECUTION INTERFACES
export interface ToolExecutionRequest {
  toolId: string;
  capability: string;
  input: any;
  context: ExecutionContext;
  options?: {
    stream?: boolean;
    timeout?: number;
    priority?: 'low' | 'normal' | 'high';
  };
}

export interface ToolExecutionLog {
  id: string;
  toolId: string;
  capability: string;
  userId: string;
  startTime: Date;
  endTime?: Date;
  duration?: number;
  success: boolean;
  error?: string;
  inputSize: number;
  outputSize?: number;
  metadata?: Record<string, any>;
}

export interface ToolPerformanceMetrics {
  toolId: string;
  totalExecutions: number;
  successRate: number;
  averageExecutionTime: number;
  errorRate: number;
  lastExecuted: Date;
  popularCapabilities: Array<{
    capability: string;
    usage: number;
  }>;
}

// 🛒 TOOL MARKETPLACE INTERFACES
export interface MarketplaceTool extends UniversalTool {
  rating: number;
  downloadCount: number;
  lastUpdated: Date;
  publisher: {
    name: string;
    verified: boolean;
    website?: string;
  };
  pricing: {
    type: 'free' | 'premium' | 'freemium';
    cost?: number;
    billingPeriod?: 'one-time' | 'monthly' | 'yearly';
  };
  screenshots?: string[];
  documentation?: string;
}

export interface ToolRating {
  toolId: string;
  averageRating: number;
  totalRatings: number;
  distribution: Record<number, number>; // rating -> count
  reviews: Array<{
    userId: string;
    rating: number;
    review?: string;
    date: Date;
    helpful: number;
  }>;
}

// 🔗 TOOL COMPOSITION INTERFACES
export interface ToolPipelineStep {
  toolId: string;
  capability: string;
  input: any;
  outputMapping?: Record<string, string>; // Map output fields to next step's input
}

export interface ToolPipeline {
  id: string;
  name: string;
  description: string;
  steps: ToolPipelineStep[];
  metadata?: Record<string, any>;
}

// 🎯 SMART ROUTING INTERFACES
export interface ToolRoutingRequest {
  intent: string;
  context: ExecutionContext;
  availableTools: string[];
  preferences?: {
    preferredTools?: string[];
    avoidTools?: string[];
    maxExecutionTime?: number;
    maxCost?: number;
  };
}

export interface ToolRoutingPlan {
  confidence: number;
  estimatedTime: number;
  estimatedCost: number;
  steps: Array<{
    toolId: string;
    capability: string;
    reasoning: string;
    confidence: number;
  }>;
  alternatives?: ToolRoutingPlan[];
}

// 🔐 SECURITY INTERFACES
export interface PermissionGrant {
  userId: string;
  permission: PermissionRequirement;
  granted: boolean;
  expiresAt?: Date;
  conditions?: string[];
  grantedBy: string;
  grantedAt: Date;
}

export interface RateLimitResult {
  allowed: boolean;
  remaining: number;
  resetTime: Date;
  retryAfter?: number; // seconds
}

// 📊 MONITORING INTERFACES
export interface ToolMonitoringData {
  activeExecutions: ToolExecutionLog[];
  recentExecutions: ToolExecutionLog[];
  performanceMetrics: ToolPerformanceMetrics[];
  errorRates: Record<string, number>;
  userUsageStats: Array<{
    userId: string;
    toolUsage: Record<string, number>;
    totalExecutions: number;
    lastActive: Date;
  }>;
  systemHealth: {
    cpu: number;
    memory: number;
    activeConnections: number;
    queueDepth: number;
  };
}