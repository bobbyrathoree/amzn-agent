import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../components/AuthProvider';
import { ThemeToggle } from '../components/ThemeToggle';
import { GlassCard } from '../components/GlassCard';
import { useApiClient } from '../lib/api';
import type { BotSummary } from '../types';
import { motion, AnimatePresence } from 'framer-motion';
import { Bot, MessageCircle, Edit3, Trash2, Plus, Filter, Sparkles, Zap } from 'lucide-react';

export function BotsPage() {
  const { user, signOut, getAccessToken } = useAuth();
  const getUserId = useCallback(() => user?.userId || user?.username || null, [user?.userId, user?.username]);
  const apiClient = useApiClient(getAccessToken, getUserId);
  const [bots, setBots] = useState<BotSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deletingBotId, setDeletingBotId] = useState<string | null>(null);

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

  const deleteBot = async (botId: string, botTitle: string) => {
    if (!apiClient) return;
    
    const confirmed = window.confirm(`Are you sure you want to delete "${botTitle}"? This action cannot be undone.`);
    if (!confirmed) return;

    try {
      setDeletingBotId(botId);
      setError(null);

      const response = await apiClient.delete(`bots/${botId}`);
      
      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`Failed to delete bot: ${response.status} ${errorText}`);
      }

      // Remove the bot from the local state
      setBots(prevBots => prevBots.filter(bot => bot.id !== botId));
      
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error';
      setError(`Failed to delete bot: ${errorMessage}`);
      console.error('Error deleting bot:', err);
    } finally {
      setDeletingBotId(null);
    }
  };

  return (
    <div className="min-h-screen bg-background text-foreground overflow-hidden">
      {/* Animated Background */}
      <div className="fixed inset-0 gradient-mesh opacity-30" />
      <div className="fixed inset-0">
        {[...Array(15)].map((_, i) => (
          <motion.div
            key={i}
            className="absolute w-1 h-1 bg-primary/30 rounded-full"
            animate={{
              x: [0, Math.random() * 50 - 25],
              y: [0, Math.random() * 50 - 25],
              scale: [1, Math.random() * 0.5 + 0.5, 1],
            }}
            transition={{
              duration: Math.random() * 15 + 15,
              repeat: Infinity,
              ease: "linear",
            }}
            style={{
              left: `${Math.random() * 100}%`,
              top: `${Math.random() * 100}%`,
            }}
          />
        ))}
      </div>
      
      {/* Header */}
      <motion.header 
        className="relative z-10 glass-card border-b border-glass-border"
        initial={{ y: -100, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        transition={{ duration: 0.8, ease: "easeOut" }}
      >
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center space-x-4">
              <motion.div 
                className="flex items-center gap-3"
                whileHover={{ scale: 1.05 }}
                transition={{ type: "spring", stiffness: 400, damping: 10 }}
              >
                <Link to="/" className="flex items-center gap-3">
                  <div className="p-2 rounded-lg gradient-primary">
                    <Sparkles className="h-5 w-5 text-white" />
                  </div>
                  <span className="text-xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent">
                    Foundry
                  </span>
                </Link>
              </motion.div>
              <span className="text-muted-foreground">•</span>
              <div className="flex items-center gap-2">
                <Bot className="h-5 w-5 text-primary" />
                <span className="text-lg font-semibold text-foreground">My Bots</span>
              </div>
            </div>
            <div>
              {user && (
                <div className="flex items-center space-x-4">
                  <span className="text-sm text-muted-foreground">Welcome, {user.username}</span>
                  <ThemeToggle />
                  <motion.button
                    onClick={signOut}
                    className="bg-secondary text-secondary-foreground px-4 py-2 rounded-lg text-sm font-medium hover:bg-accent hover:text-accent-foreground transition-all duration-300 hover-lift"
                    whileHover={{ scale: 1.05 }}
                    whileTap={{ scale: 0.95 }}
                  >
                    Sign Out
                  </motion.button>
                </div>
              )}
            </div>
          </div>
        </div>
      </motion.header>

      {/* Main Content */}
      <main className="relative z-10 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <motion.div 
          className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-6 mb-12"
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.3 }}
        >
          <div>
            <motion.h1 
              className="text-4xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent mb-2"
              initial={{ opacity: 0, x: -30 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.6, delay: 0.5 }}
            >
              My AI Bots
            </motion.h1>
            <motion.p 
              className="text-muted-foreground text-lg"
              initial={{ opacity: 0, x: -30 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.6, delay: 0.7 }}
            >
              Manage and chat with your AI assistants
            </motion.p>
          </div>
          
          <motion.div 
            className="flex gap-4"
            initial={{ opacity: 0, x: 30 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.6, delay: 0.9 }}
          >
            <GlassCard
              as={motion.button}
              className="px-4 py-2 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors hover-lift flex items-center gap-2"
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              <Filter className="h-4 w-4" />
              Filter
            </GlassCard>
            
            <motion.div
              whileHover={{ scale: 1.05, y: -2 }}
              whileTap={{ scale: 0.95 }}
            >
              <Link
                to="/bots/create"
                className="inline-flex items-center gap-3 px-6 py-3 bg-gradient-to-r from-blue-500 to-purple-600 text-white font-semibold rounded-xl hover:from-blue-600 hover:to-purple-700 transition-all duration-300 shadow-lg hover:shadow-xl hover-glow"
              >
                <Plus className="h-5 w-5" />
                Create New Bot
              </Link>
            </motion.div>
          </motion.div>
        </motion.div>

        <AnimatePresence>
          {error && (
            <motion.div 
              className="mb-8 glass-card p-6 border border-red-400/20 bg-red-500/5 backdrop-blur-sm"
              initial={{ opacity: 0, y: -20, scale: 0.95 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: -20, scale: 0.95 }}
              transition={{ duration: 0.3 }}
            >
              <div className="flex items-center gap-3 text-red-400">
                <Zap className="h-5 w-5" />
                <span className="font-medium">Something went wrong</span>
              </div>
              <p className="text-red-300 mt-2">{error}</p>
              <motion.button
                onClick={loadBots}
                className="mt-4 px-4 py-2 bg-red-500/20 text-red-300 rounded-lg hover:bg-red-500/30 transition-colors"
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
              >
                Try Again
              </motion.button>
            </motion.div>
          )}
        </AnimatePresence>

        {loading ? (
          <motion.div 
            className="text-center py-16"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
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
              Loading your AI bots...
            </motion.p>
          </motion.div>
        ) : bots.length === 0 ? (
          <motion.div 
            className="text-center py-20"
            initial={{ opacity: 0, y: 50 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.8 }}
          >
            <motion.div
              className="glass-card w-32 h-32 rounded-full mx-auto mb-8 flex items-center justify-center"
              initial={{ scale: 0 }}
              animate={{ scale: 1 }}
              transition={{ duration: 0.6, delay: 0.3, type: "spring", stiffness: 200 }}
            >
              <Bot className="h-16 w-16 text-muted-foreground" />
            </motion.div>

            <motion.h3 
              className="text-2xl font-bold text-foreground mb-4"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.5 }}
            >
              No bots yet
            </motion.h3>
            
            <motion.p 
              className="text-muted-foreground text-lg mb-8 max-w-md mx-auto"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.7 }}
            >
              Create your first AI bot to get started on your journey
            </motion.p>
            
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6, delay: 0.9 }}
              whileHover={{ scale: 1.05, y: -2 }}
              whileTap={{ scale: 0.95 }}
            >
              <Link
                to="/bots/create"
                className="inline-flex items-center gap-3 px-8 py-4 bg-gradient-to-r from-blue-500 to-purple-600 text-white font-semibold rounded-xl hover:from-blue-600 hover:to-purple-700 transition-all duration-300 shadow-lg hover:shadow-xl hover-glow"
              >
                <Plus className="h-5 w-5" />
                Create Your First Bot
              </Link>
            </motion.div>
          </motion.div>
        ) : (
          <motion.div 
            className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-8"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.6, delay: 0.3 }}
          >
            {bots.map((bot, index) => (
              <GlassCard 
                as={motion.div} 
                key={bot.id} 
                className="p-6 hover-lift group cursor-pointer"
                initial={{ opacity: 0, y: 30 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.6, delay: 0.5 + index * 0.1 }}
                whileHover={{ scale: 1.02, rotateY: 2 }}
                layout
              >
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center group-hover:scale-110 transition-transform duration-300">
                      <Bot className="h-6 w-6 text-white" />
                    </div>
                    <div>
                      <h3 className="text-lg font-semibold text-card-foreground group-hover:text-primary transition-colors duration-300">
                        {bot.title}
                      </h3>
                      <div className="flex items-center gap-2 mt-1">
                        {bot.ownerUserId === (user?.userId || user?.username) ? (
                          <span className="bg-blue-500/20 text-blue-400 text-xs px-2 py-1 rounded-full font-medium">Owned</span>
                        ) : (
                          <span className="bg-muted/50 text-muted-foreground text-xs px-2 py-1 rounded-full font-medium">Shared</span>
                        )}
                      </div>
                    </div>
                  </div>
                </div>

                <p className="text-muted-foreground text-sm mb-6 line-clamp-3 leading-relaxed">
                  {bot.description}
                </p>
                
                <div className="flex items-center justify-between text-xs mb-6">
                  <div className="flex items-center gap-2">
                    <div className={`w-2 h-2 rounded-full ${
                      bot.sharedScope === 'public' ? 'bg-green-400' : 
                      bot.sharedScope === 'partial' ? 'bg-yellow-400' : 'bg-gray-400'
                    }`} />
                    <span className="text-muted-foreground font-medium">
                      {bot.sharedScope === 'public' ? 'Public' : bot.sharedScope === 'partial' ? 'Shared' : 'Private'}
                    </span>
                  </div>
                  <div className="flex items-center gap-1">
                    <Sparkles className="h-3 w-3 text-primary" />
                    <span className="text-muted-foreground font-mono">
                      {bot.activeModels?.[0]?.split('.')[1] || 'No model'}
                    </span>
                  </div>
                </div>
                
                {/* Consistent Action Buttons Layout */}
                <div className="grid grid-cols-3 gap-3">
                  {/* Chat Button - Always present */}
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
                      <span className="hidden sm:inline">Chat</span>
                    </Link>
                  </motion.div>
                  
                  {/* Action Menu Button - Always present, shows different actions based on ownership */}
                  <motion.div 
                    className="relative"
                    whileHover={{ scale: 1.05 }} 
                    whileTap={{ scale: 0.95 }}
                  >
                    {bot.ownerUserId === (user?.userId || user?.username) ? (
                      <div className="flex flex-col gap-1">
                        <Link
                          to={`/bots/${bot.id}/edit`}
                          className="inline-flex items-center justify-center w-full h-6 bg-glass border border-border/50 text-muted-foreground rounded-lg hover:text-primary hover:border-primary/50 transition-all duration-300 text-xs"
                          title="Edit Bot"
                        >
                          <Edit3 className="h-3 w-3" />
                        </Link>
                        <motion.button
                          onClick={() => deleteBot(bot.id, bot.title)}
                          disabled={deletingBotId === bot.id}
                          className="inline-flex items-center justify-center w-full h-6 bg-red-500/10 border border-red-500/30 text-red-400 rounded-lg hover:bg-red-500/20 hover:border-red-500/50 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed text-xs"
                          whileHover={{ scale: deletingBotId === bot.id ? 1 : 1.02 }}
                          title="Delete Bot"
                        >
                          {deletingBotId === bot.id ? (
                            <motion.div
                              className="w-3 h-3 border border-red-400/30 border-t-red-400 rounded-full"
                              animate={{ rotate: 360 }}
                              transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
                            />
                          ) : (
                            <Trash2 className="h-3 w-3" />
                          )}
                        </motion.button>
                      </div>
                    ) : (
                      <GlassCard className="w-full h-12 flex items-center justify-center text-muted-foreground/50 text-xs">
                        <span className="font-medium">Shared</span>
                      </GlassCard>
                    )}
                  </motion.div>
                </div>
              </GlassCard>
            ))}
          </motion.div>
        )}
      </main>
    </div>
  );
}