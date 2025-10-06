// 🔐 API KEY VAULT SERVICE
// Client-side encryption and vault management

export interface EncryptedKeyData {
  encryptedKey: number[];
  nonce: number[];
  serviceId: string;
}

export interface VaultKeyRequest {
  serviceId: string;
  keyName: string;
  encryptedKey: string;
  nonce: string;
  salt: string;
  expiresAt?: Date;
  description?: string;
  tags?: string[];
  metadata?: Record<string, any>;
}

export interface VaultKeyResponse {
  serviceId: string;
  keyName: string;
  encryptedKey: string;
  nonce: string;
  salt: string;
  createdAt: Date;
  expiresAt?: Date;
  lastUsed?: Date;
  usageCount: number;
  isActive: boolean;
  description?: string;
  tags?: string[];
  metadata?: Record<string, any>;
}

export interface SupportedService {
  id: string;
  name: string;
  category: string;
  description: string;
  website: string;
  pricing: string;
  icon: string;
  color: string;
  isActive: boolean;
  setupSteps: string[];
}

export interface ServiceCategory {
  id: string;
  name: string;
  description: string;
  icon: string;
}

export interface KeyTestResponse {
  serviceId: string;
  valid: boolean;
  message: string;
  details?: Record<string, any>;
  testedAt: Date;
}

/**
 * 🔐 API Key Vault Service
 * Handles client-side encryption and vault operations
 */
export class APIKeyVaultService {
  private cryptoKey: CryptoKey | null = null;
  private isUnlocked = false;
  private apiClient: any; // ApiClient instance

  constructor(apiClient: any) {
    this.apiClient = apiClient;
  }

  // 🔓 VAULT UNLOCK/LOCK

  /**
   * Unlock the vault with user credentials
   */
  async unlockVault(password: string, vaultPin: string): Promise<boolean> {
    try {
      console.log('🔓 Unlocking vault...');
      
      // Generate or retrieve user salt
      const salt = await this.getUserSalt();
      
      // Derive master key
      this.cryptoKey = await this.deriveMasterKey(password, vaultPin, salt);
      this.isUnlocked = true;
      
      console.log('✅ Vault unlocked successfully');
      return true;
    } catch (error) {
      console.error('❌ Failed to unlock vault:', error);
      return false;
    }
  }

  /**
   * Lock the vault and clear sensitive data
   */
  lockVault(): void {
    this.cryptoKey = null;
    this.isUnlocked = false;
    console.log('🔒 Vault locked');
  }

  /**
   * Check if vault is unlocked
   */
  isVaultUnlocked(): boolean {
    return this.isUnlocked && this.cryptoKey !== null;
  }

  // 🔑 KEY MANAGEMENT

  /**
   * Store an encrypted API key
   */
  async storeKey(serviceId: string, apiKey: string, options: {
    keyName?: string;
    expiresAt?: Date;
    description?: string;
    tags?: string[];
  } = {}): Promise<VaultKeyResponse> {
    if (!this.isVaultUnlocked()) {
      throw new Error('Vault is locked. Please unlock vault first.');
    }

    console.log(`🔐 Storing API key for service: ${serviceId}`);

    // Encrypt the API key
    const encryptedData = await this.encryptAPIKey(apiKey, serviceId);

    // Prepare request
    const request: VaultKeyRequest = {
      serviceId,
      keyName: options.keyName || `${serviceId} API Key`,
      encryptedKey: this.arrayToBase64(encryptedData.encryptedKey),
      nonce: this.arrayToBase64(encryptedData.nonce),
      salt: await this.getSaltString(),
      expiresAt: options.expiresAt,
      description: options.description,
      tags: options.tags,
      metadata: {
        createdBy: 'Foundry',
        version: '1.0'
      }
    };

    // Send to backend
    const response = await this.apiClient.post('vault/keys', request);
    
    if (!response.ok) {
      throw new Error(`Failed to store API key: ${response.status}`);
    }

    const result = await response.json();
    console.log(`✅ API key stored successfully for ${serviceId}`);
    
    return {
      ...result,
      createdAt: new Date(result.createdAt),
      expiresAt: result.expiresAt ? new Date(result.expiresAt) : undefined,
      lastUsed: result.lastUsed ? new Date(result.lastUsed) : undefined
    };
  }

