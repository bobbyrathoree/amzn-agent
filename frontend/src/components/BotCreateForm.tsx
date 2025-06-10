'use client';

import { useState, FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '../hooks/useAuth';
import { fetchAuthSession } from 'aws-amplify/auth';

interface BotCreateFormProps {
  onSuccess?: (botId: string) => void;
  onCancel?: () => void;
}

export default function BotCreateForm({ onSuccess, onCancel }: BotCreateFormProps) {
  const router = useRouter();
  const { user } = useAuth();
  
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  const [formData, setFormData] = useState({
    title: '',
    description: '',
    instruction: '',
    isPublic: false,
    knowledgeBaseId: '',
    displayRetrievedChunks: true,
    generationParams: {
      maxTokens: 2048,
      temperature: 0.7,
      topP: 0.9,
      topK: 250,
    },
    knowledgeBaseConfig: {
      searchType: 'HYBRID',
      maxResults: 20,
      scoreThreshold: 0.7,
    },
    activeModels: ['anthropic.claude-3-5-sonnet-20240620-v1:0'],
    conversationStarters: ['Hi! How can I help you?'],
    agentTools: [],
  });

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value, type } = e.target;
    
    if (name === 'isPublic') {
      const inputEl = e.target as HTMLInputElement;
      setFormData({
        ...formData,
        [name]: inputEl.checked,
      });
      return;
    }
    
    if (name.startsWith('generationParams.')) {
      const paramName = name.split('.')[1];
      let paramValue: string | number = value;
      
      // Convert to number for numeric fields
      if (['maxTokens', 'topK'].includes(paramName)) {
        paramValue = parseInt(value, 10) || 0;
      } else if (['temperature', 'topP'].includes(paramName)) {
        paramValue = parseFloat(value) || 0;
      }
      
      setFormData({
        ...formData,
        generationParams: {
          ...formData.generationParams,
          [paramName]: paramValue,
        },
      });
      return;
    }
    
    if (name.startsWith('knowledgeBaseConfig.')) {
      const configName = name.split('.')[1];
      let configValue: string | number = value;
      
      // Convert to number for numeric fields
      if (['maxResults'].includes(configName)) {
        configValue = parseInt(value, 10) || 0;
      } else if (['scoreThreshold'].includes(configName)) {
        configValue = parseFloat(value) || 0;
      }
      
      setFormData({
        ...formData,
        knowledgeBaseConfig: {
          ...formData.knowledgeBaseConfig,
          [configName]: configValue,
        },
      });
      return;
    }
    
    setFormData({
      ...formData,
      [name]: value,
    });
  };
  
  const handleCheckboxChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, checked } = e.target;
    setFormData({
      ...formData,
      [name]: checked,
    });
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);
    
    try {
      // Validate required fields
      if (!formData.title) {
        throw new Error('Title is required');
      }
      
      if (!formData.instruction) {
        throw new Error('Instructions are required');
      }
      
      if (!formData.knowledgeBaseId) {
        throw new Error('Knowledge Base ID is required');
      }
      
      // Get the JWT token from Amplify
      const session = await fetchAuthSession();
      const accessToken = session.tokens?.accessToken?.toString();
      
      if (!accessToken) {
        throw new Error('Authentication required. Please sign in again.');
      }
      
      // Call backend directly with proper JWT authorization
      const apiUrl = 'https://c9wu2knteb.execute-api.us-east-1.amazonaws.com/prod/bots';
      const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${accessToken}`,
        'x-user-id': user?.userId || user?.username || 'unknown',
      };
      
      const response = await fetch(apiUrl, {
        method: 'POST',
        headers,
        body: JSON.stringify(formData),
      });
      
      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.message || 'Failed to create bot');
      }
      
      const data = await response.json();
      
      if (onSuccess) {
        onSuccess(data.bot.id);
      } else {
        router.push('/bots');
      }
    } catch (err: any) {
      setError(err.message || 'An error occurred');
    } finally {
      setIsLoading(false);
    }
  };
  
  return (
    <div className="max-w-4xl mx-auto p-6 bg-white rounded-lg shadow-md">
      <h2 className="text-2xl font-bold mb-6">Create a New Bot</h2>
      
      {error && (
        <div className="mb-4 p-3 bg-red-100 border border-red-400 text-red-700 rounded">
          {error}
        </div>
      )}
      
      <form onSubmit={handleSubmit}>
        <div className="space-y-6">
          {/* Basic Information */}
          <div>
            <h3 className="text-lg font-semibold mb-4">Basic Information</h3>
            <div className="space-y-4">
              <div>
                <label htmlFor="title" className="block text-sm font-medium text-gray-700">
                  Title *
                </label>
                <input
                  type="text"
                  id="title"
                  name="title"
                  value={formData.title}
                  onChange={handleInputChange}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 text-gray-900 bg-white"
                  maxLength={100}
                  required
                />
              </div>
              
              <div>
                <label htmlFor="description" className="block text-sm font-medium text-gray-700">
                  Description
                </label>
                <textarea
                  id="description"
                  name="description"
                  value={formData.description}
                  onChange={handleInputChange}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 text-gray-900 bg-white"
                  maxLength={500}
                  rows={2}
                />
              </div>
              
              <div>
                <label htmlFor="instruction" className="block text-sm font-medium text-gray-700">
                  Instructions *
                </label>
                <textarea
                  id="instruction"
                  name="instruction"
                  value={formData.instruction}
                  onChange={handleInputChange}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 text-gray-900 bg-white"
                  maxLength={4000}
                  rows={4}
                  required
                  placeholder="Instructions for how the bot should behave (e.g., 'You are a helpful marketing assistant specialized in email campaigns...')"
                />
              </div>
              
              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="isPublic"
                  name="isPublic"
                  checked={formData.isPublic}
                  onChange={handleCheckboxChange}
                  className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
                />
                <label htmlFor="isPublic" className="ml-2 block text-sm font-medium text-gray-700">
                  Make bot publicly available to other users
                </label>
              </div>
            </div>
          </div>
          
          {/* Knowledge Base Configuration */}
          <div>
            <h3 className="text-lg font-semibold mb-4">Knowledge Base</h3>
            <div className="space-y-4">
              <div>
                <label htmlFor="knowledgeBaseId" className="block text-sm font-medium text-gray-700">
                  Bedrock Knowledge Base ID *
                </label>
                <input
                  type="text"
                  id="knowledgeBaseId"
                  name="knowledgeBaseId"
                  value={formData.knowledgeBaseId}
                  onChange={handleInputChange}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 text-gray-900 bg-white"
                  required
                  placeholder="e.g. KB-12345678"
                />
                <p className="mt-1 text-sm text-gray-500">
                  Enter the ID of your existing AWS Bedrock Knowledge Base
                </p>
              </div>
              
              <div>
                <label htmlFor="searchType" className="block text-sm font-medium text-gray-700">
                  Search Type
                </label>
                <select
                  id="searchType"
                  name="knowledgeBaseConfig.searchType"
                  value={formData.knowledgeBaseConfig.searchType}
                  onChange={handleInputChange}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 text-gray-900 bg-white"
                >
                  <option value="HYBRID">Hybrid</option>
                  <option value="SEMANTIC">Semantic</option>
                </select>
              </div>
              
              <div>
                <label htmlFor="maxResults" className="block text-sm font-medium text-gray-700">
                  Max Results
                </label>
                <input
                  type="number"
                  id="maxResults"
                  name="knowledgeBaseConfig.maxResults"
                  value={formData.knowledgeBaseConfig.maxResults}
                  onChange={handleInputChange}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 text-gray-900 bg-white"
                  min={1}
                  max={100}
                />
              </div>
              
              <div>
                <label htmlFor="scoreThreshold" className="block text-sm font-medium text-gray-700">
                  Score Threshold
                </label>
                <input
                  type="number"
                  id="scoreThreshold"
                  name="knowledgeBaseConfig.scoreThreshold"
                  value={formData.knowledgeBaseConfig.scoreThreshold}
                  onChange={handleInputChange}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 text-gray-900 bg-white"
                  min={0}
                  max={1}
                  step={0.01}
                />
              </div>
              
              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="displayRetrievedChunks"
                  name="displayRetrievedChunks"
                  checked={formData.displayRetrievedChunks}
                  onChange={handleCheckboxChange}
                  className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
                />
                <label htmlFor="displayRetrievedChunks" className="ml-2 block text-sm font-medium text-gray-700">
                  Display retrieved chunks to users
                </label>
              </div>
            </div>
          </div>
          
          {/* Generation Parameters */}
          <div>
            <h3 className="text-lg font-semibold mb-4">Generation Parameters</h3>
            <div className="space-y-4 grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label htmlFor="maxTokens" className="block text-sm font-medium text-gray-700">
                  Max Tokens
                </label>
                <input
                  type="number"
                  id="maxTokens"
                  name="generationParams.maxTokens"
                  value={formData.generationParams.maxTokens}
                  onChange={handleInputChange}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 text-gray-900 bg-white"
                  min={1}
                  max={4096}
                />
              </div>
              
              <div>
                <label htmlFor="temperature" className="block text-sm font-medium text-gray-700">
                  Temperature
                </label>
                <input
                  type="number"
                  id="temperature"
                  name="generationParams.temperature"
                  value={formData.generationParams.temperature}
                  onChange={handleInputChange}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 text-gray-900 bg-white"
                  min={0}
                  max={1}
                  step={0.01}
                />
              </div>
              
              <div>
                <label htmlFor="topP" className="block text-sm font-medium text-gray-700">
                  Top-P
                </label>
                <input
                  type="number"
                  id="topP"
                  name="generationParams.topP"
                  value={formData.generationParams.topP}
                  onChange={handleInputChange}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 text-gray-900 bg-white"
                  min={0}
                  max={1}
                  step={0.01}
                />
              </div>
              
              <div>
                <label htmlFor="topK" className="block text-sm font-medium text-gray-700">
                  Top-K
                </label>
                <input
                  type="number"
                  id="topK"
                  name="generationParams.topK"
                  value={formData.generationParams.topK}
                  onChange={handleInputChange}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 text-gray-900 bg-white"
                  min={0}
                  max={500}
                />
              </div>
            </div>
          </div>
          
          <div className="flex justify-end space-x-3 pt-5">
            {onCancel && (
              <button
                type="button"
                onClick={onCancel}
                className="px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
                disabled={isLoading}
              >
                Cancel
              </button>
            )}
            
            <button
              type="submit"
              className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
              disabled={isLoading}
            >
              {isLoading ? 'Creating...' : 'Create Bot'}
            </button>
          </div>
        </div>
      </form>
    </div>
  );
}