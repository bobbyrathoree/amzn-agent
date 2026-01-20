import { createContext, useContext, useEffect, useState, useRef } from 'react';
import type { ReactNode } from 'react';
import { useNavigate } from 'react-router-dom';
import { getCurrentUser, signIn, signOut, signUp, confirmSignUp, fetchAuthSession } from 'aws-amplify/auth';
import type { AuthUser } from 'aws-amplify/auth';
import { configureAmplify } from '../lib/amplify';
import type { User, Config } from '../types';

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  signIn: (email: string, password: string) => Promise<boolean>;
  signOut: () => Promise<void>;
  signUp: (email: string, password: string, fullName?: string) => Promise<{ nextStep: any }>;
  confirmSignUp: (email: string, code: string) => Promise<void>;
  showLogin: boolean;
  setShowLogin: (show: boolean) => void;
  error: string | null;
  setError: (error: string | null) => void;
  getAccessToken: () => Promise<string | null>;
  environment: string;
}

const AuthContext = createContext<AuthContextType>({
  user: null,
  isLoading: false,
  signIn: async () => false,
  signOut: async () => {},
  signUp: async () => ({ nextStep: null }),
  confirmSignUp: async () => {},
  showLogin: false,
  setShowLogin: () => {},
  error: null,
  setError: () => {},
  getAccessToken: async () => null,
  environment: 'dev',
});

interface AuthProviderProps {
  children: ReactNode;
  config: Config | null;
}

export function AuthProvider({ children, config }: AuthProviderProps) {
  const navigate = useNavigate();
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [showLogin, setShowLogin] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Use ref to track mounted state for non-effect async operations
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    let cancelled = false;

    const checkAuthState = async () => {
      if (!config) return;

      try {
        const authUser = await getCurrentUser();
        const userInfo = await convertAuthUserToUser(authUser);

        if (!cancelled) {
          setUser(userInfo);
          setShowLogin(false);
        }
      } catch (error) {
        console.log('No authenticated user found');
        if (!cancelled) {
          setUser(null);
          setShowLogin(true);
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    };

    if (config) {
      configureAmplify(config);
      checkAuthState();
    }

    return () => {
      cancelled = true;
      mountedRef.current = false;
    };
  }, [config]);

  const convertAuthUserToUser = async (authUser: AuthUser): Promise<User> => {
    try {
      const userData: User = {
        username: authUser.username,
        userId: authUser.userId,
        email: authUser.signInDetails?.loginId || authUser.username,
        groups: [],
      };
      
      return userData;
    } catch (error) {
      console.error('Error converting auth user:', error);
      throw error;
    }
  };

  const getAccessToken = async (): Promise<string | null> => {
    try {
      const session = await fetchAuthSession();
      
      if (!session.tokens?.idToken) {
        return null;
      }
      
      const token = session.tokens.idToken.toString();
      return token;
    } catch (error) {
      console.error('Error getting access token:', error);
      return null;
    }
  };

  const handleSignIn = async (email: string, password: string): Promise<boolean> => {
    if (!mountedRef.current) return false;
    
    setIsLoading(true);
    setError(null);
    
    try {
      // Attempting sign in
      const { isSignedIn } = await signIn({
        username: email,
        password,
        options: {
          authFlowType: 'USER_PASSWORD_AUTH'
        }
      });
      
      // Sign in completed
      
      if (isSignedIn) {
        // Test token availability immediately after sign in
        const testToken = await getAccessToken();
        console.log('Token received in AuthProvider:', testToken ? 'Present' : 'Not present');

        if (mountedRef.current) {
          // Refresh auth state after successful sign in
          try {
            const authUser = await getCurrentUser();
            const userInfo = await convertAuthUserToUser(authUser);
            setUser(userInfo);
            setShowLogin(false);
          } catch (err) {
            console.error('Error refreshing auth state:', err);
          }
        }
        return true;
      } else {
        if (mountedRef.current) {
          setError('Sign in failed. Please check your credentials.');
        }
        return false;
      }
    } catch (error: any) {
      console.error('Sign in error:', error);
      if (mountedRef.current) {
        setError(error.message || 'Sign in failed. Please try again.');
      }
      return false;
    } finally {
      if (mountedRef.current) {
        setIsLoading(false);
      }
    }
  };

  const handleSignUp = async (email: string, password: string, fullName?: string) => {
    if (!mountedRef.current) throw new Error('Component unmounted');
    
    setError(null);
    
    try {
      const result = await signUp({
        username: email,
        password,
        options: {
          userAttributes: {
            email,
            name: fullName || '',
          },
        },
      });
      
      return result;
    } catch (error: any) {
      console.error('Sign up error:', error);
      if (mountedRef.current) {
        if (error.message.includes('email domain')) {
          setError('Only Amazon employees can create accounts. Please use your @amazon.com email address.');
        } else {
          setError(error.message || 'Sign up failed. Please try again.');
        }
      }
      throw error;
    }
  };

  const handleConfirmSignUp = async (email: string, code: string) => {
    if (!mountedRef.current) throw new Error('Component unmounted');
    
    setError(null);
    
    try {
      await confirmSignUp({
        username: email,
        confirmationCode: code,
      });
    } catch (error: any) {
      console.error('Confirmation error:', error);
      if (mountedRef.current) {
        setError(error.message || 'Email confirmation failed. Please try again.');
      }
      throw error;
    }
  };

  const handleSignOut = async () => {
    try {
      await signOut();
      if (mountedRef.current) {
        setUser(null);
        setShowLogin(true);
      }
      navigate('/'); // Redirect to homepage on logout
    } catch (error) {
      console.error('Sign out error:', error);
    }
  };

  return (
    <AuthContext.Provider value={{
      user,
      isLoading,
      signIn: handleSignIn,
      signOut: handleSignOut,
      signUp: handleSignUp,
      confirmSignUp: handleConfirmSignUp,
      showLogin,
      setShowLogin,
      error,
      setError,
      getAccessToken,
      environment: config?.environment || 'dev'
    }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}