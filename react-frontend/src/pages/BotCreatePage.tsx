import { useState, useEffect, useCallback } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../components/AuthProvider';
import { useApiClient } from '../lib/api';
import type { 
  CreateBotRequest, 
  AgentTool,
  PresignedUploadRequest,
  PresignedUploadResponse,
  DocumentInfo
} from '../types';

export function BotCreatePage() {
  const { user, signOut, getAccessToken } = useAuth();
  const getUserId = useCallback(() => user?.userId || user?.username || null, [user?.userId, user?.username]);
  const apiClient = useApiClient(getAccessToken, getUserId);
  const navigate = useNavigate();
  
  // Form state
  const [currentStep, setCurrentStep] = useState(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // Bot creation form data
  const [formData, setFormData] = useState<CreateBotRequest>({
    title: '',
    description: '',
    instruction: '',
    sharedScope: 'private',
    allowedUsers: [],
    allowedGroups: [],
    generationParams: {
      maxTokens: 2048,
      temperature: 0.7,
      topP: 0.9,
      topK: 250,
      stopSequences: []
    },
    knowledgeBaseConfig: {
      searchType: 'HYBRID',
      maxResults: 20,
      scoreThreshold: 0.7
    },
    conversationStarters: [],
    activeModels: ['us.anthropic.claude-3-7-sonnet-20250219-v1:0'],
    agentTools: [],
    displayRetrievedChunks: false
  });

  // Knowledge Base options
  const [knowledgeBaseOption, setKnowledgeBaseOption] = useState<'none' | 'existing' | 'new'>('none');
  const [existingKBs, setExistingKBs] = useState<any[]>([]);
  const [uploadedDocuments, setUploadedDocuments] = useState<DocumentInfo[]>([]);
  const [uploadProgress, setUploadProgress] = useState<Record<string, number>>({});

  // Guardrails state
  const [guardrailsEnabled, setGuardrailsEnabled] = useState(false);
  const [hateThreshold, setHateThreshold] = useState(0);
  const [insultsThreshold, setInsultsThreshold] = useState(0);
  const [sexualThreshold, setSexualThreshold] = useState(0);
  const [violenceThreshold, setViolenceThreshold] = useState(0);
  const [misconductThreshold, setMisconductThreshold] = useState(0);
  const [groundingThreshold, setGroundingThreshold] = useState(0);
  const [relevanceThreshold, setRelevanceThreshold] = useState(0);

  // Available models with proper names and families - matches ChatPage structure
  const availableModels = [
    // Claude Models (US inference profiles)
    { id: 'us.anthropic.claude-opus-4-20250514-v1:0', name: 'Claude 4 Opus', family: 'claude' },
    { id: 'us.anthropic.claude-sonnet-4-20250514-v1:0', name: 'Claude 4 Sonnet', family: 'claude' },
    { id: 'us.anthropic.claude-3-7-sonnet-20250219-v1:0', name: 'Claude 3.7 Sonnet', family: 'claude' },
    { id: 'us.anthropic.claude-3-5-sonnet-20241022-v2:0', name: 'Claude 3.5 Sonnet v2', family: 'claude' },
    { id: 'us.anthropic.claude-3-5-sonnet-20240620-v1:0', name: 'Claude 3.5 Sonnet', family: 'claude' },
    { id: 'us.anthropic.claude-3-5-haiku-20241022-v1:0', name: 'Claude 3.5 Haiku', family: 'claude' },
    { id: 'us.anthropic.claude-3-haiku-20240307-v1:0', name: 'Claude 3 Haiku', family: 'claude' },
    { id: 'us.anthropic.claude-3-opus-20240229-v1:0', name: 'Claude 3 Opus', family: 'claude' },
    
    // Amazon Nova Models
    { id: 'us.amazon.nova-pro-v1:0', name: 'Nova Pro', family: 'nova' },
    { id: 'us.amazon.nova-lite-v1:0', name: 'Nova Lite', family: 'nova' },
    { id: 'us.amazon.nova-micro-v1:0', name: 'Nova Micro', family: 'nova' },
    
    // Mistral Models
    { id: 'us.mistral.mistral-large-2407-v1:0', name: 'Mistral Large 2407', family: 'mistral' },
    { id: 'us.mistral.mistral-large-2402-v1:0', name: 'Mistral Large 2402', family: 'mistral' },
    { id: 'us.mistral.mixtral-8x7b-instruct-v0:1', name: 'Mixtral 8x7B', family: 'mistral' },
    { id: 'us.mistral.mistral-7b-instruct-v0:2', name: 'Mistral 7B', family: 'mistral' },
    
    // DeepSeek Models
    { id: 'us.deepseek.r1-v1:0', name: 'DeepSeek R1', family: 'deepseek' },
    
    // Meta Llama Models
    { id: 'us.meta.llama3-3-70b-instruct-v1:0', name: 'Llama 3.3 70B', family: 'llama' },
    { id: 'us.meta.llama3-2-90b-instruct-v1:0', name: 'Llama 3.2 90B', family: 'llama' },
    { id: 'us.meta.llama3-2-11b-instruct-v1:0', name: 'Llama 3.2 11B', family: 'llama' },
    { id: 'us.meta.llama3-2-3b-instruct-v1:0', name: 'Llama 3.2 3B', family: 'llama' },
    { id: 'us.meta.llama3-2-1b-instruct-v1:0', name: 'Llama 3.2 1B', family: 'llama' }
  ];

  const availableTools = [
    { type: 'plain', name: 'calculator', description: 'Perform mathematical calculations' },
    { type: 'plain', name: 'code_interpreter', description: 'Execute and analyze code' },
    { type: 'internet', name: 'web_search', description: 'Search the internet for information' },
    { type: 'plain', name: 'file_analysis', description: 'Analyze file contents' },
  ];

  useEffect(() => {
    // Load existing Knowledge Bases when component mounts
    loadExistingKnowledgeBases();
  }, []);

  const loadExistingKnowledgeBases = async () => {
    if (!apiClient) return;
    
    try {
      const response = await apiClient.get('knowledge-bases?activeOnly=true&maxResults=50');
      if (response.ok) {
        const data = await response.json();
        setExistingKBs(data.knowledgeBases || []);
        if (data.knowledgeBases.length === 0) {
          console.log('No Knowledge Bases found:', data.message);
        }
      } else {
        console.error('Failed to load Knowledge Bases:', response.status);
        // Fallback to empty array
        setExistingKBs([]);
      }
    } catch (error) {
      console.error('Error loading Knowledge Bases:', error);
      // Fallback to empty array
      setExistingKBs([]);
    }
  };

  const handleInputChange = (field: string, value: any) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));
  };

  const handleNestedInputChange = (parent: string, field: string, value: any) => {
    setFormData(prev => ({
      ...prev,
      [parent]: {
        ...(prev[parent as keyof CreateBotRequest] as any),
        [field]: value
      }
    }));
  };

  const addConversationStarter = () => {
    setFormData(prev => ({
      ...prev,
      conversationStarters: [
        ...prev.conversationStarters,
        { title: '', example: '' }
      ]
    }));
  };

  const updateConversationStarter = (index: number, field: 'title' | 'example', value: string) => {
    setFormData(prev => ({
      ...prev,
      conversationStarters: prev.conversationStarters.map((starter, i) => 
        i === index ? { ...starter, [field]: value } : starter
      )
    }));
  };

  const removeConversationStarter = (index: number) => {
    setFormData(prev => ({
      ...prev,
      conversationStarters: prev.conversationStarters.filter((_, i) => i !== index)
    }));
  };

  const handleToolToggle = (tool: any) => {
    const isSelected = formData.agentTools.some(t => t.name === tool.name);
    
    if (isSelected) {
      setFormData(prev => ({
        ...prev,
        agentTools: prev.agentTools.filter(t => t.name !== tool.name)
      }));
    } else {
      const newTool: AgentTool = {
        type: tool.type as 'plain' | 'internet' | 'bedrock_agent',
        name: tool.name,
        description: tool.description,
        config: {}
      };
      
      setFormData(prev => ({
        ...prev,
        agentTools: [...prev.agentTools, newTool]
      }));
    }
  };

  const handleFileUpload = async (files: FileList | null) => {
    if (!files || !apiClient) return;

    for (const file of Array.from(files)) {
      try {
        setUploadProgress(prev => ({ ...prev, [file.name]: 0 }));

        // Request presigned URL
        const uploadRequest: PresignedUploadRequest = {
          botId: 'temp-' + Date.now(), // Temporary ID for uploads
          fileName: file.name,
          contentType: file.type,
          fileSize: file.size
        };

        const response = await apiClient.post('bots/temp/documents/presigned-url', uploadRequest);
        if (!response.ok) {
          throw new Error('Failed to get upload URL');
        }

        const uploadResponse: PresignedUploadResponse = await response.json();

        // Upload file to S3
        setUploadProgress(prev => ({ ...prev, [file.name]: 50 }));
        const uploadResult = await apiClient.upload(uploadResponse.uploadUrl, file, file.type);
        
        if (!uploadResult.ok) {
          throw new Error('Failed to upload file');
        }

        setUploadProgress(prev => ({ ...prev, [file.name]: 100 }));

        // Add to uploaded documents list
        const docInfo: DocumentInfo = {
          s3Key: uploadResponse.s3Key,
          fileName: file.name,
          contentType: file.type,
          size: file.size,
          lastModified: new Date(),
          s3Url: uploadResponse.s3Url
        };

        setUploadedDocuments(prev => [...prev, docInfo]);
        
        // Remove from progress tracker
        setTimeout(() => {
          setUploadProgress(prev => {
            const newProgress = { ...prev };
            delete newProgress[file.name];
            return newProgress;
          });
        }, 1000);

      } catch (err) {
        console.error('Upload failed:', err);
        setError(`Failed to upload ${file.name}: ${err instanceof Error ? err.message : 'Unknown error'}`);
        setUploadProgress(prev => {
          const newProgress = { ...prev };
          delete newProgress[file.name];
          return newProgress;
        });
      }
    }
  };

  const handleSubmit = async () => {
    if (!apiClient) return;

    setLoading(true);
    setError(null);

    try {
      // Prepare the request based on knowledge base option
      const requestData: CreateBotRequest = { 
        ...formData,
        // Add guardrails configuration
        guardrails: guardrailsEnabled ? {
          enabled: true,
          hateThreshold,
          insultsThreshold,
          sexualThreshold,
          violenceThreshold,
          misconductThreshold,
          groundingThreshold,
          relevanceThreshold
        } : {
          enabled: false
        }
      };

      if (knowledgeBaseOption === 'existing' && formData.existingKnowledgeBaseId) {
        requestData.existingKnowledgeBaseId = formData.existingKnowledgeBaseId;
      } else if (knowledgeBaseOption === 'new') {
        requestData.knowledgeBaseCreation = {
          embeddingsModel: 'amazon.titan-embed-text-v2:0',
          chunkingStrategy: 'FIXED_SIZE',
          maxTokens: 512,
          overlapPercentage: 20,
          existingS3Urls: uploadedDocuments.map(doc => doc.s3Url),
          sourceUrls: [],
          enableRagReplicas: false
        };
      }

      const response = await apiClient.post('bots', requestData);
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || 'Failed to create bot');
      }

      await response.json();
      
      // Redirect to bots listing page
      navigate('/bots');
      
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create bot');
    } finally {
      setLoading(false);
    }
  };

  const nextStep = () => {
    if (currentStep < 4) {
      setCurrentStep(currentStep + 1);
    }
  };

  const prevStep = () => {
    if (currentStep > 1) {
      setCurrentStep(currentStep - 1);
    }
  };

  if (!user) {
    return <div>Please log in to create a bot.</div>;
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center space-x-4">
              <Link to="/" className="text-xl font-bold text-gray-900">
                AI Chat Platform
              </Link>
              <span className="text-gray-500">|</span>
              <Link to="/bots" className="text-gray-700 hover:text-gray-900">
                Bots
              </Link>
              <span className="text-gray-500">|</span>
              <span className="text-gray-700">Create Bot</span>
            </div>
            <div>
              <span className="text-sm text-gray-700 mr-4">Welcome, {user.username}</span>
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
        <div className="bg-white rounded-lg shadow-md p-8">
          <h1 className="text-2xl font-bold text-gray-900 mb-8">Create New Bot</h1>

          {/* Progress Steps */}
          <div className="flex items-center justify-center mb-8">
            {[1, 2, 3, 4].map((step) => (
              <div key={step} className="flex items-center">
                <div
                  className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
                    step <= currentStep
                      ? 'bg-blue-600 text-white'
                      : 'bg-gray-200 text-gray-600'
                  }`}
                >
                  {step}
                </div>
                {step < 4 && (
                  <div
                    className={`w-16 h-1 mx-2 ${
                      step < currentStep ? 'bg-blue-600' : 'bg-gray-200'
                    }`}
                  />
                )}
              </div>
            ))}
          </div>

          {/* Step Labels */}
          <div className="flex justify-center mb-8">
            <div className="flex space-x-8 text-sm text-gray-600">
              <span className={currentStep === 1 ? 'font-medium text-blue-600' : ''}>
                Basic Info
              </span>
              <span className={currentStep === 2 ? 'font-medium text-blue-600' : ''}>
                Knowledge Base
              </span>
              <span className={currentStep === 3 ? 'font-medium text-blue-600' : ''}>
                Configuration
              </span>
              <span className={currentStep === 4 ? 'font-medium text-blue-600' : ''}>
                Review
              </span>
            </div>
          </div>

          {error && (
            <div className="mb-6 p-4 bg-red-100 border border-red-400 text-red-700 rounded">
              {error}
            </div>
          )}

          {/* Step Content */}
          <div className="space-y-6">
            {/* Step 1: Basic Info */}
            {currentStep === 1 && (
              <div className="space-y-6">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Bot Name *
                  </label>
                  <input
                    type="text"
                    value={formData.title}
                    onChange={(e) => handleInputChange('title', e.target.value)}
                    className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="Enter bot name"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Description
                  </label>
                  <textarea
                    value={formData.description}
                    onChange={(e) => handleInputChange('description', e.target.value)}
                    rows={3}
                    className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="Brief description of what this bot does"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    System Instructions *
                  </label>
                  <textarea
                    value={formData.instruction}
                    onChange={(e) => handleInputChange('instruction', e.target.value)}
                    rows={6}
                    className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="Detailed instructions for how the bot should behave..."
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Sharing
                  </label>
                  <select
                    value={formData.sharedScope}
                    onChange={(e) => handleInputChange('sharedScope', e.target.value)}
                    className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="private">Private (only you)</option>
                    <option value="partial">Shared (specific users/groups)</option>
                    <option value="public">Public (everyone)</option>
                  </select>
                </div>
              </div>
            )}

            {/* Step 2: Knowledge Base */}
            {currentStep === 2 && (
              <div className="space-y-6">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-4">
                    Knowledge Base Options
                  </label>
                  <div className="space-y-3">
                    <label className="flex items-center">
                      <input
                        type="radio"
                        name="knowledgeBase"
                        value="none"
                        checked={knowledgeBaseOption === 'none'}
                        onChange={(e) => setKnowledgeBaseOption(e.target.value as 'none')}
                        className="mr-3"
                      />
                      <span>No Knowledge Base (general AI assistant)</span>
                    </label>
                    
                    <label className="flex items-center">
                      <input
                        type="radio"
                        name="knowledgeBase"
                        value="existing"
                        checked={knowledgeBaseOption === 'existing'}
                        onChange={(e) => setKnowledgeBaseOption(e.target.value as 'existing')}
                        className="mr-3"
                      />
                      <span>Use Existing Knowledge Base</span>
                    </label>
                    
                    <label className="flex items-center">
                      <input
                        type="radio"
                        name="knowledgeBase"
                        value="new"
                        checked={knowledgeBaseOption === 'new'}
                        onChange={(e) => setKnowledgeBaseOption(e.target.value as 'new')}
                        className="mr-3"
                      />
                      <span>Create New Knowledge Base</span>
                    </label>
                  </div>
                </div>

                {knowledgeBaseOption === 'existing' && (
                  <div>
                    <div className="flex items-center justify-between mb-2">
                      <label className="block text-sm font-medium text-gray-700">
                        Select Knowledge Base
                      </label>
                      <button
                        type="button"
                        onClick={loadExistingKnowledgeBases}
                        className="text-sm text-blue-600 hover:text-blue-800 flex items-center"
                        disabled={loading}
                      >
                        <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                        </svg>
                        Refresh
                      </button>
                    </div>
                    <select
                      value={formData.existingKnowledgeBaseId || ''}
                      onChange={(e) => handleInputChange('existingKnowledgeBaseId', e.target.value)}
                      className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                      <option value="">
                        {existingKBs.length === 0 ? 'No Knowledge Bases available' : 'Select a Knowledge Base'}
                      </option>
                      {existingKBs.map((kb) => (
                        <option key={kb.id} value={kb.id}>
                          {kb.name} {kb.description ? `- ${kb.description}` : ''} ({kb.status})
                        </option>
                      ))}
                    </select>
                    {existingKBs.length === 0 && (
                      <p className="text-sm text-gray-500 mt-1">
                        No Knowledge Bases found. Create one in the AWS Bedrock console first.
                      </p>
                    )}
                  </div>
                )}

                {knowledgeBaseOption === 'new' && (
                  <div className="space-y-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        Upload Documents
                      </label>
                      <div className="border-2 border-dashed border-gray-300 rounded-lg p-6 text-center">
                        <input
                          type="file"
                          multiple
                          accept=".pdf,.doc,.docx,.txt,.md"
                          onChange={(e) => handleFileUpload(e.target.files)}
                          className="hidden"
                          id="file-upload"
                        />
                        <label
                          htmlFor="file-upload"
                          className="cursor-pointer inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                        >
                          <svg className="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                          </svg>
                          Choose Files
                        </label>
                        <p className="mt-2 text-sm text-gray-500">
                          Drag and drop files here, or click to select files
                        </p>
                        <p className="text-xs text-gray-400 mt-1">
                          Supports PDF, DOC, DOCX, TXT, MD (max 10MB each)
                        </p>
                      </div>
                    </div>

                    {/* Upload Progress */}
                    {Object.keys(uploadProgress).length > 0 && (
                      <div className="space-y-2">
                        {Object.entries(uploadProgress).map(([fileName, progress]) => (
                          <div key={fileName} className="bg-gray-50 p-3 rounded">
                            <div className="flex justify-between text-sm mb-1">
                              <span>{fileName}</span>
                              <span>{progress}%</span>
                            </div>
                            <div className="w-full bg-gray-200 rounded-full h-2">
                              <div
                                className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                                style={{ width: `${progress}%` }}
                              />
                            </div>
                          </div>
                        ))}
                      </div>
                    )}

                    {/* Uploaded Documents */}
                    {uploadedDocuments.length > 0 && (
                      <div>
                        <h4 className="text-sm font-medium text-gray-700 mb-2">
                          Uploaded Documents ({uploadedDocuments.length})
                        </h4>
                        <div className="space-y-2">
                          {uploadedDocuments.map((doc, index) => (
                            <div key={index} className="flex items-center justify-between bg-green-50 p-3 rounded border border-green-200">
                              <div>
                                <span className="text-sm font-medium text-green-800">{doc.fileName}</span>
                                <span className="text-xs text-green-600 ml-2">
                                  ({(doc.size / 1024 / 1024).toFixed(2)} MB)
                                </span>
                              </div>
                              <button
                                onClick={() => setUploadedDocuments(prev => prev.filter((_, i) => i !== index))}
                                className="text-red-600 hover:text-red-800"
                              >
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                </svg>
                              </button>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>
            )}

            {/* Step 3: Configuration */}
            {currentStep === 3 && (
              <div className="space-y-6">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    AI Models
                  </label>
                  <select
                    value={formData.activeModels[0] || ''}
                    onChange={(e) => handleInputChange('activeModels', [e.target.value])}
                    className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    {availableModels.map((model) => (
                      <option key={model.id} value={model.id}>
                        {model.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Agent Tools
                  </label>
                  <div className="grid grid-cols-2 gap-3">
                    {availableTools.map((tool) => (
                      <label
                        key={tool.name}
                        className="flex items-center p-3 border border-gray-200 rounded-lg cursor-pointer hover:bg-gray-50"
                      >
                        <input
                          type="checkbox"
                          checked={formData.agentTools.some(t => t.name === tool.name)}
                          onChange={() => handleToolToggle(tool)}
                          className="mr-3"
                        />
                        <div>
                          <div className="font-medium text-sm">{tool.name}</div>
                          <div className="text-xs text-gray-500">{tool.description}</div>
                        </div>
                      </label>
                    ))}
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Conversation Starters
                  </label>
                  <div className="space-y-3">
                    {formData.conversationStarters.map((starter, index) => (
                      <div key={index} className="flex gap-3 items-start">
                        <div className="flex-1 space-y-2">
                          <input
                            type="text"
                            placeholder="Title"
                            value={starter.title}
                            onChange={(e) => updateConversationStarter(index, 'title', e.target.value)}
                            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                          />
                          <input
                            type="text"
                            placeholder="Example message"
                            value={starter.example}
                            onChange={(e) => updateConversationStarter(index, 'example', e.target.value)}
                            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                          />
                        </div>
                        <button
                          onClick={() => removeConversationStarter(index)}
                          className="mt-1 text-red-600 hover:text-red-800"
                        >
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                          </svg>
                        </button>
                      </div>
                    ))}
                    
                    <button
                      onClick={addConversationStarter}
                      className="flex items-center text-sm text-blue-600 hover:text-blue-800"
                    >
                      <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                      </svg>
                      Add Conversation Starter
                    </button>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Temperature
                    </label>
                    <input
                      type="range"
                      min="0"
                      max="1"
                      step="0.1"
                      value={formData.generationParams.temperature}
                      onChange={(e) => handleNestedInputChange('generationParams', 'temperature', parseFloat(e.target.value))}
                      className="w-full"
                    />
                    <div className="text-xs text-gray-500 mt-1">
                      {formData.generationParams.temperature} (0 = focused, 1 = creative)
                    </div>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Max Tokens
                    </label>
                    <input
                      type="number"
                      min="100"
                      max="4000"
                      value={formData.generationParams.maxTokens}
                      onChange={(e) => handleNestedInputChange('generationParams', 'maxTokens', parseInt(e.target.value))}
                      className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                  </div>
                </div>

                {/* Guardrails Section */}
                <div className="mt-6 p-6 bg-gray-50 rounded-lg">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="text-lg font-medium text-gray-900">Content Safety Guardrails</h3>
                    <label className="flex items-center">
                      <input
                        type="checkbox"
                        checked={guardrailsEnabled}
                        onChange={(e) => setGuardrailsEnabled(e.target.checked)}
                        className="mr-2"
                      />
                      <span className="text-sm text-gray-700">Enable Content Filtering</span>
                    </label>
                  </div>
                  
                  {guardrailsEnabled && (
                    <div className="space-y-4">
                      <p className="text-sm text-gray-600 mb-4">
                        Set thresholds for content filtering. Higher values are more restrictive (0 = off, 1 = maximum filtering).
                      </p>
                      
                      {/* Harmful Categories */}
                      <div className="grid grid-cols-2 gap-4">
                        <div>
                          <label className="block text-sm font-medium text-gray-700 mb-2">
                            Violence: {violenceThreshold.toFixed(1)}
                          </label>
                          <input
                            type="range"
                            min="0"
                            max="1"
                            step="0.1"
                            value={violenceThreshold}
                            onChange={(e) => setViolenceThreshold(parseFloat(e.target.value))}
                            className="w-full"
                          />
                          <div className="text-xs text-gray-500 mt-1">
                            Filter violent content and imagery
                          </div>
                        </div>

                        <div>
                          <label className="block text-sm font-medium text-gray-700 mb-2">
                            Sexual: {sexualThreshold.toFixed(1)}
                          </label>
                          <input
                            type="range"
                            min="0"
                            max="1"
                            step="0.1"
                            value={sexualThreshold}
                            onChange={(e) => setSexualThreshold(parseFloat(e.target.value))}
                            className="w-full"
                          />
                          <div className="text-xs text-gray-500 mt-1">
                            Filter sexual content and imagery
                          </div>
                        </div>

                        <div>
                          <label className="block text-sm font-medium text-gray-700 mb-2">
                            Hate: {hateThreshold.toFixed(1)}
                          </label>
                          <input
                            type="range"
                            min="0"
                            max="1"
                            step="0.1"
                            value={hateThreshold}
                            onChange={(e) => setHateThreshold(parseFloat(e.target.value))}
                            className="w-full"
                          />
                          <div className="text-xs text-gray-500 mt-1">
                            Filter hate speech and discriminatory content
                          </div>
                        </div>

                        <div>
                          <label className="block text-sm font-medium text-gray-700 mb-2">
                            Insults: {insultsThreshold.toFixed(1)}
                          </label>
                          <input
                            type="range"
                            min="0"
                            max="1"
                            step="0.1"
                            value={insultsThreshold}
                            onChange={(e) => setInsultsThreshold(parseFloat(e.target.value))}
                            className="w-full"
                          />
                          <div className="text-xs text-gray-500 mt-1">
                            Filter insults and harassment
                          </div>
                        </div>

                        <div>
                          <label className="block text-sm font-medium text-gray-700 mb-2">
                            Misconduct: {misconductThreshold.toFixed(1)}
                          </label>
                          <input
                            type="range"
                            min="0"
                            max="1"
                            step="0.1"
                            value={misconductThreshold}
                            onChange={(e) => setMisconductThreshold(parseFloat(e.target.value))}
                            className="w-full"
                          />
                          <div className="text-xs text-gray-500 mt-1">
                            Filter illegal activities and misconduct
                          </div>
                        </div>
                      </div>

                      {/* Contextual Grounding */}
                      <div className="mt-6 pt-4 border-t border-gray-200">
                        <h4 className="text-md font-medium text-gray-800 mb-3">Contextual Grounding</h4>
                        <div className="grid grid-cols-2 gap-4">
                          <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">
                              Grounding: {groundingThreshold.toFixed(1)}
                            </label>
                            <input
                              type="range"
                              min="0"
                              max="1"
                              step="0.1"
                              value={groundingThreshold}
                              onChange={(e) => setGroundingThreshold(parseFloat(e.target.value))}
                              className="w-full"
                            />
                            <div className="text-xs text-gray-500 mt-1">
                              Ensure responses are grounded in provided context
                            </div>
                          </div>

                          <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">
                              Relevance: {relevanceThreshold.toFixed(1)}
                            </label>
                            <input
                              type="range"
                              min="0"
                              max="1"
                              step="0.1"
                              value={relevanceThreshold}
                              onChange={(e) => setRelevanceThreshold(parseFloat(e.target.value))}
                              className="w-full"
                            />
                            <div className="text-xs text-gray-500 mt-1">
                              Ensure responses are relevant to the query
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Step 4: Review */}
            {currentStep === 4 && (
              <div className="space-y-6">
                <div className="bg-gray-50 p-6 rounded-lg">
                  <h3 className="text-lg font-medium text-gray-900 mb-4">Review Your Bot</h3>
                  
                  <div className="space-y-4">
                    <div>
                      <span className="font-medium text-gray-700">Name:</span>
                      <span className="ml-2">{formData.title}</span>
                    </div>
                    
                    <div>
                      <span className="font-medium text-gray-700">Description:</span>
                      <span className="ml-2">{formData.description || 'None'}</span>
                    </div>
                    
                    <div>
                      <span className="font-medium text-gray-700">Sharing:</span>
                      <span className="ml-2 capitalize">{formData.sharedScope}</span>
                    </div>
                    
                    <div>
                      <span className="font-medium text-gray-700">Knowledge Base:</span>
                      <span className="ml-2">
                        {knowledgeBaseOption === 'none' && 'None'}
                        {knowledgeBaseOption === 'existing' && `Existing KB: ${formData.existingKnowledgeBaseId}`}
                        {knowledgeBaseOption === 'new' && `New KB with ${uploadedDocuments.length} documents`}
                      </span>
                    </div>
                    
                    <div>
                      <span className="font-medium text-gray-700">Model:</span>
                      <span className="ml-2">
                        {formData.activeModels[0] 
                          ? availableModels.find(m => m.id === formData.activeModels[0])?.name || formData.activeModels[0]
                          : 'None'
                        }
                      </span>
                    </div>
                    
                    <div>
                      <span className="font-medium text-gray-700">Tools:</span>
                      <span className="ml-2">
                        {formData.agentTools.length === 0 
                          ? 'None' 
                          : formData.agentTools.map(t => t.name).join(', ')
                        }
                      </span>
                    </div>
                    
                    <div>
                      <span className="font-medium text-gray-700">Conversation Starters:</span>
                      <span className="ml-2">{formData.conversationStarters.length}</span>
                    </div>
                    
                    <div>
                      <span className="font-medium text-gray-700">Content Safety:</span>
                      <span className="ml-2">
                        {guardrailsEnabled ? (
                          <div className="mt-1">
                            <div className="text-sm text-gray-600">
                              Violence: {violenceThreshold.toFixed(1)}, Sexual: {sexualThreshold.toFixed(1)}, 
                              Hate: {hateThreshold.toFixed(1)}, Insults: {insultsThreshold.toFixed(1)}
                            </div>
                            <div className="text-sm text-gray-600">
                              Misconduct: {misconductThreshold.toFixed(1)}, Grounding: {groundingThreshold.toFixed(1)}, 
                              Relevance: {relevanceThreshold.toFixed(1)}
                            </div>
                          </div>
                        ) : (
                          'Disabled'
                        )}
                      </span>
                    </div>
                  </div>
                </div>

                <div className="bg-yellow-50 border border-yellow-200 rounded-md p-4">
                  <div className="flex">
                    <div className="flex-shrink-0">
                      <svg className="h-5 w-5 text-yellow-400" viewBox="0 0 20 20" fill="currentColor">
                        <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                      </svg>
                    </div>
                    <div className="ml-3">
                      <h3 className="text-sm font-medium text-yellow-800">
                        Ready to Create Bot
                      </h3>
                      <div className="mt-2 text-sm text-yellow-700">
                        {knowledgeBaseOption === 'new' && (
                          <p>
                            Your bot will be created with a new Knowledge Base. This process may take a few minutes 
                            to complete. You can monitor the progress on the bot details page.
                          </p>
                        )}
                        {knowledgeBaseOption === 'existing' && (
                          <p>
                            Your bot will be created using the existing Knowledge Base and will be ready immediately.
                          </p>
                        )}
                        {knowledgeBaseOption === 'none' && (
                          <p>
                            Your bot will be created as a general AI assistant without a Knowledge Base.
                          </p>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* Navigation Buttons */}
            <div className="flex justify-between pt-6 border-t">
              <button
                onClick={prevStep}
                disabled={currentStep === 1}
                className={`px-6 py-2 rounded-md text-sm font-medium ${
                  currentStep === 1
                    ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
                    : 'bg-gray-200 text-gray-800 hover:bg-gray-300'
                }`}
              >
                Previous
              </button>

              <div className="flex space-x-3">
                <Link
                  to="/bots"
                  className="px-6 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-50"
                >
                  Cancel
                </Link>
                
                {currentStep < 4 ? (
                  <button
                    onClick={nextStep}
                    disabled={
                      (currentStep === 1 && (!formData.title || !formData.instruction)) ||
                      (currentStep === 2 && knowledgeBaseOption === 'existing' && !formData.existingKnowledgeBaseId)
                    }
                    className={`px-6 py-2 rounded-md text-sm font-medium ${
                      (currentStep === 1 && (!formData.title || !formData.instruction)) ||
                      (currentStep === 2 && knowledgeBaseOption === 'existing' && !formData.existingKnowledgeBaseId)
                        ? 'bg-gray-300 text-gray-500 cursor-not-allowed'
                        : 'bg-blue-600 text-white hover:bg-blue-700'
                    }`}
                  >
                    Next
                  </button>
                ) : (
                  <button
                    onClick={handleSubmit}
                    disabled={loading}
                    className={`px-6 py-2 rounded-md text-sm font-medium ${
                      loading
                        ? 'bg-gray-300 text-gray-500 cursor-not-allowed'
                        : 'bg-green-600 text-white hover:bg-green-700'
                    }`}
                  >
                    {loading ? 'Creating Bot...' : 'Create Bot'}
                  </button>
                )}
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}