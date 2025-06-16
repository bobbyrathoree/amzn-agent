import { useState, useEffect, useRef, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useAuth } from '../components/AuthProvider';
import { useApiClient } from '../lib/api';
import { KnowledgeSearchStages } from '../components/KnowledgeSearchStages';
import { SourceCitations } from '../components/SourceCitations';
import type { 
  Bot, 
  Conversation, 
  ConversationMeta, 
  Message, 
  ChatRequest, 
  MessageContent,
  ChatResponse,
  KnowledgeSearchStage 
} from '../types';

export function ChatPage() {
  const { botId } = useParams<{ botId: string }>();
  const { user, signOut, getAccessToken } = useAuth();
  const getUserId = useCallback(() => user?.userId || user?.username || null, [user?.userId, user?.username]);
  const apiClient = useApiClient(getAccessToken, getUserId);
  
  // State management
  const [bot, setBot] = useState<Bot | null>(null);
  const [conversations, setConversations] = useState<ConversationMeta[]>([]);
  const [currentConversation, setCurrentConversation] = useState<Conversation | null>(null);
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
  
  // Session model state (not persisted to DB)
  const [sessionModel, setSessionModel] = useState<string | null>(null);
  const [showModelChangeWarning, setShowModelChangeWarning] = useState(false);
  const [showNewChatDialog, setShowNewChatDialog] = useState(false);
  const [pendingModelChange, setPendingModelChange] = useState<string | null>(null);
  
  // UI state
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // 🚀 INGENIOUS ENHANCEMENT: Knowledge search stages tracking
  const [currentSearchStages, setCurrentSearchStages] = useState<KnowledgeSearchStage[]>([]);
  const [lastChatResponse, setLastChatResponse] = useState<ChatResponse | null>(null);

  // Available models for dropdown
  const availableModels = [
    // Claude Models (US inference profiles)
    { id: 'us.anthropic.claude-opus-4-20250514-v1:0', name: 'Claude 4 Opus', family: 'claude' },
    { id: 'us.anthropic.claude-sonnet-4-20250514-v1:0', name: 'Claude 4 Sonnet', family: 'claude' },
    { id: 'us.anthropic.claude-3-7-sonnet-20250219-v1:0', name: 'Claude 3.7 Sonnet', family: 'claude' },
    { id: 'us.anthropic.claude-3-5-sonnet-20241022-v2:0', name: 'Claude 3.5 Sonnet v2', family: 'claude' },
    { id: 'us.anthropic.claude-3-5-sonnet-20240620-v1:0', name: 'Claude 3.5 Sonnet', family: 'claude' },
    { id: 'us.anthropic.claude-3-5-haiku-20241022-v1:0', name: 'Claude 3.5 Haiku', family: 'claude' },
    { id: 'us.anthropic.claude-3-haiku-20240307-v1:0', name: 'Claude 3 Haiku', family: 'claude' },
    { id: 'us.anthropic.claude-3-opus-20240229-v1:0', name: 'Claude 3 Opus', family: 'claude' },
    
    // Amazon Nova Models
    { id: 'us.amazon.nova-pro-v1:0', name: 'Nova Pro', family: 'nova' },
    { id: 'us.amazon.nova-lite-v1:0', name: 'Nova Lite', family: 'nova' },
    { id: 'us.amazon.nova-micro-v1:0', name: 'Nova Micro', family: 'nova' },
    
    // Mistral Models
    { id: 'us.mistral.mistral-large-2407-v1:0', name: 'Mistral Large 2407', family: 'mistral' },
    { id: 'us.mistral.mistral-large-2402-v1:0', name: 'Mistral Large 2402', family: 'mistral' },
    { id: 'us.mistral.mixtral-8x7b-instruct-v0:1', name: 'Mixtral 8x7B', family: 'mistral' },
    { id: 'us.mistral.mistral-7b-instruct-v0:2', name: 'Mistral 7B', family: 'mistral' },
    
    // DeepSeek Models
    { id: 'us.deepseek.r1-v1:0', name: 'DeepSeek R1', family: 'deepseek' },
    
    // Meta Llama Models
    { id: 'us.meta.llama3-3-70b-instruct-v1:0', name: 'Llama 3.3 70B', family: 'llama' },
    { id: 'us.meta.llama3-2-90b-instruct-v1:0', name: 'Llama 3.2 90B', family: 'llama' },
    { id: 'us.meta.llama3-2-11b-instruct-v1:0', name: 'Llama 3.2 11B', family: 'llama' },
    { id: 'us.meta.llama3-2-3b-instruct-v1:0', name: 'Llama 3.2 3B', family: 'llama' },
    { id: 'us.meta.llama3-2-1b-instruct-v1:0', name: 'Llama 3.2 1B', family: 'llama' }
  ];

  // Load bot and conversations on mount
  useEffect(() => {
    if (apiClient && botId) {
      loadBot();
      loadConversations();
    }
  }, [apiClient, botId]);

  // Auto-scroll to bottom when messages change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

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
      setError(err instanceof Error ? err.message : 'Failed to load bot');
    }
  }, [apiClient, botId]);

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
      setMessagesDebug(Array.isArray(data.messages) ? data.messages : []);
      setSelectedConversationId(conversationId);
      
      // Update session model to match conversation's session model (if any)
      if (data.conversation.sessionModelId) {
        setSessionModel(data.conversation.sessionModelId);
      } else if (bot?.activeModels && bot.activeModels.length > 0) {
        // Fall back to bot's default model if conversation has no session model
        setSessionModel(bot.activeModels[0]);
      }
    } catch (err) {
      console.error('Error loading conversation:', err);
      setError(err instanceof Error ? err.message : 'Failed to load conversation');
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
      setError(err instanceof Error ? err.message : 'Failed to create conversation');
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
      setError(err instanceof Error ? err.message : 'Failed to delete conversation');
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

  const handleModelChange = (newModelId: string) => {
    if (selectedConversationId && messages.length > 0) {
      // Warn user about switching models mid-conversation
      setPendingModelChange(newModelId);
      setShowModelChangeWarning(true);
    } else {
      // No active conversation, safe to switch
      setSessionModel(newModelId);
    }
  };

  const confirmModelChange = () => {
    if (pendingModelChange) {
      setSessionModel(pendingModelChange);
      setPendingModelChange(null);
      setShowModelChangeWarning(false);
      
      // Start a new conversation with the new model
      startNewConversationWithModel(pendingModelChange);
    }
  };

  const cancelModelChange = () => {
    setPendingModelChange(null);
    setShowModelChangeWarning(false);
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

  const sendMessage = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || !apiClient || !botId || isLoading) return;

    const userMessage = input.trim();
    setInput('');
    setError(null);

    // 🚀 INGENIOUS ENHANCEMENT: Reset search stages for new query
    setCurrentSearchStages([]);
    setLastChatResponse(null);

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
        sessionModelId: sessionModel || undefined
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
      
      // 🚀 INGENIOUS ENHANCEMENT: Capture search stages and sources
      if (chatResponse.knowledgeSearchStages) {
        setCurrentSearchStages(chatResponse.knowledgeSearchStages);
      }
      setLastChatResponse(chatResponse);
      
      // Reload conversation to get the latest messages
      if (conversationId) {
        try {
          console.log('Reloading conversation after chat response:', conversationId);
          await loadConversation(conversationId, true); // Force reload after chat
          console.log('Successfully reloaded conversation');
        } catch (loadErr) {
          console.warn('Failed to reload conversation after successful chat, but message was sent:', loadErr);
          // Don't throw - the message was successfully sent even if we can't reload
          // Keep the optimistic message in the UI since the chat was successful
        }
      }

    } catch (err) {
      console.error('Error sending message:', err);
      setError(err instanceof Error ? err.message : 'Failed to send message');
      
      // Remove the optimistic message on error
      setMessagesDebug(prev => prev.filter(msg => msg.id !== tempUserMessage.id));
      
      // Restore the input text so user can retry
      setInput(userMessage);
    } finally {
      setIsLoading(false);
    }
  };

  const renderMessageContent = (content: MessageContent[]) => {
    if (!content) {
      console.warn('renderMessageContent received null/undefined content');
      return <p className="text-gray-500 italic">No content</p>;
    }
    
    if (!Array.isArray(content)) {
      console.warn('renderMessageContent received non-array content:', content);
      return <p className="text-red-500">Invalid message content</p>;
    }
    
    if (content.length === 0) {
      console.log('renderMessageContent received empty content array');
      return <p className="text-gray-500 italic">Empty message</p>;
    }
    
    return content.map((item, index) => {
      switch (item.type) {
        case 'text':
          return <p key={index} className="whitespace-pre-wrap">{item.text}</p>;
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
    <div className="h-screen bg-gray-50 flex">
      {/* Conversation Sidebar */}
      <div className={`bg-white border-r border-gray-200 flex flex-col transition-all duration-300 ${
        sidebarOpen ? 'w-80' : 'w-16'
      }`}>
        <div className="p-4 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <button
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="p-1 rounded-md hover:bg-gray-100"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </button>
            {sidebarOpen && (
              <div className="flex items-center space-x-2">
                <h2 className="text-lg font-semibold text-gray-900">Conversations</h2>
                <button
                  onClick={handleNewConversationClick}
                  className="p-1 rounded-md hover:bg-gray-100 text-blue-600"
                  title="New Conversation"
                >
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                  </svg>
                </button>
              </div>
            )}
          </div>
          {sidebarOpen && bot && (
            <div className="mt-2 text-sm text-gray-600">
              Chatting with <span className="font-medium">{bot.title}</span>
              {currentConversation && (
                <div className="mt-1 text-xs text-gray-500">
                  {currentConversation.title}
                </div>
              )}
            </div>
          )}
        </div>

        {sidebarOpen && (
          <div className="flex-1 overflow-y-auto">
            {conversations.length === 0 ? (
              <div className="p-4 text-center text-gray-500">
                <p>No conversations yet</p>
                <button
                  onClick={handleNewConversationClick}
                  className="mt-2 text-blue-600 hover:text-blue-700 text-sm"
                >
                  Start your first conversation
                </button>
              </div>
            ) : (
              <div className="space-y-1 p-2">
                {Array.isArray(conversations) && conversations.map((conv) => (
                  <div
                    key={conv.id}
                    className={`group relative rounded-lg hover:bg-gray-50 transition-colors ${
                      selectedConversationId === conv.id ? 'bg-blue-50 border border-blue-200' : ''
                    }`}
                  >
                    <button
                      onClick={() => loadConversation(conv.id)}
                      className="w-full text-left p-3 pr-10"
                    >
                      <div className="font-medium text-gray-900 truncate">{conv.title}</div>
                      <div className="text-sm text-gray-500 mt-1 flex items-center justify-between">
                        <span>{conv.messageCount} messages • {new Date(conv.updatedAt).toLocaleDateString()}</span>
                        {conv.sessionModelId && (
                          <span className="bg-blue-100 text-blue-800 text-xs px-2 py-1 rounded-full">
                            {availableModels.find(m => m.id === conv.sessionModelId)?.name || 'Custom'}
                          </span>
                        )}
                      </div>
                      {conv.lastMessage && (
                        <div className="text-xs text-gray-400 mt-1 truncate">{conv.lastMessage}</div>
                      )}
                    </button>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        if (confirm('Are you sure you want to delete this conversation?')) {
                          deleteConversation(conv.id);
                        }
                      }}
                      className="absolute top-3 right-3 opacity-0 group-hover:opacity-100 p-1 hover:bg-red-100 rounded-full transition-all"
                      title="Delete conversation"
                    >
                      <svg className="w-4 h-4 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col">
        {/* Header */}
        <header className="bg-white border-b border-gray-200 px-6 py-4">
          <div className="flex justify-between items-center">
            <div>
              {bot ? (
                <div>
                  <h1 className="text-xl font-semibold text-gray-900">{bot.title}</h1>
                  <p className="text-sm text-gray-600">{bot.description}</p>
                </div>
              ) : (
                <div className="animate-pulse">
                  <div className="h-6 bg-gray-200 rounded w-48 mb-2"></div>
                  <div className="h-4 bg-gray-200 rounded w-32"></div>
                </div>
              )}
            </div>
            <div className="flex items-center space-x-4">
              {/* Model Selector Dropdown */}
              <div className="relative">
                <label className="text-xs text-gray-500 mb-1 block">Model</label>
                <select
                  value={sessionModel || ''}
                  onChange={(e) => handleModelChange(e.target.value)}
                  className="bg-white border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  {availableModels.map((model) => (
                    <option key={model.id} value={model.id}>
                      {model.name}
                    </option>
                  ))}
                </select>
              </div>
              
              <Link
                to="/bots"
                className="text-gray-600 hover:text-gray-900 transition-colors"
              >
                Back to Bots
              </Link>
              <button
                onClick={signOut}
                className="bg-gray-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-gray-700 transition-colors"
              >
                Sign Out
              </button>
            </div>
          </div>
        </header>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto p-6 space-y-4">
          {!selectedConversationId && messages.length === 0 && (
            <div className="text-center py-12">
              <div className="text-gray-400 mb-4">
                <svg className="mx-auto h-16 w-16" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                </svg>
              </div>
              <h3 className="text-lg font-medium text-gray-900 mb-2">Start a conversation</h3>
              <p className="text-gray-600 mb-4">Ask me anything about {bot?.title || 'this bot'}!</p>
              {bot?.conversationStarters && bot.conversationStarters.length > 0 && (
                <div className="space-y-2">
                  <p className="text-sm text-gray-500">Try one of these:</p>
                  <div className="space-y-2">
                    {bot.conversationStarters.slice(0, 3).map((starter, index) => (
                      <button
                        key={index}
                        onClick={() => setInput(starter.example)}
                        className="block w-full max-w-md mx-auto p-3 text-left bg-white border border-gray-200 rounded-lg hover:border-blue-300 hover:bg-blue-50 transition-colors"
                      >
                        <div className="font-medium text-gray-900">{starter.title}</div>
                        <div className="text-sm text-gray-600 mt-1">{starter.example}</div>
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {Array.isArray(messages) && messages
            // Filter out system/instruction messages with null content - they're not meant for display
            .filter(message => {
              // Filter out system/instruction messages and null content (these are internal)
              return message.content && 
                     message.content !== null && 
                     message.role !== 'system' && 
                     message.role !== 'instruction';
            })
            .map((message) => (
            <div
              key={message.id}
              className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
            >
              <div className={`max-w-3xl px-4 py-3 rounded-lg ${
                message.role === 'user'
                  ? 'bg-blue-600 text-white'
                  : 'bg-white border border-gray-200 text-gray-900'
              }`}>
                <div className="space-y-2">
                  {renderMessageContent(message.content)}
                </div>
                <div className={`text-xs mt-2 ${
                  message.role === 'user' ? 'text-blue-100' : 'text-gray-500'
                }`}>
                  {new Date(message.createdAt).toLocaleTimeString()}
                  {message.tokenCount && (
                    <span className="ml-2">• {message.tokenCount} tokens</span>
                  )}
                </div>
              </div>
            </div>
          ))}

          {/* 🚀 INGENIOUS ENHANCEMENT: Knowledge Search Stages Display */}
          {(isLoading || currentSearchStages.length > 0) && (
            <KnowledgeSearchStages 
              stages={currentSearchStages} 
              isLoading={isLoading && currentSearchStages.length === 0}
            />
          )}

          {/* 🚀 INGENIOUS ENHANCEMENT: Source Citations Display */}
          {lastChatResponse?.sources && lastChatResponse.sources.length > 0 && (
            <SourceCitations sources={lastChatResponse.sources} className="mb-4" />
          )}

          {isLoading && (
            <div className="flex justify-start">
              <div className="bg-white border border-gray-200 px-4 py-3 rounded-lg">
                <div className="flex items-center space-x-2">
                  <div className="flex space-x-1">
                    <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce"></div>
                    <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0.1s' }}></div>
                    <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0.2s' }}></div>
                  </div>
                  <span className="text-sm text-gray-500">Thinking...</span>
                </div>
              </div>
            </div>
          )}

          {error && (
            <div className="bg-red-50 border border-red-200 rounded-lg p-4">
              <div className="text-red-800 text-sm">{error}</div>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>

        {/* Input Form */}
        <div className="border-t border-gray-200 bg-white p-4">
          <form onSubmit={sendMessage} className="flex space-x-3">
            <input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder={`Message ${bot?.title || 'bot'}...`}
              className="flex-1 px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
              disabled={isLoading}
            />
            <button
              type="submit"
              disabled={isLoading || !input.trim()}
              className="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors font-medium"
            >
              Send
            </button>
          </form>
        </div>
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

      {/* Model Change Warning Dialog */}
      {showModelChangeWarning && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
            <div className="flex items-center mb-4">
              <div className="flex-shrink-0">
                <svg className="h-6 w-6 text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.728-.833-2.498 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
                </svg>
              </div>
              <h3 className="ml-3 text-lg font-semibold text-gray-900">Switch Model?</h3>
            </div>
            
            <p className="text-sm text-gray-600 mb-6">
              You're currently in the middle of a conversation. Switching models will start a new conversation with the selected model. 
              Your current conversation will be saved.
            </p>
            
            <div className="mb-4">
              <p className="text-sm font-medium text-gray-700 mb-1">Current Model:</p>
              <p className="text-sm text-gray-600">{getCurrentModelName()}</p>
            </div>
            
            <div className="mb-6">
              <p className="text-sm font-medium text-gray-700 mb-1">New Model:</p>
              <p className="text-sm text-gray-600">
                {availableModels.find(m => m.id === pendingModelChange)?.name || 'Unknown'}
              </p>
            </div>
            
            <div className="flex justify-end space-x-3">
              <button
                onClick={cancelModelChange}
                className="px-4 py-2 text-gray-600 hover:text-gray-800 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={confirmModelChange}
                className="px-4 py-2 bg-amber-600 text-white rounded-md hover:bg-amber-700 transition-colors"
              >
                Start New Conversation
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}