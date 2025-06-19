// Simple logger utility for conditional logging
// Only logs in development environment

const isDevelopment = import.meta.env.DEV;

export const logger = {
  debug: (message: string, ...args: any[]) => {
    if (isDevelopment) {
      console.log(message, ...args);
    }
  },
  
  info: (message: string, ...args: any[]) => {
    if (isDevelopment) {
      console.info(message, ...args);
    }
  },
  
  warn: (message: string, ...args: any[]) => {
    console.warn(message, ...args);
  },
  
  error: (message: string, ...args: any[]) => {
    console.error(message, ...args);
  }
};