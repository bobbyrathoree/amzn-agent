
import { motion, MotionProps } from 'framer-motion';
import React from 'react';

interface GlassCardProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode;
  className?: string;
  initial?: MotionProps['initial'];
  animate?: MotionProps['animate'];
  transition?: MotionProps['transition'];
  whileHover?: MotionProps['whileHover'];
  as?: React.ElementType;
}

export const GlassCard: React.FC<GlassCardProps> = ({
  children,
  className = '',
  initial,
  animate,
  transition,
  whileHover,
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
      {...props}
    >
      {children}
    </MotionComponent>
  );
};
