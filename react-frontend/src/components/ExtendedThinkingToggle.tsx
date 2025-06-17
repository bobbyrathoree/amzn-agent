import React, { useState } from 'react';
import type { ReasoningParams } from '../types';

interface ExtendedThinkingToggleProps {
  enabled: boolean;
  onToggle: (enabled: boolean, params?: ReasoningParams) => void;
  disabled?: boolean;
  className?: string;
}

export const ExtendedThinkingToggle: React.FC<ExtendedThinkingToggleProps> = ({
  enabled,
  onToggle,
  disabled = false,
  className = ''
}) => {
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [budgetTokens, setBudgetTokens] = useState(1024);

  const handleToggle = () => {
    const newEnabled = !enabled;
    const params = newEnabled ? { budgetTokens } : undefined;
    onToggle(newEnabled, params);
  };

  const handleBudgetChange = (newBudget: number) => {
    setBudgetTokens(newBudget);
    if (enabled) {
      onToggle(true, { budgetTokens: newBudget });
    }
  };

  if (disabled) {
    return null;
  }

  return (
    <div className={`space-y-2 ${className}`}>
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <button
            onClick={handleToggle}
            className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${
              enabled ? 'bg-blue-600' : 'bg-gray-200'
            }`}
            role="switch"
            aria-checked={enabled}
          >
            <span
              className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                enabled ? 'translate-x-4' : 'translate-x-0'
              }`}
            />
          </button>
          <div>
            <label className="text-sm font-medium text-gray-700">
              Extended Thinking
            </label>
            <p className="text-xs text-gray-500">
              Let the model think step-by-step before responding
            </p>
          </div>
        </div>
        
        {enabled && (
          <button
            onClick={() => setShowAdvanced(!showAdvanced)}
            className="text-xs text-blue-600 hover:text-blue-800 transition-colors"
          >
            {showAdvanced ? 'Hide' : 'Advanced'}
          </button>
        )}
      </div>

      {enabled && showAdvanced && (
        <div className="bg-gray-50 p-3 rounded-md space-y-3">
          <div>
            <label className="block text-xs font-medium text-gray-700 mb-1">
              Thinking Budget: {budgetTokens.toLocaleString()} tokens
            </label>
            <input
              type="range"
              min={1024}
              max={64000}
              step={1024}
              value={budgetTokens}
              onChange={(e) => handleBudgetChange(parseInt(e.target.value))}
              className="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>1K</span>
              <span>32K</span>
              <span>64K</span>
            </div>
          </div>
          
          <div className="text-xs text-gray-600 space-y-1">
            <p>• Higher budgets allow more detailed reasoning</p>
            <p>• Temperature is automatically set to 1.0 for reasoning</p>
            <p>• Reasoning tokens count towards total usage</p>
          </div>
        </div>
      )}
    </div>
  );
};