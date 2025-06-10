# Next.js Frontend Structure with Vercel AI SDK

This document outlines the structure and key components of the Next.js frontend application that integrates with the Vercel AI SDK for chat functionality.

## Directory Structure

```
/chat-ai-app/
├── .env.local               # Environment variables
├── .gitignore
├── next.config.js           # Next.js configuration
├── package.json             # Project dependencies
├── public/                  # Static assets
│   ├── fonts/
│   ├── images/
│   └── favicon.ico
├── src/
│   ├── app/                 # App Router structure
│   │   ├── api/             # API routes
│   │   │   ├── chat/
│   │   │   │   └── route.ts # Chat API endpoint
│   │   │   └── models/
│   │   │       └── route.ts # Models API endpoint
│   │   ├── bots/
│   │   │   ├── [botId]/
│   │   │   │   └── page.tsx # Bot-specific chat page
│   │   │   └── page.tsx     # Bot selection page
│   │   ├── settings/
│   │   │   └── page.tsx     # User settings page
│   │   ├── globals.css      # Global styles
│   │   ├── layout.tsx       # Root layout
│   │   └── page.tsx         # Home page
│   ├── components/          # Reusable components
│   │   ├── chat/            # Chat-related components
│   │   │   ├── ChatInput.tsx
│   │   │   ├── ChatMessage.tsx
│   │   │   ├── ChatMessageList.tsx
│   │   │   ├── MessageAttachment.tsx
│   │   │   └── ToolInvocation.tsx
│   │   ├── layout/          # Layout components
│   │   │   ├── Navbar.tsx
│   │   │   ├── Sidebar.tsx
│   │   │   └── ThemeToggle.tsx
│   │   ├── ui/              # UI components
│   │   │   ├── Button.tsx
│   │   │   ├── Card.tsx
│   │   │   ├── Dropdown.tsx
│   │   │   ├── Input.tsx
│   │   │   ├── Loading.tsx
│   │   │   └── Modal.tsx
│   │   └── bots/            # Bot-related components
│   │       ├── BotCard.tsx
│   │       ├── BotSelector.tsx
│   │       └── ModelSelector.tsx
│   ├── hooks/               # Custom hooks
│   │   ├── useBots.tsx      # For bot selection
│   │   └── useModels.tsx    # For model selection
│   ├── lib/                 # Utility functions
│   │   ├── api.ts           # API client functions
│   │   ├── types.ts         # TypeScript interfaces
│   │   ├── utils.ts         # Helper functions
│   │   └── tools/           # Chat tools
│   │       ├── weather.ts
│   │       └── search.ts
│   ├── context/             # Context providers
│   │   ├── BotContext.tsx   # Bot selection context
│   │   └── ThemeContext.tsx # Theme context
│   └── styles/              # Component styles
│       └── theme.ts         # Theme variables
└── tailwind.config.js       # Tailwind CSS configuration
```

## Key Features

1. **Bot Selection**: Users can browse and select from available bots on the home page
2. **Model Selection**: Each bot can be configured to use different language models
3. **Real-time Chat**: Streaming responses with Vercel AI SDK's useChat hook
4. **Tool Integration**: Support for various tools like weather and search
5. **Responsive Design**: Mobile-friendly interface with Tailwind CSS
6. **Dark/Light Mode**: Theme support for user preference
7. **File Attachments**: Support for uploading images and documents
8. **Error Handling**: Graceful error states and retry functionality

## API Integration

1. **Chat API**: Integrates with the backend using the Vercel AI SDK's streamText
2. **Bot Management**: Fetch, display, and interact with custom bots
3. **Model Configuration**: Select and configure different language models for each bot

## Component Highlights

1. **ChatMessage**: Displays messages with support for different content types (text, tools, files)
2. **ChatInput**: Message input with file attachment support
3. **ToolInvocation**: Visualizes tool calls and results in the chat
4. **BotCard**: Card component for bot selection
5. **ModelSelector**: Dropdown for model selection

## Context and State Management

1. **BotContext**: Manages bot selection state
2. **ThemeContext**: Manages theme preferences
3. **useChat**: Vercel AI SDK hook for chat functionality
4. **useBots**: Custom hook for bot management
5. **useModels**: Custom hook for model management

## Key Files Implementation

### Chat API (src/app/api/chat/route.ts)

