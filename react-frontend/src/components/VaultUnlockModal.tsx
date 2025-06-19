// 🔓 VAULT UNLOCK MODAL
// Secure vault authentication interface

import { useState } from 'react';
import { XMarkIcon, EyeIcon, EyeSlashIcon, LockClosedIcon } from '@heroicons/react/24/outline';

interface VaultUnlockModalProps {
  isOpen: boolean;
  onClose: () => void;
  onUnlock: (password: string, vaultPin: string) => Promise<boolean>;
}

export function VaultUnlockModal({ isOpen, onClose, onUnlock }: VaultUnlockModalProps) {
  const [password, setPassword] = useState('');
  const [vaultPin, setVaultPin] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showPin, setShowPin] = useState(false);
  const [error, setError] = useState('');
  const [isUnlocking, setIsUnlocking] = useState(false);

  const handleUnlock = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!password.trim() || !vaultPin.trim()) {
      setError('Please enter both password and vault PIN');
      return;
    }

    setIsUnlocking(true);
    setError('');

    try {
      const success = await onUnlock(password, vaultPin);
      
      if (success) {
        // Clear form and close
        setPassword('');
        setVaultPin('');
        onClose();
      } else {
        setError('Invalid credentials. Please try again.');
      }
    } catch (err) {
      setError('Failed to unlock vault. Please try again.');
    } finally {
      setIsUnlocking(false);
    }
  };

  const handleClose = () => {
    setPassword('');
    setVaultPin('');
    setError('');
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md mx-4">
        {/* Header */}
        <div className="flex justify-between items-center mb-6">
          <div className="flex items-center space-x-3">
            <div className="p-2 bg-blue-100 dark:bg-blue-900 rounded-lg">
              <LockClosedIcon className="w-6 h-6 text-blue-600 dark:text-blue-400" />
            </div>
            <div>
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                Unlock Vault
              </h2>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Enter your credentials to access stored API keys
              </p>
            </div>
          </div>
          <button
            onClick={handleClose}
            className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            disabled={isUnlocking}
          >
            <XMarkIcon className="w-6 h-6" />
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleUnlock} className="space-y-4">
          {/* Password Field */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Account Password
            </label>
            <div className="relative">
              <input
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-3 py-2 pr-10 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                placeholder="Enter your account password"
                disabled={isUnlocking}
                autoComplete="current-password"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                disabled={isUnlocking}
              >
                {showPassword ? (
                  <EyeSlashIcon className="w-5 h-5" />
                ) : (
                  <EyeIcon className="w-5 h-5" />
                )}
              </button>
            </div>
          </div>

          {/* Vault PIN Field */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Vault PIN
            </label>
            <div className="relative">
              <input
                type={showPin ? 'text' : 'password'}
                value={vaultPin}
                onChange={(e) => setVaultPin(e.target.value)}
                className="w-full px-3 py-2 pr-10 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                placeholder="Enter your 4-6 digit vault PIN"
                disabled={isUnlocking}
                maxLength={6}
              />
              <button
                type="button"
                onClick={() => setShowPin(!showPin)}
                className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                disabled={isUnlocking}
              >
                {showPin ? (
                  <EyeSlashIcon className="w-5 h-5" />
                ) : (
                  <EyeIcon className="w-5 h-5" />
                )}
              </button>
            </div>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Your vault PIN provides an extra layer of security for API keys
            </p>
          </div>

          {/* Error Message */}
          {error && (
            <div className="p-3 bg-red-50 dark:bg-red-900/50 border border-red-200 dark:border-red-800 rounded-lg">
              <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
            </div>
          )}

          {/* Security Notice */}
          <div className="p-3 bg-blue-50 dark:bg-blue-900/50 border border-blue-200 dark:border-blue-800 rounded-lg">
            <p className="text-xs text-blue-600 dark:text-blue-400">
              🔒 Your credentials are used locally to decrypt stored API keys. 
              They are never sent to our servers.
            </p>
          </div>

          {/* Actions */}
          <div className="flex space-x-3 pt-4">
            <button
              type="button"
              onClick={handleClose}
              className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 font-medium"
              disabled={isUnlocking}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-medium disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center"
              disabled={isUnlocking || !password.trim() || !vaultPin.trim()}
            >
              {isUnlocking ? (
                <>
                  <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin mr-2" />
                  Unlocking...
                </>
              ) : (
                'Unlock Vault'
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}