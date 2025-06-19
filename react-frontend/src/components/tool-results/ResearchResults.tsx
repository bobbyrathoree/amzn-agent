// 🔬 RESEARCH RESULTS VISUALIZATION
// Comprehensive display of research findings with analysis

import { useState } from 'react';
import { 
  DocumentTextIcon,
  AcademicCapIcon,
  NewspaperIcon,
  ChartBarIcon,
  LightBulbIcon,
  ArrowTrendingUpIcon,
  ExclamationTriangleIcon,
  CheckCircleIcon,
  ChevronDownIcon,
  ChevronUpIcon,
  LinkIcon,
  CalendarIcon,
  StarIcon,
  KeyIcon
} from '@heroicons/react/24/outline';

interface ResearchSource {
  type: 'web' | 'academic' | 'news' | 'report' | 'industry';
  title: string;
  url: string;
  author?: string;
  publishedDate?: string;
  summary: string;
  credibilityScore?: number;
  relevanceScore?: number;
  citations?: number;
}

interface ResearchAnalysis {
  keyFindings: string[];
  trends: string[];
  gaps: string[];
  recommendations: string[];
  confidence: number;
  methodology?: string;
  limitations?: string[];
}

interface ResearchResultsProps {
  topic: string;
  sources: ResearchSource[];
  analysis: ResearchAnalysis;
  executionTime: number;
  usedPremiumSources: boolean;
  depth: 'surface' | 'detailed' | 'comprehensive';
  className?: string;
}

