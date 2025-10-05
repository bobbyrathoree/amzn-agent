import { useState, useEffect, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useAuth } from '../components/AuthProvider';
import { useApiClient } from '../lib/api';
import type { Bot, GenerationParams } from '../types';

export function BotEditPage() {
  const { botId } = useParams<{ botId: string }>();
  const { user, signOut, getAccessToken } = useAuth();
  const getUserId = useCallback(() => user?.userId || user?.username || null, [user?.userId, user?.username]);
  const apiClient = useApiClient(getAccessToken, getUserId);
  
  const [bot, setBot] = useState<Bot | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  
  // Form state
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [instruction, setInstruction] = useState('');
  const [activeModels, setActiveModels] = useState<string[]>([]);
  const [sharedScope, setSharedScope] = useState<'private' | 'partial' | 'public'>('private');
  const [allowedUsers, setAllowedUsers] = useState<string[]>([]);
  const [allowedGroups, setAllowedGroups] = useState<string[]>([]);
  const [conversationStarters, setConversationStarters] = useState<string[]>([]);
  const [generationParams, setGenerationParams] = useState<GenerationParams>({
    maxTokens: 4000,
    temperature: 0.7,
    topP: 0.9,
    topK: 250,
    stopSequences: []
  });

  // Available models for selection
  const availableModels = [
    // Claude Models (US inference profiles)
    { id: 'us.anthropic.claude-opus-4-20250514-v1:0', name: 'Claude 4 Opus' },
    { id: 'us.anthropic.claude-sonnet-4-20250514-v1:0', name: 'Claude 4 Sonnet' },
    { id: 'us.anthropic.claude-3-7-sonnet-20250219-v1:0', name: 'Claude 3.7 Sonnet' },
    { id: 'us.anthropic.claude-3-5-sonnet-20241022-v2:0', name: 'Claude 3.5 Sonnet v2' },
    { id: 'us.anthropic.claude-3-5-sonnet-20240620-v1:0', name: 'Claude 3.5 Sonnet' },
    { id: 'us.anthropic.claude-3-5-haiku-20241022-v1:0', name: 'Claude 3.5 Haiku' },
    { id: 'us.anthropic.claude-3-haiku-20240307-v1:0', name: 'Claude 3 Haiku' },
    { id: 'us.anthropic.claude-3-opus-20240229-v1:0', name: 'Claude 3 Opus' },
    
    // Amazon Nova Models
    { id: 'us.amazon.nova-pro-v1:0', name: 'Nova Pro' },
    { id: 'us.amazon.nova-lite-v1:0', name: 'Nova Lite' },
    { id: 'us.amazon.nova-micro-v1:0', name: 'Nova Micro' },
    
    // Mistral Models
    { id: 'us.mistral.mistral-large-2407-v1:0', name: 'Mistral Large 2407' },
    { id: 'us.mistral.mistral-large-2402-v1:0', name: 'Mistral Large 2402' },
    { id: 'us.mistral.mixtral-8x7b-instruct-v0:1', name: 'Mixtral 8x7B' },
    { id: 'us.mistral.mistral-7b-instruct-v0:2', name: 'Mistral 7B' },
    
    // DeepSeek Models
    { id: 'us.deepseek.r1-v1:0', name: 'DeepSeek R1' },
    
    // Meta Llama Models
    { id: 'us.meta.llama3-3-70b-instruct-v1:0', name: 'Llama 3.3 70B' },
    { id: 'us.meta.llama3-2-90b-instruct-v1:0', name: 'Llama 3.2 90B' },
    { id: 'us.meta.llama3-2-11b-instruct-v1:0', name: 'Llama 3.2 11B' },
    { id: 'us.meta.llama3-2-3b-instruct-v1:0', name: 'Llama 3.2 3B' },
    { id: 'us.meta.llama3-2-1b-instruct-v1:0', name: 'Llama 3.2 1B' }
  ];

  useEffect(() => {
    if (apiClient && botId) {
      loadBot();
    }
  }, [apiClient, botId]);

  const loadBot = async () => {
    if (!apiClient || !botId) return;

    try {
      setLoading(true);
      setError(null);

      const response = await apiClient.get(`bots/${botId}`);
      if (!response.ok) {
        if (response.status === 404) {
          setError('Bot not found');
          return;
        }
        if (response.status === 403) {
          setError('Access denied - you can only edit bots you own');
          return;
        }
        throw new Error(`Failed to load bot: ${response.status}`);
      }

      const data = await response.json();
      const botData = data.bot;
      setBot(botData);
      
      // Populate form fields
      setTitle(botData.title || '');
      setDescription(botData.description || '');
      setInstruction(botData.instruction || '');
      setActiveModels(botData.activeModels || []);
      setSharedScope(botData.sharedScope || 'private');
      setAllowedUsers(botData.allowedUsers || []);
      setAllowedGroups(botData.allowedGroups || []);
      setConversationStarters(botData.conversationStarters || []);
      setGenerationParams(botData.generationParams || {
        maxTokens: 4000,
        temperature: 0.7,
        topP: 0.9,
        topK: 250,
        stopSequences: []
      });
      
    } catch (err) {
      console.error('Error loading bot:', err);
      setError(err instanceof Error ? err.message : 'Failed to load bot');
    } finally {
      setLoading(false);
    }
  };

  const saveBot = async () => {
    if (!apiClient || !botId) return;

    try {
      setSaving(true);
      setError(null);
      setSuccess(null);

      const updateData = {
        title,
        description,
        instruction,
        activeModels,
        sharedScope,
        allowedUsers,
        allowedGroups,
        conversationStarters,
        generationParams
      };

      const response = await apiClient.put(`bots/${botId}`, updateData);
      
      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`Failed to update bot: ${response.status} - ${errorText}`);
      }

      setSuccess('Bot updated successfully!');
      
      // Reload bot data to get the updated version
      await loadBot();
      
    } catch (err) {
      console.error('Error saving bot:', err);
      setError(err instanceof Error ? err.message : 'Failed to save bot');
    } finally {
      setSaving(false);
    }
  };

  const handleModelToggle = (modelId: string) => {
    if (activeModels.includes(modelId)) {
      setActiveModels(activeModels.filter(id => id !== modelId));
    } else {
      setActiveModels([...activeModels, modelId]);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading bot...</p>
        </div>
      </div>
    );
  }

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
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center space-x-4">
              <Link to="/bots" className="text-xl font-bold text-gray-900">
                Foundry
              </Link>
              <span className="text-gray-500">|</span>
              <span className="text-gray-700">Edit Bot</span>
            </div>
            <div className="flex items-center space-x-4">
              <span className="text-sm text-gray-700">Welcome, {user?.username}</span>
              <button
                onClick={signOut}
                className="bg-gray-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-gray-700 transition-colors"
              >
                Sign Out
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="bg-white shadow rounded-lg">
          <div className="px-6 py-4 border-b border-gray-200">
            <h1 className="text-xl font-semibold text-gray-900">Edit Bot: {bot?.title}</h1>
          </div>

          <div className="p-6 space-y-6">
            {/* Success/Error Messages */}
            {success && (
              <div className="bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded">
                {success}
              </div>
            )}
            
            {error && (
              <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
                {error}
              </div>
            )}

            {/* Basic Information */}
            <div className="space-y-4">
              <h2 className="text-lg font-medium text-gray-900">Basic Information</h2>
              
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Title *
                </label>
                <input
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="Enter bot title"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Description *
                </label>
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={3}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="Describe what your bot does"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  System Instruction *
                </label>
                <textarea
                  value={instruction}
                  onChange={(e) => setInstruction(e.target.value)}
                  rows={6}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="Enter the system instruction that defines your bot's behavior"
                  required
                />
              </div>
            </div>

            {/* Access & Sharing */}
            <div className="space-y-4">
              <h2 className="text-lg font-medium text-gray-900">Access & Sharing</h2>
              
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Sharing Level
                </label>
                <select
                  value={sharedScope}
                  onChange={(e) => setSharedScope(e.target.value as 'private' | 'partial' | 'public')}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="private">Private (only you)</option>
                  <option value="partial">Shared (specific users/groups)</option>
                  <option value="public">Public (everyone)</option>
                </select>
              </div>

              {sharedScope === 'partial' && (
                <>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      Allowed Users (comma-separated email addresses)
                    </label>
                    <input
                      type="text"
                      value={allowedUsers.join(', ')}
                      onChange={(e) => setAllowedUsers(e.target.value.split(',').map(u => u.trim()).filter(u => u))}
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      placeholder="user1@example.com, user2@example.com"
                    />
                  </div>
                  
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      Allowed Groups (comma-separated group names)
                    </label>
                    <input
                      type="text"
                      value={allowedGroups.join(', ')}
                      onChange={(e) => setAllowedGroups(e.target.value.split(',').map(g => g.trim()).filter(g => g))}
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      placeholder="admins, developers, testers"
                    />
                  </div>
                </>
              )}
            </div>

            {/* Conversation Starters */}
            <div className="space-y-4">
              <h2 className="text-lg font-medium text-gray-900">Conversation Starters</h2>
              <p className="text-sm text-gray-600">Add suggested conversation starters to help users get started with your bot.</p>
              
              <div className="space-y-2">
                {conversationStarters.map((starter, index) => (
                  <div key={index} className="flex items-center space-x-2">
                    <input
                      type="text"
                      value={starter}
                      onChange={(e) => {
                        const newStarters = [...conversationStarters];
                        newStarters[index] = e.target.value;
                        setConversationStarters(newStarters);
                      }}
                      className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      placeholder="Enter a conversation starter..."
                    />
                    <button
                      type="button"
                      onClick={() => {
                        const newStarters = conversationStarters.filter((_, i) => i !== index);
                        setConversationStarters(newStarters);
                      }}
                      className="px-3 py-2 text-red-600 hover:text-red-700 hover:bg-red-50 rounded-md transition-colors"
                    >
                      Remove
                    </button>
                  </div>
                ))}
                
                <button
                  type="button"
                  onClick={() => setConversationStarters([...conversationStarters, ''])}
                  className="w-full px-3 py-2 border border-gray-300 border-dashed rounded-md text-gray-600 hover:text-gray-700 hover:bg-gray-50 transition-colors"
                >
                  + Add Conversation Starter
                </button>
              </div>
            </div>

            {/* Active Models */}
            <div className="space-y-4">
              <h2 className="text-lg font-medium text-gray-900">Active Models</h2>
              <p className="text-sm text-gray-600">Select which models this bot can use. At least one model must be selected.</p>
              
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                {availableModels.map((model) => (
                  <label key={model.id} className="flex items-center space-x-2 p-2 border border-gray-200 rounded hover:bg-gray-50">
                    <input
                      type="checkbox"
                      checked={activeModels.includes(model.id)}
                      onChange={() => handleModelToggle(model.id)}
                      className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                    />
                    <span className="text-sm text-gray-700">{model.name}</span>
                  </label>
                ))}
              </div>
              
              {activeModels.length === 0 && (
                <p className="text-sm text-red-600">Please select at least one model.</p>
              )}
            </div>

            {/* Generation Parameters */}
            <div className="space-y-4">
              <h2 className="text-lg font-medium text-gray-900">Generation Parameters</h2>
              
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Max Tokens
                  </label>
                  <input
                    type="number"
                    value={generationParams.maxTokens}
                    onChange={(e) => setGenerationParams({
                      ...generationParams,
                      maxTokens: parseInt(e.target.value) || 0
                    })}
                    min="1"
                    max="8192"
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Temperature ({generationParams.temperature})
                  </label>
                  <input
                    type="range"
                    value={generationParams.temperature}
                    onChange={(e) => setGenerationParams({
                      ...generationParams,
                      temperature: parseFloat(e.target.value)
                    })}
                    min="0"
                    max="1"
                    step="0.1"
                    className="w-full"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Top P ({generationParams.topP})
                  </label>
                  <input
                    type="range"
                    value={generationParams.topP}
                    onChange={(e) => setGenerationParams({
                      ...generationParams,
                      topP: parseFloat(e.target.value)
                    })}
                    min="0"
                    max="1"
                    step="0.1"
                    className="w-full"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Top K
                  </label>
                  <input
                    type="number"
                    value={generationParams.topK}
                    onChange={(e) => setGenerationParams({
                      ...generationParams,
                      topK: parseInt(e.target.value) || 0
                    })}
                    min="1"
                    max="500"
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
              </div>
            </div>

            {/* Action Buttons */}
            <div className="flex justify-between pt-6 border-t border-gray-200">
              <Link
                to="/bots"
                className="bg-gray-200 text-gray-800 px-6 py-2 rounded-md hover:bg-gray-300 transition-colors"
              >
                Cancel
              </Link>
              
              <button
                onClick={saveBot}
                disabled={saving || !title.trim() || !description.trim() || !instruction.trim() || activeModels.length === 0}
                className="bg-blue-600 text-white px-6 py-2 rounded-md hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {saving ? 'Saving...' : 'Save Changes'}
              </button>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}