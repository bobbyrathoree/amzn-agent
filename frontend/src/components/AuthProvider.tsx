'use client';

import { createContext, useContext, useEffect, useState } from 'react';
import { getCurrentUser, signIn, signOut, signUp, confirmSignUp, AuthUser } from 'aws-amplify/auth';
import '../lib/amplify'; // Initialize Amplify configuration

interface User {
  username: string;
  userId: string;
  email?: string;
  groups?: string[];
  accessToken?: string;
}

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
}

export const AuthContext = createContext<AuthContextType>({
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
});

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [showLogin, setShowLogin] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    checkAuthState();
  }, []);

  const checkAuthState = async () => {
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
      // Get additional user attributes and access token if needed
      const userData: User = {
        username: authUser.username,
        userId: authUser.userId,
        email: authUser.signInDetails?.loginId || authUser.username,
        groups: [], // Groups will be in JWT token
      };
      
      return userData;
    } catch (error) {
      console.error('Error converting auth user:', error);
      throw error;
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
      
      // After successful confirmation, the user will be automatically added to the BotCreators group
      // by our post-confirmation Lambda trigger
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
      setError
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