export function ResearchResults({
  topic,
  sources,
  analysis,
  executionTime,
  usedPremiumSources,
  depth,
  className = ''
}: ResearchResultsProps) {
  const [activeTab, setActiveTab] = useState<'sources' | 'analysis' | 'findings'>('analysis');
  const [isExpanded, setIsExpanded] = useState(true);

  const getSourceIcon = (type: string) => {
    switch (type) {
      case 'academic': return <AcademicCapIcon className="w-4 h-4" />;
      case 'news': return <NewspaperIcon className="w-4 h-4" />;
      case 'report': return <ChartBarIcon className="w-4 h-4" />;
      case 'industry': return <DocumentTextIcon className="w-4 h-4" />;
      default: return <DocumentTextIcon className="w-4 h-4" />;
    }
  };

  const getSourceColor = (type: string) => {
    switch (type) {
      case 'academic': return 'text-purple-600';
      case 'news': return 'text-blue-600';
      case 'report': return 'text-green-600';
      case 'industry': return 'text-indigo-600';
      default: return 'text-gray-600';
    }
  };

  const getDepthInfo = () => {
    switch (depth) {
      case 'comprehensive': return { label: 'Comprehensive', color: 'text-green-600', bg: 'bg-green-100' };
      case 'detailed': return { label: 'Detailed', color: 'text-blue-600', bg: 'bg-blue-100' };
      case 'surface': return { label: 'Surface', color: 'text-yellow-600', bg: 'bg-yellow-100' };
    }
  };

  const depthInfo = getDepthInfo();

  return (
    <div className={`bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden ${className}`}>
      {/* Header */}
      <div className="bg-gradient-to-r from-purple-50 to-blue-50 dark:from-purple-900/20 dark:to-blue-900/20 p-4 border-b border-gray-200 dark:border-gray-700">
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="w-full flex items-center justify-between text-left"
        >
          <div className="flex items-center space-x-3">
            <span className="text-2xl">🔬</span>
            <div>
              <h3 className="font-semibold text-gray-900 dark:text-white">
                Research Analysis: {topic}
              </h3>
              <div className="flex items-center space-x-2 mt-1">
                <span className={`px-2 py-1 ${depthInfo.bg} ${depthInfo.color} text-xs rounded-full`}>
                  {depthInfo.label} Research
                </span>
                {usedPremiumSources && (
                  <span className="inline-flex items-center space-x-1 px-2 py-1 bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 text-xs rounded-full">
                    <KeyIcon className="w-3 h-3" />
                    <span>Premium Sources</span>
                  </span>
                )}
                <span className="text-xs text-gray-500">
                  {sources.length} sources • {Math.round(analysis.confidence * 100)}% confidence
                </span>
              </div>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <div className="text-right text-xs text-gray-500 dark:text-gray-400">
              <div>{Math.round(executionTime / 1000)}s</div>
            </div>
            {isExpanded ? (
              <ChevronUpIcon className="w-5 h-5 text-gray-400" />
            ) : (
              <ChevronDownIcon className="w-5 h-5 text-gray-400" />
            )}
          </div>
        </button>
      </div>

      {/* Content */}
      {isExpanded && (
        <div>
          {/* Tabs */}
          <div className="border-b border-gray-200 dark:border-gray-700">
            <nav className="flex space-x-6 px-4">
              {[
                { id: 'analysis', label: 'Analysis', icon: LightBulbIcon },
                { id: 'findings', label: 'Key Findings', icon: CheckCircleIcon },
                { id: 'sources', label: 'Sources', icon: DocumentTextIcon }
              ].map(tab => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id as any)}
                  className={`flex items-center space-x-2 py-3 border-b-2 text-sm font-medium transition-colors ${
                    activeTab === tab.id
                      ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                      : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
                  }`}
                >
                  <tab.icon className="w-4 h-4" />
                  <span>{tab.label}</span>
                </button>
              ))}
            </nav>
          </div>

          {/* Tab Content */}
          <div className="p-4">
            {activeTab === 'analysis' && (
              <div className="space-y-6">
                {/* Key Findings */}
                {analysis.keyFindings.length > 0 && (
                  <div>
                    <h4 className="flex items-center space-x-2 font-medium text-gray-900 dark:text-white mb-3">
                      <CheckCircleIcon className="w-5 h-5 text-green-600" />
                      <span>Key Findings</span>
                    </h4>
                    <ul className="space-y-2">
                      {analysis.keyFindings.map((finding, index) => (
                        <li key={index} className="flex items-start space-x-2">
                          <span className="text-green-500 font-bold mt-1">•</span>
                          <span className="text-sm text-gray-700 dark:text-gray-300">{finding}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {/* Trends */}
                {analysis.trends.length > 0 && (
                  <div>
                    <h4 className="flex items-center space-x-2 font-medium text-gray-900 dark:text-white mb-3">
                      <ArrowTrendingUpIcon className="w-5 h-5 text-blue-600" />
                      <span>Trends & Patterns</span>
                    </h4>
                    <ul className="space-y-2">
                      {analysis.trends.map((trend, index) => (
                        <li key={index} className="flex items-start space-x-2">
                          <span className="text-blue-500 font-bold mt-1">•</span>
                          <span className="text-sm text-gray-700 dark:text-gray-300">{trend}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {/* Recommendations */}
                {analysis.recommendations.length > 0 && (
                  <div>
                    <h4 className="flex items-center space-x-2 font-medium text-gray-900 dark:text-white mb-3">
                      <LightBulbIcon className="w-5 h-5 text-yellow-600" />
                      <span>Recommendations</span>
                    </h4>
                    <ul className="space-y-2">
                      {analysis.recommendations.map((rec, index) => (
                        <li key={index} className="flex items-start space-x-2">
                          <span className="text-yellow-500 font-bold mt-1">•</span>
                          <span className="text-sm text-gray-700 dark:text-gray-300">{rec}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {/* Gaps */}
                {analysis.gaps.length > 0 && (
                  <div>
                    <h4 className="flex items-center space-x-2 font-medium text-gray-900 dark:text-white mb-3">
                      <ExclamationTriangleIcon className="w-5 h-5 text-orange-600" />
                      <span>Research Gaps</span>
                    </h4>
                    <ul className="space-y-2">
                      {analysis.gaps.map((gap, index) => (
                        <li key={index} className="flex items-start space-x-2">
                          <span className="text-orange-500 font-bold mt-1">•</span>
                          <span className="text-sm text-gray-700 dark:text-gray-300">{gap}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            )}

            {activeTab === 'findings' && (
              <div className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-lg">
                    <h5 className="font-medium text-blue-900 dark:text-blue-100 mb-2">Confidence Level</h5>
                    <div className="flex items-center space-x-2">
                      <div className="flex-1 bg-blue-200 dark:bg-blue-800 rounded-full h-2">
                        <div 
                          className="bg-blue-600 h-2 rounded-full transition-all duration-500"
                          style={{ width: `${analysis.confidence * 100}%` }}
                        />
                      </div>
                      <span className="text-sm font-medium text-blue-900 dark:text-blue-100">
                        {Math.round(analysis.confidence * 100)}%
                      </span>
                    </div>
                  </div>
                  <div className="bg-green-50 dark:bg-green-900/20 p-4 rounded-lg">
                    <h5 className="font-medium text-green-900 dark:text-green-100 mb-2">Source Quality</h5>
                    <div className="flex items-center space-x-2">
                      <StarIcon className="w-4 h-4 text-yellow-500" />
                      <span className="text-sm text-green-800 dark:text-green-200">
                        {usedPremiumSources ? 'Premium Sources' : 'Standard Sources'}
                      </span>
                    </div>
                  </div>
                </div>
                
                <div className="space-y-3">
                  <h5 className="font-medium text-gray-900 dark:text-white">Summary of Findings:</h5>
                  {analysis.keyFindings.map((finding, index) => (
                    <div key={index} className="p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                      <p className="text-sm text-gray-700 dark:text-gray-300">{finding}</p>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {activeTab === 'sources' && (
              <div className="space-y-4">
                {sources.map((source, index) => (
                  <div key={index} className="border border-gray-200 dark:border-gray-600 rounded-lg p-4">
                    <div className="flex items-start justify-between mb-2">
                      <div className="flex items-center space-x-2">
                        <span className={getSourceColor(source.type)}>
                          {getSourceIcon(source.type)}
                        </span>
                        <span className={`text-xs px-2 py-1 rounded-full ${
                          source.type === 'academic' ? 'bg-purple-100 text-purple-800' :
                          source.type === 'news' ? 'bg-blue-100 text-blue-800' :
                          source.type === 'report' ? 'bg-green-100 text-green-800' :
                          'bg-gray-100 text-gray-800'
                        }`}>
                          {source.type}
                        </span>
                      </div>
                      <div className="flex items-center space-x-2">
                        {source.credibilityScore && (
                          <div className="flex items-center space-x-1">
                            <StarIcon className="w-3 h-3 text-yellow-500" />
                            <span className="text-xs text-gray-500">
                              {Math.round(source.credibilityScore * 100)}%
                            </span>
                          </div>
                        )}
                        {source.citations && (
                          <span className="text-xs text-gray-500">
                            {source.citations} citations
                          </span>
                        )}
                      </div>
                    </div>
                    
                    <h5 className="font-medium text-gray-900 dark:text-white mb-2">
                      <a 
                        href={source.url} 
                        target="_blank" 
                        rel="noopener noreferrer"
                        className="hover:text-blue-600 dark:hover:text-blue-400"
                      >
                        {source.title}
                      </a>
                    </h5>
                    
                    <p className="text-sm text-gray-600 dark:text-gray-300 mb-2">
                      {source.summary}
                    </p>
                    
                    <div className="flex items-center space-x-4 text-xs text-gray-500 dark:text-gray-400">
                      {source.author && (
                        <span>By {source.author}</span>
                      )}
                      {source.publishedDate && (
                        <div className="flex items-center space-x-1">
                          <CalendarIcon className="w-3 h-3" />
                          <span>{new Date(source.publishedDate).toLocaleDateString()}</span>
                        </div>
                      )}
                      <div className="flex items-center space-x-1">
                        <LinkIcon className="w-3 h-3" />
                        <span>{new URL(source.url).hostname}</span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}