
import { motion, type MotionProps } from 'framer-motion';
import React from 'react';

interface GlassCardProps {
  children?: React.ReactNode;
  className?: string;
  initial?: MotionProps['initial'];
  animate?: MotionProps['animate'];
  transition?: MotionProps['transition'];
  whileHover?: MotionProps['whileHover'];
  whileTap?: MotionProps['whileTap'];
  exit?: MotionProps['exit'];
  layout?: boolean;
  as?: React.ElementType;
  [key: string]: any;
}

export const GlassCard: React.FC<GlassCardProps> = ({
  children,
  className = '',
  initial,
  animate,
  transition,
  whileHover,
  whileTap,
  exit,
  layout,
  as: Component = 'div',
  ...props
}) => {
  const MotionComponent = motion(Component);

  return (
    <MotionComponent
      className={`glass-card ${className}`}
      initial={initial}
      animate={animate}
      transition={transition}
      whileHover={whileHover}
      whileTap={whileTap}
      exit={exit}
      layout={layout}
      {...props}
    >
      {children}
    </MotionComponent>
  );
};
