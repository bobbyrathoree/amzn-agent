import React from 'react';
import { motion } from 'framer-motion';
import { BookOpen, BookOpenCheck } from 'lucide-react';

interface KnowledgeBaseToggleProps {
  enabled: boolean;
  onToggle: (enabled: boolean) => void;
  className?: string;
}

export const KnowledgeBaseToggle: React.FC<KnowledgeBaseToggleProps> = ({
  enabled,
  onToggle,
  className = ''
}) => {
  return (
    <motion.div 
      className={`glass-card p-4 rounded-xl border border-border/30 ${className}`}
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4 }}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className={`w-8 h-8 rounded-lg flex items-center justify-center transition-all duration-300 ${
            enabled 
              ? 'bg-gradient-to-br from-blue-500 to-purple-600 text-white shadow-md' 
              : 'bg-muted/30 text-muted-foreground'
          }`}>
            {enabled ? (
              <BookOpenCheck className="w-4 h-4" />
            ) : (
              <BookOpen className="w-4 h-4" />
            )}
          </div>
          <div>
            <div className="text-sm font-semibold text-foreground">
              Knowledge Base Search
            </div>
            <div className="text-xs text-muted-foreground">
              {enabled ? 'Using contextual knowledge' : 'General AI conversation'}
            </div>
          </div>
        </div>
        
        <motion.button
          onClick={() => onToggle(!enabled)}
          className={`relative w-11 h-6 rounded-full transition-all duration-300 focus:outline-none focus:ring-2 focus:ring-primary/50 ${
            enabled 
              ? 'bg-gradient-to-r from-blue-500 to-purple-600 shadow-md' 
              : 'bg-muted/50'
          }`}
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
        >
          <motion.div 
            className={`absolute top-0.5 w-5 h-5 rounded-full shadow-sm transition-all duration-300 ${
              enabled ? 'bg-white' : 'bg-white/80'
            }`}
            animate={{ 
              x: enabled ? 22 : 2,
            }}
            transition={{ 
              type: "spring", 
              stiffness: 500, 
              damping: 30 
            }}
          />
        </motion.button>
      </div>
      
      {!enabled && (
        <motion.div
          className="mt-3 pt-3 border-t border-border/30"
          initial={{ opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: 'auto' }}
          transition={{ duration: 0.3 }}
        >
          <div className="text-xs text-muted-foreground">
            💡 <strong>Direct AI Mode:</strong> Ask general questions without knowledge base context. No sources or document search will be performed.
          </div>
        </motion.div>
      )}
    </motion.div>
  );
};