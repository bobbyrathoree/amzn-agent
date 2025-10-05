import { useState, useEffect, useRef, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useAuth } from '../components/AuthProvider';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  MessageCircle, 
  Send, 
  Plus, 
  Menu, 
  X, 
  Bot as BotIcon, 
  Sparkles, 
  Trash2,
  Zap,
  Brain,
  Clock,
  User,
  RefreshCw
} from 'lucide-react';
import { useApiClient, ApiClient } from '../lib/api';
import { BotToolsPanel } from '../components/BotToolsPanel';
import { KnowledgeSearchStages } from '../components/KnowledgeSearchStages';
import { SourceCitations } from '../components/SourceCitations';
import { ExtendedThinkingToggle } from '../components/ExtendedThinkingToggle';
import { KnowledgeBaseToggle } from '../components/KnowledgeBaseToggle';
import { BotSelector } from '../components/BotSelector';
import { Breadcrumbs } from '../components/Breadcrumbs';
import { ThemeToggle } from '../components/ThemeToggle';
import { ToolExecutionResults } from '../components/ToolExecutionResults';
import { KeyRecommendations } from '../components/KeyRecommendations';
import { ToolResultRenderer } from '../components/tool-results';
import { VaultUnlockModal } from '../components/VaultUnlockModal';
import { APIKeySetupModal } from '../components/APIKeySetupModal';
import { StreamingProgressIndicator, CompactStreamingIndicator } from '../components/StreamingProgressIndicator';
import { useStreamingExecution } from '../hooks/useStreamingExecution';
import { MarkdownRenderer } from '../components/MarkdownRenderer';
import { useTheme } from '../components/ThemeProvider';
import { useUnifiedTools } from '../hooks/useUnifiedTools';
import { getTool } from '../tools';
import { APIKeyVaultService } from '../services/vaultService';
import type { 
  Bot, 
  Conversation, 
  ConversationMeta, 
  Message, 
  ChatRequest, 
  MessageContent,
  ChatResponse,
  KnowledgeSearchStage,
  ReasoningParams,
  KnowledgeBaseChunk,
  ToolExecution,
  ToolSkipped,
  KeyRecommendation,
  ConversationStarter
} from '../types';