  /**
   * Retrieve and decrypt an API key
   */
  async getKey(serviceId: string): Promise<string> {
    if (!this.isVaultUnlocked()) {
      throw new Error('Vault is locked. Please unlock vault first.');
    }

    console.log(`🔍 Retrieving API key for service: ${serviceId}`);

    // Get encrypted key from backend
    const response = await this.apiClient.get(`vault/keys/${serviceId}`);
    
    if (!response.ok) {
      if (response.status === 404) {
        throw new Error(`API key not found for service: ${serviceId}`);
      }
      throw new Error(`Failed to retrieve API key: ${response.status}`);
    }

    const keyData: VaultKeyResponse = await response.json();

    // Decrypt the key
    const encryptedData: EncryptedKeyData = {
      encryptedKey: this.base64ToArray(keyData.encryptedKey),
      nonce: this.base64ToArray(keyData.nonce),
      serviceId
    };

    const decryptedKey = await this.decryptAPIKey(encryptedData);
    console.log(`✅ API key retrieved and decrypted for ${serviceId}`);
    
    return decryptedKey;
  }

  /**
   * List all stored keys (metadata only)
   */
  async listKeys(): Promise<VaultKeyResponse[]> {
    console.log('📋 Listing stored API keys...');

    const response = await this.apiClient.get('vault/keys');
    
    if (!response.ok) {
      throw new Error(`Failed to list keys: ${response.status}`);
    }

    const result = await response.json();
    
    return result.keys.map((key: any) => ({
      ...key,
      createdAt: new Date(key.createdAt),
      expiresAt: key.expiresAt ? new Date(key.expiresAt) : undefined,
      lastUsed: key.lastUsed ? new Date(key.lastUsed) : undefined
    }));
  }

  /**
   * Update an existing API key
   */
  async updateKey(serviceId: string, newApiKey: string, options: {
    keyName?: string;
    expiresAt?: Date;
    description?: string;
    tags?: string[];
  } = {}): Promise<VaultKeyResponse> {
    if (!this.isVaultUnlocked()) {
      throw new Error('Vault is locked. Please unlock vault first.');
    }

    console.log(`🔄 Updating API key for service: ${serviceId}`);

    // Encrypt the new API key
    const encryptedData = await this.encryptAPIKey(newApiKey, serviceId);

    // Prepare request
    const request: VaultKeyRequest = {
      serviceId,
      keyName: options.keyName || `${serviceId} API Key`,
      encryptedKey: this.arrayToBase64(encryptedData.encryptedKey),
      nonce: this.arrayToBase64(encryptedData.nonce),
      salt: await this.getSaltString(),
      expiresAt: options.expiresAt,
      description: options.description,
      tags: options.tags
    };

    // Send to backend
    const response = await this.apiClient.put(`vault/keys/${serviceId}`, request);
    
    if (!response.ok) {
      throw new Error(`Failed to update API key: ${response.status}`);
    }

    const result = await response.json();
    console.log(`✅ API key updated successfully for ${serviceId}`);
    
    return {
      ...result,
      createdAt: new Date(result.createdAt),
      expiresAt: result.expiresAt ? new Date(result.expiresAt) : undefined,
      lastUsed: result.lastUsed ? new Date(result.lastUsed) : undefined
    };
  }

  /**
   * Delete an API key
   */
  async deleteKey(serviceId: string): Promise<void> {
    console.log(`🗑️ Deleting API key for service: ${serviceId}`);

    const response = await this.apiClient.delete(`vault/keys/${serviceId}`);
    
    if (!response.ok) {
      throw new Error(`Failed to delete API key: ${response.status}`);
    }

    console.log(`✅ API key deleted successfully for ${serviceId}`);
  }

