# AI Chat Platform Infrastructure

This directory contains the AWS CDK infrastructure code for the AI Chat Platform.

## Overview

The infrastructure is organized into multiple stacks:

- **NetworkStack**: VPC, subnets, security groups
- **StorageStack**: DynamoDB tables, S3 buckets
- **AuthStack**: Cognito User Pool, Identity Pool, IAM roles
- **ApiStack**: API Gateway, WebSocket API, Lambda functions
- **FrontendStack**: S3 website, CloudFront distribution
- **MonitoringStack**: CloudWatch dashboards, alarms, SNS topics

## Prerequisites

- Node.js 18 or later
- AWS CLI configured
- AWS CDK installed globally (`npm install -g aws-cdk`)

## Getting Started

1. Install dependencies:
   ```
   cd infrastructure
   npm install
   ```

2. Bootstrap CDK (if not already done):
   ```
   npx cdk bootstrap
   ```

3. Configure environment variables:
   ```
   # Required for deployment
   export CDK_DEFAULT_ACCOUNT=<your-aws-account-id>
   export CDK_DEFAULT_REGION=<your-aws-region>
   
   # Optional for custom domain
   export DOMAIN_NAME=<your-domain-name>
   export CERTIFICATE_ARN=<your-certificate-arn>
   
   # Optional for alerts
   export ALERT_EMAIL=<your-email-address>
   ```

4. Deploy to development environment:
   ```
   npm run deploy:dev
   ```

5. Deploy to production environment:
   ```
   npm run deploy:prod
   ```

6. Review changes before deployment:
   ```
   npm run diff
   ```

7. Destroy the stacks:
   ```
   npm run destroy:dev
   ```

## Lambda Functions

The Go Lambda functions should be built and placed in the following directory structure before deployment:

```
/lambda
└── functions/
    ├── chat/
    │   └── bootstrap
    ├── bots/
    │   └── bootstrap
    ├── knowledge/
    │   └── bootstrap
    └── websocket/
        └── bootstrap
```

## Frontend Deployment

The frontend Next.js application should be built and placed in the `frontend/out` directory before deployment. This allows the CDK to automatically deploy the static assets to the S3 bucket.

## Security Considerations

- All data is encrypted at rest and in transit
- JWT-based authentication with Cognito
- Role-based access control with user groups
- API Gateway throttling to prevent abuse
- CloudWatch alarms for suspicious activity

## Monitoring

The MonitoringStack sets up:

- CloudWatch dashboards for service metrics
- Custom alarms for error conditions
- Email notifications via SNS

## Environment Variables

| Variable            | Description                              | Required?  |
|---------------------|------------------------------------------|------------|
| CDK_DEFAULT_ACCOUNT | AWS account ID for deployment            | Yes        |
| CDK_DEFAULT_REGION  | AWS region for deployment                | Yes        |
| DOMAIN_NAME         | Custom domain for CloudFront distribution| No         |
| CERTIFICATE_ARN     | ACM certificate ARN for HTTPS            | No         |
| ALERT_EMAIL         | Email address to receive alerts          | No         |