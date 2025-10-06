// 🛠️ TOOLS REGISTRY - Initialize All Available Tools
// Central registration point for all Foundry tools

import { toolRegistry } from '../services/toolRegistry';
import { WebSearchTool } from './webSearchTool';
import { ResearchAssistantTool } from './researchAssistant';

/**
 * Initialize and register all available tools
 * This function should be called during app startup
 */
// Flag to track initialization state
let isInitialized = false;

export function initializeTools(): void {
  // Prevent multiple initializations
  if (isInitialized) {
    return;
  }
  
  try {
    // Clear any existing registrations first
    toolRegistry.clearRegistry();
    
    // Register core information tools
    toolRegistry.registerTool(WebSearchTool);
    
    // Register research tools
    toolRegistry.registerTool(ResearchAssistantTool);
    
    // Mark as initialized
    isInitialized = true;
    
  } catch (error) {
    console.error('Failed to initialize tools framework:', error);
    throw error;
  }
}

/**
 * Force re-initialization of tools (for development/debugging)
 */
export function reinitializeTools(): void {
  isInitialized = false;
  initializeTools();
}

/**
 * Check if tools are initialized
 */
export function areToolsInitialized(): boolean {
  return isInitialized;
}

/**
 * Get all registered tools
 */
export function getAllTools() {
  return toolRegistry.getAllTools();
}

/**
 * Get tools by category
 */
export function getToolsByCategory(category: string) {
  return toolRegistry.getToolsByCategory(category as any);
}

/**
 * Get a specific tool by ID
 */
export function getTool(toolId: string) {
  return toolRegistry.getTool(toolId);
}

/**
 * Search for tools based on intent
 */
export async function findToolsForIntent(intent: string, context: any) {
  return toolRegistry.findBestTool(intent, context);
}

// Export individual tools for direct import if needed
export { WebSearchTool } from './webSearchTool';
export { ResearchAssistantTool } from './researchAssistant';

// Export tool registry for advanced usage
export { toolRegistry } from '../services/toolRegistry';
export { toolExecutor } from '../services/toolExecutor';

// Export types
export type { UniversalTool, ToolCategory } from '../types/tools';