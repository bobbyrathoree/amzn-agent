package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/bobbyrathore/go-lambda-backend/internal/handlers"
	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
	"github.com/bobbyrathore/go-lambda-backend/pkg/repositories"
	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
	appConfig "github.com/bobbyrathore/go-lambda-backend/pkg/config"
)

var (
	// Initialize these at package level for reuse across invocations
	itemService *services.ItemService
	cfg         *appConfig.Config
	sqsClient   *sqs.Client
)

func init() {
	// Load configuration
	cfg = appConfig.NewConfig()

	// Initialize AWS SDK
	awsConfig, err := config.LoadDefaultConfig(context.Background(), 
		config.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		log.Fatalf("Unable to load AWS SDK config: %v", err)
	}

	// Create DynamoDB client
	dynamoClient := dynamodb.NewFromConfig(awsConfig)
	
	// Create SQS client
	sqsClient = sqs.NewFromConfig(awsConfig)

	// Initialize repository
	itemRepo := repositories.NewDynamoDBItemRepository(dynamoClient, cfg.DynamoDBTable)

	// Initialize service
	itemService = services.NewItemService(itemRepo)
}

// Message structure for expected SQS messages
type Message struct {
	Action string                  `json:"action"`
	Data   models.CreateItemRequest `json:"data"`
}

// Handler is the Lambda function handler for SQS events
func Handler(ctx context.Context, event events.SQSEvent) error {
	for _, record := range event.Records {
		log.Printf("Processing SQS message: %s", record.MessageId)
		
		// Parse the message
		var message Message
		err := json.Unmarshal([]byte(record.Body), &message)
		if err != nil {
			log.Printf("Error parsing message: %v", err)
			continue // Process next message even if this one fails
		}
		
		// Process based on the action
		switch message.Action {
		case "create_item":
			// Create a new item
			item, err := itemService.CreateItem(ctx, message.Data)
			if err != nil {
				log.Printf("Error creating item: %v", err)
				continue
			}
			log.Printf("Successfully created item with ID: %s", item.ID)
			
		case "delete_items_batch":
			// This would handle batch deletion logic
			// Implementation omitted for brevity
			log.Printf("Batch deletion not implemented yet")
			
		default:
			log.Printf("Unknown action: %s", message.Action)
		}
	}
	
	return nil
}

func main() {
	lambda.Start(Handler)
}