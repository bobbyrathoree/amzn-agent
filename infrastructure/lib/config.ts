import * as cdk from 'aws-cdk-lib';

export class Config {
  public readonly prefix: string;
  public readonly envProps: cdk.Environment;
  public readonly domainName?: string;
  public readonly certificateArn?: string;
  public readonly lambdaMemory: number;
  public readonly lambdaTimeout: cdk.Duration;
  public readonly dynamoTableProps: { [key: string]: any };
  public readonly apiProps: { [key: string]: any };
  public readonly env: string;
  
  constructor(environment: string) {
    this.env = environment;
    this.prefix = `${this.env}-AmazonBuddy-`;
    
    this.envProps = {
      account: process.env.CDK_DEFAULT_ACCOUNT,
      region: process.env.CDK_DEFAULT_REGION || 'us-west-2',
    };
    
    // Domain configuration (if available)
    this.domainName = process.env.DOMAIN_NAME;
    this.certificateArn = process.env.CERTIFICATE_ARN;
    
    // Lambda settings
    this.lambdaMemory = this.env === 'prod' ? 1024 : 512;
    this.lambdaTimeout = this.env === 'prod' 
      ? cdk.Duration.seconds(60) 
      : cdk.Duration.seconds(30);
    
    // DynamoDB settings
    this.dynamoTableProps = {
      billingMode: this.env === 'prod' 
        ? cdk.aws_dynamodb.BillingMode.PROVISIONED 
        : cdk.aws_dynamodb.BillingMode.PAY_PER_REQUEST,
      readCapacity: this.env === 'prod' ? 5 : undefined,
      writeCapacity: this.env === 'prod' ? 5 : undefined,
    };
    
    // API Gateway settings
    this.apiProps = {
      throttlingBurstLimit: this.env === 'prod' ? 1000 : 500,
      throttlingRateLimit: this.env === 'prod' ? 2000 : 1000,
    };
  }
}