import type { Message } from '../types';

/**
 * Chat storage utilities that provide AI SDK-like persistence
 * This replaces what Vercel AI SDK would handle automatically
 */

const STORAGE_PREFIX = 'amazonbuddy_chat_';

export class ChatStorage {
  private static getKey(conversationId: string, suffix: string): string {
    return `${STORAGE_PREFIX}${conversationId}_${suffix}`;
  }

  // Message storage
  static saveMessages(conversationId: string, messages: Message[]): void {
    try {
      const key = this.getKey(conversationId, 'messages');
      localStorage.setItem(key, JSON.stringify(messages));
    } catch (error) {
      console.warn('Failed to save messages to localStorage:', error);
    }
  }

  static loadMessages(conversationId: string): Message[] {
    try {
      const key = this.getKey(conversationId, 'messages');
      const stored = localStorage.getItem(key);
      if (!stored) return [];
      
      const messages = JSON.parse(stored);
      // Ensure dates are properly restored
      return messages.map((msg: any) => ({
        ...msg,
        createdAt: new Date(msg.createdAt)
      }));
    } catch (error) {
      console.warn('Failed to load messages from localStorage:', error);
      return [];
    }
  }

  static clearMessages(conversationId: string): void {
    try {
      const key = this.getKey(conversationId, 'messages');
      localStorage.removeItem(key);
    } catch (error) {
      console.warn('Failed to clear messages from localStorage:', error);
    }
  }

  // Draft message storage (unsent messages)
  static saveDraft(conversationId: string, draft: string): void {
    try {
      const key = this.getKey(conversationId, 'draft');
      if (draft.trim()) {
        localStorage.setItem(key, draft);
      } else {
        localStorage.removeItem(key);
      }
    } catch (error) {
      console.warn('Failed to save draft to localStorage:', error);
    }
  }

  static loadDraft(conversationId: string): string {
    try {
      const key = this.getKey(conversationId, 'draft');
      return localStorage.getItem(key) || '';
    } catch (error) {
      console.warn('Failed to load draft from localStorage:', error);
      return '';
    }
  }

  static clearDraft(conversationId: string): void {
    try {
      const key = this.getKey(conversationId, 'draft');
      localStorage.removeItem(key);
    } catch (error) {
      console.warn('Failed to clear draft from localStorage:', error);
    }
  }

  // Conversation metadata storage
  static saveConversationState(conversationId: string, state: {
    scrollPosition?: number;
    selectedModel?: string;
    extendedThinking?: boolean;
  }): void {
    try {
      const key = this.getKey(conversationId, 'state');
      localStorage.setItem(key, JSON.stringify(state));
    } catch (error) {
      console.warn('Failed to save conversation state to localStorage:', error);
    }
  }

  static loadConversationState(conversationId: string): {
    scrollPosition?: number;
    selectedModel?: string;
    extendedThinking?: boolean;
  } {
    try {
      const key = this.getKey(conversationId, 'state');
      const stored = localStorage.getItem(key);
      return stored ? JSON.parse(stored) : {};
    } catch (error) {
      console.warn('Failed to load conversation state from localStorage:', error);
      return {};
    }
  }

  // Cleanup old storage entries
  static cleanup(maxAge: number = 7 * 24 * 60 * 60 * 1000): void {
    try {
      const cutoff = Date.now() - maxAge;
      const keysToRemove: string[] = [];
      
      for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i);
        if (key?.startsWith(STORAGE_PREFIX)) {
          const item = localStorage.getItem(key);
          if (item) {
            try {
              const data = JSON.parse(item);
              const timestamp = data.timestamp || data.createdAt || data[0]?.createdAt;
              if (timestamp && new Date(timestamp).getTime() < cutoff) {
                keysToRemove.push(key);
              }
            } catch (e) {
              // If we can't parse it, remove it
              keysToRemove.push(key);
            }
          }
        }
      }
      
      keysToRemove.forEach(key => localStorage.removeItem(key));
      
      if (keysToRemove.length > 0) {
        console.log(`Cleaned up ${keysToRemove.length} old chat storage entries`);
      }
    } catch (error) {
      console.warn('Failed to cleanup chat storage:', error);
    }
  }

  // Get storage usage info
  static getStorageInfo(): {
    totalEntries: number;
    totalSize: number;
    oldestEntry: Date | null;
    newestEntry: Date | null;
  } {
    let totalEntries = 0;
    let totalSize = 0;
    let oldestEntry: Date | null = null;
    let newestEntry: Date | null = null;

    try {
      for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i);
        if (key?.startsWith(STORAGE_PREFIX)) {
          totalEntries++;
          const item = localStorage.getItem(key);
          if (item) {
            totalSize += item.length;
            
            try {
              const data = JSON.parse(item);
              const timestamp = data.timestamp || data.createdAt || data[0]?.createdAt;
              if (timestamp) {
                const date = new Date(timestamp);
                if (!oldestEntry || date < oldestEntry) {
                  oldestEntry = date;
                }
                if (!newestEntry || date > newestEntry) {
                  newestEntry = date;
                }
              }
            } catch (e) {
              // Skip invalid entries
            }
          }
        }
      }
    } catch (error) {
      console.warn('Failed to get storage info:', error);
    }

    return {
      totalEntries,
      totalSize,
      oldestEntry,
      newestEntry
    };
  }
}

// Auto-cleanup on module load (runs once per session)
ChatStorage.cleanup();