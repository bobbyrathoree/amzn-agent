import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../components/AuthProvider';
import { useApiClient } from '../lib/api';
import type { Bot } from '../types';

export function BotsPage() {
  const { user, signOut, getAccessToken } = useAuth();
  const apiClient = useApiClient(getAccessToken, () => user?.userId || user?.username || null);
  const [bots, setBots] = useState<Bot[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (apiClient) {
      loadBots();
    }
  }, [user]); // Only reload when user changes, not apiClient

  const loadBots = async () => {
    if (!apiClient) return;

    try {
      setLoading(true);
      setError(null);

      const response = await apiClient.get('bots');

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`Failed to load bots: ${response.status} ${errorText}`);
      }

      const data = await response.json();
      setBots(data.bots || []);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error';
      setError(errorMessage);
      console.error('Error loading bots:', err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center space-x-4">
              <Link to="/" className="text-xl font-bold text-gray-900">
                AI Chat Platform
              </Link>
              <span className="text-gray-500">|</span>
              <span className="text-gray-700">Bots</span>
            </div>
            <div>
              {user && (
                <div className="flex items-center space-x-4">
                  <span className="text-sm text-gray-700">Welcome, {user.username}</span>
                  <Link
                    to="/chat"
                    className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700 transition-colors"
                  >
                    Go to Chat
                  </Link>
                  <button
                    onClick={signOut}
                    className="bg-gray-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-gray-700 transition-colors"
                  >
                    Sign Out
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-2xl font-bold text-gray-900">My Bots</h1>
          <Link
            to="/bots/create"
            className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors"
          >
            Create New Bot
          </Link>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-100 border border-red-400 text-red-700 rounded">
            {error}
            <button
              onClick={loadBots}
              className="ml-4 text-red-800 underline hover:no-underline"
            >
              Retry
            </button>
          </div>
        )}

        {loading ? (
          <div className="text-center py-8">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
            <p className="mt-4 text-gray-600">Loading bots...</p>
          </div>
        ) : bots.length === 0 ? (
          <div className="text-center py-8">
            <div className="text-gray-400 mb-4">
              <svg className="mx-auto h-16 w-16" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
              </svg>
            </div>
            <h3 className="text-lg font-medium text-gray-900 mb-2">No bots yet</h3>
            <p className="text-gray-600 mb-4">Create your first bot to get started</p>
            <Link
              to="/bots/create"
              className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 transition-colors"
            >
              Create Your First Bot
            </Link>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {bots.map((bot) => (
              <div key={bot.id} className="bg-white rounded-lg shadow-md p-6 hover:shadow-lg transition-shadow">
                <h3 className="text-lg font-medium text-gray-900 mb-2">{bot.title}</h3>
                <p className="text-gray-600 text-sm mb-4 line-clamp-3">{bot.description}</p>
                
                <div className="flex items-center justify-between text-xs text-gray-500 mb-4">
                  <span>{bot.isPublic ? 'Public' : 'Private'}</span>
                  <span>{bot.activeModels?.[0]?.split('.')[1] || 'No model'}</span>
                </div>
                
                <div className="flex space-x-2">
                  <Link
                    to={`/chat?bot=${bot.id}`}
                    className="flex-1 bg-blue-600 text-white px-3 py-2 rounded text-sm text-center hover:bg-blue-700 transition-colors"
                  >
                    Chat
                  </Link>
                  <Link
                    to={`/bots/${bot.id}/edit`}
                    className="flex-1 bg-gray-200 text-gray-800 px-3 py-2 rounded text-sm text-center hover:bg-gray-300 transition-colors"
                  >
                    Edit
                  </Link>
                </div>
              </div>
            ))}
          </div>
        )}
      </main>
    </div>
  );
}