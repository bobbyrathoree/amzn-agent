// 🛠️ BOT TOOLS SELECTOR
// Tool selection component for bot creation/editing

import { useState, useEffect } from 'react';
import { getAllTools } from '../tools';
import type { UniversalTool, ToolCategory } from '../types/tools';
import type { AgentTool } from '../types';

interface BotToolsSelectorProps {
  selectedTools: AgentTool[];
  onToolsChange: (tools: AgentTool[]) => void;
  className?: string;
}

export function BotToolsSelector({ selectedTools, onToolsChange, className = '' }: BotToolsSelectorProps) {
  const [availableTools, setAvailableTools] = useState<UniversalTool[]>([]);
  const [selectedCategory, setSelectedCategory] = useState<ToolCategory | 'all'>('all');

  useEffect(() => {
    // Initialize available tools
    const tools = getAllTools();
    setAvailableTools(tools);
  }, []);

  const handleToolToggle = (tool: UniversalTool) => {
    const isSelected = selectedTools.some(t => t.name === tool.id);
    
    if (isSelected) {
      // Remove tool
      onToolsChange(selectedTools.filter(t => t.name !== tool.id));
    } else {
      // Add tool
      const newTool: AgentTool = {
        type: 'plain', // Our new tools are 'plain' type
        name: tool.id,
        description: tool.description,
        config: {
          toolId: tool.id,
          capabilities: tool.capabilities.map(cap => cap.name),
          apiRequirements: tool.apiRequirements || {}
        }
      };
      onToolsChange([...selectedTools, newTool]);
    }
  };

  const filteredTools = availableTools.filter(tool => 
    selectedCategory === 'all' || tool.category === selectedCategory
  );

  const categories = Array.from(new Set(availableTools.map(tool => tool.category)));

  return (
    <div className={`space-y-4 ${className}`}>
      <div>
        <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
          🛠️ Select Bot Tools
        </h3>
        <p className="text-sm text-gray-600 dark:text-gray-300">
          Choose which tools this bot can use during conversations
        </p>
      </div>

      {/* Category Filter */}
      <div className="flex flex-wrap gap-2">
        <button
          onClick={() => setSelectedCategory('all')}
          className={`px-3 py-1 text-sm rounded-full transition-colors ${
            selectedCategory === 'all'
              ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
              : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
          }`}
        >
          All ({availableTools.length})
        </button>
        {categories.map(category => (
          <button
            key={category}
            onClick={() => setSelectedCategory(category)}
            className={`px-3 py-1 text-sm rounded-full transition-colors ${
              selectedCategory === category
                ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
                : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
            }`}
          >
            {category} ({availableTools.filter(t => t.category === category).length})
          </button>
        ))}
      </div>

      {/* Tools Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        {filteredTools.map(tool => {
          const isSelected = selectedTools.some(t => t.name === tool.id);
          
          return (
            <div
              key={tool.id}
              className={`p-4 border rounded-lg cursor-pointer transition-all ${
                isSelected
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                  : 'border-gray-200 dark:border-gray-600 hover:border-gray-300 dark:hover:border-gray-500'
              }`}
              onClick={() => handleToolToggle(tool)}
            >
              <div className="flex items-start space-x-3">
                <input
                  type="checkbox"
                  checked={isSelected}
                  onChange={() => handleToolToggle(tool)}
                  className="mt-1 w-4 h-4 text-blue-600 rounded focus:ring-blue-500"
                />
                <div className="flex-1">
                  <div className="flex items-center space-x-2 mb-2">
                    <span className="text-xl">{tool.icon}</span>
                    <h4 className="font-medium text-gray-900 dark:text-white">
                      {tool.name}
                    </h4>
                    {tool.apiRequirements && (
                      <span className="text-xs px-2 py-1 bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 rounded">
                        Premium
                      </span>
                    )}
                  </div>
                  <p className="text-sm text-gray-600 dark:text-gray-300 mb-2">
                    {tool.description}
                  </p>
                  <div className="flex items-center space-x-2 text-xs text-gray-500">
                    <span>{tool.capabilities.length} capabilities</span>
                    <span>•</span>
                    <span>{tool.category}</span>
                    {tool.config?.rateLimit && (
                      <>
                        <span>•</span>
                        <span>{tool.config.rateLimit.requests}/min</span>
                      </>
                    )}
                  </div>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {/* Selected Tools Summary */}
      {selectedTools.length > 0 && (
        <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
          <h4 className="font-medium text-gray-900 dark:text-white mb-2">
            Selected Tools ({selectedTools.length})
          </h4>
          <div className="flex flex-wrap gap-2">
            {selectedTools.map(tool => {
              const toolInfo = availableTools.find(t => t.id === tool.name);
              return (
                <span
                  key={tool.name}
                  className="inline-flex items-center space-x-1 px-3 py-1 bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 rounded-full text-sm"
                >
                  <span>{toolInfo?.icon}</span>
                  <span>{toolInfo?.name}</span>
                </span>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}