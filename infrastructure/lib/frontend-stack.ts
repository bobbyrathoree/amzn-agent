import * as cdk from 'aws-cdk-lib';
import * as s3 from 'aws-cdk-lib/aws-s3';
import * as cloudfront from 'aws-cdk-lib/aws-cloudfront';
import * as origins from 'aws-cdk-lib/aws-cloudfront-origins';
import * as s3deploy from 'aws-cdk-lib/aws-s3-deployment';
import * as cognito from 'aws-cdk-lib/aws-cognito';
import * as acm from 'aws-cdk-lib/aws-certificatemanager';
import * as iam from 'aws-cdk-lib/aws-iam';
import { Construct } from 'constructs';
import * as path from 'path';
import { Config } from './config';

export interface FrontendStackProps extends cdk.StackProps {
  config: Config;
  userPool: cognito.UserPool;
  userPoolClient: cognito.UserPoolClient;
  identityPool: cognito.CfnIdentityPool;
  apiEndpoint: string;
  websocketEndpoint: string;
}

export class FrontendStack extends cdk.Stack {
  public readonly websiteBucket: s3.Bucket;
  public readonly distribution: cloudfront.Distribution;
  public readonly websiteUrl: string;
  
  constructor(scope: Construct, id: string, props: FrontendStackProps) {
    super(scope, id, props);
    
    // Create S3 bucket for website hosting
    this.websiteBucket = new s3.Bucket(this, 'WebsiteBucket', {
      bucketName: `${props.config.prefix.toLowerCase()}website-${this.account}`.toLowerCase(),
      removalPolicy: cdk.RemovalPolicy.RETAIN,
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
      encryption: s3.BucketEncryption.S3_MANAGED,
      cors: [
        {
          allowedMethods: [s3.HttpMethods.GET, s3.HttpMethods.HEAD],
          allowedOrigins: ['*'],
          allowedHeaders: ['*'],
        },
      ],
    });
    
    // Create Origin Access Identity
    const originAccessIdentity = new cloudfront.OriginAccessIdentity(this, 'OAI', {
      comment: 'OAI for website bucket',
    });
    
    // Grant access to the bucket
    this.websiteBucket.grantRead(originAccessIdentity);
    
    // Determine if custom domain is configured
    let certificate: acm.ICertificate | undefined;
    let domainNames: string[] | undefined;
    
    if (props.config.domainName && props.config.certificateArn) {
      certificate = acm.Certificate.fromCertificateArn(
        this, 'Certificate', props.config.certificateArn
      );
      domainNames = [props.config.domainName];
    }
    
    // Create CloudFront distribution
    this.distribution = new cloudfront.Distribution(this, 'Distribution', {
      defaultBehavior: {
        origin: new origins.S3Origin(this.websiteBucket, {
          originAccessIdentity,
        }),
        compress: true,
        allowedMethods: cloudfront.AllowedMethods.ALLOW_GET_HEAD,
        viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
        cachePolicy: cloudfront.CachePolicy.CACHING_OPTIMIZED,
      },
      defaultRootObject: 'index.html',
      errorResponses: [
        {
          httpStatus: 404,
          responseHttpStatus: 200,
          responsePagePath: '/index.html',
          ttl: cdk.Duration.seconds(10),
        },
        {
          httpStatus: 403,
          responseHttpStatus: 200,
          responsePagePath: '/index.html',
          ttl: cdk.Duration.seconds(10),
        },
      ],
      domainNames,
      certificate,
      minimumProtocolVersion: cloudfront.SecurityPolicyProtocol.TLS_V1_2_2021,
    });
    
    // Additional SPA routing configuration is now handled in errorResponses above
    
    // Generate config file for frontend
    const configFileContent = JSON.stringify({
      environment: props.config.env,
      userPoolId: props.userPool.userPoolId,
      userPoolClientId: props.userPoolClient.userPoolClientId,
      identityPoolId: props.identityPool.ref,
      apiEndpoint: props.apiEndpoint,
      websocketEndpoint: props.websocketEndpoint,
      region: this.region,
    }, null, 2);
    
    // Deploy website assets to S3
    const sources = [s3deploy.Source.data('config.json', configFileContent)];
    
    // Include the built Next.js app
    const frontendPath = path.join(__dirname, '../../frontend/out');
    try {
      // Check if the Next.js build output exists
      if (require('fs').existsSync(frontendPath)) {
        sources.push(s3deploy.Source.asset(frontendPath));
        console.log('✅ Including built Next.js app from:', frontendPath);
      } else {
        console.log('⚠️  Next.js build output not found at:', frontendPath);
        console.log('📝 Including fallback index.html from frontend directory');
        // Fallback to the current HTML file if Next.js build doesn't exist
        const fallbackPath = path.join(__dirname, '../../frontend');
        if (require('fs').existsSync(path.join(fallbackPath, 'index.html'))) {
          sources.push(s3deploy.Source.asset(fallbackPath, { exclude: ['node_modules/**/*', 'src/**/*', '*.json', '*.js', '*.ts'] }));
        }
      }
    } catch (error) {
      console.log('Error checking frontend paths:', error);
    }

    new s3deploy.BucketDeployment(this, 'WebsiteDeployment', {
      sources,
      destinationBucket: this.websiteBucket,
      distribution: this.distribution,
      distributionPaths: ['/*'],
      memoryLimit: 512,
    });
    
    // Set website URL based on domain configuration
    this.websiteUrl = props.config.domainName || `https://${this.distribution.distributionDomainName}`;
    
    // Export outputs
    new cdk.CfnOutput(this, 'WebsiteUrl', {
      value: this.websiteUrl,
      description: 'Website URL',
      exportName: `${props.config.prefix}WebsiteUrl`,
    });
    
    new cdk.CfnOutput(this, 'BucketName', {
      value: this.websiteBucket.bucketName,
      description: 'Website Bucket Name',
      exportName: `${props.config.prefix}WebsiteBucketName`,
    });
    
    new cdk.CfnOutput(this, 'DistributionId', {
      value: this.distribution.distributionId,
      description: 'CloudFront Distribution ID',
      exportName: `${props.config.prefix}DistributionId`,
    });
  }
}