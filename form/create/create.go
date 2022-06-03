package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-xray-sdk-go/xray"
	"log"
	"time"
)

// aws config
var region = "us-east-1"
var formsTable = "jubla-forms-responses"

// Form to receive, is map because is dynamic
type Form map[string]interface{}

// method to insert a form item to dynamodb
func insertItem(item Form) error {
	cfg, err := config.LoadDefaultConfig(context.TODO(), func(opts *config.LoadOptions) error {
		opts.Region = region
		return nil
	})

	if err != nil {
		log.Printf("ERROR ON insertInput METHOD: %s", err)
		panic(err)
	}

	svc := dynamodb.NewFromConfig(cfg)

	data, err := attributevalue.MarshalMap(item)

	if err != nil {
		return fmt.Errorf("MarshalMap: %v\n", err)
	}

	_, err = svc.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(formsTable),
		Item:      data,
	})

	if err != nil {
		log.Printf("ERROR ON insertInput METHOD: %s", err)
		return fmt.Errorf("PutItem: %v\n", err)
	}

	return nil
}

type Response events.APIGatewayProxyResponse

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {

	ctx, seg := xray.BeginSubsegment(ctx, "CREATE-FORM-SEGMENT")

	seg.AddMetadata("BODY", request.Body)
	log.Printf("REQUEST BODY: %s", request.Body)

	var form Form
	json.Unmarshal([]byte(request.Body), &form)
	form["type"] = "response"
	form["createdAt"] = time.Now()

	err := insertItem(form)

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
		Body:            "Successfully created!",
		Headers: map[string]string{
			"Content-Type":                 "application/json",
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Headers": "Content-Type,access-control-allow-origin, access-control-allow-headers",
		},
	}

	seg.AddMetadata("RESPONSE", resp)
	seg.Close(err)
	log.Printf("SUCCESS")

	return resp, nil
}

func main() {
	lambda.Start(Handler)
}
