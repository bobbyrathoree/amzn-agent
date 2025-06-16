import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../components/AuthProvider';
import { useApiClient } from '../lib/api';
import type { BotSummary, Bot } from '../types';

interface DiscoverFilters {
  search: string;
  category: string;
  sharedScope: 'all' | 'public' | 'shared';
  sortBy: 'popular' | 'recent' | 'alphabetical';
}

export function DiscoverPage() {
  const { user, signOut, getAccessToken } = useAuth();
  const getUserId = useCallback(() => user?.userId || user?.username || null, [user?.userId, user?.username]);
  const apiClient = useApiClient(getAccessToken, getUserId);
  
  // State management
  const [publicBots, setPublicBots] = useState<BotSummary[]>([]);
  const [sharedBots, setSharedBots] = useState<BotSummary[]>([]);
  const [filteredBots, setFilteredBots] = useState<BotSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedBot, setSelectedBot] = useState<Bot | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  
  // Filter state
  const [filters, setFilters] = useState<DiscoverFilters>({
    search: '',
    category: '',
    sharedScope: 'all',
    sortBy: 'popular'
  });

  // Load discoverable bots
  useEffect(() => {
    if (apiClient) {
      loadDiscoverableBots();
    }
  }, [apiClient]);

  // Apply filters whenever filters change or bots update
  useEffect(() => {
    applyFilters();
  }, [filters, publicBots, sharedBots]);

  const loadDiscoverableBots = async () => {
    if (!apiClient) return;

    try {
      setLoading(true);
      setError(null);

      // Load public bots
      const publicResponse = await apiClient.get('bots?scope=public&limit=50');
      let publicBotsData = [];
      if (publicResponse.ok) {
        const data = await publicResponse.json();
        publicBotsData = data.bots || [];
      }

      // Load shared bots (bots shared with user)
      const sharedResponse = await apiClient.get('bots?scope=shared&limit=50');
      let sharedBotsData = [];
      if (sharedResponse.ok) {
        const data = await sharedResponse.json();
        sharedBotsData = data.bots || [];
      }

      setPublicBots(publicBotsData);
      setSharedBots(sharedBotsData);
    } catch (err) {
      console.error('Error loading discoverable bots:', err);
      setError(err instanceof Error ? err.message : 'Failed to load bots');
    } finally {
      setLoading(false);
    }
  };

  const applyFilters = () => {
    let allBots = [];
    
    // Combine bots based on scope filter
    switch (filters.sharedScope) {
      case 'public':
        allBots = [...publicBots];
        break;
      case 'shared':
        allBots = [...sharedBots];
        break;
      default:
        // Remove duplicates when combining
        const botIds = new Set();
        allBots = [...publicBots, ...sharedBots].filter(bot => {
          if (botIds.has(bot.id)) return false;
          botIds.add(bot.id);
          return true;
        });
    }

    // Apply search filter
    if (filters.search.trim()) {
      const searchTerm = filters.search.toLowerCase();
      allBots = allBots.filter(bot => 
        bot.title.toLowerCase().includes(searchTerm) ||
        bot.description.toLowerCase().includes(searchTerm)
      );
    }

    // Apply sorting
    switch (filters.sortBy) {
      case 'recent':
        allBots.sort((a, b) => new Date(b.lastUsedTime).getTime() - new Date(a.lastUsedTime).getTime());
        break;
      case 'alphabetical':
        allBots.sort((a, b) => a.title.localeCompare(b.title));
        break;
      case 'popular':
      default:
        // Sort by usage activity (assuming this reflects popularity)
        allBots.sort((a, b) => new Date(b.lastUsedTime).getTime() - new Date(a.lastUsedTime).getTime());
    }

    setFilteredBots(allBots);
  };

  const handleAddToMyBots = async (botId: string) => {
    if (!apiClient) return;

    try {
      const response = await apiClient.post(`bots/${botId}/access`);
      if (response.ok) {
        const responseData = await response.json();
        setSuccessMessage(responseData.message || 'Bot added to your collection successfully!');
        // Clear success message after 5 seconds
        setTimeout(() => setSuccessMessage(null), 5000);
        // Refresh the bots list to show updated access status
        loadDiscoverableBots();
      } else {
        const errorData = await response.json().catch(() => ({}));
        setError(errorData.error || 'Failed to add bot');
      }
    } catch (err) {
      console.error('Error adding bot:', err);
      setError('Failed to add bot to your collection');
    }
  };

  const loadBotDetails = async (botId: string) => {
    if (!apiClient) return;

    try {
      const response = await apiClient.get(`bots/${botId}`);
      if (response.ok) {
        const data = await response.json();
        setSelectedBot(data.bot);
      }
    } catch (err) {
      console.error('Error loading bot details:', err);
    }
  };

  const getBotStatusBadge = (bot: BotSummary) => {
    if (bot.sharedScope === 'public') {
      return <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">🌍 Public</span>;
    }
    if (bot.sharedScope === 'partial') {
      return <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800">👥 Shared</span>;
    }
    return <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800">🔒 Private</span>;
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center space-x-4">
              <Link to="/" className="text-xl font-bold text-gray-900">
                AI Bot Platform
              </Link>
              <span className="text-gray-500">|</span>
              <h1 className="text-xl font-semibold text-gray-900">Discover Bots</h1>
            </div>
            <div className="flex items-center space-x-4">
              <Link
                to="/bots"
                className="text-gray-600 hover:text-gray-900 transition-colors"
              >
                My Bots
              </Link>
              <Link
                to="/bots/create"
                className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700 transition-colors"
              >
                Create Bot
              </Link>
              <button
                onClick={signOut}
                className="bg-gray-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-gray-700 transition-colors"
              >
                Sign Out
              </button>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Search and Filters */}
        <div className="mb-8 bg-white rounded-lg shadow-sm p-6">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            {/* Search */}
            <div className="md:col-span-2">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Search Bots
              </label>
              <div className="relative">
                <input
                  type="text"
                  value={filters.search}
                  onChange={(e) => setFilters(prev => ({ ...prev, search: e.target.value }))}
                  placeholder="Search by name or description..."
                  className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
                <svg className="absolute left-3 top-2.5 h-5 w-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
              </div>
            </div>

            {/* Scope Filter */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Scope
              </label>
              <select
                value={filters.sharedScope}
                onChange={(e) => setFilters(prev => ({ ...prev, sharedScope: e.target.value as any }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              >
                <option value="all">All Bots</option>
                <option value="public">Public Only</option>
                <option value="shared">Shared Only</option>
              </select>
            </div>

            {/* Sort */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Sort By
              </label>
              <select
                value={filters.sortBy}
                onChange={(e) => setFilters(prev => ({ ...prev, sortBy: e.target.value as any }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              >
                <option value="popular">Popular</option>
                <option value="recent">Recently Used</option>
                <option value="alphabetical">Alphabetical</option>
              </select>
            </div>
          </div>
        </div>

        {/* Results Header */}
        <div className="flex justify-between items-center mb-6">
          <div>
            <h2 className="text-2xl font-bold text-gray-900">
              {filters.search ? `Search Results (${filteredBots.length})` : 'Available Bots'}
            </h2>
            <p className="text-gray-600 mt-1">
              Discover and chat with {filters.sharedScope === 'all' ? 'all' : filters.sharedScope} bots
            </p>
          </div>
        </div>

        {/* Loading State */}
        {loading && (
          <div className="text-center py-12">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
            <p className="mt-4 text-gray-600">Loading bots...</p>
          </div>
        )}

        {/* Success State */}
        {successMessage && (
          <div className="bg-green-50 border border-green-200 rounded-lg p-4 mb-6">
            <div className="flex items-center">
              <svg className="w-5 h-5 text-green-400 mr-3" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
              </svg>
              <div className="text-green-800">{successMessage}</div>
            </div>
          </div>
        )}

        {/* Error State */}
        {error && (
          <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
            <div className="text-red-800">{error}</div>
            <button
              onClick={loadDiscoverableBots}
              className="mt-2 text-red-600 hover:text-red-700 text-sm font-medium"
            >
              Try Again
            </button>
          </div>
        )}

        {/* Bot Grid */}
        {!loading && filteredBots.length === 0 ? (
          <div className="text-center py-12">
            <div className="text-gray-400 mb-4">
              <svg className="mx-auto h-16 w-16" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
            </div>
            <h3 className="text-lg font-medium text-gray-900 mb-2">
              {filters.search ? 'No bots found' : 'No discoverable bots available'}
            </h3>
            <p className="text-gray-600">
              {filters.search ? 'Try adjusting your search terms or filters' : 'Check back later for new bots'}
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
            {filteredBots.map((bot) => (
              <div key={bot.id} className="bg-white rounded-lg shadow-md hover:shadow-lg transition-shadow duration-300 overflow-hidden">
                {/* Bot Card Header */}
                <div className="p-6">
                  <div className="flex items-start justify-between mb-3">
                    <h3 className="text-lg font-semibold text-gray-900 line-clamp-2">{bot.title}</h3>
                    {getBotStatusBadge(bot)}
                  </div>
                  
                  <p className="text-gray-600 text-sm mb-4 line-clamp-3">{bot.description}</p>
                  
                  {/* Bot Metadata */}
                  <div className="space-y-2 mb-4">
                    <div className="flex items-center text-xs text-gray-500">
                      <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                      </svg>
                      by {bot.ownerUserId}
                    </div>
                    
                    {bot.hasKnowledgeBase && (
                      <div className="flex items-center text-xs text-gray-500">
                        <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.746 0 3.332.477 4.5 1.253v13C19.832 18.477 18.246 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
                        </svg>
                        Knowledge Base
                      </div>
                    )}
                    
                    <div className="flex items-center text-xs text-gray-500">
                      <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                      Updated {new Date(bot.lastUsedTime).toLocaleDateString()}
                    </div>
                  </div>

                  {/* Conversation Starters Preview */}
                  {bot.conversationStarters && bot.conversationStarters.length > 0 && (
                    <div className="mb-4">
                      <div className="text-xs font-medium text-gray-700 mb-2">Try asking:</div>
                      <div className="text-xs text-gray-600 bg-gray-50 rounded p-2 line-clamp-2">
                        "{bot.conversationStarters[0].example}"
                      </div>
                    </div>
                  )}
                </div>

                {/* Action Buttons */}
                <div className="border-t border-gray-100 px-6 py-4">
                  <div className="flex space-x-2">
                    <Link
                      to={`/bots/${bot.id}/chat`}
                      className="flex-1 bg-blue-600 text-white px-3 py-2 rounded-md text-sm text-center font-medium hover:bg-blue-700 transition-colors"
                    >
                      Chat Now
                    </Link>
                    <button
                      onClick={() => loadBotDetails(bot.id)}
                      className="bg-gray-100 text-gray-700 px-3 py-2 rounded-md text-sm font-medium hover:bg-gray-200 transition-colors"
                    >
                      Details
                    </button>
                    {/* Show Add to My Bots button only if user doesn't own the bot */}
                    {bot.ownerUserId !== (user?.userId || user?.username) && (
                      <button
                        onClick={() => handleAddToMyBots(bot.id)}
                        className="bg-green-600 text-white px-3 py-2 rounded-md text-sm font-medium hover:bg-green-700 transition-colors"
                      >
                        Add Bot
                      </button>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Bot Details Modal */}
      {selectedBot && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="p-6">
              <div className="flex items-start justify-between mb-4">
                <div>
                  <h2 className="text-2xl font-bold text-gray-900">{selectedBot.title}</h2>
                  <p className="text-gray-600 mt-1">by {selectedBot.ownerUserId}</p>
                </div>
                <button
                  onClick={() => setSelectedBot(null)}
                  className="text-gray-400 hover:text-gray-600 p-1"
                >
                  <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>

              <div className="space-y-6">
                <div>
                  <h3 className="text-lg font-medium text-gray-900 mb-2">Description</h3>
                  <p className="text-gray-600">{selectedBot.description}</p>
                </div>

                {selectedBot.conversationStarters && selectedBot.conversationStarters.length > 0 && (
                  <div>
                    <h3 className="text-lg font-medium text-gray-900 mb-3">Conversation Starters</h3>
                    <div className="space-y-2">
                      {selectedBot.conversationStarters.slice(0, 3).map((starter, index) => (
                        <div key={index} className="bg-gray-50 rounded-lg p-3">
                          <div className="font-medium text-gray-900">{starter.title}</div>
                          <div className="text-sm text-gray-600 mt-1">"{starter.example}"</div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                <div className="flex space-x-4 pt-4 border-t">
                  <Link
                    to={`/bots/${selectedBot.id}/chat`}
                    className="flex-1 bg-blue-600 text-white px-4 py-2 rounded-md text-center font-medium hover:bg-blue-700 transition-colors"
                  >
                    Start Chatting
                  </Link>
                  <button
                    onClick={() => setSelectedBot(null)}
                    className="flex-1 bg-gray-100 text-gray-700 px-4 py-2 rounded-md font-medium hover:bg-gray-200 transition-colors"
                  >
                    Close
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}