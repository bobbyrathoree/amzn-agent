export interface User {
  username: string;
  userId: string;
  email?: string;
  groups?: string[];
  accessToken?: string;
}

export interface Config {
  environment: string;
  userPoolId: string;
  userPoolClientId: string;
  apiEndpoint: string;
  websocketEndpoint: string;
  region: string;
}

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: Date;
}

export interface Bot {
  id: string;
  title: string;
  description: string;
  instruction: string;
  isPublic: boolean;
  knowledgeBaseId: string;
  displayRetrievedChunks: boolean;
  generationParams: {
    maxTokens: number;
    temperature: number;
    topP: number;
    topK: number;
  };
  knowledgeBaseConfig: {
    searchType: 'HYBRID' | 'SEMANTIC';
    maxResults: number;
    scoreThreshold: number;
  };
  activeModels: string[];
  conversationStarters: string[];
  agentTools: string[];
  createdAt?: Date;
  updatedAt?: Date;
}