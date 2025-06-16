export interface User {
  username: string;
  userId: string;
  email?: string;
  groups?: string[];
  accessToken?: string;
}

export interface Config {
  environment: string;
  userPoolId: string;
  userPoolClientId: string;
  apiEndpoint: string;
  websocketEndpoint: string;
  region: string;
}

// Enhanced conversation types matching sophisticated backend
export interface MessageContent {
  type: 'text' | 'image' | 'tool_use' | 'tool_result' | 'reasoning';
  text?: string;
  imageUrl?: string;
  toolUse?: ToolUseContent;
  toolResult?: ToolResultContent;
  metadata?: Record<string, any>;
}

export interface ToolUseContent {
  toolUseId: string;
  name: string;
  input: Record<string, any>;
}

export interface ToolResultContent {
  toolUseId: string;
  content: string;
  isError?: boolean;
}

export interface MessageFeedback {
  thumbsUp?: boolean;
  thumbsDown?: boolean;
  comment?: string;
  createdAt?: Date;
}

export interface Message {
  id: string;
  conversationId: string;
  role: 'user' | 'assistant' | 'system' | 'instruction';
  content: MessageContent[];
  model?: string;
  parentId?: string;
  children?: string[];
  createdAt: Date;
  tokenCount?: number;
  feedback?: MessageFeedback;
  metadata?: Record<string, any>;
}

export interface Conversation {
  id: string;
  userId: string;
  botId?: string;
  title: string;
  sessionModelId?: string; // Override model for this conversation
  messageMap: Record<string, Message>;
  lastMessageId: string;
  shouldContinue?: boolean;
  createdAt: Date;
  updatedAt: Date;
  totalTokens?: number;
  metadata?: Record<string, any>;
  isLargeConversation?: boolean;
  s3Location?: string;
}

export interface ConversationMeta {
  id: string;
  userId: string;
  botId?: string;
  title: string;
  sessionModelId?: string; // Override model for this conversation
  createdAt: Date;
  updatedAt: Date;
  messageCount: number;
  lastMessage?: string;
}

export interface ConversationWithMessages {
  conversation: Conversation;
  messages: Message[];
}

// Chat API types
export interface ChatRequest {
  message: string;
  conversationId?: string;
  stream?: boolean;
  context?: Record<string, string>;
  sessionModelId?: string; // Override model for this session
}

export interface KnowledgeBaseChunk {
  content: string;
  score: number;
  source: string;
  metadata?: Record<string, any>;
}

export interface ChatResponse {
  response: string;
  conversationId: string;
  sources?: KnowledgeBaseChunk[];
  toolsUsed?: string[];
  guardrailApplied?: boolean;
  knowledgeSearchStages?: KnowledgeSearchStage[]; // 🚀 INGENIOUS ENHANCEMENT
  metadata?: Record<string, any>;
}

// 🚀 INGENIOUS ENHANCEMENT: Knowledge Search Stages for Progressive Display
export interface KnowledgeSearchStage {
  stage: string;
  query: string;
  strategy: string;
  result_count: number;
  duration: string;
  success: boolean;
  metadata?: Record<string, any>;
}

// Legacy simple chat message for backwards compatibility
export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: Date;
}

// Enhanced Bot interface matching Go models
export interface Bot {
  // Core properties
  id: string;
  title: string;
  description: string;
  instruction: string;
  ownerUserId: string;
  
  // Timestamps
  createTime: Date;
  lastUsedTime: Date;
  updateTime: Date;
  
  // Knowledge Base configuration
  knowledgeBaseId?: string;
  knowledgeBaseConfig: KnowledgeBaseConfig;
  
  // CloudFormation stack tracking (for dynamically created KBs)
  cloudFormationStackName?: string;
  stackStatus?: string;
  documentBucketName?: string;
  guardrailArn?: string;
  guardrailVersion?: string;
  
  // Async processing status
  syncStatus: 'QUEUED' | 'RUNNING' | 'SUCCEEDED' | 'FAILED';
  syncStatusReason?: string;
  syncLastExecId?: string;
  
