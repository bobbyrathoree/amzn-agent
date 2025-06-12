import { Amplify } from 'aws-amplify';
import type { Config } from '../types';

let isConfigured = false;

export const configureAmplify = (config: Config) => {
  if (isConfigured) return;

  const amplifyConfig = {
    Auth: {
      Cognito: {
        userPoolId: config.userPoolId,
        userPoolClientId: config.userPoolClientId,
        region: config.region,
        signUpVerificationMethod: 'code' as const,
        loginWith: {
          email: true,
          username: false,
        },
      },
    },
  };

  Amplify.configure(amplifyConfig);
  isConfigured = true;
  console.log('🔧 Amplify configured with:', { 
    userPoolId: config.userPoolId,
    userPoolClientId: config.userPoolClientId.substring(0, 10) + '...',
    region: config.region 
  });
  console.log('🔧 Full Amplify config:', amplifyConfig);
};

export { Amplify };