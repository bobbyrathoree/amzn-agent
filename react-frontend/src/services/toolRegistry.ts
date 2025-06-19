// 🏛️ TOOL REGISTRY - Discovery & Management Engine
// The heart of our tools framework

import type { 
  UniversalTool, 
  ToolRoutingPlan,
  ExecutionContext
} from '../types/tools';
import type { ToolCategory } from '../types/tools';

const TOOL_CATEGORIES: ToolCategory[] = [
  'information',
  'creative',
  'business',
  'development',
  'research',
  'automation',
  'communication'
];

export class ToolRegistry {
  private tools: Map<string, UniversalTool> = new Map();
  private toolsByCategory: Map<ToolCategory, UniversalTool[]> = new Map();
  private toolsByCapability: Map<string, UniversalTool[]> = new Map();

  constructor() {
    this.initializeCategories();
  }

  private initializeCategories() {
    TOOL_CATEGORIES.forEach(category => {
      this.toolsByCategory.set(category, []);
    });
  }

  // 🔧 TOOL REGISTRATION
  registerTool(tool: UniversalTool): void {
    // Validate tool
    this.validateTool(tool);
    
    // Skip if already registered
    if (this.tools.has(tool.id)) {
      return;
    }
    
    // Store tool
    this.tools.set(tool.id, tool);
    
    // Index by category
    const categoryTools = this.toolsByCategory.get(tool.category) || [];
    categoryTools.push(tool);
    this.toolsByCategory.set(tool.category, categoryTools);
    
    // Index by capabilities
    tool.capabilities.forEach(capability => {
      const capabilityTools = this.toolsByCapability.get(capability.name) || [];
      capabilityTools.push(tool);
      this.toolsByCapability.set(capability.name, capabilityTools);
    });
  }

  private validateTool(tool: UniversalTool): void {
    if (!tool.id || !tool.name || !tool.handler) {
      throw new Error(`Invalid tool: missing required fields`);
    }
    
    if (this.tools.has(tool.id)) {
      console.warn(`Tool ${tool.id} already registered - skipping duplicate registration`);
      return; // Don't throw error for duplicate registration
    }
    
    if (!tool.capabilities || tool.capabilities.length === 0) {
      throw new Error(`Tool ${tool.id} must have at least one capability`);
    }
  }

  // 🔍 TOOL DISCOVERY
  async discoverTools(filters?: {
    category?: ToolCategory;
    search?: string;
    tags?: string[];
    featured?: boolean;
  }): Promise<UniversalTool[]> {
    let tools = Array.from(this.tools.values());
    
    if (filters?.category) {
      tools = this.toolsByCategory.get(filters.category) || [];
    }
    
    if (filters?.search) {
      const searchLower = filters.search.toLowerCase();
      tools = tools.filter(tool => 
        tool.name.toLowerCase().includes(searchLower) ||
        tool.description.toLowerCase().includes(searchLower) ||
        tool.keywords.some(keyword => keyword.toLowerCase().includes(searchLower)) ||
        tool.tags.some(tag => tag.toLowerCase().includes(searchLower))
      );
    }
    
    if (filters?.tags) {
      tools = tools.filter(tool => 
        filters.tags!.some(tag => tool.tags.includes(tag))
      );
    }
    
    if (filters?.featured !== undefined) {
      tools = tools.filter(tool => tool.featured === filters.featured);
    }
    
    // Sort by relevance (featured first, then alphabetical)
    return tools.sort((a, b) => {
      if (a.featured && !b.featured) return -1;
      if (!a.featured && b.featured) return 1;
      return a.name.localeCompare(b.name);
    });
  }

  // 🎯 SMART TOOL ROUTING
  async findBestTool(intent: string, _context: ExecutionContext): Promise<ToolRoutingPlan | null> {
    const tools = Array.from(this.tools.values());
    
    // Simple intent matching for now (can be enhanced with AI)
    const matches = this.matchIntent(intent, tools);
    
    if (matches.length === 0) {
      return null;
    }
    
    // Score and rank matches
    const scoredMatches = matches.map(match => ({
      ...match,
      confidence: this.calculateConfidence(intent, match.tool, match.capability)
    }));
    
    // Sort by confidence
    scoredMatches.sort((a, b) => b.confidence - a.confidence);
    
    const bestMatch = scoredMatches[0];
    
    return {
      confidence: bestMatch.confidence,
      estimatedTime: bestMatch.capability.timeEstimate || 5000,
      estimatedCost: bestMatch.capability.costEstimate?.credits || 0,
      steps: [{
        toolId: bestMatch.tool.id,
        capability: bestMatch.capability.name,
        reasoning: `Best match for intent: "${intent}"`,
        confidence: bestMatch.confidence
      }],
      alternatives: scoredMatches.slice(1, 3).map(match => ({
        confidence: match.confidence,
        estimatedTime: match.capability.timeEstimate || 5000,
        estimatedCost: match.capability.costEstimate?.credits || 0,
        steps: [{
          toolId: match.tool.id,
          capability: match.capability.name,
          reasoning: `Alternative match for intent: "${intent}"`,
          confidence: match.confidence
        }]
      }))
    };
  }