  // Sharing configuration (3-tier system)
  sharedScope: 'private' | 'partial' | 'public';
  sharedStatus: 'unshared' | 'shared' | 'pinned';
  allowedUsers?: string[];
  allowedGroups?: string[];
  
  // User preferences
  isStarred: boolean;
  
  // Usage analytics
  usageCount: number;
  
  // Generation and behavior configuration
  generationParams: GenerationParams;
  conversationStarters: ConversationStarter[];
  activeModels: string[];
  agentTools: AgentTool[];
  displayRetrievedChunks: boolean;
}

export interface GenerationParams {
  maxTokens: number;
  temperature: number;
  topP: number;
  topK: number;
  stopSequences: string[];
}

export interface KnowledgeBaseConfig {
  searchType: 'SEMANTIC' | 'HYBRID';
  maxResults: number;
  scoreThreshold: number;
}

export interface ConversationStarter {
  title: string;
  example: string;
}

// Agent Tools (matching Go typed system)
export interface AgentTool {
  type: 'plain' | 'internet' | 'bedrock_agent';
  name: string;
  description: string;
  config?: Record<string, any>;
}

export interface PlainTool {
  name: string;
  description: string;
}

export interface InternetTool {
  name: string;
  description: string;
  searchEngine: 'tavily' | 'duckduckgo' | 'google';
  apiKey?: string;
  maxResults: number;
}

export interface BedrockAgentTool {
  name: string;
  description: string;
  agentId: string;
  agentAlias: string;
  region: string;
}

// Bot Creation Request
export interface CreateBotRequest {
  // Core properties
  title: string;
  description: string;
  instruction: string;
  
  // Conditional Knowledge Base options
  existingKnowledgeBaseId?: string;
  knowledgeBaseCreation?: KnowledgeBaseCreationConfig;
  
  // Sharing configuration
  sharedScope: 'private' | 'partial' | 'public';
  allowedUsers?: string[];
  allowedGroups?: string[];
  
  // Configuration
  generationParams: GenerationParams;
  knowledgeBaseConfig: KnowledgeBaseConfig;
  conversationStarters: ConversationStarter[];
  activeModels: string[];
  agentTools: AgentTool[];
  displayRetrievedChunks: boolean;
  
  // Content Safety
  guardrails?: GuardrailConfig;
}

export interface KnowledgeBaseCreationConfig {
  embeddingsModel?: string;
  chunkingStrategy?: 'FIXED_SIZE' | 'NONE' | 'HIERARCHICAL' | 'SEMANTIC';
  maxTokens?: number;
  overlapPercentage?: number;
  existingS3Urls?: string[];
  sourceUrls?: string[];
  guardrailConfig?: GuardrailConfig;
  enableRagReplicas?: boolean;
}

export interface GuardrailConfig {
  enabled: boolean;
  hateThreshold?: number;
  insultsThreshold?: number;
  sexualThreshold?: number;
  violenceThreshold?: number;
  misconductThreshold?: number;
  groundingThreshold?: number;
  relevanceThreshold?: number;
}

// Document Upload Types
export interface PresignedUploadRequest {
  botId: string;
  fileName: string;
  contentType: string;
  fileSize: number;
}

export interface PresignedUploadResponse {
  uploadUrl: string;
  s3Key: string;
  s3Url: string;
  expiresAt: number;
}

export interface DocumentInfo {
  s3Key: string;
  fileName: string;
  contentType: string;
  size: number;
  lastModified: Date;
  s3Url: string;
}

export interface DocumentListResponse {
  documents: DocumentInfo[];
  count: number;
}

// Bot Summaries for listings
export interface BotSummary {
  id: string;
  title: string;
  description: string;
  isStarred: boolean;
  ownerUserId: string;
  createTime: Date;
  lastUsedTime: Date;
  hasKnowledgeBase: boolean;
  conversationStarters: ConversationStarter[];
  sharedScope: string;
  sharedStatus: string;
  activeModels: string[]; // Add this for compatibility
}