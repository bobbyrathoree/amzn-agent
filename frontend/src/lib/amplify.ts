import { Amplify } from 'aws-amplify';

// This will be populated by the deployment process
// For now, we'll use environment variables or config.json
const amplifyConfig = {
  Auth: {
    Cognito: {
      userPoolId: process.env.NEXT_PUBLIC_USER_POOL_ID || '',
      userPoolClientId: process.env.NEXT_PUBLIC_USER_POOL_CLIENT_ID || '',
      identityPoolId: process.env.NEXT_PUBLIC_IDENTITY_POOL_ID || '',
      signUpVerificationMethod: 'code' as const,
      loginWith: {
        email: true,
        username: false,
      },
    },
  },
};

// Configure Amplify
if (typeof window !== 'undefined') {
  // Only configure on client side
  Amplify.configure(amplifyConfig);
}

export { amplifyConfig };