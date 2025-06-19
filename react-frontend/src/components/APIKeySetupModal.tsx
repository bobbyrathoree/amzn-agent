// 🔑 API KEY SETUP MODAL
// Beautiful, user-friendly API key configuration interface

import { useState, useEffect } from 'react';
import { XMarkIcon, KeyIcon, CheckCircleIcon, ArrowTopRightOnSquareIcon } from '@heroicons/react/24/outline';
import type { SupportedService, KeyTestResponse } from '../services/vaultService';
import { TestResultIndicator } from './TestResultIndicator';

interface APIKeySetupModalProps {
  isOpen: boolean;
  onClose: () => void;
  serviceId: string;
  service?: SupportedService;
  onKeyAdded: () => void;
  vaultService: any; // APIKeyVaultService instance
}

export function APIKeySetupModal({ 
  isOpen, 
  onClose, 
  serviceId, 
  service, 
  onKeyAdded, 
  vaultService 
}: APIKeySetupModalProps) {
  const [apiKey, setApiKey] = useState('');
  const [keyName, setKeyName] = useState('');
  const [description, setDescription] = useState('');
  const [showKey, setShowKey] = useState(false);
  const [isStoring, setIsStoring] = useState(false);
  const [isTesting, setIsTesting] = useState(false);
  const [testResult, setTestResult] = useState<KeyTestResponse | null>(null);
  const [error, setError] = useState('');
  const [step, setStep] = useState<'instructions' | 'input' | 'success'>('instructions');

  useEffect(() => {
    if (service) {
      setKeyName(`${service.name} API Key`);
      setDescription(`API key for ${service.name} integration`);
    }
  }, [service]);

  const handleNext = () => {
    setStep('input');
  };

  const handleTestKey = async () => {
    if (!apiKey.trim()) {
      setError('Please enter an API key first');
      return;
    }

    setIsTesting(true);
    setError('');
    setTestResult(null);

    try {
      // First store the key temporarily to test it
      await vaultService.storeKey(serviceId, apiKey, {
        keyName,
        description
      });

      // Then test it
      const result = await vaultService.testKey(serviceId);
      setTestResult(result);

      if (result.valid) {
        setStep('success');
        onKeyAdded();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to test API key');
      setTestResult({
        serviceId,
        valid: false,
        message: 'Key test failed',
        testedAt: new Date()
      });
    } finally {
      setIsTesting(false);
    }
  };

  const handleStoreWithoutTest = async () => {
    if (!apiKey.trim()) {
      setError('Please enter an API key');
      return;
    }

    setIsStoring(true);
    setError('');

    try {
      await vaultService.storeKey(serviceId, apiKey, {
        keyName,
        description
      });

      setStep('success');
      onKeyAdded();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to store API key');
    } finally {
      setIsStoring(false);
    }
  };

  const handleClose = () => {
    setApiKey('');
    setKeyName(service ? `${service.name} API Key` : '');
    setDescription('');
    setError('');
    setTestResult(null);
    setStep('instructions');
    onClose();
  };

  if (!isOpen || !service) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-2xl mx-4 max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="flex justify-between items-start mb-6">
          <div className="flex items-center space-x-3">
            <div className="p-2 rounded-lg" style={{ backgroundColor: service.color + '20' }}>
              <span className="text-2xl">{service.icon}</span>
            </div>
            <div>
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                Setup {service.name}
              </h2>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                {service.description}
              </p>
            </div>
          </div>
          <button
            onClick={handleClose}
            className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
          >
            <XMarkIcon className="w-6 h-6" />
          </button>
        </div>

        {/* Instructions Step */}
        {step === 'instructions' && (
          <div className="space-y-6">
            {/* Service Info */}
            <div className="bg-gray-50 dark:bg-gray-700 rounded-lg p-4">
              <div className="flex justify-between items-center mb-3">
                <h3 className="font-medium text-gray-900 dark:text-white">Service Information</h3>
                <a
                  href={service.website}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-600 hover:text-blue-700 dark:text-blue-400 flex items-center text-sm"
                >
                  Visit Website
                  <ArrowTopRightOnSquareIcon className="w-4 h-4 ml-1" />
                </a>
              </div>
              <p className="text-sm text-gray-600 dark:text-gray-300 mb-2">
                {service.description}
              </p>
              <p className="text-sm text-green-600 dark:text-green-400 font-medium">
                💰 {service.pricing}
              </p>
            </div>

            {/* Setup Instructions */}
            <div>
              <h3 className="font-medium text-gray-900 dark:text-white mb-3">
                How to get your API key:
              </h3>
              <ol className="space-y-2">
                {service.setupSteps.map((instruction: string, index: number) => (
                  <li key={index} className="flex items-start space-x-3">
                    <span className="flex-shrink-0 w-6 h-6 bg-blue-100 dark:bg-blue-900 text-blue-600 dark:text-blue-400 rounded-full flex items-center justify-center text-sm font-medium">
                      {index + 1}
                    </span>
                    <span className="text-sm text-gray-600 dark:text-gray-300">
                      {instruction}
                    </span>
                  </li>
                ))}
              </ol>
            </div>

            {/* Sign Up Link */}
            {service.website && (
              <div className="bg-blue-50 dark:bg-blue-900/50 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
                <p className="text-sm text-blue-700 dark:text-blue-300 mb-2">
                  Need to create an account?
                </p>
                <a
                  href={service.website}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center text-blue-600 hover:text-blue-700 dark:text-blue-400 font-medium text-sm"
                >
                  Sign up for {service.name}
                  <ArrowTopRightOnSquareIcon className="w-4 h-4 ml-1" />
                </a>
              </div>
            )}

            {/* Actions */}
            <div className="flex space-x-3 pt-4 border-t border-gray-200 dark:border-gray-700">
              <button
                onClick={handleClose}
                className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 font-medium"
              >
                Cancel
              </button>
              <button
                onClick={handleNext}
                className="flex-1 px-4 py-2 text-white rounded-lg font-medium"
                style={{ backgroundColor: service.color }}
              >
                I have my API key
              </button>
            </div>
          </div>
        )}

        {/* Key Input Step */}
        {step === 'input' && (
          <div className="space-y-6">
            {/* Key Input */}
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                API Key
              </label>
              <div className="relative">
                <KeyIcon className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
                <input
                  type={showKey ? 'text' : 'password'}
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  className="w-full pl-10 pr-20 py-3 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                  placeholder={`Enter your ${service.name} API key`}
                />
                <button
                  type="button"
                  onClick={() => setShowKey(!showKey)}
                  className="absolute right-3 top-1/2 transform -translate-y-1/2 text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400"
                >
                  {showKey ? 'Hide' : 'Show'}
                </button>
              </div>
            </div>

            {/* Key Name */}
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Key Name (Optional)
              </label>
              <input
                type="text"
                value={keyName}
                onChange={(e) => setKeyName(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                placeholder="Give this key a name"
              />
            </div>

            {/* Description */}
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Description (Optional)
              </label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={2}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                placeholder="Brief description of this key's purpose"
              />
            </div>

            {/* Test Result */}
            {testResult && (
              <TestResultIndicator result={testResult} isLoading={isTesting} />
            )}

            {/* Error */}
            {error && (
              <div className="p-3 bg-red-50 dark:bg-red-900/50 border border-red-200 dark:border-red-800 rounded-lg">
                <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
              </div>
            )}

            {/* Actions */}
            <div className="flex space-x-3 pt-4 border-t border-gray-200 dark:border-gray-700">
              <button
                onClick={() => setStep('instructions')}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 font-medium"
              >
                Back
              </button>
              <button
                onClick={handleStoreWithoutTest}
                disabled={!apiKey.trim() || isStoring}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 font-medium disabled:opacity-50"
              >
                {isStoring ? 'Storing...' : 'Store Key'}
              </button>
              <button
                onClick={handleTestKey}
                disabled={!apiKey.trim() || isTesting}
                className="flex-1 px-4 py-2 text-white rounded-lg font-medium disabled:opacity-50 flex items-center justify-center"
                style={{ backgroundColor: service.color }}
              >
                {isTesting ? (
                  <>
                    <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin mr-2" />
                    Testing...
                  </>
                ) : (
                  'Test & Store Key'
                )}
              </button>
            </div>
          </div>
        )}

        {/* Success Step */}
        {step === 'success' && (
          <div className="text-center space-y-6">
            <div className="w-16 h-16 bg-green-100 dark:bg-green-900 rounded-full flex items-center justify-center mx-auto">
              <CheckCircleIcon className="w-8 h-8 text-green-600 dark:text-green-400" />
            </div>
            
            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
                API Key Added Successfully!
              </h3>
              <p className="text-gray-600 dark:text-gray-300">
                Your {service.name} API key has been encrypted and stored securely. 
                You can now use advanced features powered by {service.name}.
              </p>
            </div>

            <div className="bg-green-50 dark:bg-green-900/50 border border-green-200 dark:border-green-800 rounded-lg p-4">
              <p className="text-sm text-green-700 dark:text-green-300">
                🔒 Your API key is encrypted with AES-256-GCM and stored securely. 
                Only you can decrypt and use it.
              </p>
            </div>

            <button
              onClick={handleClose}
              className="px-6 py-2 text-white rounded-lg font-medium"
              style={{ backgroundColor: service.color }}
            >
              Continue
            </button>
          </div>
        )}
      </div>
    </div>
  );
}