  /**
   * Test an API key
   */
  async testKey(serviceId: string): Promise<KeyTestResponse> {
    console.log(`🧪 Testing API key for service: ${serviceId}`);

    try {
      // Step 1: Check backend validation (key exists and is active)
      const backendResponse = await this.apiClient.post(`vault/keys/${serviceId}/test`, {
        serviceId
      });
      
      if (!backendResponse.ok) {
        throw new Error(`Backend test failed: ${backendResponse.status}`);
      }

      const backendResult = await backendResponse.json();
      
      // If backend validation fails, return early
      if (!backendResult.valid) {
        return {
          serviceId,
          valid: false,
          message: backendResult.message,
          details: backendResult.details,
          testedAt: new Date(backendResult.testedAt)
        };
      }

      // Step 2: Frontend validation - decrypt key and test with actual service
      if (this.isVaultUnlocked()) {
        try {
          const { APIKeyTester } = await import('./apiKeyTester');
          const decryptedKey = await this.getKey(serviceId);
          const testResult = await APIKeyTester.testAPIKey(serviceId, decryptedKey);
          
          return {
            serviceId,
            valid: testResult.valid,
            message: testResult.message,
            details: {
              ...backendResult.details,
              frontendTest: testResult.success ? 'completed' : 'failed',
              actualValidation: testResult.valid,
              responseTime: testResult.responseTime,
              testDetails: testResult.details,
              error: testResult.error
            },
            testedAt: new Date()
          };
        } catch (decryptError) {
          console.warn('Frontend key testing failed:', decryptError);
          return {
            serviceId,
            valid: backendResult.valid,
            message: `${backendResult.message} (Frontend testing unavailable: ${decryptError instanceof Error ? decryptError.message : 'Unknown error'})`,
            details: {
              ...backendResult.details,
              frontendTest: 'failed',
              frontendError: decryptError instanceof Error ? decryptError.message : 'Unknown error'
            },
            testedAt: new Date(backendResult.testedAt)
          };
        }
      } else {
        // Vault is locked, return backend result only
        return {
          serviceId,
          valid: backendResult.valid,
          message: `${backendResult.message} (Full testing requires unlocked vault)`,
          details: {
            ...backendResult.details,
            frontendTest: 'skipped',
            reason: 'Vault is locked'
          },
          testedAt: new Date(backendResult.testedAt)
        };
      }
    } catch (error) {
      console.error(`❌ API key test failed for ${serviceId}:`, error);
      throw error;
    }
  }


  // 🔒 ENCRYPTION METHODS

  /**
   * Generate master key from user credentials
   */
  private async deriveMasterKey(password: string, vaultPin: string, salt: Uint8Array): Promise<CryptoKey> {
    const encoder = new TextEncoder();
    const keyMaterial = await crypto.subtle.importKey(
      'raw',
      encoder.encode(password + vaultPin),
      'PBKDF2',
      false,
      ['deriveKey']
    );

    return crypto.subtle.deriveKey(
      {
        name: 'PBKDF2',
        salt: salt,
        iterations: 100000,
        hash: 'SHA-256'
      },
      keyMaterial,
      { name: 'AES-GCM', length: 256 },
      false,
      ['encrypt', 'decrypt']
    );
  }

  /**
   * Encrypt API key for storage
   */
  private async encryptAPIKey(apiKey: string, serviceId: string): Promise<EncryptedKeyData> {
    if (!this.cryptoKey) throw new Error('Vault not unlocked');

    const encoder = new TextEncoder();
    const data = encoder.encode(apiKey);
    const nonce = crypto.getRandomValues(new Uint8Array(12));

    const encrypted = await crypto.subtle.encrypt(
      { name: 'AES-GCM', iv: nonce },
      this.cryptoKey,
      data
    );

    return {
      encryptedKey: Array.from(new Uint8Array(encrypted)),
      nonce: Array.from(nonce),
      serviceId
    };
  }

  /**
   * Decrypt API key for use
   */
  private async decryptAPIKey(encryptedData: EncryptedKeyData): Promise<string> {
    if (!this.cryptoKey) throw new Error('Vault not unlocked');

    const decrypted = await crypto.subtle.decrypt(
      { name: 'AES-GCM', iv: new Uint8Array(encryptedData.nonce) },
      this.cryptoKey,
      new Uint8Array(encryptedData.encryptedKey)
    );

    return new TextDecoder().decode(decrypted);
  }

  // 🛠️ UTILITY METHODS

