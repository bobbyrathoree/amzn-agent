import { useRef, useEffect } from 'react';

/**
 * Custom hook to track if a component is still mounted
 * Prevents setState calls on unmounted components
 */
export function useMountedRef() {
  const mountedRef = useRef(true);
  
  useEffect(() => {
    return () => {
      mountedRef.current = false;
    };
  }, []);
  
  return mountedRef;
}