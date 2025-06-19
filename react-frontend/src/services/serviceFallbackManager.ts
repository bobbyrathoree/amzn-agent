// 🔧 SERVICE FALLBACK MANAGER
// Netflix-like intelligent service degradation system

import type { ServiceFallback, ToolCapability, ExecutionContext } from '../types/tools';

export interface ServiceStatus {
  serviceId: string;
  available: boolean;
  mode: 'premium' | 'free';
  hasApiKey: boolean;
  lastChecked: Date;
  limitations?: {
    maxRequests?: number;
    maxResults?: number;
    features?: string[];
    quality?: 'high' | 'medium' | 'low';
  };
}

export interface FallbackSelection {
  selectedService: ServiceFallback;
  fallbackLevel: number;
  reasoning: string;
  availableUpgrades: Array<{
    serviceId: string;
    benefits: string[];
    setupRequired: boolean;
  }>;
}

/**
 * 🔧 Service Fallback Manager
 * Intelligently manages service fallbacks and mode detection
 */
export class ServiceFallbackManager {
  private vaultService: any;
  private serviceStatusCache = new Map<string, ServiceStatus>();
  private cacheExpiry = 5 * 60 * 1000; // 5 minutes

  constructor(vaultService: any) {
    this.vaultService = vaultService;
  }

  /**
   * Select the best available service from fallback hierarchy
   */
  async selectBestService(
    capability: ToolCapability,
    _context: ExecutionContext
  ): Promise<FallbackSelection> {
    if (!capability.fallbackHierarchy || capability.fallbackHierarchy.length === 0) {
      throw new Error('No fallback hierarchy defined for this capability');
    }

    console.log(`🔧 Selecting best service for capability: ${capability.name}`);

    // Get service statuses
    const serviceStatuses = await this.getServiceStatuses(
      capability.fallbackHierarchy.map(f => f.serviceId)
    );

    // Find the best available service
    const sortedServices = capability.fallbackHierarchy
      .sort((a, b) => a.priority - b.priority);

    for (let i = 0; i < sortedServices.length; i++) {
      const fallback = sortedServices[i];
      const status = serviceStatuses.get(fallback.serviceId);

      if (this.isServiceUsable(fallback, status)) {
        console.log(`✅ Selected service: ${fallback.serviceId} (level ${i})`);
        
        return {
          selectedService: fallback,
          fallbackLevel: i,
          reasoning: this.generateReasoning(fallback, status, i),
          availableUpgrades: this.getAvailableUpgrades(
            sortedServices,
            serviceStatuses,
            i
          )
        };
      }
    }

    throw new Error('No usable services available in fallback hierarchy');
  }

  /**
   * Get service statuses with caching
   */
  private async getServiceStatuses(serviceIds: string[]): Promise<Map<string, ServiceStatus>> {
    const statuses = new Map<string, ServiceStatus>();
    const now = new Date();

    for (const serviceId of serviceIds) {
      // Check cache first
      const cached = this.serviceStatusCache.get(serviceId);
      if (cached && (now.getTime() - cached.lastChecked.getTime()) < this.cacheExpiry) {
        statuses.set(serviceId, cached);
        continue;
      }

      // Check service status
      const status = await this.checkServiceStatus(serviceId);
      this.serviceStatusCache.set(serviceId, status);
      statuses.set(serviceId, status);
    }

    return statuses;
  }

  /**
   * Check individual service status
   */
  private async checkServiceStatus(serviceId: string): Promise<ServiceStatus> {
    console.log(`🔍 Checking status for service: ${serviceId}`);

    try {
      // Check if API key exists in vault
      const hasApiKey = await this.checkApiKeyExists(serviceId);
      
      // Get service limitations from supported services
      const limitations = await this.getServiceLimitations(serviceId);

      const status: ServiceStatus = {
        serviceId,
        available: true,
        mode: hasApiKey ? 'premium' : 'free',
        hasApiKey,
        lastChecked: new Date(),
        limitations
      };

      console.log(`✅ Service ${serviceId} status:`, status);
      return status;
    } catch (error) {
      console.error(`❌ Error checking service ${serviceId}:`, error);
      
      return {
        serviceId,
        available: false,
        mode: 'free',
        hasApiKey: false,
        lastChecked: new Date()
      };
    }
  }

  /**
   * Check if API key exists for service
   */
  private async checkApiKeyExists(serviceId: string): Promise<boolean> {
    try {
      if (!this.vaultService.isVaultUnlocked()) {
        return false;
      }

      await this.vaultService.getKey(serviceId);
      return true;
    } catch (error) {
      return false;
    }
  }

  /**
   * Get service limitations from supported services
   */
  private async getServiceLimitations(serviceId: string): Promise<ServiceStatus['limitations']> {
    try {
      const service = await this.vaultService.getSupportedService(serviceId);
      
      // Parse limitations from service pricing/description
      return this.parseLimitationsFromService(service);
    } catch (error) {
      return undefined;
    }
  }

