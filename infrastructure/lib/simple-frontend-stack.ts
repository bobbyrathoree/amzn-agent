import * as cdk from 'aws-cdk-lib';
import * as s3 from 'aws-cdk-lib/aws-s3';
import * as cloudfront from 'aws-cdk-lib/aws-cloudfront';
import * as origins from 'aws-cdk-lib/aws-cloudfront-origins';
import * as s3deploy from 'aws-cdk-lib/aws-s3-deployment';
import { Construct } from 'constructs';
import { Config } from './config';

export interface SimpleFrontendStackProps extends cdk.StackProps {
  config: Config;
}

export class SimpleFrontendStack extends cdk.Stack {
  public readonly websiteBucket: s3.Bucket;
  public readonly distribution: cloudfront.Distribution;
  public readonly websiteUrl: string;
  
  constructor(scope: Construct, id: string, props: SimpleFrontendStackProps) {
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
        },
      ],
      minimumProtocolVersion: cloudfront.SecurityPolicyProtocol.TLS_V1_2_2021,
    });
    
    // Create custom error response for SPAs
    const cfnDistribution = this.distribution.node.defaultChild as cloudfront.CfnDistribution;
    cfnDistribution.addPropertyOverride('DistributionConfig.CustomErrorResponses', [
      {
        ErrorCode: 403,
        ResponseCode: 200,
        ResponsePagePath: '/index.html',
        ErrorCachingMinTTL: 10,
      },
    ]);
    
    // Generate placeholder config file for frontend
    const configFileContent = JSON.stringify({
      environment: props.config.env,
      region: this.region,
    }, null, 2);
    
    // Deploy website assets to S3
    try {
      new s3deploy.BucketDeployment(this, 'WebsiteDeployment', {
        sources: [
          s3deploy.Source.data('config.json', configFileContent),
        ],
        destinationBucket: this.websiteBucket,
        distribution: this.distribution,
        distributionPaths: ['/*'],
      });
    } catch (error) {
      // In case of issues, log message
      console.log('Note: Frontend deployment issue:', error);
    }
    
    // Set website URL
    this.websiteUrl = `https://${this.distribution.distributionDomainName}`;
    
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