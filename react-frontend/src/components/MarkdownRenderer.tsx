import React, { useState, useEffect, useRef } from 'react';
import ReactMarkdown from 'react-markdown';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneDark, oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism';
import remarkGfm from 'remark-gfm';
import remarkBreaks from 'remark-breaks';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import rehypeExternalLinks from 'rehype-external-links';
import mermaid from 'mermaid';
import { EyeIcon, CodeBracketIcon } from '@heroicons/react/24/outline';

// Import KaTeX CSS
import 'katex/dist/katex.min.css';

interface MarkdownRendererProps {
  content: string;
  className?: string;
  isDark?: boolean;
}

interface MermaidProps {
  chart: string;
  isDark?: boolean;
}

// Mermaid diagram component
const MermaidDiagram: React.FC<MermaidProps> = ({ chart, isDark }) => {
  const elementRef = useRef<HTMLDivElement>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const renderMermaid = async () => {
      if (!elementRef.current) return;

      try {
        // Configure mermaid
        mermaid.initialize({
          startOnLoad: false,
          theme: isDark ? 'dark' : 'default',
          themeVariables: {
            primaryColor: isDark ? '#3b82f6' : '#2563eb',
            primaryTextColor: isDark ? '#ffffff' : '#000000',
            primaryBorderColor: isDark ? '#374151' : '#d1d5db',
            lineColor: isDark ? '#6b7280' : '#374151',
            secondaryColor: isDark ? '#374151' : '#f3f4f6',
            tertiaryColor: isDark ? '#1f2937' : '#ffffff',
          },
        });

        // Generate unique ID
        const id = `mermaid-${Math.random().toString(36).substr(2, 9)}`;
        
        // Render the diagram
        const { svg } = await mermaid.render(id, chart);
        
        if (elementRef.current) {
          elementRef.current.innerHTML = svg;
        }
        
        setError(null);
      } catch (err) {
        console.error('Mermaid rendering error:', err);
        setError('Failed to render diagram');
      }
    };

    renderMermaid();
  }, [chart, isDark]);

  if (error) {
    return (
      <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
        <p className="text-red-700 dark:text-red-300 text-sm">{error}</p>
        <pre className="text-xs text-red-600 dark:text-red-400 mt-2 overflow-x-auto">{chart}</pre>
      </div>
    );
  }

  return <div ref={elementRef} className="mermaid-diagram flex justify-center" />;
};

export const MarkdownRenderer: React.FC<MarkdownRendererProps> = ({ 
  content, 
  className = '', 
  isDark = false 
}) => {
  // Detect if content has markdown syntax
  const hasMarkdownSyntax = (text: string): boolean => {
    const markdownPatterns = [
      /^#{1,6}\s/m,           // Headers
      /\*\*.*?\*\*/,          // Bold
      /\*.*?\*/,              // Italic
      /`.*?`/,                // Inline code
      /```[\s\S]*?```/,       // Code blocks
      /^\s*[-*+]\s/m,         // Lists
      /^\s*\d+\.\s/m,         // Numbered lists
      /\[.*?\]\(.*?\)/,       // Links
      /!\[.*?\]\(.*?\)/,      // Images
      /^\s*>/m,               // Blockquotes
      /\|.*?\|/,              // Tables
      /^```mermaid[\s\S]*?```/m, // Mermaid diagrams
    ];
    
    return markdownPatterns.some(pattern => pattern.test(text));
  };

  // Default to markdown if syntax detected, otherwise plain
  const shouldShowMarkdown = hasMarkdownSyntax(content);
  const [viewMode, setViewMode] = useState<'markdown' | 'plain'>(
    shouldShowMarkdown ? 'markdown' : 'plain'
  );

  // Toggle between markdown and plain text view
  const toggleViewMode = () => {
    setViewMode(viewMode === 'markdown' ? 'plain' : 'markdown');
  };

  // If no markdown syntax detected, just show plain text
  if (!shouldShowMarkdown) {
    return (
      <div className={`prose prose-sm max-w-none ${isDark ? 'prose-invert' : ''} ${className}`}>
        <p className="whitespace-pre-wrap">{content}</p>
      </div>
    );
  }

  return (
    <div className={className}>
      {/* Toggle button */}
      <div className="flex justify-end mb-2">
        <button
          onClick={toggleViewMode}
          className="flex items-center space-x-1 px-2 py-1 text-xs bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors"
          title={`Switch to ${viewMode === 'markdown' ? 'plain text' : 'markdown'} view`}
        >
          {viewMode === 'markdown' ? (
            <>
              <CodeBracketIcon className="w-3 h-3" />
              <span>Raw</span>
            </>
          ) : (
            <>
              <EyeIcon className="w-3 h-3" />
              <span>Markdown</span>
            </>
          )}
        </button>
      </div>

      {/* Content */}
      {viewMode === 'plain' ? (
        <div className="bg-gray-50 dark:bg-gray-800 p-4 rounded border font-mono text-sm">
          <pre className="whitespace-pre-wrap">{content}</pre>
        </div>
      ) : (
        <div className={`prose prose-sm max-w-none ${isDark ? 'prose-invert' : ''}`}>
          <ReactMarkdown
            remarkPlugins={[remarkGfm, remarkBreaks, remarkMath]}
            rehypePlugins={[
              rehypeKatex,
              [rehypeExternalLinks, { target: '_blank', rel: 'noopener noreferrer' }]
            ]}
            components={{
              // Custom code block renderer with syntax highlighting
              code: ({ node, className, children, ...props }) => {
                const inline = !className;
                const match = /language-(\w+)/.exec(className || '');
                const language = match ? match[1] : '';
                
                // Handle mermaid diagrams
                if (language === 'mermaid') {
                  return <MermaidDiagram chart={String(children).replace(/\n$/, '')} isDark={isDark} />;
                }
                
                // Regular code blocks
                if (!inline && language) {
                  return (
                    <SyntaxHighlighter
                      style={isDark ? oneDark : oneLight}
                      language={language}
                      PreTag="div"
                      className="rounded-md"
                    >
                      {String(children).replace(/\n$/, '')}
                    </SyntaxHighlighter>
                  );
                }
                
                // Inline code
                return (
                  <code className={`${className} bg-gray-100 dark:bg-gray-800 px-1 py-0.5 rounded text-sm`} {...props}>
                    {children}
                  </code>
                );
              },
              
              // Custom table styling
              table: ({ children }) => (
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                    {children}
                  </table>
                </div>
              ),
              
              // Custom blockquote styling
              blockquote: ({ children }) => (
                <blockquote className="border-l-4 border-blue-500 pl-4 py-2 bg-blue-50 dark:bg-blue-900/20">
                  {children}
                </blockquote>
              ),
              
              // Custom heading anchors
              h1: ({ children }) => (
                <h1 className="text-2xl font-bold mb-4 text-gray-900 dark:text-white">
                  {children}
                </h1>
              ),
              h2: ({ children }) => (
                <h2 className="text-xl font-semibold mb-3 text-gray-900 dark:text-white">
                  {children}
                </h2>
              ),
              h3: ({ children }) => (
                <h3 className="text-lg font-medium mb-2 text-gray-900 dark:text-white">
                  {children}
                </h3>
              ),
            }}
          >
            {content}
          </ReactMarkdown>
        </div>
      )}
    </div>
  );
};

export default MarkdownRenderer;