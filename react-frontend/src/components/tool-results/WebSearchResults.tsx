// 🔍 WEB SEARCH RESULTS VISUALIZATION
// Beautiful, interactive display of web search results

import { useState } from 'react';
import { 
  LinkIcon, 
  CalendarIcon, 
  StarIcon,
  ChevronDownIcon,
  ChevronUpIcon,
  ArrowTopRightOnSquareIcon,
  ClockIcon,
  KeyIcon
} from '@heroicons/react/24/outline';

interface SearchResult {
  title: string;
  url: string;
  snippet: string;
  relevanceScore?: number;
  publishedDate?: string;
  author?: string;
  domain?: string;
  searchQuery?: string;
  searchEngine?: string;
  retrievedAt?: string;
}

interface WebSearchResultsProps {
  results: SearchResult[];
  query: string;
  engine: string;
  totalResults: number;
  searchTime: number;
  usedApiKey: boolean;
  fallbackUsed?: boolean;
  requestedEngine?: string;
  className?: string;
}

export function WebSearchResults({
  results,
  query,
  engine,
  totalResults,
  searchTime,
  usedApiKey,
  fallbackUsed = false,
  requestedEngine,
  className = ''
}: WebSearchResultsProps) {
  const [isExpanded, setIsExpanded] = useState(true);

  const getEngineInfo = () => {
    switch (engine) {
      case 'google':
        return { name: 'Google', icon: '🔍', color: 'text-blue-600', premium: true };
      case 'tavily':
        return { name: 'Tavily', icon: '🔎', color: 'text-green-600', premium: true };
      case 'serpapi':
        return { name: 'SerpAPI', icon: '🐍', color: 'text-purple-600', premium: true };
      case 'duckduckgo':
      default:
        return { name: 'DuckDuckGo', icon: '🦆', color: 'text-orange-600', premium: false };
    }
  };

  const engineInfo = getEngineInfo();

  return (
    <div className={`bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden ${className}`}>
      {/* Header */}
      <div className="bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-900/20 dark:to-indigo-900/20 p-4 border-b border-gray-200 dark:border-gray-700">
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="w-full flex items-center justify-between text-left"
        >
          <div className="flex items-center space-x-3">
            <span className="text-2xl">{engineInfo.icon}</span>
            <div>
              <h3 className="font-semibold text-gray-900 dark:text-white">
                🔍 Web Search Results
              </h3>
              <div className="flex items-center space-x-2 mt-1">
                <span className="text-sm text-gray-600 dark:text-gray-300">
                  "{query}" via {engineInfo.name}
                </span>
                {usedApiKey && (
                  <span className="inline-flex items-center space-x-1 px-2 py-1 bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 text-xs rounded-full">
                    <KeyIcon className="w-3 h-3" />
                    <span>Premium</span>
                  </span>
                )}
                {fallbackUsed && requestedEngine && (
                  <span className="inline-flex items-center px-2 py-1 bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 text-xs rounded-full">
                    Fallback from {requestedEngine}
                  </span>
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <div className="text-right text-xs text-gray-500 dark:text-gray-400">
              <div>{totalResults} results</div>
              <div className="flex items-center space-x-1">
                <ClockIcon className="w-3 h-3" />
                <span>{searchTime}ms</span>
              </div>
            </div>
            {isExpanded ? (
              <ChevronUpIcon className="w-5 h-5 text-gray-400" />
            ) : (
              <ChevronDownIcon className="w-5 h-5 text-gray-400" />
            )}
          </div>
        </button>
      </div>

      {/* Results */}
      {isExpanded && (
        <div className="divide-y divide-gray-200 dark:divide-gray-700">
          {results.map((result, index) => (
            <div key={index} className="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  {/* Title and URL */}
                  <div className="flex items-start space-x-2 mb-2">
                    <div className="flex-1 min-w-0">
                      <a
                        href={result.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="group flex items-start space-x-2 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
                      >
                        <h4 className="font-medium text-gray-900 dark:text-white line-clamp-2 group-hover:underline">
                          {result.title}
                        </h4>
                        <ArrowTopRightOnSquareIcon className="w-4 h-4 text-gray-400 group-hover:text-blue-500 flex-shrink-0 mt-0.5 opacity-0 group-hover:opacity-100 transition-opacity" />
                      </a>
                      <div className="flex items-center space-x-2 mt-1">
                        <LinkIcon className="w-3 h-3 text-gray-400 flex-shrink-0" />
                        <span className="text-xs text-gray-500 dark:text-gray-400 truncate">
                          {result.domain || new URL(result.url).hostname}
                        </span>
                        {result.publishedDate && (
                          <>
                            <span className="text-gray-300">•</span>
                            <div className="flex items-center space-x-1">
                              <CalendarIcon className="w-3 h-3 text-gray-400" />
                              <span className="text-xs text-gray-500 dark:text-gray-400">
                                {new Date(result.publishedDate).toLocaleDateString()}
                              </span>
                            </div>
                          </>
                        )}
                      </div>
                    </div>
                    {result.relevanceScore && (
                      <div className="flex items-center space-x-1 flex-shrink-0">
                        <StarIcon className="w-4 h-4 text-yellow-500" />
                        <span className="text-xs font-medium text-gray-600 dark:text-gray-300">
                          {Math.round(result.relevanceScore * 100)}%
                        </span>
                      </div>
                    )}
                  </div>

                  {/* Snippet */}
                  <p className="text-sm text-gray-600 dark:text-gray-300 line-clamp-3 leading-relaxed">
                    {result.snippet}
                  </p>

                  {/* Additional Info */}
                  {result.author && (
                    <div className="mt-2 text-xs text-gray-500 dark:text-gray-400">
                      By {result.author}
                    </div>
                  )}
                </div>
              </div>
            </div>
          ))}

          {/* Footer */}
          <div className="p-3 bg-gray-50 dark:bg-gray-800 text-center">
            <div className="text-xs text-gray-500 dark:text-gray-400">
              Powered by {engineInfo.name} • Retrieved {new Date().toLocaleTimeString()}
              {!usedApiKey && engineInfo.premium && (
                <span className="ml-2 text-blue-600 dark:text-blue-400">
                  • Add API key for premium results
                </span>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}