import { useState, useEffect, useCallback } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  Check, 
  X, 
  FileText, 
  Settings, 
  Eye, 
  Sparkles,
  Plus,
  BookOpen
} from 'lucide-react';
import { useAuth } from '../components/AuthProvider';
import { useApiClient } from '../lib/api';
import { BotToolsSelector } from '../components/BotToolsSelector';
import { ThemeToggle } from '../components/ThemeToggle';
import { GlassCard } from '../components/GlassCard';
import { initializeTools } from '../tools';
import type { 
  CreateBotRequest, 
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

  // Removed: old availableTools - now handled by BotToolsSelector

  useEffect(() => {
    // Load existing Knowledge Bases when component mounts
    loadExistingKnowledgeBases();
    // Initialize tools framework
    initializeTools();
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
    <div className="min-h-screen bg-background relative overflow-hidden">
      {/* Animated Background */}
      <div className="fixed inset-0 -z-10">
        <div className="absolute inset-0 bg-gradient-to-br from-background via-background to-background/95" />
        {Array.from({ length: 6 }, (_, i) => (
          <motion.div
            key={i}
            className="absolute w-2 h-2 bg-primary/20 rounded-full"
            initial={{
              x: Math.random() * (typeof window !== 'undefined' ? window.innerWidth : 1200),
              y: Math.random() * (typeof window !== 'undefined' ? window.innerHeight : 800),
            }}
            animate={{
              x: Math.random() * (typeof window !== 'undefined' ? window.innerWidth : 1200),
              y: Math.random() * (typeof window !== 'undefined' ? window.innerHeight : 800),
            }}
            transition={{
              duration: 25 + Math.random() * 20,
              repeat: Infinity,
              repeatType: "reverse",
              ease: "linear",
            }}
          />
        ))}
      </div>

      {/* Modern Header */}
      <motion.header 
        className="glass-card border-b border-border/50 relative z-10"
        initial={{ y: -100, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        transition={{ duration: 0.8 }}
      >
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-20">
            <motion.div 
              className="flex items-center space-x-6"
              initial={{ opacity: 0, x: -30 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.8, delay: 0.2 }}
            >
              <Link 
                to="/" 
                className="flex items-center gap-3 text-2xl font-bold bg-gradient-to-r from-primary to-primary/70 bg-clip-text text-transparent hover:from-primary/80 hover:to-primary/90 transition-all duration-300"
              >
                <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center">
                  <Sparkles className="w-5 h-5 text-white" />
                </div>
                Foundry
              </Link>
              <div className="h-8 w-px bg-border/50" />
              <GlassCard 
                as={Link} 
                to="/bots" 
                className="px-3 py-2 text-sm font-medium text-muted-foreground hover:text-primary transition-all duration-300 hover-lift"
              >
                Bots
              </GlassCard>
              <div className="flex items-center gap-2 text-foreground font-semibold">
                <Plus className="w-4 h-4" />
                Create Bot
              </div>
            </motion.div>
            
            <motion.div 
              className="flex items-center space-x-4"
              initial={{ opacity: 0, x: 30 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.8, delay: 0.4 }}
            >
              <span className="text-sm text-muted-foreground font-medium">Welcome, {user.username}</span>
              <ThemeToggle />
              <GlassCard
                as={motion.button}
                onClick={signOut}
                className="px-4 py-2 text-sm font-medium text-foreground hover:text-primary transition-all duration-300 hover-lift"
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
              >
                Sign Out
              </GlassCard>
            </motion.div>
          </div>
        </div>
      </motion.header>

      {/* Main Content */}
      <main className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8 relative z-10">
        <motion.div 
          className="glass-card p-8 md:p-12"
          initial={{ opacity: 0, y: 50 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8, delay: 0.6 }}
        >
          <motion.div
            className="text-center mb-12"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.8 }}
          >
            <h1 className="text-4xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent mb-4">
              Create New Bot
            </h1>
            <p className="text-muted-foreground text-lg">
              Build your AI assistant with custom knowledge and capabilities
            </p>
          </motion.div>

          {/* Modern Progress Steps */}
          <motion.div 
            className="flex items-center justify-center mb-12"
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.6, delay: 1.0 }}
          >
            {[
              { num: 1, label: 'Basic Info', icon: FileText },
              { num: 2, label: 'Knowledge Base', icon: BookOpen },
              { num: 3, label: 'Configuration', icon: Settings },
              { num: 4, label: 'Review', icon: Eye }
            ].map((step, index) => {
              const Icon = step.icon;
              const isActive = step.num === currentStep;
              const isCompleted = step.num < currentStep;
              // const isUpcoming = step.num > currentStep;
              
              return (
                <div key={step.num} className="flex items-center">
                  <motion.div
                    className="flex flex-col items-center"
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.4, delay: 1.2 + index * 0.1 }}
                  >
                    <GlassCard
                      as={motion.div}
                      className={`relative w-12 h-12 rounded-xl flex items-center justify-center text-sm font-semibold transition-all duration-300 ${
                        isCompleted
                          ? 'bg-gradient-to-r from-green-500 to-emerald-600 text-white shadow-lg'
                          : isActive
                          ? 'bg-gradient-to-r from-blue-500 to-purple-600 text-white shadow-lg'
                          : 'text-muted-foreground'
                      }`}
                      whileHover={{ scale: 1.05 }}
                      whileTap={{ scale: 0.95 }}
                    >
                      {isCompleted ? (
                        <Check className="w-5 h-5" />
                      ) : (
                        <Icon className="w-5 h-5" />
                      )}
                      
                      {isActive && (
                        <motion.div
                          className="absolute inset-0 rounded-xl bg-gradient-to-r from-blue-500/20 to-purple-600/20"
                          animate={{ scale: [1, 1.2, 1] }}
                          transition={{ duration: 2, repeat: Infinity }}
                        />
                      )}
                    </GlassCard>

                    <span className={`mt-3 text-xs font-medium transition-colors ${
                      isActive ? 'text-primary' : isCompleted ? 'text-green-400' : 'text-muted-foreground'
                    }`}>
                      {step.label}
                    </span>
                  </motion.div>
                  
                  {step.num < 4 && (
                    <motion.div
                      className={`w-20 h-1 mx-4 rounded-full transition-all duration-500 ${
                        isCompleted ? 'bg-gradient-to-r from-green-500 to-emerald-600' : 'bg-border/30'
                      }`}
                      initial={{ scaleX: 0 }}
                      animate={{ scaleX: 1 }}
                      transition={{ duration: 0.5, delay: 1.4 + index * 0.1 }}
                    />
                  )}
                </div>
              );
            })}
          </motion.div>

          {/* Error State */}
          <AnimatePresence>
            {error && (
              <motion.div
                className="mb-8 glass-card border border-red-400/30 bg-red-500/10 p-4"
                initial={{ opacity: 0, y: -20, scale: 0.95 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, y: -20, scale: 0.95 }}
                transition={{ duration: 0.3 }}
              >
                <div className="flex items-center gap-3 text-red-400">
                  <div className="w-8 h-8 rounded-full bg-red-400/20 flex items-center justify-center">
                    <X className="w-5 h-5" />
                  </div>
                  <div className="font-medium">{error}</div>
                </div>
              </motion.div>
            )}
          </AnimatePresence>

          {/* Step Content */}
          <div className="space-y-6">
            {/* Step 1: Basic Info */}
            {currentStep === 1 && (
              <div className="space-y-6">
                <div>
                  <label className="block text-sm font-semibold text-foreground mb-3">
                    Bot Name *
                  </label>
                  <GlassCard
                    as="input"
                    type="text"
                    value={formData.title}
                    onChange={(e: any) => handleInputChange('title', e.target.value)}
                    className="w-full px-4 py-3 text-foreground placeholder-muted-foreground/60 focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300 text-base"
                    placeholder="Enter bot name"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-semibold text-foreground mb-3">
                    Description
                  </label>
                  <GlassCard
                    as="textarea"
                    value={formData.description}
                    onChange={(e: any) => handleInputChange('description', e.target.value)}
                    rows={3}
                    className="w-full px-4 py-3 text-foreground placeholder-muted-foreground/60 focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300 text-base resize-none"
                    placeholder="Brief description of what this bot does"
                  />
                </div>

                <div>
                  <label className="block text-sm font-semibold text-foreground mb-3">
                    System Instructions *
                  </label>
                  <GlassCard
                    as="textarea"
                    value={formData.instruction}
                    onChange={(e: any) => handleInputChange('instruction', e.target.value)}
                    rows={6}
                    className="w-full px-4 py-3 text-foreground placeholder-muted-foreground/60 focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300 text-base resize-none"
                    placeholder="Detailed instructions for how the bot should behave..."
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-semibold text-foreground mb-3">
                    Sharing
                  </label>
                  <GlassCard
                    as="select"
                    value={formData.sharedScope}
                    onChange={(e: any) => handleInputChange('sharedScope', e.target.value)}
                    className="w-full px-4 py-3 text-foreground focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300 text-base"
                  >
                    <option value="private">Private (only you)</option>
                    <option value="partial">Shared (specific users/groups)</option>
                    <option value="public">Public (everyone)</option>
                  </GlassCard>
                </div>
              </div>
            )}

            {/* Step 2: Knowledge Base */}
            {currentStep === 2 && (
              <div className="space-y-6">
                <div>
                  <label className="block text-sm font-semibold text-foreground mb-4">
                    Knowledge Base Options
                  </label>
                  <div className="space-y-3">
                    <label className="flex items-center">
                      <input
                        type="radio"
                        name="knowledgeBase"
                        value="none"
                        checked={knowledgeBaseOption === 'none'}
                        onChange={(e: any) => setKnowledgeBaseOption(e.target.value as 'none')}
                        className="mr-3"
                      />
                      <span className="text-foreground">No Knowledge Base (general AI assistant)</span>
                    </label>
                    
                    <label className="flex items-center">
                      <input
                        type="radio"
                        name="knowledgeBase"
                        value="existing"
                        checked={knowledgeBaseOption === 'existing'}
                        onChange={(e: any) => setKnowledgeBaseOption(e.target.value as 'existing')}
                        className="mr-3"
                      />
                      <span className="text-foreground">Use Existing Knowledge Base</span>
                    </label>
                    
                    <label className="flex items-center">
                      <input
                        type="radio"
                        name="knowledgeBase"
                        value="new"
                        checked={knowledgeBaseOption === 'new'}
                        onChange={(e: any) => setKnowledgeBaseOption(e.target.value as 'new')}
                        className="mr-3"
                      />
                      <span className="text-foreground">Create New Knowledge Base</span>
                    </label>
                  </div>
                </div>

                {knowledgeBaseOption === 'existing' && (
                  <div>
                    <div className="flex items-center justify-between mb-2">
                      <label className="block text-sm font-semibold text-foreground">
                        Select Knowledge Base
                      </label>
                      <button
                        type="button"
                        onClick={loadExistingKnowledgeBases}
                        className="text-sm text-primary hover:text-primary/80 flex items-center"
                        disabled={loading}
                      >
                        <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                        </svg>
                        Refresh
                      </button>
                    </div>
                    <GlassCard
                      as="select"
                      value={formData.existingKnowledgeBaseId || ''}
                      onChange={(e: any) => handleInputChange('existingKnowledgeBaseId', e.target.value)}
                      className="w-full px-4 py-3 text-foreground focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300 text-base"
                    >
                      <option value="">
                        {existingKBs.length === 0 ? 'No Knowledge Bases available' : 'Select a Knowledge Base'}
                      </option>
                      {existingKBs.map((kb) => (
                        <option key={kb.id} value={kb.id}>
                          {kb.name} {kb.description ? `- ${kb.description}` : ''} ({kb.status})
                        </option>
                      ))}
                    </GlassCard>
                    {existingKBs.length === 0 && (
                      <p className="text-sm text-muted-foreground mt-1">
                        No Knowledge Bases found. Create one in the AWS Bedrock console first.
                      </p>
                    )}
                  </div>
                )}

                {knowledgeBaseOption === 'new' && (
                  <div className="space-y-4">
                    <div>
                      <label className="block text-sm font-semibold text-foreground mb-3">
                        Upload Documents
                      </label>
                      <div className="border-2 border-dashed border-border/50 rounded-lg p-6 text-center hover:border-primary/50 transition-colors">
                        <input
                          type="file"
                          multiple
                          accept=".pdf,.doc,.docx,.txt,.md"
                          onChange={(e: any) => handleFileUpload(e.target.files)}
                          className="hidden"
                          id="file-upload"
                        />
                        <label
                          htmlFor="file-upload"
                          className="cursor-pointer inline-flex items-center px-4 py-2 bg-gradient-to-r from-blue-500 to-purple-600 text-white rounded-md hover:from-blue-600 hover:to-purple-700 transition-all duration-300 hover-lift"
                        >
                          <svg className="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                          </svg>
                          Choose Files
                        </label>
                        <p className="mt-2 text-sm text-muted-foreground">
                          Drag and drop files here, or click to select files
                        </p>
                        <p className="text-xs text-muted-foreground/70 mt-1">
                          Supports PDF, DOC, DOCX, TXT, MD (max 10MB each)
                        </p>
                      </div>
                    </div>

                    {/* Upload Progress */}
                    {Object.keys(uploadProgress).length > 0 && (
                      <div className="space-y-2">
                        {Object.entries(uploadProgress).map(([fileName, progress]) => (
                          <div key={fileName} className="glass-card p-3 rounded">
                            <div className="flex justify-between text-sm mb-1 text-foreground">
                              <span>{fileName}</span>
                              <span>{progress}%</span>
                            </div>
                            <div className="w-full bg-muted/30 rounded-full h-2">
                              <div
                                className="bg-gradient-to-r from-blue-500 to-purple-600 h-2 rounded-full transition-all duration-300"
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
                        <h4 className="text-sm font-semibold text-foreground mb-2">
                          Uploaded Documents ({uploadedDocuments.length})
                        </h4>
                        <div className="space-y-2">
                          {uploadedDocuments.map((doc, index) => (
                            <div key={index} className="flex items-center justify-between glass-card p-3 rounded border border-green-500/30 bg-green-500/10">
                              <div>
                                <span className="text-sm font-medium text-green-400">{doc.fileName}</span>
                                <span className="text-xs text-green-400/80 ml-2">
                                  ({(doc.size / 1024 / 1024).toFixed(2)} MB)
                                </span>
                              </div>
                              <button
                                onClick={() => setUploadedDocuments(prev => prev.filter((_, i) => i !== index))}
                                className="text-red-400 hover:text-red-300 transition-colors"
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
                  <label className="block text-sm font-semibold text-foreground mb-2">
                    AI Models
                  </label>
                  <GlassCard
                    as="select"
                    value={formData.activeModels[0] || ''}
                    onChange={(e: any) => handleInputChange('activeModels', [e.target.value])}
                    className="w-full px-4 py-3 text-foreground focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300 text-base"
                  >
                    {availableModels.map((model) => (
                      <option key={model.id} value={model.id}>
                        {model.name}
                      </option>
                    ))}
                  </GlassCard>
                </div>

                {/* Enhanced Bot Tools Selection */}
                <BotToolsSelector
                  selectedTools={formData.agentTools}
                  onToolsChange={(tools) => setFormData(prev => ({ ...prev, agentTools: tools }))}
                />

                <div>
                  <label className="block text-sm font-semibold text-foreground mb-2">
                    Conversation Starters
                  </label>
                  <div className="space-y-3">
                    {formData.conversationStarters.map((starter, index) => (
                      <div key={index} className="flex gap-3 items-start">
                        <div className="flex-1 space-y-2">
                          <GlassCard
                            as="input"
                            type="text"
                            placeholder="Title"
                            value={starter.title}
                            onChange={(e: any) => updateConversationStarter(index, 'title', e.target.value)}
                            className="w-full px-4 py-3 text-foreground placeholder-muted-foreground/60 focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300 text-sm"
                          />
                          <GlassCard
                            as="input"
                            type="text"
                            placeholder="Example message"
                            value={starter.example}
                            onChange={(e: any) => updateConversationStarter(index, 'example', e.target.value)}
                            className="w-full px-4 py-3 text-foreground placeholder-muted-foreground/60 focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300 text-sm"
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
                      className="flex items-center text-sm text-primary hover:text-primary/80 transition-colors duration-300"
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
                    <label className="block text-sm font-semibold text-foreground mb-2">
                      Temperature
                    </label>
                    <input
                      type="range"
                      min="0"
                      max="1"
                      step="0.1"
                      value={formData.generationParams.temperature}
                      onChange={(e: any) => handleNestedInputChange('generationParams', 'temperature', parseFloat(e.target.value))}
                      className="w-full accent-primary"
                    />
                    <div className="text-xs text-muted-foreground mt-1">
                      {formData.generationParams.temperature} (0 = focused, 1 = creative)
                    </div>
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-foreground mb-2">
                      Max Tokens
                    </label>
                    <GlassCard
                      as="input"
                      type="number"
                      min="100"
                      max="4000"
                      value={formData.generationParams.maxTokens}
                      onChange={(e: any) => handleNestedInputChange('generationParams', 'maxTokens', parseInt(e.target.value))}
                      className="w-full px-4 py-3 text-foreground focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300 text-base"
                    />
                  </div>
                </div>

                {/* Guardrails Section */}
                <div className="mt-6 p-6 glass-card border border-border/30 rounded-lg">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="text-lg font-semibold text-foreground">Content Safety Guardrails</h3>
                    <label className="flex items-center">
                      <input
                        type="checkbox"
                        checked={guardrailsEnabled}
                        onChange={(e: any) => setGuardrailsEnabled(e.target.checked)}
                        className="mr-2 accent-primary"
                      />
                      <span className="text-sm text-foreground">Enable Content Filtering</span>
                    </label>
                  </div>
                  
                  {guardrailsEnabled && (
                    <div className="space-y-4">
                      <p className="text-sm text-muted-foreground mb-4">
                        Set thresholds for content filtering. Higher values are more restrictive (0 = off, 1 = maximum filtering).
                      </p>
                      
                      {/* Harmful Categories */}
                      <div className="grid grid-cols-2 gap-4">
                        <div>
                          <label className="block text-sm font-semibold text-foreground mb-2">
                            Violence: {violenceThreshold.toFixed(1)}
                          </label>
                          <input
                            type="range"
                            min="0"
                            max="1"
                            step="0.1"
                            value={violenceThreshold}
                            onChange={(e: any) => setViolenceThreshold(parseFloat(e.target.value))}
                            className="w-full accent-primary"
                          />
                          <div className="text-xs text-muted-foreground mt-1">
                            Filter violent content and imagery
                          </div>
                        </div>

                        <div>
                          <label className="block text-sm font-semibold text-foreground mb-2">
                            Sexual: {sexualThreshold.toFixed(1)}
                          </label>
                          <input
                            type="range"
                            min="0"
                            max="1"
                            step="0.1"
                            value={sexualThreshold}
                            onChange={(e: any) => setSexualThreshold(parseFloat(e.target.value))}
                            className="w-full accent-primary"
                          />
                          <div className="text-xs text-muted-foreground mt-1">
                            Filter sexual content and imagery
                          </div>
                        </div>

                        <div>
                          <label className="block text-sm font-semibold text-foreground mb-2">
                            Hate: {hateThreshold.toFixed(1)}
                          </label>
                          <input
                            type="range"
                            min="0"
                            max="1"
                            step="0.1"
                            value={hateThreshold}
                            onChange={(e: any) => setHateThreshold(parseFloat(e.target.value))}
                            className="w-full accent-primary"
                          />
                          <div className="text-xs text-muted-foreground mt-1">
                            Filter hate speech and discriminatory content
                          </div>
                        </div>

                        <div>
                          <label className="block text-sm font-semibold text-foreground mb-2">
                            Insults: {insultsThreshold.toFixed(1)}
                          </label>
                          <input
                            type="range"
                            min="0"
                            max="1"
                            step="0.1"
                            value={insultsThreshold}
                            onChange={(e: any) => setInsultsThreshold(parseFloat(e.target.value))}
                            className="w-full accent-primary"
                          />
                          <div className="text-xs text-muted-foreground mt-1">
                            Filter insults and harassment
                          </div>
                        </div>

                        <div>
                          <label className="block text-sm font-semibold text-foreground mb-2">
                            Misconduct: {misconductThreshold.toFixed(1)}
                          </label>
                          <input
                            type="range"
                            min="0"
                            max="1"
                            step="0.1"
                            value={misconductThreshold}
                            onChange={(e: any) => setMisconductThreshold(parseFloat(e.target.value))}
                            className="w-full accent-primary"
                          />
                          <div className="text-xs text-muted-foreground mt-1">
                            Filter illegal activities and misconduct
                          </div>
                        </div>
                      </div>

                      {/* Contextual Grounding */}
                      <div className="mt-6 pt-4 border-t border-border">
                        <h4 className="text-md font-semibold text-foreground mb-3">Contextual Grounding</h4>
                        <div className="grid grid-cols-2 gap-4">
                          <div>
                            <label className="block text-sm font-semibold text-foreground mb-2">
                              Grounding: {groundingThreshold.toFixed(1)}
                            </label>
                            <input
                              type="range"
                              min="0"
                              max="1"
                              step="0.1"
                              value={groundingThreshold}
                              onChange={(e: any) => setGroundingThreshold(parseFloat(e.target.value))}
                              className="w-full accent-primary"
                            />
                            <div className="text-xs text-muted-foreground mt-1">
                              Ensure responses are grounded in provided context
                            </div>
                          </div>

                          <div>
                            <label className="block text-sm font-semibold text-foreground mb-2">
                              Relevance: {relevanceThreshold.toFixed(1)}
                            </label>
                            <input
                              type="range"
                              min="0"
                              max="1"
                              step="0.1"
                              value={relevanceThreshold}
                              onChange={(e: any) => setRelevanceThreshold(parseFloat(e.target.value))}
                              className="w-full accent-primary"
                            />
                            <div className="text-xs text-muted-foreground mt-1">
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
                <div className="glass-card p-6 rounded-lg border border-border/30">
                  <h3 className="text-lg font-semibold text-foreground mb-4">Review Your Bot</h3>
                  
                  <div className="space-y-4">
                    <div>
                      <span className="font-semibold text-foreground">Name:</span>
                      <span className="ml-2 text-foreground">{formData.title}</span>
                    </div>
                    
                    <div>
                      <span className="font-semibold text-foreground">Description:</span>
                      <span className="ml-2 text-muted-foreground">{formData.description || 'None'}</span>
                    </div>
                    
                    <div>
                      <span className="font-semibold text-foreground">Sharing:</span>
                      <span className="ml-2 capitalize text-muted-foreground">{formData.sharedScope}</span>
                    </div>
                    
                    <div>
                      <span className="font-semibold text-foreground">Knowledge Base:</span>
                      <span className="ml-2 text-muted-foreground">
                        {knowledgeBaseOption === 'none' && 'None'}
                        {knowledgeBaseOption === 'existing' && `Existing KB: ${formData.existingKnowledgeBaseId}`}
                        {knowledgeBaseOption === 'new' && `New KB with ${uploadedDocuments.length} documents`}
                      </span>
                    </div>
                    
                    <div>
                      <span className="font-semibold text-foreground">Model:</span>
                      <span className="ml-2 text-muted-foreground">
                        {formData.activeModels[0] 
                          ? availableModels.find(m => m.id === formData.activeModels[0])?.name || formData.activeModels[0]
                          : 'None'
                        }
                      </span>
                    </div>
                    
                    <div>
                      <span className="font-semibold text-foreground">Tools:</span>
                      <span className="ml-2 text-muted-foreground">
                        {formData.agentTools.length === 0 
                          ? 'None' 
                          : formData.agentTools.map(t => t.name).join(', ')
                        }
                      </span>
                    </div>
                    
                    <div>
                      <span className="font-semibold text-foreground">Conversation Starters:</span>
                      <span className="ml-2 text-muted-foreground">{formData.conversationStarters.length}</span>
                    </div>
                    
                    <div>
                      <span className="font-semibold text-foreground">Content Safety:</span>
                      <span className="ml-2">
                        {guardrailsEnabled ? (
                          <div className="mt-1">
                            <div className="text-sm text-muted-foreground">
                              Violence: {violenceThreshold.toFixed(1)}, Sexual: {sexualThreshold.toFixed(1)}, 
                              Hate: {hateThreshold.toFixed(1)}, Insults: {insultsThreshold.toFixed(1)}
                            </div>
                            <div className="text-sm text-muted-foreground">
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

                <div className="glass-card border border-amber-500/30 bg-amber-500/10 rounded-md p-4">
                  <div className="flex">
                    <div className="flex-shrink-0">
                      <svg className="h-5 w-5 text-amber-500" viewBox="0 0 20 20" fill="currentColor">
                        <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                      </svg>
                    </div>
                    <div className="ml-3">
                      <h3 className="text-sm font-semibold text-amber-600 dark:text-amber-400">
                        Ready to Create Bot
                      </h3>
                      <div className="mt-2 text-sm text-amber-700 dark:text-amber-300">
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
              <GlassCard
                as="button"
                onClick={prevStep}
                disabled={currentStep === 1}
                className={`px-6 py-2 rounded-md text-sm font-medium transition-all duration-300 ${
                  currentStep === 1
                    ? 'opacity-50 cursor-not-allowed text-muted-foreground'
                    : 'text-foreground hover:bg-muted/30 hover-lift'
                }`}
              >
                Previous
              </GlassCard>

              <div className="flex space-x-3">
                <GlassCard
                  as={Link}
                  to="/bots"
                  className="px-6 py-2 border border-border/50 rounded-md text-sm font-medium text-foreground hover:bg-muted/30 transition-all duration-300 hover-lift"
                >
                  Cancel
                </GlassCard>
                
                {currentStep < 4 ? (
                  <button
                    onClick={nextStep}
                    disabled={
                      (currentStep === 1 && (!formData.title || !formData.instruction)) ||
                      (currentStep === 2 && knowledgeBaseOption === 'existing' && !formData.existingKnowledgeBaseId)
                    }
                    className={`px-6 py-2 rounded-md text-sm font-medium transition-all duration-300 ${
                      (currentStep === 1 && (!formData.title || !formData.instruction)) ||
                      (currentStep === 2 && knowledgeBaseOption === 'existing' && !formData.existingKnowledgeBaseId)
                        ? 'glass-card opacity-50 cursor-not-allowed text-muted-foreground'
                        : 'bg-gradient-to-r from-blue-500 to-purple-600 text-white hover:from-blue-600 hover:to-purple-700 shadow-lg hover:shadow-xl hover-lift'
                    }`}
                  >
                    Next
                  </button>
                ) : (
                  <button
                    onClick={handleSubmit}
                    disabled={loading}
                    className={`px-6 py-2 rounded-md text-sm font-medium transition-all duration-300 ${
                      loading
                        ? 'glass-card opacity-50 cursor-not-allowed text-muted-foreground'
                        : 'bg-gradient-to-r from-green-500 to-emerald-600 text-white hover:from-green-600 hover:to-emerald-700 shadow-lg hover:shadow-xl hover-lift'
                    }`}
                  >
                    {loading ? 'Creating Bot...' : 'Create Bot'}
                  </button>
                )}
              </div>
            </div>
          </div>
        </motion.div>
      </main>
    </div>
  );
}