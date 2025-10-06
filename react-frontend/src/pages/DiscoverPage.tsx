import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  Search, 
  Filter, 
  Bot as BotIcon, 
  MessageCircle, 
  Plus, 
  User, 
  Book, 
  Clock, 
  X,
  Sparkles,
  Globe,
  Users,
  Lock,
  ArrowRight
} from 'lucide-react';
import { useAuth } from '../components/AuthProvider';
import { useApiClient } from '../lib/api';
import { ThemeToggle } from '../components/ThemeToggle';
import { GlassCard } from '../components/GlassCard';
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
      console.log('🔍 Loading public bots...');
      const publicResponse = await apiClient.get('bots?scope=public&limit=50');
      console.log('📡 Public API Response Status:', publicResponse.status, publicResponse.ok);
      
      let publicBotsData = [];
      if (publicResponse.ok) {
        const data = await publicResponse.json();
        console.log('📊 Public API Response Data:', data);
        publicBotsData = data.bots || [];
        console.log('🤖 Public Bots Found:', publicBotsData.length, publicBotsData);
      } else {
        console.error('❌ Public API Response Error:', publicResponse.status, publicResponse.statusText);
        const errorData = await publicResponse.text();
        console.error('❌ Error Details:', errorData);
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
    console.log('🔧 Applying filters...', filters);
    console.log('📦 Available bots - Public:', publicBots.length, 'Shared:', sharedBots.length);
    
    let allBots = [];
    
    // Combine bots based on scope filter
    switch (filters.sharedScope) {
      case 'public':
        allBots = [...publicBots];
        console.log('🌍 Filtering for public bots only:', allBots.length);
        break;
      case 'shared':
        allBots = [...sharedBots];
        console.log('👥 Filtering for shared bots only:', allBots.length);
        break;
      default:
        // Remove duplicates when combining
        const botIds = new Set();
        allBots = [...publicBots, ...sharedBots].filter(bot => {
          if (botIds.has(bot.id)) return false;
          botIds.add(bot.id);
          return true;
        });
        console.log('🔀 Combined all bots (after deduplication):', allBots.length);
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

    console.log('✅ Final filtered bots:', allBots.length, allBots);
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


  return (
    <div className="min-h-screen bg-background relative overflow-hidden">
      {/* Animated Background */}
      <div className="fixed inset-0 -z-10">
        <div className="absolute inset-0 bg-gradient-to-br from-background via-background to-background/95" />
        {Array.from({ length: 8 }, (_, i) => (
          <motion.div
            key={i}
            className="absolute w-2 h-2 bg-primary/20 rounded-full"
            initial={{
              x: Math.random() * (typeof window !== 'undefined' ? window.innerWidth : 1200),
              y: Math.random() * (typeof window !== 'undefined' ? window.innerHeight : 800),
            }}
            animate={{
              x: Math.random() * (typeof window !== 'undefined' ? window.innerWidth : 1200),
              y: Math.random() * (typeof window !== 'undefined' ? window.innerHeight : 800),
            }}
            transition={{
              duration: 20 + Math.random() * 20,
              repeat: Infinity,
              repeatType: "reverse",
              ease: "linear",
            }}
          />
        ))}
      </div>

      {/* Modern Header */}
      <motion.header 
        className="glass-card border-b border-border/50 relative z-10"
        initial={{ y: -100, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        transition={{ duration: 0.8 }}
      >
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-20">
            <motion.div 
              className="flex items-center space-x-6"
              initial={{ opacity: 0, x: -30 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.8, delay: 0.2 }}
            >
              <Link 
                to="/" 
                className="flex items-center gap-3 text-2xl font-bold bg-gradient-to-r from-primary to-primary/70 bg-clip-text text-transparent hover:from-primary/80 hover:to-primary/90 transition-all duration-300"
              >
                <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center">
                  <Sparkles className="w-5 h-5 text-white" />
                </div>
                Foundry
              </Link>
              <div className="h-8 w-px bg-border/50" />
              <h1 className="text-xl font-semibold text-foreground">Discover Bots</h1>
            </motion.div>
            
            <motion.div 
              className="flex items-center space-x-4"
              initial={{ opacity: 0, x: 30 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.8, delay: 0.4 }}
            >
              <Link
                to="/bots"
                className="glass-card px-4 py-2 text-sm font-medium text-foreground hover:text-primary transition-all duration-300 hover-lift"
              >
                My Bots
              </Link>
              <Link
                to="/bots/create"
                className="inline-flex items-center gap-2 bg-gradient-to-r from-blue-500 to-purple-600 text-white px-6 py-2 rounded-xl text-sm font-semibold hover:from-blue-600 hover:to-purple-700 transition-all duration-300 shadow-lg hover:shadow-xl hover-glow"
              >
                <Plus className="w-4 h-4" />
                Create Bot
              </Link>
              <ThemeToggle />
              <GlassCard
                as={motion.button}
                onClick={signOut}
                className="px-4 py-2 text-sm font-medium text-foreground hover:text-primary transition-all duration-300 hover-lift"
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
              >
                Sign Out
              </GlassCard>
            </motion.div>
          </div>
        </div>
      </motion.header>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 relative z-10">
        {/* Modern Search and Filters */}
        <motion.div 
          className="mb-12 glass-card p-8"
          initial={{ opacity: 0, y: 50 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8, delay: 0.6 }}
        >
          <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
            {/* Search */}
            <motion.div 
              className="md:col-span-2"
              initial={{ opacity: 0, x: -30 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.6, delay: 0.8 }}
            >
              <label className="block text-sm font-semibold text-foreground mb-3">
                Search Bots
              </label>
              <div className="relative">
                <input
                  type="text"
                  value={filters.search}
                  onChange={(e) => setFilters(prev => ({ ...prev, search: e.target.value }))}
                  placeholder="Search by name or description..."
                  className="w-full glass-card pl-12 pr-4 py-4 text-foreground placeholder-muted-foreground/60 focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300 text-base"
                />
                <Search className="absolute left-4 top-4 h-5 w-5 text-muted-foreground" />
              </div>
            </motion.div>

            {/* Scope Filter */}
            <motion.div
              initial={{ opacity: 0, x: -15 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.6, delay: 0.9 }}
            >
              <label className="block text-sm font-semibold text-foreground mb-3">
                <Filter className="inline w-4 h-4 mr-2" />
                Scope
              </label>
              <select
                value={filters.sharedScope}
                onChange={(e) => setFilters(prev => ({ ...prev, sharedScope: e.target.value as any }))}
                className="w-full glass-card px-4 py-4 text-foreground focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300 text-base"
              >
                <option value="all">All Bots</option>
                <option value="public">Public Only</option>
                <option value="shared">Shared Only</option>
              </select>
            </motion.div>

            {/* Sort */}
            <motion.div
              initial={{ opacity: 0, x: 15 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.6, delay: 1.0 }}
            >
              <label className="block text-sm font-semibold text-foreground mb-3">
                Sort By
              </label>
              <select
                value={filters.sortBy}
                onChange={(e) => setFilters(prev => ({ ...prev, sortBy: e.target.value as any }))}
                className="w-full glass-card px-4 py-4 text-foreground focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300 text-base"
              >
                <option value="popular">Popular</option>
                <option value="recent">Recently Used</option>
                <option value="alphabetical">Alphabetical</option>
              </select>
            </motion.div>
          </div>
        </motion.div>

        {/* Results Header */}
        <motion.div 
          className="flex justify-between items-center mb-8"
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 1.2 }}
        >
          <div>
            <h2 className="text-3xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent">
              {filters.search ? `Search Results (${filteredBots.length})` : 'Available Bots'}
            </h2>
            <p className="text-muted-foreground text-lg mt-2">
              Discover and chat with {filters.sharedScope === 'all' ? 'all' : filters.sharedScope} bots
            </p>
          </div>
        </motion.div>

        {/* Loading State */}
        <AnimatePresence>
        {loading && (
          <motion.div 
            className="text-center py-20"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.3 }}
          >
            <motion.div 
              className="w-16 h-16 border-4 border-primary/20 border-t-primary rounded-full mx-auto"
              animate={{ rotate: 360 }}
              transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
            />
            <motion.p 
              className="mt-6 text-muted-foreground text-lg"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.3 }}
            >
              Discovering amazing bots...
            </motion.p>
          </motion.div>
        )}
        </AnimatePresence>

        {/* Success State */}
        <AnimatePresence>
        {successMessage && (
          <motion.div 
            className="glass-card border border-green-400/30 bg-green-500/10 p-6 mb-8"
            initial={{ opacity: 0, y: -20, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -20, scale: 0.95 }}
            transition={{ duration: 0.3 }}
          >
            <div className="flex items-center gap-4 text-green-400">
              <div className="w-8 h-8 rounded-full bg-green-400/20 flex items-center justify-center">
                <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                </svg>
              </div>
              <div className="text-sm font-semibold">{successMessage}</div>
            </div>
          </motion.div>
        )}
        </AnimatePresence>

        {/* Error State */}
        <AnimatePresence>
        {error && (
          <motion.div 
            className="glass-card border border-red-400/30 bg-red-500/10 p-6 mb-8"
            initial={{ opacity: 0, y: -20, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -20, scale: 0.95 }}
            transition={{ duration: 0.3 }}
          >
            <div className="flex items-start gap-4">
              <div className="w-8 h-8 rounded-full bg-red-400/20 flex items-center justify-center flex-shrink-0">
                <X className="w-5 h-5 text-red-400" />
              </div>
              <div>
                <div className="text-red-400 font-medium">{error}</div>
                <motion.button
                  onClick={loadDiscoverableBots}
                  className="mt-3 text-red-400 hover:text-red-300 text-sm font-semibold underline transition-colors"
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                >
                  Try Again
                </motion.button>
              </div>
            </div>
          </motion.div>
        )}
        </AnimatePresence>

        {/* Bot Grid */}
        {!loading && filteredBots.length === 0 ? (
          <motion.div 
            className="text-center py-20"
            initial={{ opacity: 0, y: 50 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.8, delay: 0.3 }}
          >
            <motion.div 
              className="glass-card w-32 h-32 rounded-full mx-auto mb-8 flex items-center justify-center"
              initial={{ scale: 0 }}
              animate={{ scale: 1 }}
              transition={{ duration: 0.6, delay: 0.5, type: "spring", stiffness: 200 }}
            >
              <Search className="h-16 w-16 text-muted-foreground" />
            </motion.div>
            
            <motion.h3 
              className="text-2xl font-bold text-foreground mb-4"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.7 }}
            >
              {filters.search ? 'No bots found' : 'No discoverable bots available'}
            </motion.h3>
            
            <motion.p 
              className="text-muted-foreground text-lg max-w-md mx-auto"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.9 }}
            >
              {filters.search ? 'Try adjusting your search terms or filters' : 'Check back later for new bots'}
            </motion.p>
          </motion.div>
        ) : (
          <motion.div 
            className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-8"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.6, delay: 0.3 }}
          >
            {filteredBots.map((bot, index) => (
              <motion.div 
                key={bot.id} 
                className="glass-card p-6 hover-lift group cursor-pointer overflow-hidden"
                initial={{ opacity: 0, y: 30 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.6, delay: 0.5 + index * 0.1 }}
                whileHover={{ scale: 1.02, rotateY: 2 }}
                layout
              >
                {/* Bot Card Header */}
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center group-hover:scale-110 transition-transform duration-300">
                      <BotIcon className="h-6 w-6 text-white" />
                    </div>
                    <div>
                      <h3 className="text-lg font-semibold text-card-foreground group-hover:text-primary transition-colors duration-300 line-clamp-2">
                        {bot.title}
                      </h3>
                      <div className="flex items-center gap-2 mt-1">
                        {bot.sharedScope === 'public' ? (
                          <span className="bg-green-500/20 text-green-400 text-xs px-2 py-1 rounded-full font-medium flex items-center gap-1">
                            <Globe className="w-3 h-3" /> Public
                          </span>
                        ) : bot.sharedScope === 'partial' ? (
                          <span className="bg-blue-500/20 text-blue-400 text-xs px-2 py-1 rounded-full font-medium flex items-center gap-1">
                            <Users className="w-3 h-3" /> Shared
                          </span>
                        ) : (
                          <span className="bg-muted/50 text-muted-foreground text-xs px-2 py-1 rounded-full font-medium flex items-center gap-1">
                            <Lock className="w-3 h-3" /> Private
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
                  
                <p className="text-muted-foreground text-sm mb-6 line-clamp-3 leading-relaxed">{bot.description}</p>
                
                {/* Bot Metadata */}
                <div className="flex items-center justify-between text-xs mb-6">
                  <div className="flex items-center gap-2">
                    <User className="h-3 w-3 text-muted-foreground" />
                    <span className="text-muted-foreground font-medium">by {bot.ownerUserId}</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <Clock className="h-3 w-3 text-primary" />
                    <span className="text-muted-foreground font-mono text-xs">
                      {new Date(bot.lastUsedTime).toLocaleDateString()}
                    </span>
                  </div>
                </div>
                
                {bot.hasKnowledgeBase && (
                  <div className="flex items-center gap-2 mb-4">
                    <div className="w-6 h-6 rounded-lg bg-primary/20 flex items-center justify-center">
                      <Book className="w-3 h-3 text-primary" />
                    </div>
                    <span className="text-sm text-muted-foreground font-medium">Knowledge Base Available</span>
                  </div>
                )}

                {/* Conversation Starters Preview */}
                {bot.conversationStarters && bot.conversationStarters.length > 0 && (
                  <div className="mb-6">
                    <div className="text-xs font-semibold text-muted-foreground mb-3 flex items-center gap-2">
                      <MessageCircle className="w-3 h-3" />
                      Try asking:
                    </div>
                    <div className="glass-card bg-muted/5 p-3 text-xs text-muted-foreground leading-relaxed line-clamp-2 italic">
                      "{bot.conversationStarters[0].example}"
                    </div>
                  </div>
                )}

                {/* Modern Action Buttons */}
                <div className="border-t border-border/30 pt-6 mt-6">
                  <div className="grid grid-cols-3 gap-3">
                    {/* Chat Button - Always primary */}
                    <motion.div 
                      className="col-span-2"
                      whileHover={{ scale: 1.02 }} 
                      whileTap={{ scale: 0.98 }}
                    >
                      <Link
                        to={`/bots/${bot.id}/chat`}
                        className="w-full inline-flex items-center justify-center gap-2 bg-gradient-to-r from-blue-500 to-purple-600 text-white px-4 py-3 rounded-xl text-sm font-medium hover:from-blue-600 hover:to-purple-700 transition-all duration-300 shadow-md hover:shadow-lg hover-glow"
                      >
                        <MessageCircle className="h-4 w-4" />
                        <span className="hidden sm:inline">Chat Now</span>
                      </Link>
                    </motion.div>
                    
                    {/* Secondary Actions */}
                    <motion.div 
                      className="flex flex-col gap-1"
                      whileHover={{ scale: 1.05 }} 
                      whileTap={{ scale: 0.95 }}
                    >
                      <motion.button
                        onClick={() => loadBotDetails(bot.id)}
                        className="w-full glass-card border border-border/50 text-muted-foreground hover:text-primary hover:border-primary/50 px-2 py-2 rounded-lg text-xs font-medium transition-all duration-300"
                        title="View Details"
                      >
                        <span className="hidden sm:inline">Details</span>
                        <ArrowRight className="w-3 h-3 sm:hidden" />
                      </motion.button>
                      
                      {/* Add Bot button only if user doesn't own it */}
                      {bot.ownerUserId !== (user?.userId || user?.username) && (
                        <motion.button
                          onClick={() => handleAddToMyBots(bot.id)}
                          className="w-full bg-green-500/20 border border-green-500/30 text-green-400 hover:bg-green-500/30 hover:border-green-500/50 px-2 py-2 rounded-lg text-xs font-medium transition-all duration-300"
                          title="Add to My Bots"
                        >
                          <Plus className="w-3 h-3" />
                        </motion.button>
                      )}
                    </motion.div>
                  </div>
                </div>
              </motion.div>
            ))}
          </motion.div>
        )}
      </div>

      {/* Modern Bot Details Modal */}
      <AnimatePresence>
      {selectedBot && (
        <motion.div 
          className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 z-50"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.3 }}
          onClick={() => setSelectedBot(null)}
        >
          <motion.div 
            className="glass-card max-w-3xl w-full max-h-[90vh] overflow-y-auto scrollbar-thin"
            initial={{ scale: 0.9, opacity: 0, y: 50 }}
            animate={{ scale: 1, opacity: 1, y: 0 }}
            exit={{ scale: 0.9, opacity: 0, y: 50 }}
            transition={{ duration: 0.3 }}
            onClick={(e) => e.stopPropagation()}
          >
            <div className="p-8">
              {/* Modal Header */}
              <div className="flex items-start justify-between mb-8">
                <div className="flex items-center gap-4">
                  <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center">
                    <BotIcon className="h-8 w-8 text-white" />
                  </div>
                  <div>
                    <h2 className="text-3xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent">
                      {selectedBot.title}
                    </h2>
                    <p className="text-muted-foreground mt-2 flex items-center gap-2">
                      <User className="w-4 h-4" />
                      by {selectedBot.ownerUserId}
                    </p>
                  </div>
                </div>
                <motion.button
                  onClick={() => setSelectedBot(null)}
                  className="glass-card w-10 h-10 flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
                  whileHover={{ scale: 1.1 }}
                  whileTap={{ scale: 0.9 }}
                >
                  <X className="w-5 h-5" />
                </motion.button>
              </div>

              {/* Modal Content */}
              <div className="space-y-8">
                <motion.div
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: 0.1 }}
                >
                  <h3 className="text-xl font-semibold text-foreground mb-4 flex items-center gap-2">
                    <Book className="w-5 h-5 text-primary" />
                    Description
                  </h3>
                  <p className="text-muted-foreground leading-relaxed text-lg">{selectedBot.description}</p>
                </motion.div>

                {selectedBot.conversationStarters && selectedBot.conversationStarters.length > 0 && (
                  <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.2 }}
                  >
                    <h3 className="text-xl font-semibold text-foreground mb-6 flex items-center gap-2">
                      <MessageCircle className="w-5 h-5 text-primary" />
                      Conversation Starters
                    </h3>
                    <div className="grid gap-4">
                      {selectedBot.conversationStarters.slice(0, 3).map((starter: any, index: number) => (
                        <motion.div 
                          key={index} 
                          className="glass-card p-4 hover-lift cursor-pointer group"
                          initial={{ opacity: 0, x: -20 }}
                          animate={{ opacity: 1, x: 0 }}
                          transition={{ delay: 0.3 + index * 0.1 }}
                          whileHover={{ scale: 1.02 }}
                        >
                          <div className="font-semibold text-foreground group-hover:text-primary transition-colors mb-2">
                            {starter.title}
                          </div>
                          <div className="text-sm text-muted-foreground italic leading-relaxed">
                            "{starter.example}"
                          </div>
                        </motion.div>
                      ))}
                    </div>
                  </motion.div>
                )}

                {/* Modal Actions */}
                <motion.div 
                  className="flex gap-4 pt-8 border-t border-border/30"
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: 0.4 }}
                >
                  <Link
                    to={`/bots/${selectedBot.id}/chat`}
                    className="flex-1 inline-flex items-center justify-center gap-3 bg-gradient-to-r from-blue-500 to-purple-600 text-white px-8 py-4 rounded-xl text-lg font-semibold hover:from-blue-600 hover:to-purple-700 transition-all duration-300 shadow-lg hover:shadow-xl hover-glow"
                  >
                    <MessageCircle className="w-5 h-5" />
                    Start Chatting
                  </Link>
                  <motion.button
                    onClick={() => setSelectedBot(null)}
                    className="flex-1 glass-card px-8 py-4 text-lg font-semibold text-foreground hover:text-primary transition-all duration-300 hover-lift"
                    whileHover={{ scale: 1.02 }}
                    whileTap={{ scale: 0.98 }}
                  >
                    Close
                  </motion.button>
                </motion.div>
              </div>
            </div>
          </motion.div>
        </motion.div>
      )}
      </AnimatePresence>
    </div>
  );
}