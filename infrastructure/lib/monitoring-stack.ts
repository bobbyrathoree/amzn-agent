import * as cdk from 'aws-cdk-lib';
import * as cloudwatch from 'aws-cdk-lib/aws-cloudwatch';
import * as cloudwatchActions from 'aws-cdk-lib/aws-cloudwatch-actions';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as sns from 'aws-cdk-lib/aws-sns';
import * as subscriptions from 'aws-cdk-lib/aws-sns-subscriptions';
import { Construct } from 'constructs';
import { Config } from './config';

export interface MonitoringStackProps extends cdk.StackProps {
  config: Config;
  apiGateway: apigateway.RestApi;
  lambdaFunctions: lambda.Function[];
  dynamoTables: dynamodb.Table[];
}

export class MonitoringStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props: MonitoringStackProps) {
    super(scope, id, props);
    
    // Create SNS topic for alerts
    const alertTopic = new sns.Topic(this, 'AlertTopic', {
      displayName: `${props.config.prefix}Alerts`,
      topicName: `${props.config.prefix}Alerts`,
    });
    
    // Add email subscription if provided
    if (process.env.ALERT_EMAIL) {
      alertTopic.addSubscription(
        new subscriptions.EmailSubscription(process.env.ALERT_EMAIL)
      );
    }
    
    // Create dashboard
    const dashboard = new cloudwatch.Dashboard(this, 'Dashboard', {
      dashboardName: `${props.config.prefix}Dashboard`,
    });
    
    // Add API Gateway metrics to dashboard
    const apiMetrics = this.createApiGatewayWidgets(props.apiGateway);
    dashboard.addWidgets(...apiMetrics);
    
    // Add Lambda metrics to dashboard
    const lambdaMetrics = this.createLambdaWidgets(props.lambdaFunctions);
    dashboard.addWidgets(...lambdaMetrics);
    
    // Add DynamoDB metrics to dashboard
    const dynamoMetrics = this.createDynamoWidgets(props.dynamoTables);
    dashboard.addWidgets(...dynamoMetrics);
    
    // Create alarms for critical services
    
    // API Gateway 5xx errors alarm
    const api5xxErrorAlarm = new cloudwatch.Alarm(this, 'Api5xxErrorAlarm', {
      metric: props.apiGateway.metricServerError({
        period: cdk.Duration.minutes(1),
        statistic: 'sum',
      }),
      evaluationPeriods: 3,
      threshold: 5,
      alarmDescription: 'API Gateway is returning 5xx errors',
      actionsEnabled: true,
    });
    
    api5xxErrorAlarm.addAlarmAction(new cloudwatchActions.SnsAction(alertTopic));
    
    // Lambda error rate alarms
    props.lambdaFunctions.forEach((fn, index) => {
      const errorAlarm = new cloudwatch.Alarm(this, `LambdaErrorAlarm-${index}`, {
        metric: fn.metricErrors({
          period: cdk.Duration.minutes(1),
        }),
        evaluationPeriods: 3,
        threshold: 5,
        alarmDescription: `Lambda function ${fn.functionName} has a high error rate`,
        actionsEnabled: true,
      });
      
      errorAlarm.addAlarmAction(new cloudwatchActions.SnsAction(alertTopic));
    });
    
    // DynamoDB throttling alarms
    props.dynamoTables.forEach((table, index) => {
      const readThrottleAlarm = new cloudwatch.Alarm(this, `DynamoReadThrottleAlarm-${index}`, {
        metric: new cloudwatch.Metric({
          namespace: 'AWS/DynamoDB',
          metricName: 'ReadThrottleEvents',
          dimensionsMap: { TableName: table.tableName },
          period: cdk.Duration.minutes(5),
          statistic: 'sum',
        }),
        evaluationPeriods: 3,
        threshold: 10,
        alarmDescription: `DynamoDB table ${table.tableName} has read throttling events`,
        actionsEnabled: true,
      });
      
      readThrottleAlarm.addAlarmAction(new cloudwatchActions.SnsAction(alertTopic));
    });
    
    // Export outputs
    new cdk.CfnOutput(this, 'DashboardURL', {
      value: `https://${this.region}.console.aws.amazon.com/cloudwatch/home?region=${this.region}#dashboards:name=${dashboard.dashboardName}`,
      description: 'Dashboard URL',
      exportName: `${props.config.prefix}DashboardURL`,
    });
    