  /**
   * Get or generate user salt from backend
   */
  private async getUserSalt(): Promise<Uint8Array> {
    console.log('🧂 Getting user salt from backend...');
    
    try {
      // Try to get salt from backend first
      const response = await this.apiClient.get('vault/salt');
      
      if (response.ok) {
        const data = await response.json();
        console.log('✅ Retrieved salt from backend');
        return this.base64ToUint8Array(data.salt);
      } else if (response.status === 404) {
        // Salt doesn't exist, create one
        console.log('🔧 Creating new salt...');
        return await this.createNewUserSalt();
      } else {
        console.error('❌ Failed to get salt from backend:', response.status);
        // Fallback to localStorage temporarily
        return await this.getLocalStorageSalt();
      }
    } catch (error) {
      console.error('❌ Error getting salt from backend:', error);
      // Fallback to localStorage temporarily
      return await this.getLocalStorageSalt();
    }
  }

  /**
   * Create a new user salt in the backend
   */
  private async createNewUserSalt(): Promise<Uint8Array> {
    // Generate new salt
    const salt = crypto.getRandomValues(new Uint8Array(32));
    const saltString = this.arrayToBase64(Array.from(salt));
    
    try {
      const response = await this.apiClient.post('vault/salt', {
        salt: saltString
      });
      
      if (response.ok) {
        console.log('✅ Created new salt in backend');
        // Remove localStorage salt if it exists
        localStorage.removeItem('vault_salt');
        return salt;
      } else {
        console.error('❌ Failed to create salt in backend:', response.status);
        throw new Error(`Failed to create salt: ${response.status}`);
      }
    } catch (error) {
      console.error('❌ Error creating salt in backend:', error);
      throw error;
    }
  }

  /**
   * Fallback to localStorage salt (temporary migration support)
   */
  private async getLocalStorageSalt(): Promise<Uint8Array> {
    console.log('⚠️ Using localStorage salt as fallback');
    
    const saltString = localStorage.getItem('vault_salt');
    
    if (saltString) {
      const salt = this.base64ToUint8Array(saltString);
      
      // Try to migrate to backend
      try {
        await this.migrateLocalSaltToBackend(saltString);
      } catch (error) {
        console.warn('Failed to migrate salt to backend:', error);
      }
      
      return salt;
    }
    
    // Generate new salt and store in localStorage as fallback
    const salt = crypto.getRandomValues(new Uint8Array(32));
    const newSaltString = this.arrayToBase64(Array.from(salt));
    localStorage.setItem('vault_salt', newSaltString);
    
    return salt;
  }

  /**
   * Migrate localStorage salt to backend
   */
  private async migrateLocalSaltToBackend(saltString: string): Promise<void> {
    console.log('🔄 Migrating localStorage salt to backend...');
    
    try {
      const response = await this.apiClient.post('vault/salt', {
        salt: saltString
      });
      
      if (response.ok) {
        console.log('✅ Successfully migrated salt to backend');
        localStorage.removeItem('vault_salt');
      } else if (response.status === 409) {
        console.log('ℹ️ Salt already exists in backend, removing localStorage');
        localStorage.removeItem('vault_salt');
      } else {
        console.error('❌ Failed to migrate salt:', response.status);
      }
    } catch (error) {
      console.error('❌ Error migrating salt:', error);
    }
  }

  private async getSaltString(): Promise<string> {
    const salt = await this.getUserSalt();
    return this.arrayToBase64(Array.from(salt));
  }

  private arrayToBase64(array: number[]): string {
    return btoa(String.fromCharCode(...array));
  }

  private base64ToArray(base64: string): number[] {
    return Array.from(atob(base64), c => c.charCodeAt(0));
  }

  private base64ToUint8Array(base64: string): Uint8Array {
    const binaryString = atob(base64);
    const bytes = new Uint8Array(binaryString.length);
    for (let i = 0; i < binaryString.length; i++) {
      bytes[i] = binaryString.charCodeAt(i);
    }
    return bytes;
  }

  // 🧂 SALT ROTATION METHODS

  /**
   * Rotate user salt (for enhanced security)
   */
  async rotateSalt(): Promise<boolean> {
    console.log('🔄 Starting salt rotation...');
    
    try {
      // Generate new salt
      const newSalt = crypto.getRandomValues(new Uint8Array(32));
      const newSaltString = this.arrayToBase64(Array.from(newSalt));
      
      // Update salt in backend
      const response = await this.apiClient.put('vault/salt', {
        salt: newSaltString
      });
      
      if (!response.ok) {
        throw new Error(`Failed to rotate salt: ${response.status}`);
      }
      
      console.log('✅ Salt rotation completed successfully');
      
      // Note: After salt rotation, existing API keys will need to be re-encrypted
      // This is intentional for security but should be communicated to the user
      
      return true;
    } catch (error) {
      console.error('❌ Salt rotation failed:', error);
      throw error;
    }
  }

