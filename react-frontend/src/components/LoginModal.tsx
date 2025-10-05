import { useState } from 'react';
import type { FormEvent } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Sparkles, Mail, Lock, User, CheckCircle, AlertCircle } from 'lucide-react';
import { useAuth } from './AuthProvider';

export function LoginModal() {
  const { signIn, signUp, confirmSignUp, error, setError } = useAuth();
  const [mode, setMode] = useState<'signin' | 'signup' | 'confirm'>('signin');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [fullName, setFullName] = useState('');
  const [confirmationCode, setConfirmationCode] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSignIn = async (e: FormEvent) => {
    e.preventDefault();
    setLoading(true);
    await signIn(email, password);
    setLoading(false);
  };

  const handleSignUp = async (e: FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await signUp(email, password, fullName);
      setMode('confirm');
    } catch (error) {
      // Error is handled by AuthProvider
    }
    setLoading(false);
  };

  const handleConfirmSignUp = async (e: FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await confirmSignUp(email, confirmationCode);
      setMode('signin');
      setError(null);
    } catch (error) {
      // Error is handled by AuthProvider
    }
    setLoading(false);
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted/20 flex items-center justify-center p-4 relative overflow-hidden">
      {/* Animated Background Elements */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <motion.div 
          className="absolute -top-40 -right-40 w-80 h-80 bg-primary/10 rounded-full blur-3xl"
          animate={{ 
            x: [0, 30, -30, 0],
            y: [0, -20, 20, 0],
            scale: [1, 1.1, 0.9, 1]
          }}
          transition={{ duration: 20, repeat: Infinity, ease: "easeInOut" }}
        />
        <motion.div 
          className="absolute -bottom-40 -left-40 w-80 h-80 bg-purple-500/10 rounded-full blur-3xl"
          animate={{ 
            x: [0, -30, 30, 0],
            y: [0, 20, -20, 0],
            scale: [1, 0.9, 1.1, 1]
          }}
          transition={{ duration: 25, repeat: Infinity, ease: "easeInOut" }}
        />
      </div>

      <motion.div 
        className="max-w-md w-full glass-card border border-border/30 p-8 relative"
        initial={{ opacity: 0, y: 20, scale: 0.95 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        transition={{ duration: 0.6, ease: "easeOut" }}
      >
        {/* Header */}
        <motion.div 
          className="text-center mb-8"
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.2 }}
        >
          <div className="flex items-center justify-center gap-3 mb-4">
            <motion.div
              className="w-12 h-12 bg-gradient-to-br from-blue-500 to-purple-600 rounded-xl flex items-center justify-center shadow-lg"
              whileHover={{ scale: 1.1, rotate: 5 }}
              transition={{ type: "spring", stiffness: 300, damping: 20 }}
            >
              <Sparkles className="w-6 h-6 text-white" />
            </motion.div>
            <h1 className="text-3xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent">
              Foundry
            </h1>
          </div>
          <motion.p 
            className="text-muted-foreground"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.6, delay: 0.4 }}
          >
            {mode === 'signin' && 'Welcome back to your AI workspace'}
            {mode === 'signup' && 'Join the future of AI conversation'}
            {mode === 'confirm' && 'Almost there! Verify your email'}
          </motion.p>
        </motion.div>

        {/* Error Display */}
        <AnimatePresence>
          {error && (
            <motion.div 
              className="mb-6 p-4 glass-card border border-red-500/30 bg-red-500/10 rounded-xl"
              initial={{ opacity: 0, y: -10, scale: 0.95 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: -10, scale: 0.95 }}
              transition={{ duration: 0.3 }}
            >
              <div className="flex items-center gap-3 text-red-500 dark:text-red-400">
                <AlertCircle className="w-5 h-5 flex-shrink-0" />
                <span className="text-sm font-medium">{error}</span>
              </div>
            </motion.div>
          )}
        </AnimatePresence>

        {mode === 'signin' && (
          <motion.form 
            onSubmit={handleSignIn}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.4, delay: 0.6 }}
          >
            <div className="space-y-6">
              <motion.div
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.4, delay: 0.7 }}
              >
                <label htmlFor="email" className="block text-sm font-semibold text-foreground mb-2">
                  <Mail className="w-4 h-4 inline mr-2" />
                  Email Address
                </label>
                <div className="relative">
                  <GlassCard
                    as="input"
                    type="email"
                    id="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className="w-full px-4 py-3 text-foreground placeholder-muted-foreground/60 focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300"
                    placeholder="Enter your email"
                    required
                  />
                </div>
              </motion.div>
              
              <motion.div
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.4, delay: 0.8 }}
              >
                <label htmlFor="password" className="block text-sm font-semibold text-foreground mb-2">
                  <Lock className="w-4 h-4 inline mr-2" />
                  Password
                </label>
                <div className="relative">
                  <GlassCard
                    as="input"
                    type="password"
                    id="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="w-full px-4 py-3 text-foreground placeholder-muted-foreground/60 focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300"
                    placeholder="Enter your password"
                    required
                  />
                </div>
              </motion.div>
              
              <motion.button
                type="submit"
                disabled={loading}
                className="w-full bg-gradient-to-r from-blue-500 to-purple-600 text-white py-3 px-6 rounded-xl font-semibold transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed hover:from-blue-600 hover:to-purple-700 shadow-lg hover:shadow-xl flex items-center justify-center gap-2"
                whileHover={{ scale: loading ? 1 : 1.02 }}
                whileTap={{ scale: loading ? 1 : 0.98 }}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.4, delay: 0.9 }}
              >
                {loading ? (
                  <>
                    <motion.div
                      className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full"
                      animate={{ rotate: 360 }}
                      transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
                    />
                    Signing In...
                  </>
                ) : (
                  <>
                    <CheckCircle className="w-5 h-5" />
                    Sign In
                  </>
                )}
              </motion.button>
            </div>
          </motion.form>
        )}

        {mode === 'signup' && (
          <motion.form 
            onSubmit={handleSignUp}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.4, delay: 0.6 }}
          >
            <div className="space-y-6">
              <motion.div
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.4, delay: 0.7 }}
              >
                <label htmlFor="fullName" className="block text-sm font-semibold text-foreground mb-2">
                  <User className="w-4 h-4 inline mr-2" />
                  Full Name
                </label>
                <div className="relative">
                  <GlassCard
                    as="input"
                    type="text"
                    id="fullName"
                    value={fullName}
                    onChange={(e) => setFullName(e.target.value)}
                    className="w-full px-4 py-3 text-foreground placeholder-muted-foreground/60 focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300"
                    placeholder="Enter your full name"
                  />
                </div>
              </motion.div>
              
              <motion.div
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.4, delay: 0.8 }}
              >
                <label htmlFor="email" className="block text-sm font-semibold text-foreground mb-2">
                  <Mail className="w-4 h-4 inline mr-2" />
                  Email Address
                </label>
                <div className="relative">
                  <GlassCard
                    as="input"
                    type="email"
                    id="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className="w-full px-4 py-3 text-foreground placeholder-muted-foreground/60 focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300"
                    placeholder="Enter your email"
                    required
                  />
                </div>
              </motion.div>
              
              <motion.div
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.4, delay: 0.9 }}
              >
                <label htmlFor="password" className="block text-sm font-semibold text-foreground mb-2">
                  <Lock className="w-4 h-4 inline mr-2" />
                  Password
                </label>
                <div className="relative">
                  <GlassCard
                    as="input"
                    type="password"
                    id="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="w-full px-4 py-3 text-foreground placeholder-muted-foreground/60 focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300"
                    placeholder="Create a secure password"
                    required
                  />
                </div>
              </motion.div>
              
              <motion.button
                type="submit"
                disabled={loading}
                className="w-full bg-gradient-to-r from-green-500 to-emerald-600 text-white py-3 px-6 rounded-xl font-semibold transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed hover:from-green-600 hover:to-emerald-700 shadow-lg hover:shadow-xl flex items-center justify-center gap-2"
                whileHover={{ scale: loading ? 1 : 1.02 }}
                whileTap={{ scale: loading ? 1 : 0.98 }}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.4, delay: 1.0 }}
              >
                {loading ? (
                  <>
                    <motion.div
                      className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full"
                      animate={{ rotate: 360 }}
                      transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
                    />
                    Creating Account...
                  </>
                ) : (
                  <>
                    <Sparkles className="w-5 h-5" />
                    Create Account
                  </>
                )}
              </motion.button>
            </div>
          </motion.form>
        )}

        {mode === 'confirm' && (
          <motion.form 
            onSubmit={handleConfirmSignUp}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.4, delay: 0.6 }}
          >
            <div className="space-y-6">
              <motion.div
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.4, delay: 0.7 }}
              >
                <label htmlFor="confirmationCode" className="block text-sm font-semibold text-foreground mb-2">
                  <Mail className="w-4 h-4 inline mr-2" />
                  Confirmation Code
                </label>
                <div className="relative">
                  <GlassCard
                    as="input"
                    type="text"
                    id="confirmationCode"
                    value={confirmationCode}
                    onChange={(e) => setConfirmationCode(e.target.value)}
                    className="w-full px-4 py-3 text-foreground placeholder-muted-foreground/60 focus:ring-2 focus:ring-primary/50 focus:border-primary/50 outline-none transition-all duration-300 text-center text-lg tracking-widest"
                    placeholder="000000"
                    required
                  />
                </div>
                <motion.p 
                  className="mt-3 text-sm text-muted-foreground bg-muted/10 rounded-lg p-3 border border-border/20"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ duration: 0.4, delay: 0.8 }}
                >
                  <Mail className="w-4 h-4 inline mr-2 text-primary" />
                  Check your email at <strong className="text-foreground">{email}</strong> for the 6-digit confirmation code
                </motion.p>
              </motion.div>
              
              <motion.button
                type="submit"
                disabled={loading}
                className="w-full bg-gradient-to-r from-emerald-500 to-teal-600 text-white py-3 px-6 rounded-xl font-semibold transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed hover:from-emerald-600 hover:to-teal-700 shadow-lg hover:shadow-xl flex items-center justify-center gap-2"
                whileHover={{ scale: loading ? 1 : 1.02 }}
                whileTap={{ scale: loading ? 1 : 0.98 }}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.4, delay: 0.9 }}
              >
                {loading ? (
                  <>
                    <motion.div
                      className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full"
                      animate={{ rotate: 360 }}
                      transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
                    />
                    Confirming...
                  </>
                ) : (
                  <>
                    <CheckCircle className="w-5 h-5" />
                    Confirm Email
                  </>
                )}
              </motion.button>
            </div>
          </motion.form>
        )}

        <motion.div 
          className="mt-8 text-center"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.4, delay: 1.2 }}
        >
          {mode === 'signin' && (
            <motion.button
              onClick={() => setMode('signup')}
              className="text-primary hover:text-primary/80 text-sm font-medium transition-all duration-300 hover:underline underline-offset-4"
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              Need an account? <strong>Sign up</strong>
            </motion.button>
          )}
          {(mode === 'signup' || mode === 'confirm') && (
            <motion.button
              onClick={() => setMode('signin')}
              className="text-primary hover:text-primary/80 text-sm font-medium transition-all duration-300 hover:underline underline-offset-4"
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              <strong>Sign in</strong>
            </motion.button>
          )}
        </motion.div>
      </GlassCard>
    </div>
        </motion.div>
      </motion.div>
    </div>
  );
}