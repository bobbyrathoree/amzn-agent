import { createContext, useContext, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
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
});

interface AuthProviderProps {
  children: ReactNode;
  config: Config | null;
}

export function AuthProvider({ children, config }: AuthProviderProps) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [showLogin, setShowLogin] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (config) {
      configureAmplify(config);
      checkAuthState();
    }
  }, [config]);

  const checkAuthState = async () => {
    if (!config) return;
    
    try {
      const authUser = await getCurrentUser();
      const userInfo = await convertAuthUserToUser(authUser);
      setUser(userInfo);
      setShowLogin(false);
    } catch (error) {
      console.log('No authenticated user found');
      setUser(null);
      setShowLogin(true);
    } finally {
      setIsLoading(false);
    }
  };

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
      return session.tokens?.accessToken?.toString() || null;
    } catch (error) {
      console.error('Error getting access token:', error);
      return null;
    }
  };

  const handleSignIn = async (email: string, password: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const { isSignedIn } = await signIn({ 
        username: email, 
        password,
        options: {
          authFlowType: 'USER_SRP_AUTH'
        }
      });
      
      if (isSignedIn) {
        await checkAuthState();
        return true;
      } else {
        setError('Sign in failed. Please check your credentials.');
        return false;
      }
    } catch (error: any) {
      console.error('Sign in error:', error);
      setError(error.message || 'Sign in failed. Please try again.');
      return false;
    } finally {
      setIsLoading(false);
    }
  };

  const handleSignUp = async (email: string, password: string, fullName?: string) => {
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
      if (error.message.includes('email domain')) {
        setError('Only Amazon employees can create accounts. Please use your @amazon.com email address.');
      } else {
        setError(error.message || 'Sign up failed. Please try again.');
      }
      throw error;
    }
  };

  const handleConfirmSignUp = async (email: string, code: string) => {
    setError(null);
    
    try {
      await confirmSignUp({
        username: email,
        confirmationCode: code,
      });
    } catch (error: any) {
      console.error('Confirmation error:', error);
      setError(error.message || 'Email confirmation failed. Please try again.');
      throw error;
    }
  };

  const handleSignOut = async () => {
    try {
      await signOut();
      setUser(null);
      setShowLogin(true);
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
      getAccessToken
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