  /**
   * Parse limitations from service info
   */
  private parseLimitationsFromService(service: any): ServiceStatus['limitations'] {
    if (!service) return undefined;

    // Extract limitations from pricing info
    const limitations: ServiceStatus['limitations'] = {};

    if (service.pricing.includes('100')) {
      limitations.maxRequests = 100;
    }
    if (service.pricing.includes('1000')) {
      limitations.maxRequests = 1000;
    }

    // Categorize quality based on service type
    switch (service.id) {
      case 'google-search':
      case 'openai':
      case 'anthropic':
        limitations.quality = 'high';
        break;
      case 'weather-api':
      case 'news-api':
        limitations.quality = 'medium';
        break;
      default:
        limitations.quality = 'medium';
    }

    return limitations;
  }

  /**
   * Check if service is usable based on requirements
   */
  private isServiceUsable(fallback: ServiceFallback, status?: ServiceStatus): boolean {
    if (!status || !status.available) {
      return false;
    }

    // Check API key requirement
    if (fallback.requirements?.apiKey && !status.hasApiKey) {
      return false;
    }

    // Check mode compatibility
    if (fallback.mode === 'premium' && status.mode !== 'premium') {
      return false;
    }

    return true;
  }

  /**
   * Generate reasoning for service selection
   */
  private generateReasoning(
    fallback: ServiceFallback,
    status?: ServiceStatus,
    level: number = 0
  ): string {
    if (level === 0) {
      return `Using premium ${fallback.serviceId} service for best quality results`;
    }

    const reasons = [];
    
    if (fallback.fallbackReason) {
      reasons.push(fallback.fallbackReason);
    }

    if (status?.mode === 'free') {
      reasons.push('using free tier');
    }

    if (fallback.limitations?.quality) {
      reasons.push(`${fallback.limitations.quality} quality results`);
    }

    return `Fallback to ${fallback.serviceId} (${reasons.join(', ')})`;
  }

  /**
   * Get available upgrades from current service
   */
  private getAvailableUpgrades(
    hierarchy: ServiceFallback[],
    statuses: Map<string, ServiceStatus>,
    currentLevel: number
  ): FallbackSelection['availableUpgrades'] {
    const upgrades = [];

    // Check all higher priority services
    for (let i = 0; i < currentLevel; i++) {
      const service = hierarchy[i];
      const status = statuses.get(service.serviceId);

      if (!this.isServiceUsable(service, status)) {
        const benefits = this.getServiceBenefits(service, hierarchy[currentLevel]);
        upgrades.push({
          serviceId: service.serviceId,
          benefits,
          setupRequired: service.requirements?.apiKey === true && !status?.hasApiKey
        });
      }
    }

    return upgrades;
  }

  /**
   * Get benefits of upgrading to a service
   */
  private getServiceBenefits(targetService: ServiceFallback, currentService: ServiceFallback): string[] {
    const benefits = [];

    // Quality improvement
    if (targetService.limitations?.quality === 'high' && 
        currentService.limitations?.quality !== 'high') {
      benefits.push('Higher quality results');
    }

    // More results
    if (targetService.limitations?.maxResults && currentService.limitations?.maxResults &&
        targetService.limitations.maxResults > currentService.limitations.maxResults) {
      benefits.push(`More results (${targetService.limitations.maxResults} vs ${currentService.limitations.maxResults})`);
    }

    // More requests
    if (targetService.limitations?.maxRequests && currentService.limitations?.maxRequests &&
        targetService.limitations.maxRequests > currentService.limitations.maxRequests) {
      benefits.push(`Higher rate limits (${targetService.limitations.maxRequests} vs ${currentService.limitations.maxRequests})`);
    }

    // Additional features
    if (targetService.limitations?.features && currentService.limitations?.features) {
      const additionalFeatures = targetService.limitations.features
        .filter(f => !currentService.limitations!.features!.includes(f));
      
      if (additionalFeatures.length > 0) {
        benefits.push(`Additional features: ${additionalFeatures.join(', ')}`);
      }
    }

    // Service-specific benefits
    switch (targetService.serviceId) {
      case 'google-search':
        benefits.push('Google\'s comprehensive search index');
        break;
      case 'openai':
        benefits.push('Latest GPT models with advanced reasoning');
        break;
      case 'anthropic':
        benefits.push('Claude\'s superior analysis capabilities');
        break;
    }

    return benefits.length > 0 ? benefits : ['Premium service features'];
  }

  /**
   * Clear service status cache
   */
  clearCache(): void {
    this.serviceStatusCache.clear();
    console.log('🗑️ Service status cache cleared');
  }

  /**
   * Get cached service status
   */
  getCachedStatus(serviceId: string): ServiceStatus | undefined {
    return this.serviceStatusCache.get(serviceId);
  }
}