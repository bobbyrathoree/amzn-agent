import React, { useState } from 'react';
import type { KnowledgeBaseChunk } from '../types';

interface SourceCitationsProps {
  sources: KnowledgeBaseChunk[];
  className?: string;
}

// 🚀 INGENIOUS ENHANCEMENT: Interactive Source Citations with Preview
// Surpasses bedrock-chat with expandable source content and metadata display
export const SourceCitations: React.FC<SourceCitationsProps> = ({ 
  sources, 
  className = '' 
}) => {
  const [expandedSources, setExpandedSources] = useState<Set<number>>(new Set());

  if (!sources || sources.length === 0) {
    return null;
  }

  const toggleSource = (index: number) => {
    const newExpanded = new Set(expandedSources);
    if (newExpanded.has(index)) {
      newExpanded.delete(index);
    } else {
      newExpanded.add(index);
    }
    setExpandedSources(newExpanded);
  };

  const getScoreColor = (score: number) => {
    if (score >= 0.8) return 'text-green-600 bg-green-100';
    if (score >= 0.6) return 'text-yellow-600 bg-yellow-100';
    return 'text-gray-600 bg-gray-100';
  };

  const formatSource = (source: string) => {
    // Extract filename from S3 URI or URL
    if (source.includes('/')) {
      const parts = source.split('/');
      return parts[parts.length - 1];
    }
    return source;
  };

  const getSourceIcon = (source: string) => {
    const lowerSource = source.toLowerCase();
    
    if (lowerSource.includes('.pdf')) {
      return (
        <svg className="w-4 h-4 text-red-600" fill="currentColor" viewBox="0 0 20 20">
          <path fillRule="evenodd" d="M4 4a2 2 0 00-2 2v8a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2H4zm2 6a1 1 0 011-1h6a1 1 0 110 2H7a1 1 0 01-1-1zm1 3a1 1 0 100 2h6a1 1 0 100-2H7z" clipRule="evenodd" />
        </svg>
      );
    }
    
    if (lowerSource.includes('.doc') || lowerSource.includes('.docx')) {
      return (
        <svg className="w-4 h-4 text-blue-600" fill="currentColor" viewBox="0 0 20 20">
          <path fillRule="evenodd" d="M4 4a2 2 0 00-2 2v8a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2H4zm2 6a1 1 0 011-1h6a1 1 0 110 2H7a1 1 0 01-1-1zm1 3a1 1 0 100 2h6a1 1 0 100-2H7z" clipRule="evenodd" />
        </svg>
      );
    }
    
    return (
      <svg className="w-4 h-4 text-gray-600" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M4 4a2 2 0 00-2 2v8a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2H4zm2 6a1 1 0 011-1h6a1 1 0 110 2H7a1 1 0 01-1-1zm1 3a1 1 0 100 2h6a1 1 0 100-2H7z" clipRule="evenodd" />
      </svg>
    );
  };

  return (
    <div className={`bg-blue-50 border border-blue-200 rounded-lg p-4 ${className}`}>
      <div className="flex items-center mb-3">
        <svg className="w-4 h-4 text-blue-600 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.746 0 3.332.477 4.5 1.253v13C19.832 18.477 18.246 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
        </svg>
        <h3 className="text-sm font-medium text-blue-900">
          Sources ({sources.length})
        </h3>
      </div>

      <div className="space-y-2">
        {sources.map((source, index) => (
          <div key={index} className="bg-white rounded-md border border-blue-200 overflow-hidden">
            <button
              onClick={() => toggleSource(index)}
              className="w-full px-3 py-2 text-left hover:bg-blue-50 transition-colors"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2 min-w-0 flex-1">
                  {getSourceIcon(source.source)}
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center space-x-2">
                      <span className="text-sm font-medium text-gray-900 truncate">
                        [{index + 1}] {formatSource(source.source)}
                      </span>
                      <span className={`px-2 py-0.5 text-xs rounded-full ${getScoreColor(source.score)}`}>
                        {(source.score * 100).toFixed(0)}%
                      </span>
                    </div>
                    {source.metadata?.page_number && (
                      <div className="text-xs text-gray-500">
                        Page {source.metadata.page_number}
                      </div>
                    )}
                  </div>
                </div>
                
                <div className="flex items-center space-x-2">
                  <svg 
                    className={`w-4 h-4 text-gray-500 transition-transform ${
                      expandedSources.has(index) ? 'rotate-180' : ''
                    }`} 
                    fill="none" 
                    stroke="currentColor" 
                    viewBox="0 0 24 24"
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                  </svg>
                </div>
              </div>
            </button>
            
            {expandedSources.has(index) && (
              <div className="px-3 pb-3 border-t border-blue-100">
                <div className="mt-2">
                  <div className="text-xs text-gray-700 leading-relaxed bg-gray-50 p-2 rounded">
                    {source.content.length > 300 ? (
                      <>
                        {source.content.substring(0, 300)}...
                        <button 
                          className="text-blue-600 hover:text-blue-800 ml-1"
                          onClick={(e) => {
                            e.stopPropagation();
                            // Could implement full content modal here
                          }}
                        >
                          [Show more]
                        </button>
                      </>
                    ) : (
                      source.content
                    )}
                  </div>
                  
                  {source.metadata && Object.keys(source.metadata).length > 0 && (
                    <div className="mt-2 pt-2 border-t border-gray-200">
                      <div className="text-xs text-gray-500">
                        <div className="grid grid-cols-2 gap-2">
                          {source.metadata.sourceLink && (
                            <div>
                              <span className="font-medium">Link:</span>{' '}
                              <a 
                                href={source.metadata.sourceLink} 
                                target="_blank" 
                                rel="noopener noreferrer"
                                className="text-blue-600 hover:text-blue-800 truncate"
                                onClick={(e) => e.stopPropagation()}
                              >
                                View Document
                              </a>
                            </div>
                          )}
                          
                          {source.metadata.rank && (
                            <div>
                              <span className="font-medium">Rank:</span> {source.metadata.rank}
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        ))}
      </div>

      <div className="mt-3 pt-3 border-t border-blue-200">
        <p className="text-xs text-blue-700">
          💡 Click on sources to view content excerpts and metadata
        </p>
      </div>
    </div>
  );
};