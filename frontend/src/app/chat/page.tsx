'use client';

import { useChat } from 'ai/react';
import { useState, useEffect } from 'react';
import { useAuth } from '@/hooks/useAuth';
import { useConfig } from '@/hooks/useConfig';
import { PaperAirplaneIcon } from '@heroicons/react/24/solid';

export default function ChatPage() {
  const { user } = useAuth();
  const { config } = useConfig();
  const [selectedBotId, setSelectedBotId] = useState<string>('default');
  const [bots, setBots] = useState<any[]>([]);

  const { messages, input, handleInputChange, handleSubmit, isLoading } = useChat({
    api: '/api/chat',
    body: {
      botId: selectedBotId,
    },
    headers: {
      'Authorization': 'Bearer demo-token', // Replace with actual auth token
    },
  });

  useEffect(() => {
    if (config && user) {
      fetchBots();
    }
  }, [config, user]);

  const fetchBots = async () => {
    try {
      const isDevelopment = process.env.NODE_ENV === 'development';
      const apiUrl = isDevelopment 
        ? '/api/bots' 
        : 'https://c9wu2knteb.execute-api.us-east-1.amazonaws.com/prod/bots';
      
      const response = await fetch(apiUrl, {
        headers: {
          'Authorization': `Bearer demo-token`, // In real app, use actual JWT token
          'X-User-ID': user?.userId || 'bobrt-user-id',
        },
      });
      if (response.ok) {
        const data = await response.json();
        setBots(data.bots || []);
      }
    } catch (error) {
      console.error('Failed to fetch bots:', error);
      // Add a default bot for demo
      setBots([
        {
          id: 'default',
          name: 'Claude Assistant',
          description: 'A helpful AI assistant powered by Claude',
        },
      ]);
    }
  };

  if (!user) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <h2 className="text-2xl font-bold text-gray-900">Please sign in to chat</h2>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Sidebar */}
      <div className="w-80 bg-white border-r border-gray-200 flex flex-col">
        {/* Header */}
        <div className="p-4 border-b border-gray-200">
          <h1 className="text-lg font-semibold text-gray-900">AI Chat</h1>
          <p className="text-sm text-gray-500">Welcome, {user.username}</p>
        </div>

        {/* Bot Selection */}
        <div className="p-4 border-b border-gray-200">
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Select Bot
          </label>
          <select
            value={selectedBotId}
            onChange={(e) => setSelectedBotId(e.target.value)}
            className="w-full p-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500 text-gray-900 bg-white"
          >
            {bots.map((bot) => (
              <option key={bot.id} value={bot.id}>
                {bot.name}
              </option>
            ))}
          </select>
        </div>

        {/* Conversation History */}
        <div className="flex-1 p-4">
          <h3 className="text-sm font-medium text-gray-700 mb-2">Recent Conversations</h3>
          <div className="space-y-2">
            <div className="p-2 rounded-md bg-blue-50 border border-blue-200">
              <p className="text-sm text-blue-900">Current Chat</p>
              <p className="text-xs text-blue-600">{messages.length} messages</p>
            </div>
          </div>
        </div>
      </div>

      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col">
        {/* Chat Header */}
        <div className="bg-white border-b border-gray-200 p-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold text-gray-900">
                {bots.find(b => b.id === selectedBotId)?.name || 'AI Assistant'}
              </h2>
              <p className="text-sm text-gray-500">
                {bots.find(b => b.id === selectedBotId)?.description || 'AI-powered chat assistant'}
              </p>
            </div>
          </div>
        </div>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {messages.length === 0 ? (
            <div className="text-center text-gray-500 mt-8">
              <h3 className="text-lg font-medium">Start a conversation</h3>
              <p className="mt-2">Send a message to begin chatting with the AI assistant.</p>
            </div>
          ) : (
            messages.map((message) => (
              <div
                key={message.id}
                className={`flex ${
                  message.role === 'user' ? 'justify-end' : 'justify-start'
                }`}
              >
                <div
                  className={`max-w-xs lg:max-w-md px-4 py-2 rounded-lg ${
                    message.role === 'user'
                      ? 'bg-blue-600 text-white'
                      : 'bg-white border border-gray-200 text-gray-900'
                  }`}
                >
                  <p className="text-sm">{message.content}</p>
                </div>
              </div>
            ))
          )}
          
          {isLoading && (
            <div className="flex justify-start">
              <div className="bg-white border border-gray-200 text-gray-900 max-w-xs lg:max-w-md px-4 py-2 rounded-lg">
                <div className="flex items-center space-x-2">
                  <div className="flex space-x-1">
                    <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce"></div>
                    <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{animationDelay: '0.1s'}}></div>
                    <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{animationDelay: '0.2s'}}></div>
                  </div>
                  <span className="text-sm text-gray-500">Thinking...</span>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Message Input */}
        <div className="bg-white border-t border-gray-200 p-4">
          <form onSubmit={handleSubmit} className="flex space-x-2">
            <input
              value={input}
              onChange={handleInputChange}
              placeholder="Type your message..."
              className="flex-1 p-3 border border-gray-300 rounded-lg focus:ring-blue-500 focus:border-blue-500 text-gray-900 bg-white placeholder-gray-500"
              disabled={isLoading}
            />
            <button
              type="submit"
              disabled={isLoading || !input.trim()}
              className="px-4 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <PaperAirplaneIcon className="h-5 w-5" />
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}