'use client';

import { useState, useEffect } from 'react';
import { useAuth } from '@/hooks/useAuth';
import { useConfig } from '@/hooks/useConfig';
import { PlusIcon, ChatBubbleLeftIcon, CogIcon } from '@heroicons/react/24/outline';
import Link from 'next/link';
import BotCreateForm from '@/components/BotCreateForm';

interface Bot {
  id: string;
  title: string; // Changed from name to match backend model
  description: string;
  isPublic: boolean;
  activeModels: string[]; // Changed from defaultModel to match backend model
  createTime: string; // Changed from createdAt to match backend model
}

export default function BotsPage() {
  const { user } = useAuth();
  const { config } = useConfig();
  const [bots, setBots] = useState<Bot[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateForm, setShowCreateForm] = useState(false);

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
      // Add demo bots for development
      setBots([
        {
          id: 'demo-1',
          title: 'Claude Assistant',
          description: 'A helpful AI assistant powered by Claude',
          isPublic: true,
          activeModels: ['anthropic.claude-3-sonnet-20240620-v1:0'],
          createTime: new Date().toISOString(),
        },
        {
          id: 'demo-2',
          title: 'Code Helper',
          description: 'Specialized in helping with programming tasks',
          isPublic: false,
          activeModels: ['anthropic.claude-3-sonnet-20240620-v1:0'],
          createTime: new Date().toISOString(),
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const handleBotCreated = (botId: string) => {
    setShowCreateForm(false);
    fetchBots(); // Refresh the bot list after creating a new bot
  };

  if (!user) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <h2 className="text-2xl font-bold text-gray-900">Please sign in to manage bots</h2>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-6">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">My Bots</h1>
              <p className="mt-1 text-sm text-gray-500">
                Create and manage your AI chat bots
              </p>
            </div>
            <div className="flex space-x-3">
              <Link
                href="/chat"
                className="inline-flex items-center px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
              >
                <ChatBubbleLeftIcon className="h-4 w-4 mr-2" />
                Go to Chat
              </Link>
              <button
                onClick={() => setShowCreateForm(true)}
                className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700"
              >
                <PlusIcon className="h-4 w-4 mr-2" />
                Create Bot
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {loading ? (
          <div className="text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
            <p className="mt-4 text-gray-600">Loading bots...</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {bots.map((bot) => (
              <div
                key={bot.id}
                className="bg-white overflow-hidden shadow rounded-lg hover:shadow-md transition-shadow"
              >
                <div className="p-6">
                  <div className="flex items-center justify-between">
                    <h3 className="text-lg font-medium text-gray-900 truncate">
                      {bot.title}
                    </h3>
                    <div className="flex items-center space-x-2">
                      {bot.isPublic && (
                        <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                          Public
                        </span>
                      )}
                      <button className="text-gray-400 hover:text-gray-500">
                        <CogIcon className="h-5 w-5" />
                      </button>
                    </div>
                  </div>
                  <p className="mt-2 text-sm text-gray-500 h-12 overflow-hidden">
                    {bot.description}
                  </p>
                  <div className="mt-4">
                    <p className="text-xs text-gray-400">
                      Model: {bot.activeModels && bot.activeModels.length > 0 ? bot.activeModels[0] : 'N/A'}
                    </p>
                    <p className="text-xs text-gray-400">
                      Created: {new Date(bot.createTime).toLocaleDateString()}
                    </p>
                  </div>
                  <div className="mt-6">
                    <Link
                      href={`/chat?bot=${bot.id}`}
                      className="w-full flex justify-center items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700"
                    >
                      <ChatBubbleLeftIcon className="h-4 w-4 mr-2" />
                      Chat with Bot
                    </Link>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {bots.length === 0 && !loading && (
          <div className="text-center py-12">
            <h3 className="mt-2 text-sm font-medium text-gray-900">No bots</h3>
            <p className="mt-1 text-sm text-gray-500">
              Get started by creating your first AI bot.
            </p>
            <div className="mt-6">
              <button
                onClick={() => setShowCreateForm(true)}
                className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700"
              >
                <PlusIcon className="h-4 w-4 mr-2" />
                Create Bot
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Create Bot Modal */}
      {showCreateForm && (
        <div className="fixed inset-0 bg-gray-600 bg-opacity-50 flex items-center justify-center p-4 z-50 overflow-y-auto">
          <div className="bg-white rounded-lg w-full max-w-4xl p-6 max-h-[90vh] overflow-y-auto">
            <BotCreateForm 
              onSuccess={handleBotCreated}
              onCancel={() => setShowCreateForm(false)}
            />
          </div>
        </div>
      )}
    </div>
  );
}