```typescript
import { bedrock } from '@ai-sdk/amazon-bedrock';
import { streamText, tool } from 'ai';
import { z } from 'zod';
import { getWeather } from '@/lib/tools/weather';
import { searchWeb } from '@/lib/tools/search';

export const maxDuration = 30;

export async function POST(req: Request) {
  const { messages, botId, model } = await req.json();

  // Initialize Bedrock client
  const selectedModel = bedrock(model || 'anthropic.claude-3-sonnet-20240229-v1:0');

  const result = streamText({
    model: selectedModel,
    messages,
    tools: {
      weather: tool({
        description: 'Get current weather information for a location',
        parameters: z.object({
          location: z.string().describe('The location to get weather for (city name)'),
        }),
        execute: async ({ location }) => {
          return await getWeather(location);
        },
      }),
      
      searchWeb: tool({
        description: 'Search the web for information',
        parameters: z.object({
          query: z.string().describe('The search query'),
        }),
        execute: async ({ query }) => {
          return await searchWeb(query);
        },
      }),
    },
  });

  return result.toDataStreamResponse();
}
```

### Chat Message Component (src/components/chat/ChatMessage.tsx)

```tsx
'use client';

import { MessagePart } from '@ai-sdk/ui-utils';
import { MessageAttachment } from './MessageAttachment';
import { ToolInvocation } from './ToolInvocation';
import { useState } from 'react';

interface ChatMessageProps {
  role: 'user' | 'assistant' | 'system';
  parts: MessagePart[];
  id: string;
  createdAt?: Date;
  addToolResult?: (params: {
    toolCallId: string;
    result: string | object;
  }) => void;
}

export function ChatMessage({ role, parts, id, createdAt, addToolResult }: ChatMessageProps) {
  const [showReasoning, setShowReasoning] = useState(false);
  
  const timestamp = createdAt ? new Intl.DateTimeFormat('en-US', {
    hour: '2-digit',
    minute: '2-digit'
  }).format(createdAt) : '';

  return (
    <div className={`py-6 ${role === 'assistant' ? 'bg-zinc-50 dark:bg-zinc-900' : 'bg-white dark:bg-zinc-950'} px-4 sm:px-6`}>
      <div className="max-w-3xl mx-auto flex gap-4">
        <div className={`shrink-0 h-8 w-8 rounded-full flex items-center justify-center ${
          role === 'user' ? 'bg-blue-600 text-white' : 'bg-green-600 text-white'
        }`}>
          {role === 'user' ? 'U' : 'AI'}
        </div>
        
        <div className="flex-1 space-y-2">
          <div className="flex items-center justify-between">
            <h3 className="font-medium text-zinc-900 dark:text-zinc-100">
              {role === 'user' ? 'You' : 'Assistant'}
            </h3>
            {timestamp && (
              <span className="text-xs text-zinc-500">{timestamp}</span>
            )}
          </div>
          
          <div className="prose prose-zinc dark:prose-invert">
            {parts.map((part, i) => {
              switch (part.type) {
                case 'text':
                  return <div key={`${id}-text-${i}`} className="whitespace-pre-wrap">{part.text}</div>;
                
                case 'file':
                  return <MessageAttachment key={`${id}-file-${i}`} file={part} />;
                
                case 'tool-invocation':
                  return (
                    <ToolInvocation
                      key={`${id}-tool-${i}`}
                      toolInvocation={part.toolInvocation}
                      addToolResult={addToolResult}
                    />
                  );
                
                case 'reasoning':
                  // Show/hide reasoning toggle
                  return (
                    <div key={`${id}-reasoning-${i}`} className="mt-2 mb-2">
                      <button
                        onClick={() => setShowReasoning(!showReasoning)}
                        className="text-sm text-blue-600 dark:text-blue-400 hover:underline flex items-center gap-1"
                      >
                        {showReasoning ? 'Hide' : 'Show'} reasoning
                        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={`transition-transform ${showReasoning ? 'rotate-180' : ''}`}>
                          <polyline points="6 9 12 15 18 9" />
                        </svg>
                      </button>
                      
                      {showReasoning && (
                        <pre className="mt-2 p-3 text-xs bg-zinc-100 dark:bg-zinc-800 rounded-md overflow-auto max-h-[300px]">
                          {part.reasoning}
                        </pre>
                      )}
                    </div>
                  );
                  
                default:
                  return null;
              }
            })}
          </div>
        </div>
      </div>
    </div>
  );
}
```

### Bot-specific Chat Page (src/app/bots/[botId]/page.tsx)