export function ChatPage() {
  const { botId } = useParams<{ botId: string }>();
  const { user, signOut, getAccessToken } = useAuth();
  const getUserId = useCallback(() => user?.userId || user?.username || null, [user?.userId, user?.username]);
  const apiClient = useApiClient(getAccessToken, getUserId);
  const { theme } = useTheme();
  
  // State management
  const [bot, setBot] = useState<Bot | null>(null);
  
  // 🔧 NEW: Unified Tools Integration
  const {
    executeToolInChat,
    executeToolWithProgress, 
    isExecuting: isToolExecuting,
    executionProgress
  } = useUnifiedTools({ 
    apiClient: apiClient!, 
    user: user!, 
    bot: bot || undefined 
  });
  const [allBots, setAllBots] = useState<Bot[]>([]);
  const [conversations, setConversations] = useState<ConversationMeta[]>([]);
  const [, setCurrentConversation] = useState<Conversation | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  
  // Debug wrapper for setMessages to catch null assignments
  const setMessagesDebug = useCallback((newMessages: Message[] | ((prev: Message[]) => Message[])) => {
    if (typeof newMessages === 'function') {
      setMessages(prev => {
        const result = newMessages(prev);
        if (!Array.isArray(result)) {
          console.error('setMessages function returned non-array:', result);
          return [];
        }
        return result;
      });
    } else {
      if (!Array.isArray(newMessages)) {
        console.error('setMessages called with non-array:', newMessages);
        setMessages([]);  
      } else {
        setMessages(newMessages);
      }
    }
  }, []);
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedConversationId, setSelectedConversationId] = useState<string | null>(null);
  
  // Race condition prevention for message operations
  const [lastMessageId, setLastMessageId] = useState<string | null>(null);
  
  // Session model state (not persisted to DB)
  const [sessionModel, setSessionModel] = useState<string | null>(null);
  const [showNewChatDialog, setShowNewChatDialog] = useState(false);
  
  // UI state
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // 🚀 INGENIOUS ENHANCEMENT: Knowledge search stages tracking
  const [currentSearchStages, setCurrentSearchStages] = useState<KnowledgeSearchStage[]>([]);
  const [lastChatResponse, setLastChatResponse] = useState<ChatResponse | null>(null);
  
  // 🚀 ENHANCEMENT: Sources persistence per conversation
  const [conversationSources, setConversationSources] = useState<Record<string, KnowledgeBaseChunk[]>>({});
  
  // 🛠️ Tool execution state management
  const [toolExecutions, setToolExecutions] = useState<ToolExecution[]>([]);
  const [toolSkipped, setToolSkipped] = useState<ToolSkipped[]>([]);
  const [keyRecommendations, setKeyRecommendations] = useState<KeyRecommendation[]>([]);
  const [toolResults, setToolResults] = useState<any[]>([]); // Store full tool result data
  
  // 🌊 Streaming tool execution state (for potential future use)
  // const [activeStreamingTools, setActiveStreamingTools] = useState<Map<string, any>>(new Map());
  
  // 🌊 Streaming execution hook
  const streamingExecution = useStreamingExecution({
    onComplete: (result) => {
      console.log('🌊 Streaming tool execution completed:', result);
      // Handle streaming completion - could update chat or show results
    },
    onError: (error) => {
      console.error('🌊 Streaming tool execution failed:', error);
    },
    onProgress: (progress) => {
      console.log('🌊 Streaming progress:', progress);
    }
  });
  
  // 🔑 Vault unlock flow state
  const [showVaultUnlock, setShowVaultUnlock] = useState(false);
  const [showAPIKeySetup, setShowAPIKeySetup] = useState(false);
  const [selectedServiceForSetup, setSelectedServiceForSetup] = useState<string>('');
  const [vaultService, setVaultService] = useState<APIKeyVaultService | null>(null);
  
  // Extended thinking state
  const [extendedThinkingEnabled, setExtendedThinkingEnabled] = useState(false);
  const [reasoningParams, setReasoningParams] = useState<ReasoningParams>({ budgetTokens: 1024 });
  
  // Knowledge Base toggle state
  const [knowledgeBaseEnabled, setKnowledgeBaseEnabled] = useState(true);

  // Available models for dropdown with extended thinking support
  const availableModels = [
    // Claude Models (US inference profiles) - Extended thinking supported
    { id: 'us.anthropic.claude-opus-4-20250514-v1:0', name: 'Claude 4 Opus', family: 'claude', supportsReasoning: true },
    { id: 'us.anthropic.claude-sonnet-4-20250514-v1:0', name: 'Claude 4 Sonnet', family: 'claude', supportsReasoning: true },
    { id: 'us.anthropic.claude-3-7-sonnet-20250219-v1:0', name: 'Claude 3.7 Sonnet', family: 'claude', supportsReasoning: true },
    { id: 'us.anthropic.claude-3-5-sonnet-20241022-v2:0', name: 'Claude 3.5 Sonnet v2', family: 'claude', supportsReasoning: false },
    { id: 'us.anthropic.claude-3-5-sonnet-20240620-v1:0', name: 'Claude 3.5 Sonnet', family: 'claude', supportsReasoning: false },
    { id: 'us.anthropic.claude-3-5-haiku-20241022-v1:0', name: 'Claude 3.5 Haiku', family: 'claude', supportsReasoning: false },
    { id: 'us.anthropic.claude-3-haiku-20240307-v1:0', name: 'Claude 3 Haiku', family: 'claude', supportsReasoning: false },
    { id: 'us.anthropic.claude-3-opus-20240229-v1:0', name: 'Claude 3 Opus', family: 'claude', supportsReasoning: false },
    
    // Amazon Nova Models - No extended thinking support
    { id: 'us.amazon.nova-pro-v1:0', name: 'Nova Pro', family: 'nova', supportsReasoning: false },
    { id: 'us.amazon.nova-lite-v1:0', name: 'Nova Lite', family: 'nova', supportsReasoning: false },
    { id: 'us.amazon.nova-micro-v1:0', name: 'Nova Micro', family: 'nova', supportsReasoning: false },
    
    // Mistral Models - No extended thinking support
    { id: 'us.mistral.mistral-large-2407-v1:0', name: 'Mistral Large 2407', family: 'mistral', supportsReasoning: false },
    { id: 'us.mistral.mistral-large-2402-v1:0', name: 'Mistral Large 2402', family: 'mistral', supportsReasoning: false },
    { id: 'us.mistral.mixtral-8x7b-instruct-v0:1', name: 'Mixtral 8x7B', family: 'mistral', supportsReasoning: false },
    { id: 'us.mistral.mistral-7b-instruct-v0:2', name: 'Mistral 7B', family: 'mistral', supportsReasoning: false },
    
    // DeepSeek Models - Built-in reasoning, always enabled
    { id: 'us.deepseek.r1-v1:0', name: 'DeepSeek R1', family: 'deepseek', supportsReasoning: true, forceReasoningEnabled: true },
    
    // Meta Llama Models - No extended thinking support
    { id: 'us.meta.llama3-3-70b-instruct-v1:0', name: 'Llama 3.3 70B', family: 'llama', supportsReasoning: false },
    { id: 'us.meta.llama3-2-90b-instruct-v1:0', name: 'Llama 3.2 90B', family: 'llama', supportsReasoning: false },
    { id: 'us.meta.llama3-2-11b-instruct-v1:0', name: 'Llama 3.2 11B', family: 'llama', supportsReasoning: false },
    { id: 'us.meta.llama3-2-3b-instruct-v1:0', name: 'Llama 3.2 3B', family: 'llama', supportsReasoning: false },
    { id: 'us.meta.llama3-2-1b-instruct-v1:0', name: 'Llama 3.2 1B', family: 'llama', supportsReasoning: false }
  ];

  // Load bot and conversations on mount
  useEffect(() => {
    if (apiClient && botId) {
      loadBot();
      loadConversations();
    }
    if (apiClient) {
      loadAllBots();
    }
  }, [apiClient, botId]);

  // Initialize vault service
  useEffect(() => {
    if (apiClient) {
      const service = new APIKeyVaultService(apiClient);
      setVaultService(service);
    }
  }, [apiClient]);

  // Auto-scroll to bottom when messages change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // Update extended thinking for DeepSeek models
  useEffect(() => {
    const model = getCurrentModel();
    if (model?.forceReasoningEnabled) {
      setExtendedThinkingEnabled(true);
    }
  }, [sessionModel]);

  const getBreadcrumbItems = () => {
    const items: Array<{ label: string; href?: string }> = [
      { label: 'Home', href: '/' },
      { label: 'Bots', href: '/bots' }
    ];
    
    if (bot) {
      items.push({ label: bot.title });
    }
    
    return items;
  };

  const loadBot = useCallback(async () => {
    if (!apiClient || !botId) return;

    try {
      const response = await apiClient.get(`bots/${botId}`);
      if (!response.ok) {
        if (response.status === 404) {
          setError('Bot not found');
          return;
        }
        if (response.status === 403) {
          setError('Access denied to this bot');
          return;
        }
        throw new Error(`Failed to load bot: ${response.status}`);
      }
      
      const data = await response.json();
      setBot(data.bot);
      
      // Initialize session model with bot's default model (first active model)
      if (data.bot.activeModels && data.bot.activeModels.length > 0 && !sessionModel) {
        setSessionModel(data.bot.activeModels[0]);
      }
    } catch (err) {
      console.error('Error loading bot:', err);
      setError(ApiClient.getErrorMessage(err));
    }
  }, [apiClient, botId]);

  const loadAllBots = useCallback(async () => {
    if (!apiClient) return;

    try {
      const response = await apiClient.get('bots');
      if (response.ok) {
        const data = await response.json();
        setAllBots(data.bots || []);
      }
    } catch (err) {
      console.error('Error loading all bots:', err);
    }
  }, [apiClient]);

  const loadConversations = useCallback(async () => {
    if (!apiClient || !botId) return;

    try {
      const response = await apiClient.get(`bots/${botId}/conversations`);
      if (response.ok) {
        const data = await response.json();
        setConversations(data.conversations || []);
      }
    } catch (err) {
      console.error('Error loading conversations:', err);
    }
  }, [apiClient, botId]);

  const loadConversation = useCallback(async (conversationId: string, force = false) => {
    if (!apiClient || !botId || (!force && isLoading)) return;

    try {
      setIsLoading(true);
      // Clear messages immediately when starting to load a new conversation
      setMessagesDebug([]);
      setCurrentConversation(null);
      setError(null);
      
      // Clear current search stages when switching conversations
      setCurrentSearchStages([]);
      setLastChatResponse(null);
      
      // 🛠️ TOOL INTEGRATION: Clear tool execution state when switching conversations
      setToolExecutions([]);
      setToolSkipped([]);
      setKeyRecommendations([]);
      setToolResults([]);
      
      const response = await apiClient.get(`bots/${botId}/conversations/${conversationId}`);
      
      if (!response.ok) {
        const errorText = await response.text();
        console.error('Load conversation failed:', response.status, errorText);
        throw new Error(`Failed to load conversation: ${response.status} - ${errorText}`);
      }

      // Check content type here too
      const contentType = response.headers.get('content-type');
      if (!contentType || !contentType.includes('application/json')) {
        const responseText = await response.text();
        console.error('Expected JSON but got:', contentType, responseText.substring(0, 500));
        throw new Error(`Server returned HTML instead of JSON when loading conversation`);
      }

      const data = await response.json();
      console.log('Conversation loaded:', conversationId, 'Messages:', data.messages?.length || 0);
      
      // Defensive programming: ensure we always have valid arrays
      setCurrentConversation(data.conversation || null);
      const validMessages = Array.isArray(data.messages) ? data.messages : [];
      console.log('Loading conversation messages:', {
        conversationId,
        totalMessages: validMessages.length,
        messageRoles: validMessages.map((m: Message) => ({ id: m.id, role: m.role, hasContent: !!m.content }))
      });
      setMessagesDebug(validMessages);
      setSelectedConversationId(conversationId);
      
      // Update session model to match conversation's session model (if any)
      if (data.conversation.sessionModelId) {
        setSessionModel(data.conversation.sessionModelId);
      } else if (bot?.activeModels && bot.activeModels.length > 0) {
        // Fall back to bot's default model if conversation has no session model
        setSessionModel(bot.activeModels[0]);
      }
      
      // FIXED: Only restore sources if we don't have a current response
      // This prevents old sources from conflicting with new responses
      if (conversationSources[conversationId] && !lastChatResponse) {
        // Restore the last chat response with sources for this conversation
        setLastChatResponse({ 
          response: '', 
          conversationId, 
          sources: conversationSources[conversationId] 
        });
      }
    } catch (err) {
      console.error('Error loading conversation:', err);
      setError(ApiClient.getErrorMessage(err));
      // Reset to safe state on error to prevent null.map() errors
      setMessages([]);
      setCurrentConversation(null);
      setSelectedConversationId(null);
    } finally {
      setIsLoading(false);
    }
  }, [apiClient, botId, isLoading]);

  const startNewConversation = async (): Promise<string | null> => {
    if (!apiClient || !botId) return null;

    try {
      setIsLoading(true);
      setError(null);
      
      // Clear current conversation state first
      setCurrentConversation(null);
      setMessages([]);
      setSelectedConversationId(null);
      
      console.log('Creating new conversation with session model:', sessionModel);
      
      const response = await apiClient.post(`bots/${botId}/conversations`, {
        title: `New conversation with ${bot?.title || 'Bot'}`,
        sessionModelId: sessionModel || undefined
      });

      if (!response.ok) {
        const errorText = await response.text();
        console.error('Create conversation failed:', response.status, errorText);
        throw new Error(`Failed to create conversation: ${response.status} - ${errorText}`);
      }

      const data = await response.json();
      const newConversationId = data.conversationId;
      console.log('New conversation created:', newConversationId);
      
      // Reload conversations list and select the new one
      await loadConversations();
      await loadConversation(newConversationId, true);
      
      return newConversationId;
    } catch (err) {
      console.error('Error creating conversation:', err);
      setError(ApiClient.getErrorMessage(err));
      return null;
    } finally {
      setIsLoading(false);
    }
  };

  const deleteConversation = async (conversationId: string) => {
    if (!apiClient || !botId) return;

    try {
      setIsLoading(true);
      const response = await apiClient.delete(`bots/${botId}/conversations/${conversationId}`);
      
      if (!response.ok) {
        const errorText = await response.text();
        console.error('Delete conversation failed:', response.status, errorText);
        throw new Error(`Failed to delete conversation: ${response.status} - ${errorText}`);
      }

      console.log('Conversation deleted:', conversationId);
      
      // If we're deleting the current conversation, clear the state
      if (selectedConversationId === conversationId) {
        setCurrentConversation(null);
        setMessages([]);
        setSelectedConversationId(null);
      }
      
      // Reload conversations list
      await loadConversations();
      
    } catch (err) {
      console.error('Error deleting conversation:', err);
      setError(ApiClient.getErrorMessage(err));
    } finally {
      setIsLoading(false);
    }
  };

  // Create conversation for chat without interfering with loading state
  const startNewConversationForChat = async (): Promise<string | null> => {
    if (!apiClient || !botId) return null;

    try {
      console.log('Creating new conversation for chat with session model:', sessionModel);
      
      const response = await apiClient.post(`bots/${botId}/conversations`, {
        title: `New conversation with ${bot?.title || 'Bot'}`,
        sessionModelId: sessionModel || undefined
      });

      if (!response.ok) {
        const errorText = await response.text();
        console.error('Create conversation failed:', response.status, errorText);
        throw new Error(`Failed to create conversation: ${response.status} - ${errorText}`);
      }

      const data = await response.json();
      const newConversationId = data.conversationId;
      console.log('New conversation created for chat:', newConversationId);
      
      // Update selected conversation ID and load conversation state 
      setSelectedConversationId(newConversationId);
      
      // Reload conversations list in background
      loadConversations(); // Don't await to keep chat flowing
      
      return newConversationId;
    } catch (err) {
      console.error('Error creating conversation for chat:', err);
      throw err; // Re-throw to be handled by sendMessage
    }
  };

  const handleNewConversationClick = () => {
    // Show model selection dialog for new conversations
    setShowNewChatDialog(true);
  };


  const startNewConversationWithModel = async (modelId: string) => {
    try {
      setSessionModel(modelId);
      await startNewConversation();
      setShowNewChatDialog(false);
    } catch (err) {
      console.error('Error starting new conversation with model:', err);
    }
  };

  const getCurrentModelName = () => {
    if (!sessionModel) return 'Loading...';
    const model = availableModels.find(m => m.id === sessionModel);
    return model ? model.name : sessionModel;
  };
  
  const getCurrentModel = () => {
    if (!sessionModel) return null;
    return availableModels.find(m => m.id === sessionModel) || null;
  };
  
  const currentModelSupportsReasoning = () => {
    const model = getCurrentModel();
    return model?.supportsReasoning ?? false;
  };
  
  const handleExtendedThinkingToggle = (enabled: boolean, params?: ReasoningParams) => {
    setExtendedThinkingEnabled(enabled);
    if (params) {
      setReasoningParams(params);
    }
  };

  // 🔑 VAULT UNLOCK FLOW - Complete integration
  const handleAddAPIKey = async (serviceId: string) => {
    console.log('🔑 Starting API key setup flow for service:', serviceId);
    
    // Step 1: Check if vault is already unlocked
    try {
      // Try to access vault - if this fails, we need to unlock first
      if (!apiClient) {
        console.warn('API client not available');
        setSelectedServiceForSetup(serviceId);
        setShowVaultUnlock(true);
        return;
      }
      
      const testResponse = await apiClient.get('vault/status');
      
      if (testResponse.ok) {
        const vaultStatus = await testResponse.json();
        
        if (vaultStatus.unlocked) {
          // Vault is already unlocked, go directly to API key setup
          console.log('✅ Vault already unlocked, proceeding to API key setup');
          setSelectedServiceForSetup(serviceId);
          setShowAPIKeySetup(true);
        } else {
          // Vault is locked, need to unlock first
          console.log('🔒 Vault locked, showing unlock modal');
          setSelectedServiceForSetup(serviceId);
          setShowVaultUnlock(true);
        }
      } else {
        // Vault not initialized or error, show unlock modal
        console.log('🔒 Vault not accessible, showing unlock modal');
        setSelectedServiceForSetup(serviceId);
        setShowVaultUnlock(true);
      }
    } catch (error) {
      console.error('Error checking vault status:', error);
      // On error, assume vault needs to be unlocked
      setSelectedServiceForSetup(serviceId);
      setShowVaultUnlock(true);
    }
  };

  // Handle vault unlock attempt
  const handleVaultUnlock = async (password: string, vaultPin: string): Promise<boolean> => {
    console.log('🔓 Attempting to unlock vault');
    
    try {
      if (!apiClient) {
        console.error('API client not available');
        return false;
      }
      
      // In a real implementation, this would call the vault unlock endpoint
      // For now, we'll simulate success to proceed with the flow
      const response = await apiClient.post('vault/unlock', {
        password,
        vaultPin
      });
      
      if (response.ok) {
        console.log('✅ Vault unlocked successfully');
        setShowVaultUnlock(false);
        
        // Proceed to API key setup for the selected service
        if (selectedServiceForSetup) {
          setShowAPIKeySetup(true);
        }
        return true;
      } else {
        console.error('Failed to unlock vault');
        return false;
      }
    } catch (error) {
      console.error('Error unlocking vault:', error);
      return false;
    }
  };

  // Handle successful API key addition
  const handleAPIKeyAdded = () => {
    console.log('🔑 API key added successfully');
    setShowAPIKeySetup(false);
    setSelectedServiceForSetup('');
    
    // Refresh tool availability status
    // This will trigger a re-check of API key availability in BotToolsPanel
    console.log('✨ API key added - refreshing tool status');
    
    // The BotToolsPanel will automatically refresh its status
    // because it re-checks API key availability when the component updates
  };

  // Handle modal dismissals
  const handleVaultUnlockCancel = () => {
    setShowVaultUnlock(false);
    setSelectedServiceForSetup('');
  };

  const handleAPIKeySetupCancel = () => {
    setShowAPIKeySetup(false);
    setSelectedServiceForSetup('');
  };

  // Handle tool upgrade requests from BotToolsPanel
  const handleToolUpgradeRequest = async (toolId: string) => {
    console.log('🚀 Tool upgrade requested for:', toolId);
    
    // Get the tool definition from the tools registry
    const tool = getTool(toolId);
    if (!tool || !tool.apiRequirements) {
      console.warn('Tool not found or no API requirements:', toolId);
      return;
    }
    
    // Get the first required service (in a real implementation, could show a choice)
    const serviceIds = Object.keys(tool.apiRequirements);
    if (serviceIds.length > 0) {
      const primaryService = serviceIds[0]; // Use first service for now
      console.log('🔑 Starting upgrade flow for service:', primaryService);
      await handleAddAPIKey(primaryService);
    }
  };


  const sendMessage = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || !apiClient || !botId || isLoading) return;

    const userMessage = input.trim();
    // Generate unique ID for this message operation to prevent race conditions
    const messageId = `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    setLastMessageId(messageId);
    
    
    setInput('');
    setError(null);

    // 🚀 INGENIOUS ENHANCEMENT: Reset search stages for new query
    setCurrentSearchStages([]);
    setLastChatResponse(null);
    
    // 🛠️ TOOL INTEGRATION: Reset tool execution state for new query
    setToolExecutions([]);
    setToolSkipped([]);
    setKeyRecommendations([]);
    setToolResults([]);

    // Set loading state FIRST to show thinking animations
    setIsLoading(true);

    // Optimistic UI update - show user message immediately
    const tempUserMessage: Message = {
      id: `temp-${Date.now()}`,
      conversationId: selectedConversationId || '',
      role: 'user',
      content: [{ type: 'text', text: userMessage }],
      createdAt: new Date(),
      tokenCount: Math.ceil(userMessage.length / 4)
    };

    // Add user message to UI immediately
    setMessagesDebug(prev => [...prev, tempUserMessage]);

    try {
      // If no conversation selected, create one (but keep loading state active)
      let conversationId = selectedConversationId;
      if (!conversationId) {
        conversationId = await startNewConversationForChat();
        if (!conversationId) {
          throw new Error('Failed to create conversation');
        }
      }

      const chatRequest: ChatRequest = {
        message: userMessage,
        conversationId: conversationId,
        stream: false,
        sessionModelId: sessionModel || undefined,
        enableReasoning: extendedThinkingEnabled && currentModelSupportsReasoning(),
        reasoningParams: extendedThinkingEnabled ? reasoningParams : undefined,
        disableKnowledgeBase: !knowledgeBaseEnabled
      };

      const response = await apiClient.post(`bots/${botId}/chat`, chatRequest);
      
      if (!response.ok) {
        const errorText = await response.text();
        console.error('Chat request failed:', response.status, errorText);
        throw new Error(`Chat failed: ${response.status} - ${errorText}`);
      }

      // Check if we actually got JSON
      const contentType = response.headers.get('content-type');
      if (!contentType || !contentType.includes('application/json')) {
        const responseText = await response.text();
        console.error('Expected JSON but got:', contentType, responseText.substring(0, 500));
        throw new Error(`Server returned HTML instead of JSON. Response: ${responseText.substring(0, 200)}...`);
      }

      const chatResponse: ChatResponse = await response.json();
      console.log('Chat response received:', chatResponse);
      console.log('Chat response details:', {
        hasResponse: !!chatResponse.response,
        responseLength: chatResponse.response?.length || 0,
        hasSources: !!chatResponse.sources,
        sourcesCount: chatResponse.sources?.length || 0,
        hasKnowledgeStages: !!chatResponse.knowledgeSearchStages,
        stagesCount: chatResponse.knowledgeSearchStages?.length || 0
      });
      
      // 🚀 INGENIOUS ENHANCEMENT: Capture search stages and sources
      if (chatResponse.knowledgeSearchStages) {
        setCurrentSearchStages(chatResponse.knowledgeSearchStages);
      }
      setLastChatResponse(chatResponse);
      
      // 🚀 ENHANCEMENT: Persist sources per conversation
      if (chatResponse.sources && chatResponse.sources.length > 0 && conversationId) {
        setConversationSources(prev => ({
          ...prev,
          [conversationId]: chatResponse.sources!
        }));
      }
      
      // 🛠️ TOOL INTEGRATION: Capture tool execution data
      if (chatResponse.toolsExecuted) {
        setToolExecutions(chatResponse.toolsExecuted);
      }
      if (chatResponse.toolsSkipped) {
        setToolSkipped(chatResponse.toolsSkipped);
      }
      if (chatResponse.keyRecommendations) {
        setKeyRecommendations(chatResponse.keyRecommendations);
      }
      // Capture full tool results for enhanced visualization
      if (chatResponse.toolResults) {
        setToolResults(chatResponse.toolResults);
      }
      
      // ULTRA-CRITICAL FIX: Display the response immediately to prevent disappearing bug
      if (conversationId && lastMessageId === messageId && chatResponse.response) {
        console.log('IMMEDIATE RESPONSE DISPLAY - Adding assistant message directly');
        
        // Create the assistant message from the response
        const assistantMessage: Message = {
          id: `assistant-${Date.now()}`,
          conversationId: conversationId,
          role: 'assistant',
          content: [{ type: 'text', text: chatResponse.response }],
          createdAt: new Date(),
          tokenCount: Math.ceil(chatResponse.response.length / 4)
        };
        
        // Update messages immediately: remove temp user message and add both real user + assistant
        setMessagesDebug(prev => {
          const withoutTemp = prev.filter(msg => msg.id !== tempUserMessage.id);
          const realUserMessage: Message = {
            ...tempUserMessage,
            id: `user-${Date.now()}`,
            conversationId: conversationId
          };
          return [...withoutTemp, realUserMessage, assistantMessage];
        });
        
        // Still reload conversation in background to sync with server, but don't wait for it
        try {
          console.log('Background conversation sync:', conversationId, 'messageId:', messageId);
          setTimeout(() => {
            loadConversation(conversationId, true);
          }, 1000); // 1 second delay to ensure server has saved the message
          console.log('Immediate response display completed for messageId:', messageId);
          
          // TITLE GENERATION FIX: Refresh conversation list to pick up title updates
          // The backend generates titles in background, so we refresh after a short delay
          setTimeout(() => {
            console.log('Refreshing conversation list for potential title updates');
            loadConversations();
          }, 2000); // 2 second delay to allow backend title generation
        } catch (loadErr) {
          console.warn('Background conversation sync failed (non-critical):', loadErr);
          // Non-critical since we already displayed the response immediately
        }
      } else if (conversationId && lastMessageId === messageId && !chatResponse.response) {
        console.log('No response text in chatResponse, falling back to conversation reload');
        
        // Fallback for cases where there's no response text (shouldn't happen normally)
        try {
          setMessagesDebug(prev => prev.filter(msg => msg.id !== tempUserMessage.id));
          await loadConversation(conversationId, true);
        } catch (loadErr) {
          console.error('Fallback conversation reload failed:', loadErr);
        }
      } else if (lastMessageId !== messageId) {
        console.log('Skipping message processing - newer message in progress:', lastMessageId, 'vs', messageId);
      }

    } catch (err) {
      console.error('Error sending message:', err);
      setError(ApiClient.getErrorMessage(err));
      
      // Remove the optimistic message on error
      setMessagesDebug(prev => prev.filter(msg => msg.id !== tempUserMessage.id));
      
      // Clear any partial response state on error
      setLastChatResponse(null);
      setCurrentSearchStages([]);
      
      // Restore the input text so user can retry
      setInput(userMessage);
    } finally {
      setIsLoading(false);
    }
  };

  const renderMessageContent = (content: MessageContent[]) => {
    if (!content) {
      console.warn('renderMessageContent received null/undefined content');
      return <p className="text-muted-foreground italic">Loading response...</p>;
    }
    
    if (!Array.isArray(content)) {
      console.warn('renderMessageContent received non-array content:', content);
      return <p className="text-red-400">Invalid message content format</p>;
    }
    
    if (content.length === 0) {
      console.log('renderMessageContent received empty content array');
      return <p className="text-muted-foreground italic">Empty response</p>;
    }
    
    // Determine if dark mode is active
    const isDark = theme === 'dark' || (theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches);
    
    return content.map((item, index) => {
      switch (item.type) {
        case 'text':
          return (
            <MarkdownRenderer
              key={index}
              content={item.text || ''}
              isDark={isDark}
              className="prose-sm"
            />
          );
        case 'tool_use':
          return (
            <div key={index} className="bg-blue-50 p-3 rounded-lg mt-2">
              <div className="text-sm font-medium text-blue-800">
                🔧 Using tool: {item.toolUse?.name}
              </div>
              {item.toolUse?.input && (
                <pre className="text-xs text-blue-600 mt-1 whitespace-pre-wrap">
                  {JSON.stringify(item.toolUse.input, null, 2)}
                </pre>
              )}
            </div>
          );
        case 'tool_result':
          return (
            <div key={index} className={`p-3 rounded-lg mt-2 ${
              item.toolResult?.isError ? 'bg-red-50' : 'bg-green-50'
            }`}>
              <div className={`text-sm font-medium ${
                item.toolResult?.isError ? 'text-red-800' : 'text-green-800'
              }`}>
                🔧 Tool result:
              </div>
              <div className={`text-sm mt-1 ${
                item.toolResult?.isError ? 'text-red-700' : 'text-green-700'
              }`}>
                {item.toolResult?.content}
              </div>
            </div>
          );
        case 'reasoning':
          return (
            <div key={index} className="bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-800 p-3 rounded-lg mt-2">
              <div className="text-sm font-medium text-purple-800 dark:text-purple-200 mb-2 flex items-center">
                🧠 Extended Thinking
                <span className="ml-2 text-xs bg-purple-100 dark:bg-purple-800 text-purple-700 dark:text-purple-200 px-2 py-1 rounded-full">
                  Reasoning
                </span>
              </div>
              <div className="bg-purple-25 dark:bg-purple-900/30 p-2 rounded border border-purple-100 dark:border-purple-700 max-h-60 overflow-y-auto">
                <MarkdownRenderer
                  content={item.reasoningContent?.text || item.text || ''}
                  isDark={isDark}
                  className="prose-sm text-purple-700 dark:text-purple-200"
                />
              </div>
            </div>
          );
        default:
          return <p key={index}>{JSON.stringify(item)}</p>;
      }
    });
  };

  if (error && !bot) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center bg-white p-8 rounded-lg shadow-md max-w-md">
          <div className="text-red-500 mb-4">
            <svg className="mx-auto h-16 w-16" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.728-.833-2.498 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
            </svg>
          </div>
          <h3 className="text-lg font-medium text-gray-900 mb-2">{error}</h3>
          <div className="flex space-x-3 justify-center">
            <Link
              to="/bots"
              className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors"
            >
              Back to Bots
            </Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="h-screen bg-background text-foreground flex overflow-hidden">
      {/* Animated Background */}
      <div className="fixed inset-0 gradient-mesh opacity-20" />
      <div className="fixed inset-0">
        {[...Array(12)].map((_, i) => (
          <motion.div
            key={i}
            className="absolute w-1 h-1 bg-primary/20 rounded-full"
            animate={{
              x: [0, Math.random() * 30 - 15],
              y: [0, Math.random() * 30 - 15],
              scale: [1, Math.random() * 0.5 + 0.5, 1],
            }}
            transition={{
              duration: Math.random() * 20 + 20,
              repeat: Infinity,
              ease: "linear",
            }}
            style={{
              left: `${Math.random() * 100}%`,
              top: `${Math.random() * 100}%`,
            }}
          />
        ))}
      </div>

      {/* Conversation Sidebar */}
      <motion.div 
        className={`relative z-10 glass-card border-r border-glass-border flex flex-col transition-all duration-300 ${
          sidebarOpen ? 'w-80' : 'w-16'
        }`}
        initial={{ x: -300, opacity: 0 }}
        animate={{ x: 0, opacity: 1 }}
        transition={{ duration: 0.5, ease: "easeOut" }}
      >
        <div className="p-4 border-b border-border/50">
          <div className="flex items-center justify-between">
            <motion.button
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="p-2 rounded-xl hover:bg-primary/10 text-muted-foreground hover:text-primary transition-all duration-300 hover-lift"
              whileHover={{ scale: 1.1 }}
              whileTap={{ scale: 0.9 }}
            >
              {sidebarOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
            </motion.button>
            
            <AnimatePresence>
              {sidebarOpen && (
                <motion.div 
                  className="flex items-center space-x-3"
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: -20 }}
                  transition={{ duration: 0.3 }}
                >
                  <div className="flex items-center gap-2">
                    <MessageCircle className="w-5 h-5 text-primary" />
                    <h2 className="text-lg font-semibold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent">
                      Conversations
                    </h2>
                  </div>
                  <div className="flex items-center gap-2">
                    <motion.button
                      onClick={loadConversations}
                      className="p-2 rounded-xl bg-muted/30 text-muted-foreground hover:bg-primary/10 hover:text-primary transition-all duration-300 hover-lift"
                      whileHover={{ scale: 1.1 }}
                      whileTap={{ scale: 0.9 }}
                      title="Refresh conversations (to see updated titles)"
                    >
                      <RefreshCw className="w-4 h-4" />
                    </motion.button>
                    <motion.button
                      onClick={handleNewConversationClick}
                      className="p-2 rounded-xl bg-primary/10 text-primary hover:bg-primary/20 transition-all duration-300 hover-lift"
                      whileHover={{ scale: 1.1, rotate: 90 }}
                      whileTap={{ scale: 0.9 }}
                      title="New Conversation"
                    >
                      <Plus className="w-4 h-4" />
                    </motion.button>
                  </div>
                </motion.div>
              )}
            </AnimatePresence>
          </div>
          
          {/* Bot Selector */}
          <AnimatePresence>
            {sidebarOpen && (
              <motion.div 
                className="mt-4 space-y-4"
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: 20 }}
                transition={{ duration: 0.3, delay: 0.1 }}
              >
                <GlassCard className="p-3 rounded-xl">
                  <BotSelector
                    bots={allBots}
                    currentBot={bot}
                    className=""
                  />
                </GlassCard>
              
                {/* Bot Tools Panel - NOW WITH CONVERSATION INTEGRATION */}
                {bot && (
                  <GlassCard className="p-3 rounded-xl">
                    <BotToolsPanel
                  bot={bot}
                  onToolExecute={async (tool, capability, input) => {
                    try {
                      // 🚀 NEW: Execute tool with conversation integration
                      if (selectedConversationId) {
                        // Execute in conversation context - saves automatically
                        const result = await executeToolInChat({
                          conversationId: selectedConversationId,
                          toolId: tool.id,
                          capability,
                          input,
                          botId: bot.id,
                        });
                        
                        // Refresh messages to show the new tool execution
                        await loadConversation(selectedConversationId, true);
                        
                        console.log('Tool executed in conversation:', result);
                      } else {
                        // No conversation selected - create one first
                        const newConvId = await startNewConversationForChat();
                        if (newConvId) {
                          const result = await executeToolInChat({
                            conversationId: newConvId,
                            toolId: tool.id,
                            capability,
                            input,
                            botId: bot.id,
                          });
                          
                          // Load the new conversation
                          setSelectedConversationId(newConvId);
                          await loadConversation(newConvId, true);
                          
                          console.log('Tool executed in new conversation:', result);
                        }
                      }
                    } catch (error) {
                      console.error('Tool execution failed:', error);
                      // TODO: Show user-friendly error message
                    }
                  }}
                  onStreamingExecute={async (toolId, capability, input) => {
                    try {
                      // 🌊 NEW: Streaming execution with conversation integration
                      if (selectedConversationId) {
                        const result = await executeToolWithProgress(
                          toolId,
                          capability,
                          input,
                          {
                            conversationId: selectedConversationId,
                            onProgress: (progress) => {
                              console.log('Tool progress:', progress);
                              // TODO: Show progress in UI
                            }
                          }
                        );
                        
                        // Refresh messages to show results
                        await loadConversation(selectedConversationId, true);
                        
                        console.log('Streaming tool completed:', result);
                      } else {
                        console.warn('No conversation selected for streaming execution');
                      }
                    } catch (error) {
                      console.error('Streaming tool execution failed:', error);
                    }
                  }}
                  getAccessToken={getAccessToken}
                  getUserId={getUserId}
                    onUpgradeRequest={handleToolUpgradeRequest}
                    className=""
                  />
                  </GlassCard>
                )}
              </motion.div>
            )}
          </AnimatePresence>
        </div>

        <AnimatePresence>
          {sidebarOpen && (
            <motion.div 
              className="flex-1 overflow-y-auto scrollbar-thin"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.3, delay: 0.2 }}
            >
              {conversations.length === 0 ? (
                <motion.div 
                  className="p-6 text-center"
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.5, delay: 0.3 }}
                >
                  <div className="glass-card p-8 rounded-xl">
                    <div className="w-16 h-16 mx-auto mb-4 rounded-full glass flex items-center justify-center">
                      <MessageCircle className="w-8 h-8 text-muted-foreground" />
                    </div>
                    <h3 className="text-lg font-semibold text-foreground mb-2">No conversations yet</h3>
                    <p className="text-muted-foreground text-sm mb-4">Start your first conversation with this AI bot</p>
                    <motion.button
                      onClick={handleNewConversationClick}
                      className="px-4 py-2 bg-primary/10 text-primary rounded-xl hover:bg-primary/20 transition-all duration-300 text-sm font-medium hover-lift"
                      whileHover={{ scale: 1.05 }}
                      whileTap={{ scale: 0.95 }}
                    >
                      Start Conversation
                    </motion.button>
                  </div>
                </motion.div>
              ) : (
                <div className="space-y-2 p-3">
                  {Array.isArray(conversations) && conversations.map((conv, index) => (
                    <motion.div
                      key={conv.id}
                      className={`group relative glass-card hover-lift transition-all duration-300 ${
                        selectedConversationId === conv.id ? 'bg-primary/10 border-primary/30' : ''
                      }`}
                      initial={{ opacity: 0, x: -20 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ duration: 0.3, delay: index * 0.05 }}
                      whileHover={{ scale: 1.02 }}
                    >
                      <button
                        onClick={() => loadConversation(conv.id)}
                        className="w-full text-left p-4 pr-12 group-hover:text-primary transition-colors duration-300"
                      >
                        <div className="flex items-center gap-3 mb-2">
                          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center flex-shrink-0">
                            <BotIcon className="w-4 h-4 text-white" />
                          </div>
                          <div className="min-w-0 flex-1">
                            <div className="font-semibold text-foreground truncate group-hover:text-primary transition-colors duration-300">
                              {conv.title}
                            </div>
                          </div>
                        </div>
                        
                        <div className="flex items-center justify-between text-xs text-muted-foreground mb-2">
                          <div className="flex items-center gap-1">
                            <MessageCircle className="w-3 h-3" />
                            <span>{conv.messageCount} messages</span>
                          </div>
                          <div className="flex items-center gap-1">
                            <Clock className="w-3 h-3" />
                            <span>{new Date(conv.updatedAt).toLocaleDateString()}</span>
                          </div>
                        </div>
                        
                        {conv.sessionModelId && (
                          <div className="mb-2">
                            <span className="bg-primary/20 text-primary text-xs px-2 py-1 rounded-full font-medium">
                              {availableModels.find(m => m.id === conv.sessionModelId)?.name || 'Custom'}
                            </span>
                          </div>
                        )}
                        
                        {conv.lastMessage && (
                          <div className="text-xs text-muted-foreground/70 truncate leading-relaxed">
                            {conv.lastMessage}
                          </div>
                        )}
                      </button>
                      <motion.button
                        onClick={(e) => {
                          e.stopPropagation();
                          if (confirm('Are you sure you want to delete this conversation?')) {
                            deleteConversation(conv.id);
                          }
                        }}
                        className="absolute top-4 right-4 opacity-0 group-hover:opacity-100 p-2 hover:bg-red-500/20 text-red-400 hover:text-red-500 rounded-xl transition-all duration-300"
                        whileHover={{ scale: 1.1 }}
                        whileTap={{ scale: 0.9 }}
                        title="Delete conversation"
                      >
                        <Trash2 className="w-4 h-4" />
                      </motion.button>
                    </GlassCard>
                  ))}
                </div>
              )}
            </motion.div>
          )}
        </AnimatePresence>
      </GlassCard>

      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col relative z-10">
        {/* Header */}
        <motion.header 
          className="glass-card border-b border-border/50 px-6 py-4"
          initial={{ y: -50, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          transition={{ duration: 0.5, delay: 0.3 }}
        >
          <div className="flex justify-between items-center">
            <div className="flex-1">
              {/* Breadcrumbs */}
              <Breadcrumbs items={getBreadcrumbItems()} className="mb-2" />
              
              {bot ? (
                <motion.div
                  initial={{ opacity: 0, x: -30 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ duration: 0.5, delay: 0.4 }}
                >
                  <div className="flex items-center gap-4 mb-2">
                    <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center">
                      <BotIcon className="w-5 h-5 text-white" />
                    </div>
                    <div>
                      <h1 className="text-xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent">
                        {bot.title}
                      </h1>
                      <p className="text-sm text-muted-foreground">{bot.description}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-3 flex-wrap">
                    <div className="flex items-center gap-2">
                      <Sparkles className="w-3 h-3 text-primary" />
                      <span className="text-xs text-muted-foreground">Model:</span>
                      <span className="text-xs bg-primary/20 text-primary px-3 py-1 rounded-full font-medium">
                        {getCurrentModelName()}
                      </span>
                    </div>
                    {currentModelSupportsReasoning() && (
                      <motion.span 
                        className="text-xs bg-purple-500/20 text-purple-400 px-3 py-1 rounded-full font-medium flex items-center gap-1"
                        initial={{ scale: 0 }}
                        animate={{ scale: 1 }}
                        transition={{ type: "spring", stiffness: 500, delay: 0.6 }}
                      >
                        <Brain className="w-3 h-3" />
                        Extended Thinking Available
                      </motion.span>
                    )}
                  </div>
                </motion.div>
              ) : (
                <div className="animate-pulse">
                  <div className="flex items-center gap-4 mb-2">
                    <div className="w-10 h-10 bg-muted/30 rounded-xl"></div>
                    <div>
                      <div className="h-5 bg-muted/30 rounded w-32 mb-2"></div>
                      <div className="h-4 bg-muted/20 rounded w-48"></div>
                    </div>
                  </div>
                  <div className="h-4 bg-muted/20 rounded w-32"></div>
                </div>
              )}
            </div>
            <motion.div 
              className="flex items-center space-x-4"
              initial={{ opacity: 0, x: 30 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.5, delay: 0.6 }}
            >
              <ThemeToggle />
              <motion.button
                onClick={signOut}
                className="glass-card px-4 py-2 text-sm font-medium text-foreground hover:text-primary transition-all duration-300 hover-lift"
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
              >
                Sign Out
              </motion.button>
            </motion.div>
          </div>
        </GlassCard>

        {/* Messages */}
        <motion.div 
          className="flex-1 overflow-y-auto p-6 space-y-4 scrollbar-thin"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.6, delay: 0.4 }}
        >
          {!selectedConversationId && messages.length === 0 && (
            <motion.div 
              className="text-center py-16"
              initial={{ opacity: 0, y: 50 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.8, delay: 0.6 }}
            >
              <motion.div 
                className="glass-card w-32 h-32 rounded-full mx-auto mb-8 flex items-center justify-center"
                initial={{ scale: 0 }}
                animate={{ scale: 1 }}
                transition={{ duration: 0.6, delay: 0.8, type: "spring", stiffness: 200 }}
              >
                <MessageCircle className="h-16 w-16 text-muted-foreground" />
              </motion.div>
              
              <motion.h3 
                className="text-2xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent mb-4"
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.6, delay: 1.0 }}
              >
                Start a conversation
              </motion.h3>
              
              <motion.p 
                className="text-muted-foreground text-lg mb-8"
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.6, delay: 1.2 }}
              >
                Ask me anything about {bot?.title || 'this bot'}!
              </motion.p>
              
              {bot?.conversationStarters && bot.conversationStarters.length > 0 && (
                <motion.div 
                  className="space-y-4 max-w-2xl mx-auto"
                  initial={{ opacity: 0, y: 30 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.8, delay: 1.4 }}
                >
                  <p className="text-sm text-muted-foreground/70 font-medium">Try one of these:</p>
                  <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                    {bot.conversationStarters.slice(0, 3).map((starter: ConversationStarter, index: number) => (
                      <motion.button
                        key={index}
                        onClick={() => setInput(starter.example)}
                        className="glass-card p-4 text-left hover-lift group transition-all duration-300"
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.5, delay: 1.6 + index * 0.1 }}
                        whileHover={{ scale: 1.02, y: -2 }}
                        whileTap={{ scale: 0.98 }}
                      >
                        <div className="font-medium text-foreground group-hover:text-primary transition-colors duration-300 mb-2">
                          {starter.title}
                        </div>
                        <div className="text-sm text-muted-foreground leading-relaxed">
                          {starter.example}
                        </div>
                      </motion.button>
                    ))}
                  </div>
                </motion.div>
              )}
            </motion.div>
          )}

          {Array.isArray(messages) && messages
            // Filter out system/instruction messages with null content - they're not meant for display
            .filter(message => {
              // More robust filtering - ensure we don't filter out valid assistant responses
              const hasValidContent = message.content && 
                                      message.content !== null && 
                                      Array.isArray(message.content) && 
                                      message.content.length > 0;
              const isDisplayableRole = message.role !== 'system' && message.role !== 'instruction';
              
              if (!hasValidContent || !isDisplayableRole) {
                console.log('Filtering out message:', { 
                  id: message.id, 
                  role: message.role, 
                  hasValidContent, 
                  isDisplayableRole,
                  contentType: typeof message.content,
                  contentLength: Array.isArray(message.content) ? message.content.length : 'not array'
                });
              }
              
              return hasValidContent && isDisplayableRole;
            })
            .map((message) => (
            <motion.div
              key={message.id}
              className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
              initial={{ opacity: 0, y: 20, scale: 0.95 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              transition={{ duration: 0.4, ease: "easeOut" }}
              layout
            >
              <motion.div 
                className={`max-w-3xl relative group ${
                  message.role === 'user'
                    ? 'glass-card bg-gradient-to-br from-blue-500/20 to-purple-600/20 border-primary/30 border-l-4 border-l-primary text-foreground shadow-lg shadow-primary/10'
                    : 'glass-card border-border/50 border-l-4 border-l-muted-foreground/30 text-foreground bg-gradient-to-br from-muted/10 to-muted/20 shadow-lg shadow-muted/10'
                } px-6 py-4 hover-lift`}
                whileHover={{ scale: 1.01, y: -1 }}
                transition={{ type: "spring", stiffness: 300, damping: 20 }}
              >
                {/* Enhanced Message Role Indicator */}
                <div className={`flex items-center gap-2 mb-3 text-xs font-medium ${
                  message.role === 'user' 
                    ? 'text-primary' 
                    : 'text-muted-foreground'
                }`}>
                  <div className={`w-6 h-6 rounded-full flex items-center justify-center ${
                    message.role === 'user'
                      ? 'bg-gradient-to-br from-blue-500 to-purple-600 text-white shadow-md'
                      : 'bg-gradient-to-br from-muted to-muted-foreground/20 text-muted-foreground shadow-md'
                  }`}>
                    {message.role === 'user' ? (
                      <User className="w-3 h-3" />
                    ) : (
                      <BotIcon className="w-3 h-3" />
                    )}
                  </div>
                  <span className={message.role === 'user' ? 'text-primary font-semibold' : 'text-muted-foreground'}>
                    {message.role === 'user' ? 'You' : (bot?.title || 'Assistant')}
                  </span>
                </div>
                
                <div className="space-y-3">
                  {renderMessageContent(message.content)}
                </div>
                
                {/* Message Metadata */}
                <motion.div 
                  className={`text-xs mt-4 pt-3 border-t border-border/30 flex items-center justify-between opacity-60 group-hover:opacity-100 transition-opacity duration-300`}
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 0.6 }}
                  whileHover={{ opacity: 1 }}
                >
                  <div className="flex items-center gap-2">
                    <Clock className="w-3 h-3" />
                    <span>{new Date(message.createdAt).toLocaleTimeString()}</span>
                  </div>
                  {message.tokenCount && (
                    <div className="flex items-center gap-1">
                      <Zap className="w-3 h-3" />
                      <span>{message.tokenCount} tokens</span>
                    </div>
                  )}
                </motion.div>
                
                {/* Floating Animation on Hover */}
                <motion.div
                  className="absolute inset-0 rounded-xl bg-gradient-to-r from-primary/5 to-primary/10 opacity-0 pointer-events-none"
                  whileHover={{ opacity: 1 }}
                  transition={{ duration: 0.3 }}
                />
              </GlassCard>
            </motion.div>
          ))}

          {/* 🚀 INGENIOUS ENHANCEMENT: Knowledge Search Stages Display */}
          {knowledgeBaseEnabled && (isLoading || currentSearchStages.length > 0) && (
            <KnowledgeSearchStages 
              stages={currentSearchStages} 
              isLoading={isLoading && currentSearchStages.length === 0}
            />
          )}

          {/* 🌊 STREAMING TOOL EXECUTION: Real-time progress indicators */}
          {streamingExecution.isExecuting && (
            <div className="flex justify-start">
              <div className="max-w-3xl px-4 py-3 rounded-lg bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-800">
                <StreamingProgressIndicator
                  isExecuting={streamingExecution.isExecuting}
                  currentStep={streamingExecution.currentStep}
                  progress={streamingExecution.getProgressPercentage()}
                  elapsedTime={streamingExecution.getElapsedTime()}
                  error={streamingExecution.error}
                  success={streamingExecution.result?.success}
                  size="md"
                  showMessage={true}
                  showElapsedTime={true}
                  className="w-full"
                />
                {streamingExecution.progressSteps.length > 0 && (
                  <div className="mt-3 pt-3 border-t border-purple-200 dark:border-purple-700">
                    <div className="text-xs text-purple-600 dark:text-purple-300 font-medium mb-2">
                      Progress Steps ({streamingExecution.getTotalSteps()})
                    </div>
                    <div className="space-y-1">
                      {streamingExecution.progressSteps.slice(-3).map((step) => (
                        <CompactStreamingIndicator
                          key={step.id}
                          isExecuting={false}
                          progress={step.progress}
                          message={step.message}
                          success={step.stage === 'complete'}
                          error={step.stage === 'error' ? step.message : undefined}
                        />
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* 🚀 INGENIOUS ENHANCEMENT: Source Citations Display */}
          {knowledgeBaseEnabled && ((lastChatResponse?.sources && lastChatResponse.sources.length > 0) || 
            (selectedConversationId && conversationSources[selectedConversationId])) && (
            <SourceCitations 
              sources={lastChatResponse?.sources || conversationSources[selectedConversationId!] || []} 
              className="mb-4"
              defaultCollapsed={true}
            />
          )}

          {/* 🛠️ TOOL INTEGRATION: Tool Execution Results */}
          {(toolExecutions.length > 0 || toolSkipped.length > 0) && (
            <ToolExecutionResults 
              toolsExecuted={toolExecutions}
              toolsSkipped={toolSkipped}
              className="mb-4"
            />
          )}

          {/* 🎨 ENHANCED TOOL RESULTS: Beautiful visualizations */}
          {toolResults.length > 0 && (
            <div className="space-y-4 mb-4">
              {toolResults.map((result, index) => (
                <ToolResultRenderer
                  key={`tool-result-${index}`}
                  toolName={result.toolName || 'Unknown Tool'}
                  toolId={result.toolId || 'unknown'}
                  capability={result.capability || 'execute'}
                  data={result.data || {}}
                  executionTime={result.executionTime || 0}
                  usedApiKey={result.usedApiKey || false}
                  success={result.success !== false}
                  errorMessage={result.errorMessage}
                  className=""
                />
              ))}
            </div>
          )}

          {/* 🔑 KEY RECOMMENDATIONS: Non-blocking upgrade suggestions */}
          {keyRecommendations.length > 0 && (
            <KeyRecommendations 
              recommendations={keyRecommendations}
              onAddKey={handleAddAPIKey}
              className="mb-4"
            />
          )}

          {isLoading && (
            <motion.div 
              className="flex justify-start"
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.9 }}
              transition={{ duration: 0.3 }}
            >
              <motion.div 
                className="glass-card px-6 py-4 max-w-xs"
                animate={{ y: [0, -2, 0] }}
                transition={{ duration: 2, repeat: Infinity, ease: "easeInOut" }}
              >
                <div className="flex items-center space-x-3">
                  <div className="flex space-x-1">
                    <motion.div 
                      className="w-2 h-2 bg-primary rounded-full"
                      animate={{ scale: [1, 1.2, 1], opacity: [1, 0.7, 1] }}
                      transition={{ duration: 1, repeat: Infinity, delay: 0 }}
                    />
                    <motion.div 
                      className="w-2 h-2 bg-primary rounded-full"
                      animate={{ scale: [1, 1.2, 1], opacity: [1, 0.7, 1] }}
                      transition={{ duration: 1, repeat: Infinity, delay: 0.2 }}
                    />
                    <motion.div 
                      className="w-2 h-2 bg-primary rounded-full"
                      animate={{ scale: [1, 1.2, 1], opacity: [1, 0.7, 1] }}
                      transition={{ duration: 1, repeat: Infinity, delay: 0.4 }}
                    />
                  </div>
                  <motion.span 
                    className="text-sm text-muted-foreground font-medium"
                    animate={{ opacity: [0.5, 1, 0.5] }}
                    transition={{ duration: 2, repeat: Infinity }}
                  >
                    {bot?.title || 'AI'} is thinking...
                  </motion.span>
                </div>
              </GlassCard>
            </motion.div>
          )}

          <AnimatePresence>
            {error && (
              <GlassCard 
                className="border border-red-400/30 bg-red-500/10 p-4"
                initial={{ opacity: 0, y: -20, scale: 0.95 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, y: -20, scale: 0.95 }}
                transition={{ duration: 0.3 }}
              >
                <div className="flex items-center gap-3 text-red-400">
                  <Zap className="w-4 h-4" />
                  <div className="text-sm font-medium">{error}</div>
                </div>
              </GlassCard>
            )}
          </AnimatePresence>

          <div ref={messagesEndRef} />
        </motion.div>

        {/* Input Form */}
        <GlassCard
          className="border-t border-border/50 p-6"
          initial={{ y: 100, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          transition={{ duration: 0.6, delay: 0.8 }}
        >
          {/* Extended Thinking Toggle */}
          {currentModelSupportsReasoning() && (
            <div className="mb-4">
              <ExtendedThinkingToggle
                enabled={extendedThinkingEnabled || getCurrentModel()?.forceReasoningEnabled || false}
                onToggle={handleExtendedThinkingToggle}
                disabled={getCurrentModel()?.forceReasoningEnabled}
                className=""
              />
              {getCurrentModel()?.forceReasoningEnabled && (
                <p className="text-xs text-gray-500 mt-1">
                  This model has built-in reasoning that cannot be disabled.
                </p>
              )}
            </div>
          )}
          
          {/* Knowledge Base Toggle */}
          <div className="mb-4">
            <KnowledgeBaseToggle
              enabled={knowledgeBaseEnabled}
              onToggle={setKnowledgeBaseEnabled}
              className=""
            />
          </div>
          
          <form onSubmit={sendMessage} className="flex gap-4">
            <motion.div 
              className="flex-1 relative"
              whileFocus={{ scale: 1.01 }}
              transition={{ type: "spring", stiffness: 300, damping: 20 }}
            >
              <input
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder={`Message ${bot?.title || 'AI assistant'}...`}
                className="w-full glass-card px-6 py-4 text-foreground placeholder-muted-foreground/60 focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300 text-base"
                disabled={isLoading}
              />
              {/* Subtle glow effect on focus */}
              <motion.div
                className="absolute inset-0 rounded-xl bg-gradient-to-r from-primary/10 to-purple-500/10 opacity-0 pointer-events-none"
                whileFocus={{ opacity: 1 }}
                transition={{ duration: 0.3 }}
              />
            </motion.div>
            
            <motion.button
              type="submit"
              disabled={isLoading || !input.trim()}
              className="px-8 py-4 bg-gradient-to-r from-blue-500 to-purple-600 text-white rounded-xl font-semibold transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed hover:from-blue-600 hover:to-purple-700 shadow-lg hover:shadow-xl hover-glow flex items-center gap-2"
              whileHover={{ scale: isLoading || !input.trim() ? 1 : 1.05 }}
              whileTap={{ scale: isLoading || !input.trim() ? 1 : 0.95 }}
            >
              {isLoading ? (
                <motion.div
                  className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full"
                  animate={{ rotate: 360 }}
                  transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
                />
              ) : (
                <Send className="w-5 h-5" />
              )}
              <span className="hidden sm:inline">
                {isLoading ? 'Sending...' : 'Send'}
              </span>
            </motion.button>
          </form>
        </GlassCard>
      </div>

      {/* New Chat Model Selection Dialog */}
      {showNewChatDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Select Model for New Conversation</h3>
            <p className="text-sm text-gray-600 mb-4">
              Choose which model to use for this new conversation. This will override the bot's default model for this session only.
            </p>
            
            <div className="mb-6">
              <label className="block text-sm font-medium text-gray-700 mb-2">Model</label>
              <select
                value={sessionModel || ''}
                onChange={(e) => setSessionModel(e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                {availableModels.map((model) => (
                  <option key={model.id} value={model.id}>
                    {model.name}
                  </option>
                ))}
              </select>
            </div>
            
            <div className="flex justify-end space-x-3">
              <button
                onClick={() => setShowNewChatDialog(false)}
                className="px-4 py-2 text-gray-600 hover:text-gray-800 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => startNewConversationWithModel(sessionModel || availableModels[0].id)}
                className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
              >
                Create Conversation
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 🔑 VAULT UNLOCK FLOW MODALS */}
      {showVaultUnlock && (
        <VaultUnlockModal
          isOpen={showVaultUnlock}
          onClose={handleVaultUnlockCancel}
          onUnlock={handleVaultUnlock}
        />
      )}

      {showAPIKeySetup && selectedServiceForSetup && vaultService && (
        <APIKeySetupModal
          isOpen={showAPIKeySetup}
          serviceId={selectedServiceForSetup}
          onClose={handleAPIKeySetupCancel}
          onKeyAdded={handleAPIKeyAdded}
          vaultService={vaultService}
        />
      )}

      {/* 🔧 NEW: Tool Execution Progress Indicator */}
      {isToolExecuting && (
        <div className="fixed bottom-4 right-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg p-4 max-w-sm">
          <div className="flex items-center space-x-3">
            <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-600"></div>
            <div>
              <p className="text-sm font-medium text-gray-900 dark:text-white">
                Executing Tool...
              </p>
              {executionProgress && (
                <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  <div className="flex items-center justify-between">
                    <span>{executionProgress.stage}</span>
                    <span>{executionProgress.progress}%</span>
                  </div>
                  <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1 mt-1">
                    <div 
                      className="bg-blue-600 h-1 rounded-full transition-all duration-300"
                      style={{ width: `${executionProgress.progress}%` }}
                    ></div>
                  </div>
                  <p className="text-xs mt-1">{executionProgress.message}</p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

    </div>
  );
}