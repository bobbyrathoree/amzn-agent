import { useState, useCallback } from 'react';
import { ApiClient } from '../lib/api';
import type { ChatRequest, ChatResponse, Message } from '../types';

interface UseChatConfig {
  apiClient: ApiClient | null;
  botId?: string;
  onError?: (error: Error) => void;
  onFinish?: (response: ChatResponse) => void;
}

interface UseChatReturn {
  messages: Message[];
  input: string;
  setInput: (input: string) => void;
  isLoading: boolean;
  error: string | null;
  sendMessage: (messageText?: string) => Promise<void>;
  reload: () => Promise<void>;
  stop: () => void;
  setMessages: (messages: Message[]) => void;
  append: (message: Message) => void;
}

/**
 * Custom hook that provides AI SDK-like functionality for chat interfaces
 * This replaces the need for Vercel AI SDK's useChat hook
 */
export function useCustomChat(config: UseChatConfig): UseChatReturn {
  const { apiClient, botId, onError, onFinish } = config;
  
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [abortController, setAbortController] = useState<AbortController | null>(null);

  const sendMessage = useCallback(async (messageText?: string) => {
    const textToSend = messageText || input.trim();
    if (!textToSend || !apiClient || !botId || isLoading) return;

    setInput('');
    setError(null);
    setIsLoading(true);

    // Create abort controller for this request
    const controller = new AbortController();
    setAbortController(controller);

    // Optimistic UI update
    const userMessage: Message = {
      id: `temp-user-${Date.now()}`,
      conversationId: '',
      role: 'user',
      content: [{ type: 'text', text: textToSend }],
      createdAt: new Date(),
      tokenCount: Math.ceil(textToSend.length / 4)
    };

    setMessages(prev => [...prev, userMessage]);

    try {
      const chatRequest: ChatRequest = {
        message: textToSend,
        stream: false,
        // Add other request parameters as needed
      };

      const response = await apiClient.postWithRetry(`bots/${botId}/chat`, chatRequest);
      
      if (!response.ok) {
        throw new Error(`Chat failed: ${response.status}`);
      }

      const chatResponse: ChatResponse = await response.json();
      
      // Create assistant message
      const assistantMessage: Message = {
        id: `assistant-${Date.now()}`,
        conversationId: chatResponse.conversationId,
        role: 'assistant',
        content: [{ type: 'text', text: chatResponse.response }],
        createdAt: new Date(),
        tokenCount: Math.ceil(chatResponse.response.length / 4)
      };

      // Replace temp message with actual messages
      setMessages(prev => [...prev.filter(m => m.id !== userMessage.id), assistantMessage]);
      
      onFinish?.(chatResponse);
      
    } catch (err) {
      const errorMessage = ApiClient.getErrorMessage(err);
      setError(errorMessage);
      
      // Remove optimistic message on error
      setMessages(prev => prev.filter(m => m.id !== userMessage.id));
      
      // Restore input text
      setInput(textToSend);
      
      onError?.(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setIsLoading(false);
      setAbortController(null);
    }
  }, [input, apiClient, botId, isLoading, onError, onFinish]);

  const reload = useCallback(async () => {
    if (messages.length === 0) return;
    
    // Find the last user message and resend it
    const lastUserMessage = [...messages].reverse().find(m => m.role === 'user');
    if (lastUserMessage && lastUserMessage.content[0]?.text) {
      // Remove assistant messages after the last user message
      const lastUserIndex = messages.findIndex(m => m.id === lastUserMessage.id);
      setMessages(prev => prev.slice(0, lastUserIndex + 1));
      
      await sendMessage(lastUserMessage.content[0].text);
    }
  }, [messages, sendMessage]);

  const stop = useCallback(() => {
    if (abortController) {
      abortController.abort();
      setAbortController(null);
    }
    setIsLoading(false);
  }, [abortController]);

  const append = useCallback((message: Message) => {
    setMessages(prev => [...prev, message]);
  }, []);

  return {
    messages,
    input,
    setInput,
    isLoading,
    error,
    sendMessage,
    reload,
    stop,
    setMessages,
    append
  };
}

/**
 * Hook for streaming chat responses (future enhancement)
 */
export function useStreamingChat(config: UseChatConfig) {
  // This can be implemented when backend supports streaming
  // For now, falls back to regular chat
  return useCustomChat(config);
}