```tsx
'use client';

import { useEffect, useRef } from 'react';
import { useChat } from '@ai-sdk/react';
import { useBots } from '@/hooks/useBots';
import { useModels } from '@/hooks/useModels';
import { ChatInput } from '@/components/chat/ChatInput';
import { ChatMessage } from '@/components/chat/ChatMessage';
import { ModelSelector } from '@/components/bots/ModelSelector';
import { Sidebar } from '@/components/layout/Sidebar';
import { Card } from '@/components/ui/Card';

export default function BotChatPage({ params }: { params: { botId: string } }) {
  const { botId } = params;
  const { getBotById } = useBots();
  const { models, isLoading: isLoadingModels } = useModels();
  const messagesEndRef = useRef<HTMLDivElement>(null);
  
  const bot = getBotById(botId);
  
  const {
    messages,
    input,
    handleInputChange,
    handleSubmit,
    status,
    setMessages,
    append,
    reload,
    stop,
    isLoading,
    error,
    addToolResult
  } = useChat({
    api: '/api/chat',
    maxSteps: 5, // Enable multi-step tool calls
    body: {
      botId,
      model: bot?.defaultModel || 'anthropic.claude-3-sonnet-20240229-v1:0',
    },
    onError: (error) => {
      console.error('Chat error:', error);
    },
  });
  
  // Auto-scroll to bottom
  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages]);

  if (!bot) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Card className="p-6 max-w-md">
          <h2 className="text-xl font-semibold mb-2">Bot not found</h2>
          <p className="text-zinc-600 dark:text-zinc-400">The bot you're looking for doesn't exist.</p>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex h-screen">
      <Sidebar>
        <div className="p-4">
          <h2 className="text-lg font-semibold mb-4">{bot.name}</h2>
          <p className="text-sm text-zinc-600 dark:text-zinc-400 mb-6">{bot.description}</p>
          
          <div className="space-y-4">
            <div>
              <h3 className="text-sm font-medium mb-2">Model</h3>
              <ModelSelector
                models={models}
                selectedModel={bot.defaultModel}
                onSelectModel={(modelId) => console.log(`Changed model to: ${modelId}`)}
                isLoading={isLoadingModels}
              />
            </div>
            
            <div>
              <h3 className="text-sm font-medium mb-2">Actions</h3>
              <button
                onClick={() => setMessages([])}
                className="w-full px-3 py-2 text-sm bg-zinc-200 hover:bg-zinc-300 dark:bg-zinc-800 dark:hover:bg-zinc-700 rounded-md transition-colors"
              >
                Clear conversation
              </button>
            </div>
          </div>
        </div>
      </Sidebar>
      
      <div className="flex-1 flex flex-col h-full overflow-hidden">
        <div className="flex-1 overflow-y-auto">
          {messages.length === 0 ? (
            <div className="h-full flex flex-col items-center justify-center text-center p-4">
              <h2 className="text-2xl font-semibold mb-2">Chat with {bot.name}</h2>
              <p className="text-zinc-600 dark:text-zinc-400 max-w-md mb-4">
                {bot.description}
              </p>
            </div>
          ) : (
            <div className="pb-20">
              {messages.map((message) => (
                <ChatMessage
                  key={message.id}
                  id={message.id}
                  role={message.role}
                  parts={message.parts}
                  createdAt={new Date()}
                  addToolResult={addToolResult}
                />
              ))}
              
              <div ref={messagesEndRef} />
            </div>
          )}
        </div>
        
        <div className="absolute bottom-0 left-0 right-0 bg-white dark:bg-zinc-950 border-t border-zinc-200 dark:border-zinc-800">
          <div className="max-w-3xl mx-auto">
            <ChatInput
              input={input}
              handleInputChange={handleInputChange}
              handleSubmit={handleSubmit}
              isLoading={isLoading}
              placeholder={`Message ${bot.name}...`}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
```

## AWS Service Integration

The frontend integrates with AWS services through:

1. **Bedrock Integration**: Using `@ai-sdk/amazon-bedrock` provider for models
2. **API Gateway**: All backend communication goes through API Gateway endpoints
3. **Authentication**: Currently using static credentials but will later integrate with Cognito
4. **S3 Hosting**: The built Next.js app will be hosted on S3 and served through CloudFront

## Data Flow

1. User selects a bot from the home page
2. User starts a conversation by sending a message
3. Message is sent to the API Gateway endpoint
4. Go Lambda processes the request and calls Bedrock
5. Response is streamed back to the frontend
6. Vercel AI SDK handles the streaming response and updates the UI in real-time
7. If tool calls are needed, they are executed and results sent back to continue the conversation