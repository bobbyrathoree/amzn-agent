import React, { useState } from 'react';
import { Settings, ChevronDown } from 'lucide-react';

interface AdvancedSettingsAccordionProps {
  children: React.ReactNode;
  defaultCollapsed?: boolean;
  title?: string;
}

export const AdvancedSettingsAccordion: React.FC<AdvancedSettingsAccordionProps> = ({
  children,
  defaultCollapsed = true,
  title = 'Advanced Settings'
}) => {
  const [isCollapsed, setIsCollapsed] = useState(defaultCollapsed);

  return (
    <div className="glass-card border border-border/30 rounded-xl overflow-hidden">
      <button
        type="button"
        onClick={() => setIsCollapsed(!isCollapsed)}
        className="w-full p-3 flex items-center justify-between hover:bg-primary/5 transition-colors duration-200"
      >
        <div className="flex items-center gap-2">
          <Settings className="w-4 h-4 text-muted-foreground" />
          <span className="text-sm font-medium text-foreground">{title}</span>
        </div>
        <ChevronDown
          className={`w-4 h-4 text-muted-foreground transition-transform duration-300 ${
            isCollapsed ? '' : 'rotate-180'
          }`}
        />
      </button>

      {!isCollapsed && (
        <div className="px-3 pb-3 space-y-3 border-t border-border/20 pt-3">
          {children}
        </div>
      )}
    </div>
  );
};

export default AdvancedSettingsAccordion;
