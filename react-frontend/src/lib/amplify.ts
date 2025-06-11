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
        identityPoolId: config.identityPoolId,
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
  console.log('Amplify configured with:', { 
    userPoolId: config.userPoolId,
    region: config.region 
  });
};

export { Amplify };