    new cdk.CfnOutput(this, 'AlertTopicArn', {
      value: alertTopic.topicArn,
      description: 'Alert Topic ARN',
      exportName: `${props.config.prefix}AlertTopicArn`,
    });
  }
  
  private createApiGatewayWidgets(api: apigateway.RestApi): cloudwatch.IWidget[] {
    return [
      new cloudwatch.GraphWidget({
        title: 'API Gateway - Requests',
        left: [
          api.metricCount({ statistic: 'sum', period: cdk.Duration.minutes(1) }),
        ],
        width: 12,
      }),
      new cloudwatch.GraphWidget({
        title: 'API Gateway - Latency',
        left: [
          api.metricLatency({ statistic: 'avg', period: cdk.Duration.minutes(1) }),
          api.metricLatency({ statistic: 'p90', period: cdk.Duration.minutes(1) }),
          api.metricLatency({ statistic: 'p99', period: cdk.Duration.minutes(1) }),
        ],
        width: 12,
      }),
      new cloudwatch.GraphWidget({
        title: 'API Gateway - Errors',
        left: [
          api.metricClientError({ statistic: 'sum', period: cdk.Duration.minutes(1) }),
          api.metricServerError({ statistic: 'sum', period: cdk.Duration.minutes(1) }),
        ],
        width: 12,
      }),
    ];
  }
  
  private createLambdaWidgets(functions: lambda.Function[]): cloudwatch.IWidget[] {
    const widgets: cloudwatch.IWidget[] = [];
    
    if (functions.length === 0) {
      return widgets;
    }
    
    // Invocations widget
    const invocationsMetrics: cloudwatch.IMetric[] = [];
    const errorsMetrics: cloudwatch.IMetric[] = [];
    const durationMetrics: cloudwatch.IMetric[] = [];
    
    functions.forEach(fn => {
      invocationsMetrics.push(fn.metricInvocations({
        statistic: 'sum',
        period: cdk.Duration.minutes(1),
      }));
      
      errorsMetrics.push(fn.metricErrors({
        statistic: 'sum',
        period: cdk.Duration.minutes(1),
      }));
      
      durationMetrics.push(fn.metricDuration({
        statistic: 'avg',
        period: cdk.Duration.minutes(1),
      }));
    });
    
    widgets.push(
      new cloudwatch.GraphWidget({
        title: 'Lambda - Invocations',
        left: invocationsMetrics,
        width: 12,
      }),
      new cloudwatch.GraphWidget({
        title: 'Lambda - Errors',
        left: errorsMetrics,
        width: 12,
      }),
      new cloudwatch.GraphWidget({
        title: 'Lambda - Duration',
        left: durationMetrics,
        width: 12,
      })
    );
    
    return widgets;
  }
  
  private createDynamoWidgets(tables: dynamodb.Table[]): cloudwatch.IWidget[] {
    const widgets: cloudwatch.IWidget[] = [];
    
    if (tables.length === 0) {
      return widgets;
    }
    
    const readCapacityMetrics: cloudwatch.IMetric[] = [];
    const writeCapacityMetrics: cloudwatch.IMetric[] = [];
    const throttleMetrics: cloudwatch.IMetric[] = [];
    
    tables.forEach(table => {
      readCapacityMetrics.push(table.metricConsumedReadCapacityUnits({
        statistic: 'sum',
        period: cdk.Duration.minutes(5),
      }));
      
      writeCapacityMetrics.push(table.metricConsumedWriteCapacityUnits({
        statistic: 'sum',
        period: cdk.Duration.minutes(5),
      }));
      
      throttleMetrics.push(
        new cloudwatch.Metric({
          namespace: 'AWS/DynamoDB',
          metricName: 'ReadThrottleEvents',
          dimensionsMap: { TableName: table.tableName },
          statistic: 'sum',
          period: cdk.Duration.minutes(5),
        }),
        new cloudwatch.Metric({
          namespace: 'AWS/DynamoDB',
          metricName: 'WriteThrottleEvents',
          dimensionsMap: { TableName: table.tableName },
          statistic: 'sum',
          period: cdk.Duration.minutes(5),
        })
      );
    });
    
    widgets.push(
      new cloudwatch.GraphWidget({
        title: 'DynamoDB - Read Capacity',
        left: readCapacityMetrics,
        width: 12,
      }),
      new cloudwatch.GraphWidget({
        title: 'DynamoDB - Write Capacity',
        left: writeCapacityMetrics,
        width: 12,
      }),
      new cloudwatch.GraphWidget({
        title: 'DynamoDB - Throttle Events',
        left: throttleMetrics,
        width: 12,
      })
    );
    
    return widgets;
  }
}