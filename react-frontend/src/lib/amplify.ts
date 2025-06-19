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
  // Amplify configured successfully
};

export { Amplify };