  private matchIntent(intent: string, tools: UniversalTool[]): Array<{
    tool: UniversalTool;
    capability: typeof tools[0]['capabilities'][0];
    score: number;
  }> {
    const intentLower = intent.toLowerCase();
    const matches: Array<{ tool: UniversalTool; capability: any; score: number }> = [];
    
    tools.forEach(tool => {
      tool.capabilities.forEach(capability => {
        let score = 0;
        
        // Check capability name and description
        if (capability.name.toLowerCase().includes(intentLower)) score += 10;
        if (capability.description.toLowerCase().includes(intentLower)) score += 8;
        
        // Check tool name and description
        if (tool.name.toLowerCase().includes(intentLower)) score += 6;
        if (tool.description.toLowerCase().includes(intentLower)) score += 4;
        
        // Check keywords and tags
        tool.keywords.forEach(keyword => {
          if (intentLower.includes(keyword.toLowerCase())) score += 5;
        });
        
        tool.tags.forEach(tag => {
          if (intentLower.includes(tag.toLowerCase())) score += 3;
        });
        
        // Check examples
        capability.examples.forEach(example => {
          if (example.toLowerCase().includes(intentLower)) score += 2;
        });
        
        if (score > 0) {
          matches.push({ tool, capability, score });
        }
      });
    });
    
    return matches.sort((a, b) => b.score - a.score);
  }

  private calculateConfidence(intent: string, tool: UniversalTool, capability: any): number {
    // Simple confidence calculation (can be enhanced with ML)
    const intentWords = intent.toLowerCase().split(' ');
    const toolWords = [
      ...tool.name.toLowerCase().split(' '),
      ...tool.description.toLowerCase().split(' '),
      ...capability.name.toLowerCase().split(' '),
      ...capability.description.toLowerCase().split(' '),
      ...tool.keywords,
      ...tool.tags
    ];
    
    const matches = intentWords.filter(word => 
      toolWords.some(toolWord => toolWord.includes(word) || word.includes(toolWord))
    );
    
    return Math.min(matches.length / intentWords.length, 1.0);
  }

  // 🔎 CAPABILITY DISCOVERY
  async findToolsWithCapability(capabilityName: string): Promise<UniversalTool[]> {
    return this.toolsByCapability.get(capabilityName) || [];
  }

  // 📊 TOOL INFORMATION
  getTool(toolId: string): UniversalTool | undefined {
    return this.tools.get(toolId);
  }

  getAllTools(): UniversalTool[] {
    return Array.from(this.tools.values());
  }

  getToolsByCategory(category: ToolCategory): UniversalTool[] {
    return this.toolsByCategory.get(category) || [];
  }

  // 🧹 REGISTRY CLEANUP
  clearRegistry(): void {
    this.tools.clear();
    this.toolsByCategory.clear();
    this.toolsByCapability.clear();
    this.initializeCategories();
  }

  // 🔍 REGISTRY STATUS
  isToolRegistered(toolId: string): boolean {
    return this.tools.has(toolId);
  }

  // 📈 ANALYTICS
  getRegistryStats(): {
    totalTools: number;
    toolsByCategory: Record<string, number>;
    totalCapabilities: number;
    averageCapabilitiesPerTool: number;
  } {
    const tools = Array.from(this.tools.values());
    const toolsByCategory: Record<string, number> = {};
    let totalCapabilities = 0;
    
    tools.forEach(tool => {
      const category = tool.category;
      toolsByCategory[category] = (toolsByCategory[category] || 0) + 1;
      totalCapabilities += tool.capabilities.length;
    });
    
    return {
      totalTools: tools.length,
      toolsByCategory,
      totalCapabilities,
      averageCapabilitiesPerTool: totalCapabilities / tools.length || 0
    };
  }
}

// 🌟 SINGLETON INSTANCE
export const toolRegistry = new ToolRegistry();