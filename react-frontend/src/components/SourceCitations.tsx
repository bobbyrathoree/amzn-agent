import React, { useState } from 'react';
import type { KnowledgeBaseChunk } from '../types';

interface SourceCitationsProps {
  sources: KnowledgeBaseChunk[];
  className?: string;
  defaultCollapsed?: boolean;
}

export const SourceCitations: React.FC<SourceCitationsProps> = ({ 
  sources, 
  className = '',
  defaultCollapsed = true
}) => {
  const [expandedSources, setExpandedSources] = useState<Set<number>>(new Set());
  const [isCollapsed, setIsCollapsed] = useState(defaultCollapsed);

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
    if (score >= 0.8) return 'text-green-600 bg-green-500/20';
    if (score >= 0.6) return 'text-yellow-600 bg-yellow-500/20';
    return 'text-muted-foreground bg-muted/30';
  };

  const formatSource = (source: string) => {
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
    
    return (
      <svg className="w-4 h-4 text-muted-foreground" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M4 4a2 2 0 00-2 2v8a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2H4zm2 6a1 1 0 011-1h6a1 1 0 110 2H7a1 1 0 01-1-1zm1 3a1 1 0 100 2h6a1 1 0 100-2H7z" clipRule="evenodd" />
      </svg>
    );
  };

  return (
    <div className={`glass-card border border-primary/30 rounded-lg p-4 ${className}`}>
      <button
        onClick={() => setIsCollapsed(!isCollapsed)}
        className="flex items-center w-full text-left hover:bg-primary/10 -m-2 p-2 rounded transition-colors"
      >
        <svg className="w-4 h-4 text-primary mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.746 0 3.332.477 4.5 1.253v13C19.832 18.477 18.246 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
        </svg>
        <h3 className="text-sm font-medium text-primary flex-1">
          Sources ({sources.length})
        </h3>
        <svg 
          className={`w-4 h-4 text-primary transition-transform ${
            isCollapsed ? 'rotate-0' : 'rotate-180'
          }`} 
          fill="none" 
          stroke="currentColor" 
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {!isCollapsed && (
        <div className="space-y-2 mt-3">
          {sources.map((source, index) => (
            <div key={index} className="glass-card rounded-md border border-border/30 overflow-hidden">
              <button
                onClick={() => toggleSource(index)}
                className="w-full px-3 py-2 text-left hover:bg-primary/10 transition-colors"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-2 min-w-0 flex-1">
                    {getSourceIcon(source.source)}
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center space-x-2">
                        <span className="text-sm font-medium text-foreground truncate">
                          [{index + 1}] {formatSource(source.source)}
                        </span>
                        <span className={`px-2 py-0.5 text-xs rounded-full ${getScoreColor(source.score)}`}>
                          {(source.score * 100).toFixed(0)}%
                        </span>
                      </div>
                      {source.metadata?.page_number && (
                        <div className="text-xs text-muted-foreground">
                          Page {source.metadata.page_number}
                        </div>
                      )}
                    </div>
                  </div>
                  
                  <svg 
                    className={`w-4 h-4 text-muted-foreground transition-transform ${
                      expandedSources.has(index) ? 'rotate-180' : ''
                    }`} 
                    fill="none" 
                    stroke="currentColor" 
                    viewBox="0 0 24 24"
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                  </svg>
                </div>
              </button>
              
              {expandedSources.has(index) && (
                <div className="px-3 pb-3 border-t border-border/30">
                  <div className="mt-2">
                    <div className="text-xs text-foreground leading-relaxed bg-muted/30 p-2 rounded">
                      {source.content.length > 300 ? (
                        <>
                          {source.content.substring(0, 300)}...
                          <button 
                            className="text-primary hover:text-primary/80 ml-1"
                            onClick={(e) => e.stopPropagation()}
                          >
                            [Show more]
                          </button>
                        </>
                      ) : (
                        source.content
                      )}
                    </div>
                    
                    {source.metadata && Object.keys(source.metadata).length > 0 && (
                      <div className="mt-2 pt-2 border-t border-border/30">
                        <div className="text-xs text-muted-foreground">
                          {source.metadata.sourceLink && (
                            <div>
                              <span className="font-medium">Link:</span>{' '}
                              <a 
                                href={source.metadata.sourceLink} 
                                target="_blank" 
                                rel="noopener noreferrer"
                                className="text-primary hover:text-primary/80"
                                onClick={(e) => e.stopPropagation()}
                              >
                                View Document
                              </a>
                            </div>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          ))}
          
          <div className="mt-3 pt-3 border-t border-border/30">
            <p className="text-xs text-primary">
              💡 Click on sources to view content excerpts and metadata
            </p>
          </div>
        </div>
      )}
    </div>
  );
};