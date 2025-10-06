import { Link } from 'react-router-dom';
import { useAuth } from '../components/AuthProvider';
import { ThemeToggle } from '../components/ThemeToggle';
import { motion } from 'framer-motion';
import { MessageCircle, Search, BookOpen, Bot, Sparkles, Zap, Shield } from 'lucide-react';

export function HomePage() {
  const { user, signOut } = useAuth();

  return (
    <div className="min-h-screen bg-background text-foreground overflow-hidden">
      {/* Animated Background */}
      <div className="fixed inset-0 gradient-mesh opacity-50" />
      <div className="fixed inset-0">
        {[...Array(20)].map((_, i) => (
          <motion.div
            key={i}
            className="absolute w-2 h-2 bg-primary/20 rounded-full"
            animate={{
              x: [0, Math.random() * 100 - 50],
              y: [0, Math.random() * 100 - 50],
              scale: [1, Math.random() * 0.5 + 0.5, 1],
            }}
            transition={{
              duration: Math.random() * 10 + 10,
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
            <div className="flex items-center">
              <motion.div 
                className="flex items-center gap-3"
                whileHover={{ scale: 1.05 }}
                transition={{ type: "spring", stiffness: 400, damping: 10 }}
              >
                <div className="p-2 rounded-lg gradient-primary">
                  <Sparkles className="h-6 w-6 text-white" />
                </div>
                <h1 className="text-xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent">
                  Foundry
                </h1>
              </motion.div>
            </div>
            <div>
              {user && (
                <div className="flex items-center space-x-4">
                  <span className="text-sm text-muted-foreground">Welcome, {user.username}</span>
                  <ThemeToggle />
                  <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
                    <Link
                      to="/discover"
                      className="bg-green-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-green-700 hover-glow transition-all duration-300 shadow-lg"
                    >
                      Discover Bots
                    </Link>
                  </motion.div>
                  <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
                    <Link
                      to="/bots"
                      className="bg-blue-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-blue-700 hover-glow transition-all duration-300 shadow-lg"
                    >
                      My Bots
                    </Link>
                  </motion.div>
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
      <main className="relative z-10 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-20">
        <motion.div 
          className="text-center"
          initial={{ opacity: 0, y: 50 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 1, delay: 0.3 }}
        >
          <motion.div
            initial={{ scale: 0.9, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            transition={{ duration: 1.2, delay: 0.5 }}
          >
            <h1 className="text-5xl font-bold text-balance sm:text-6xl md:text-7xl lg:text-8xl">
              <span className="bg-gradient-to-r from-foreground via-primary to-foreground bg-clip-text text-transparent">
                Foundry
              </span>
              <br />
              <span className="text-3xl sm:text-4xl md:text-5xl lg:text-6xl bg-gradient-to-r from-muted-foreground to-primary bg-clip-text text-transparent font-light">
                Intelligent Bot Creation & Sharing
              </span>
            </h1>
          </motion.div>
          
          <motion.p
            className="mt-8 max-w-2xl mx-auto text-lg text-muted-foreground sm:text-xl md:text-2xl leading-relaxed text-balance"
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.8, delay: 0.8 }}
          >
            Create, customize, and chat with intelligent bots. Leverage the power of large language models
            with knowledge base integration for enhanced responses.
          </motion.p>
          
          <motion.div
            className="mt-12 flex flex-wrap justify-center gap-6 text-sm text-muted-foreground"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.6, delay: 1.2 }}
          >
            {[
              { icon: Shield, text: "Enterprise Security" },
              { icon: Zap, text: "Lightning Fast" },
              { icon: Bot, text: "LLM-Powered" }
            ].map(({ icon: Icon, text }, index) => (
              <motion.div
                key={text}
                className="flex items-center gap-2 glass-card px-4 py-2"
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.5, delay: 1.4 + index * 0.1 }}
                whileHover={{ scale: 1.05, y: -2 }}
              >
                <Icon className="h-4 w-4 text-primary" />
                <span>{text}</span>
              </motion.div>
            ))}
          </motion.div>
        </motion.div>

        {/* Features */}
        <motion.div 
          className="mt-32"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.8, delay: 1.6 }}
        >
          <motion.div 
            className="grid grid-cols-1 gap-8 sm:grid-cols-2 lg:grid-cols-4"
            initial={{ opacity: 0, y: 50 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.8, delay: 1.8 }}
          >
            {[
              {
                icon: MessageCircle,
                title: "Smart Conversations",
                description: "Engage with LLM-powered bots through an intuitive chat interface with streaming responses.",
                color: "from-blue-500 to-purple-600"
              },
              {
                icon: Search,
                title: "Bot Marketplace",
                description: "Discover and chat with public bots, or share your own creations with the community.",
                color: "from-green-500 to-emerald-600"
              },
              {
                icon: BookOpen,
                title: "Knowledge Integration",
                description: "Enhance bot responses with retrieval-augmented generation using your knowledge bases.",
                color: "from-purple-500 to-pink-600"
              },
              {
                icon: Bot,
                title: "Custom Bots",
                description: "Create and customize intelligent bots with specific personalities, tools, and knowledge bases.",
                color: "from-orange-500 to-red-600"
              }
            ].map((feature, index) => {
              const Icon = feature.icon;
              return (
                <motion.div
                  key={feature.title}
                  className="glass-card p-8 hover-lift group cursor-pointer"
                  initial={{ opacity: 0, y: 30 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.6, delay: 2 + index * 0.1 }}
                  whileHover={{ scale: 1.05, rotateY: 5 }}
                >
                  <div className={`flex items-center justify-center h-16 w-16 rounded-2xl bg-gradient-to-br ${feature.color} shadow-lg group-hover:shadow-xl transition-all duration-300`}>
                    <Icon className="h-8 w-8 text-white" />
                  </div>
                  <h3 className="mt-6 text-xl font-semibold text-card-foreground group-hover:text-primary transition-colors duration-300">
                    {feature.title}
                  </h3>
                  <p className="mt-4 text-base text-muted-foreground leading-relaxed">
                    {feature.description}
                  </p>
                </motion.div>
              );
            })}
          </motion.div>
        </motion.div>


        {/* CTA Section */}
        <motion.div 
          className="mt-32 text-center"
          initial={{ opacity: 0, y: 50 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8, delay: 2.5 }}
        >
          <motion.div 
            className="glass-card p-12 max-w-4xl mx-auto"
            whileHover={{ scale: 1.02 }}
            transition={{ type: "spring", stiffness: 300, damping: 30 }}
          >
            <motion.h2 
              className="text-3xl font-bold text-foreground mb-6 text-balance"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ duration: 0.6, delay: 2.8 }}
            >
              Ready to Transform Your Workflow?
            </motion.h2>
            
            <motion.p
              className="text-lg text-muted-foreground mb-10 text-balance max-w-2xl mx-auto"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ duration: 0.6, delay: 3 }}
            >
              Join thousands of users who are already leveraging intelligent bots to boost their productivity
            </motion.p>
            
            <div className="space-y-8">
              <div className="flex flex-col sm:flex-row gap-6 justify-center">
                <motion.div
                  whileHover={{ scale: 1.05, y: -2 }}
                  whileTap={{ scale: 0.95 }}
                  initial={{ opacity: 0, x: -30 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ duration: 0.6, delay: 3.2 }}
                >
                  <Link
                    to="/discover"
                    className="inline-flex items-center px-10 py-4 text-lg font-semibold rounded-2xl text-white bg-gradient-to-r from-green-500 to-emerald-600 hover:from-green-600 hover:to-emerald-700 transition-all duration-300 shadow-xl hover:shadow-2xl hover-glow group"
                  >
                    <Search className="mr-3 h-5 w-5 group-hover:rotate-12 transition-transform duration-300" />
                    Discover Bots
                    <motion.div
                      className="ml-2"
                      animate={{ x: [0, 5, 0] }}
                      transition={{ duration: 1.5, repeat: Infinity }}
                    >
                      →
                    </motion.div>
                  </Link>
                </motion.div>
                
                <motion.div
                  whileHover={{ scale: 1.05, y: -2 }}
                  whileTap={{ scale: 0.95 }}
                  initial={{ opacity: 0, x: 30 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ duration: 0.6, delay: 3.4 }}
                >
                  <Link
                    to="/bots/create"
                    className="inline-flex items-center px-10 py-4 text-lg font-semibold rounded-2xl text-white bg-gradient-to-r from-blue-500 to-purple-600 hover:from-blue-600 hover:to-purple-700 transition-all duration-300 shadow-xl hover:shadow-2xl hover-glow group"
                  >
                    <Bot className="mr-3 h-5 w-5 group-hover:bounce transition-all duration-300" />
                    Create Your First Bot
                    <motion.div
                      className="ml-2"
                      animate={{ x: [0, 5, 0] }}
                      transition={{ duration: 1.5, repeat: Infinity, delay: 0.3 }}
                    >
                      →
                    </motion.div>
                  </Link>
                </motion.div>
              </div>
              
              <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.6, delay: 3.6 }}
              >
                <Link
                  to="/bots"
                  className="inline-flex items-center px-8 py-3 border-2 border-border text-base font-medium rounded-xl text-muted-foreground hover:text-foreground hover:border-primary/50 bg-background/50 hover:bg-background transition-all duration-300 hover-lift"
                >
                  <motion.div
                    whileHover={{ rotate: 360 }}
                    transition={{ duration: 0.5 }}
                  >
                    ⚙️
                  </motion.div>
                  <span className="ml-2">Manage Existing Bots</span>
                </Link>
              </motion.div>
            </div>
          </motion.div>
        </motion.div>
      </main>
    </div>
  );
}