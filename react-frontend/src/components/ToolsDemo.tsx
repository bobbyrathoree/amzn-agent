// 🧪 TOOLS FRAMEWORK DEMO
// Complete demonstration of the tools framework with API key vault integration

import { useState, useEffect } from 'react';
import { useAuth } from './AuthProvider';
import { useApiClient } from '../lib/api';
import { APIKeyVaultService } from '../services/vaultService';
import { toolExecutor } from '../services/toolExecutor';
import { initializeTools, getAllTools, findToolsForIntent } from '../tools';
import { VaultUnlockModal } from './VaultUnlockModal';
import { APIKeySetupModal } from './APIKeySetupModal';
import type { UniversalTool, ToolResult } from '../types/tools';
import { PlayIcon, KeyIcon, CogIcon, ChartBarIcon } from '@heroicons/react/24/outline';

export function ToolsDemo() {
  const { user, getAccessToken, environment } = useAuth();
  const getUserId = () => user?.userId || user?.username || null;
  const apiClient = useApiClient(getAccessToken, getUserId, environment);
  
  // Vault state
  const [vaultService, setVaultService] = useState<APIKeyVaultService | null>(null);
  const [isVaultUnlocked, setIsVaultUnlocked] = useState(false);
  const [showVaultModal, setShowVaultModal] = useState(false);
  const [showKeySetupModal, setShowKeySetupModal] = useState(false);
  const [selectedService, setSelectedService] = useState<string>('');
  
  // Tools state
  const [availableTools, setAvailableTools] = useState<UniversalTool[]>([]);
  const [selectedTool, setSelectedTool] = useState<UniversalTool | null>(null);
  const [executionResult, setExecutionResult] = useState<ToolResult | null>(null);
  const [isExecuting, setIsExecuting] = useState(false);
  
  // Demo inputs
  const [searchQuery, setSearchQuery] = useState('artificial intelligence trends 2024');
  const [researchTopic, setResearchTopic] = useState('sustainable energy solutions');

  useEffect(() => {
    if (apiClient && user) {
      // Initialize vault service
      const vault = new APIKeyVaultService(apiClient);
      setVaultService(vault);
      
      // Set vault service in tool executor
      toolExecutor.setVaultService(vault);
      
      // Initialize tools framework
      initializeTools();
      setAvailableTools(getAllTools());
    }
  }, [apiClient, user]);

  const handleUnlockVault = async (password: string, vaultPin: string): Promise<boolean> => {
    if (!vaultService) return false;
    
    const success = await vaultService.unlockVault(password, vaultPin);
    setIsVaultUnlocked(success);
    return success;
  };

  const handleExecuteTool = async (toolId: string, capability: string, input: any) => {
    if (!vaultService || !user) return;
    
    setIsExecuting(true);
    setExecutionResult(null);
    
    try {
      const result = await toolExecutor.execute({
        toolId,
        capability,
        input,
        context: {
          userId: user.userId,
          permissions: ['internet-access:read', 'external-api:read'],
          requestId: `req_${Date.now()}`,
          timestamp: new Date()
        }
      });
      
      setExecutionResult(result);
      
      // Handle API key required error
      if (!result.success && result.error?.code === 'API_KEY_REQUIRED') {
        const missingService = result.error.details?.missingServices?.[0];
        if (missingService) {
          setSelectedService(missingService);
          setShowKeySetupModal(true);
        }
      }
      
    } catch (error) {
      setExecutionResult({
        success: false,
        error: {
          code: 'EXECUTION_ERROR',
          message: error instanceof Error ? error.message : 'Unknown error'
        }
      });
    } finally {
      setIsExecuting(false);
    }
  };

  const handleSmartToolSearch = async (intent: string) => {
    if (!user) return;
    
    const plan = await findToolsForIntent(intent, {
      userId: user.userId,
      permissions: ['internet-access:read'],
      requestId: `req_${Date.now()}`,
      timestamp: new Date()
    });
    
    if (plan && plan.steps.length > 0) {
      const step = plan.steps[0];
      const tool = availableTools.find(t => t.id === step.toolId);
      if (tool) {
        setSelectedTool(tool);
        console.log(`🎯 Smart routing suggests: ${tool.name} (confidence: ${Math.round(plan.confidence * 100)}%)`);
      }
    }
  };

  if (!user) {
    return (
      <div className="p-6 text-center">
        <p className="text-gray-500">Please sign in to access the Tools Framework Demo</p>
      </div>
    );
  }

  return (
    <div className="max-w-6xl mx-auto p-6 space-y-8">
      {/* Header */}
      <div className="text-center">
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">
          🛠️ Tools Framework Demo
        </h1>
        <p className="text-gray-600 dark:text-gray-300">
          Experience the power of AI tools with secure API key management
        </p>
      </div>

      {/* Vault Status */}
      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 border border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white flex items-center">
            <KeyIcon className="w-6 h-6 mr-2" />
            API Key Vault
          </h2>
          <div className={`px-3 py-1 rounded-full text-sm font-medium ${
            isVaultUnlocked 
              ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
              : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
          }`}>
            {isVaultUnlocked ? '🔓 Unlocked' : '🔒 Locked'}
          </div>
        </div>
        
        <p className="text-gray-600 dark:text-gray-300 mb-4">
          {isVaultUnlocked 
            ? 'Your vault is unlocked. Premium tools can access stored API keys securely.'
            : 'Unlock your vault to use premium features with stored API keys.'
          }
        </p>
        
        {!isVaultUnlocked && (
          <button
            onClick={() => setShowVaultModal(true)}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-medium"
          >
            Unlock Vault
          </button>
        )}
        
        {isVaultUnlocked && (
          <button
            onClick={() => {
              vaultService?.lockVault();
              setIsVaultUnlocked(false);
            }}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 font-medium"
          >
            Lock Vault
          </button>
        )}
      </div>

      {/* Available Tools */}
      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 border border-gray-200 dark:border-gray-700">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
          <CogIcon className="w-6 h-6 mr-2" />
          Available Tools ({availableTools.length})
        </h2>
        
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {availableTools.map((tool) => (
            <div
              key={tool.id}
              className={`p-4 border rounded-lg cursor-pointer transition-colors ${
                selectedTool?.id === tool.id
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                  : 'border-gray-200 dark:border-gray-600 hover:border-gray-300 dark:hover:border-gray-500'
              }`}
              onClick={() => setSelectedTool(tool)}
            >
              <div className="flex items-center space-x-3">
                <span className="text-2xl">{tool.icon}</span>
                <div className="flex-1">
                  <h3 className="font-medium text-gray-900 dark:text-white">
                    {tool.name}
                  </h3>
                  <p className="text-sm text-gray-500 dark:text-gray-400">
                    {tool.description}
                  </p>
                  <div className="mt-2 flex items-center space-x-2">
                    <span className="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-700 rounded">
                      {tool.category}
                    </span>
                    <span className="text-xs text-gray-500">
                      {tool.capabilities.length} capabilities
                    </span>
                    {tool.apiRequirements && (
                      <span className="text-xs px-2 py-1 bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 rounded">
                        Premium Features
                      </span>
                    )}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Smart Tool Search */}
      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 border border-gray-200 dark:border-gray-700">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          🎯 Smart Tool Routing
        </h3>
        <div className="space-y-4">
          <div className="flex space-x-2">
            <button
              onClick={() => handleSmartToolSearch('search for AI news')}
              className="px-3 py-1 text-sm bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded hover:bg-gray-200 dark:hover:bg-gray-600"
            >
              "search for AI news"
            </button>
            <button
              onClick={() => handleSmartToolSearch('research quantum computing')}
              className="px-3 py-1 text-sm bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded hover:bg-gray-200 dark:hover:bg-gray-600"
            >
              "research quantum computing"
            </button>
            <button
              onClick={() => handleSmartToolSearch('find information about climate change')}
              className="px-3 py-1 text-sm bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded hover:bg-gray-200 dark:hover:bg-gray-600"
            >
              "find information about climate change"
            </button>
          </div>
        </div>
      </div>

      {/* Tool Execution */}
      {selectedTool && (
        <div className="bg-white dark:bg-gray-800 rounded-lg p-6 border border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
            <PlayIcon className="w-5 h-5 mr-2" />
            Execute {selectedTool.name}
          </h3>
          
          <div className="space-y-4">
            {selectedTool.id === 'web-search' && (
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Search Query
                </label>
                <input
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                  placeholder="Enter your search query..."
                />
                <div className="mt-2 flex space-x-2">
                  <button
                    onClick={() => handleExecuteTool('web-search', 'search', { 
                      query: searchQuery, 
                      engine: 'duckduckgo' 
                    })}
                    disabled={isExecuting}
                    className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 font-medium"
                  >
                    {isExecuting ? 'Searching...' : 'Search (Free)'}
                  </button>
                  <button
                    onClick={() => handleExecuteTool('web-search', 'search', { 
                      query: searchQuery, 
                      engine: 'google' 
                    })}
                    disabled={isExecuting}
                    className="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50 font-medium"
                  >
                    {isExecuting ? 'Searching...' : 'Search with Google (Premium)'}
                  </button>
                </div>
              </div>
            )}
            
            {selectedTool.id === 'research-assistant' && (
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Research Topic
                </label>
                <input
                  type="text"
                  value={researchTopic}
                  onChange={(e) => setResearchTopic(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                  placeholder="Enter research topic..."
                />
                <button
                  onClick={() => handleExecuteTool('research-assistant', 'comprehensive-research', { 
                    topic: researchTopic,
                    depth: 'detailed'
                  })}
                  disabled={isExecuting}
                  className="mt-2 px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 disabled:opacity-50 font-medium"
                >
                  {isExecuting ? 'Researching...' : 'Start Research'}
                </button>
              </div>
            )}
            
            {/* API Key Requirements */}
            {selectedTool.apiRequirements && (
              <div className="mt-4 p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
                <h4 className="font-medium text-yellow-800 dark:text-yellow-200 mb-2">
                  💎 Premium Features Available
                </h4>
                <p className="text-sm text-yellow-700 dark:text-yellow-300 mb-3">
                  This tool supports premium features with API keys. Add your keys for enhanced capabilities:
                </p>
                <div className="space-y-2">
                  {Object.entries(selectedTool.apiRequirements).map(([serviceId, req]) => (
                    <div key={serviceId} className="flex items-center justify-between">
                      <span className="text-sm text-yellow-700 dark:text-yellow-300">
                        {req.keyName}
                      </span>
                      <button
                        onClick={() => {
                          setSelectedService(serviceId);
                          setShowKeySetupModal(true);
                        }}
                        className="text-sm px-3 py-1 bg-yellow-600 text-white rounded hover:bg-yellow-700"
                      >
                        Setup
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Execution Results */}
      {executionResult && (
        <div className="bg-white dark:bg-gray-800 rounded-lg p-6 border border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
            <ChartBarIcon className="w-5 h-5 mr-2" />
            Execution Results
          </h3>
          
          <div className={`p-4 rounded-lg ${
            executionResult.success 
              ? 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800'
              : 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800'
          }`}>
            {executionResult.success ? (
              <div>
                <div className="flex items-center mb-3">
                  <span className="text-green-600 dark:text-green-400 text-lg mr-2">✅</span>
                  <span className="font-medium text-green-800 dark:text-green-200">
                    Execution Successful
                  </span>
                </div>
                <pre className="text-sm text-green-700 dark:text-green-300 overflow-x-auto">
                  {JSON.stringify(executionResult.data, null, 2)}
                </pre>
              </div>
            ) : (
              <div>
                <div className="flex items-center mb-3">
                  <span className="text-red-600 dark:text-red-400 text-lg mr-2">❌</span>
                  <span className="font-medium text-red-800 dark:text-red-200">
                    {executionResult.error?.code}: {executionResult.error?.message}
                  </span>
                </div>
                {executionResult.error?.code === 'API_KEY_REQUIRED' && (
                  <p className="text-sm text-red-700 dark:text-red-300">
                    💡 Add the required API keys to unlock premium features!
                  </p>
                )}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Modals */}
      <VaultUnlockModal
        isOpen={showVaultModal}
        onClose={() => setShowVaultModal(false)}
        onUnlock={handleUnlockVault}
      />
      
      <APIKeySetupModal
        isOpen={showKeySetupModal}
        onClose={() => setShowKeySetupModal(false)}
        serviceId={selectedService}
        onKeyAdded={() => {
          setShowKeySetupModal(false);
          // Could retry the tool execution here
        }}
        vaultService={vaultService}
      />
    </div>
  );
}