  /**
   * Get salt information (for debugging/monitoring)
   */
  async getSaltInfo(): Promise<{
    version: number;
    createdAt: Date;
    updatedAt: Date;
  } | null> {
    try {
      const response = await this.apiClient.get('vault/salt');
      
      if (!response.ok) {
        return null;
      }
      
      const data = await response.json();
      
      return {
        version: data.version,
        createdAt: new Date(data.createdAt),
        updatedAt: new Date(data.updatedAt)
      };
    } catch (error) {
      console.error('❌ Failed to get salt info:', error);
      return null;
    }
  }

  // 🛠️ SUPPORTED SERVICES MANAGEMENT

  /**
   * Get all supported services
   */
  async getSupportedServices(): Promise<SupportedService[]> {
    try {
      console.log('🔍 Fetching supported services...');
      
      const response = await this.apiClient.get('vault/services');
      if (!response.ok) {
        throw new Error(`Failed to fetch supported services: ${response.status}`);
      }
      
      const data = await response.json();
      console.log(`✅ Loaded ${data.count} supported services`);
      
      return data.services || [];
    } catch (error) {
      console.error('❌ Failed to fetch supported services:', error);
      throw error;
    }
  }

  /**
   * Get service categories
   */
  async getServiceCategories(): Promise<ServiceCategory[]> {
    try {
      console.log('🔍 Fetching service categories...');
      
      const response = await this.apiClient.get('vault/services/categories');
      if (!response.ok) {
        throw new Error(`Failed to fetch service categories: ${response.status}`);
      }
      
      const data = await response.json();
      console.log(`✅ Loaded ${data.count} service categories`);
      
      return data.categories || [];
    } catch (error) {
      console.error('❌ Failed to fetch service categories:', error);
      throw error;
    }
  }

  /**
   * Get a specific supported service by ID
   */
  async getSupportedService(serviceId: string): Promise<SupportedService | null> {
    try {
      console.log(`🔍 Fetching service details for: ${serviceId}`);
      
      const response = await this.apiClient.get(`vault/services/${serviceId}`);
      if (!response.ok) {
        if (response.status === 404) {
          return null;
        }
        throw new Error(`Failed to fetch service ${serviceId}: ${response.status}`);
      }
      
      const service = await response.json();
      console.log(`✅ Loaded service details for: ${serviceId}`);
      
      return service;
    } catch (error) {
      console.error(`❌ Failed to fetch service ${serviceId}:`, error);
      throw error;
    }
  }

  /**
   * Get supported services by category
   */
  async getSupportedServicesByCategory(category: string): Promise<SupportedService[]> {
    try {
      const allServices = await this.getSupportedServices();
      return allServices.filter(service => service.category === category);
    } catch (error) {
      console.error(`❌ Failed to fetch services for category ${category}:`, error);
      throw error;
    }
  }

  /**
   * Check if a service is supported
   */
  async isServiceSupported(serviceId: string): Promise<boolean> {
    try {
      const service = await this.getSupportedService(serviceId);
      return service !== null && service.isActive;
    } catch (error) {
      console.error(`❌ Failed to check if service ${serviceId} is supported:`, error);
      return false;
    }
  }

  /**
   * Get service setup instructions
   */
  async getServiceSetupInstructions(serviceId: string): Promise<string[]> {
    try {
      const service = await this.getSupportedService(serviceId);
      return service?.setupSteps || [];
    } catch (error) {
      console.error(`❌ Failed to get setup instructions for ${serviceId}:`, error);
      return [];
    }
  }

  // 🧹 CLEANUP

  /**
   * Clear sensitive data from memory
   */
  clearSensitiveData(): void {
    this.lockVault();
    
    // Clear any cached data
    if ('caches' in window) {
      caches.keys().then(names => {
        names.forEach(name => {
          if (name.includes('vault')) {
            caches.delete(name);
          }
        });
      });
    }
  }
}