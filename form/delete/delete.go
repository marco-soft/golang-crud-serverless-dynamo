package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-xray-sdk-go/xray"
	"log"
)

// aws config
var region = "us-east-1"
var formsTable = "jubla-forms-responses"

// Form to receive, is map because is dynamic
type Form map[string]interface{}

// method to deleteItem a form item to dynamodb
func deleteItem(item Form) (*dynamodb.DeleteItemOutput, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), func(opts *config.LoadOptions) error {
		opts.Region = region
		return nil
	})

	if err != nil {
		panic(err)
	}

	svc := dynamodb.NewFromConfig(cfg)

	selectedKeys := map[string]string{
		"form": item["form"].(string),
		"type": item["type"].(string),
	}

	key, err := attributevalue.MarshalMap(selectedKeys)
	if err != nil {
		log.Printf("ERROR ON deleteItem METHOD: %s", err)
		panic(err)
	}

	out, err := svc.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: aws.String(formsTable),
		Key:       key,
	})

	if err != nil {
		log.Printf("ERROR ON deleteItem METHOD: %s", err)
		panic(err)
	}

	return out, nil
}

type Response events.APIGatewayProxyResponse

func HandlerDelete(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {

	ctx, seg := xray.BeginSubsegment(ctx, "DELETE-FORM-SEGMENT")
	seg.AddMetadata("BODY", request.Body)

	log.Printf("REQUEST BODY: %s", request.Body)

	var form Form
	json.Unmarshal([]byte(request.Body), &form)

	out, err := deleteItem(form)
	log.Printf("DELETE ITEM OUTPUT: %s", out)

	if err != nil {
		log.Fatalf("LoadDefaultConfig: %v\n", err)
		resp := Response{
			StatusCode:      500,
			IsBase64Encoded: false,
		}
		return resp, nil
	}

	resp := Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            "Successfully deleted!",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	seg.AddMetadata("RESPONSE", resp)
	seg.Close(err)
	log.Printf("SUCCESS")

	return resp, nil
}

func main() {
	lambda.Start(HandlerDelete)
}
