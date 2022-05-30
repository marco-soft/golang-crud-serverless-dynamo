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

func insertItem(item Form) error {

	cfg, err := config.LoadDefaultConfig(context.TODO(), func(opts *config.LoadOptions) error {
		opts.Region = "us-east-1"
		return nil
	})
	if err != nil {
		panic(err)
	}

	svc := dynamodb.NewFromConfig(cfg)

	data, err := attributevalue.MarshalMap(item)

	if err != nil {
		return fmt.Errorf("MarshalMap: %v\n", err)
	}

	_, err = svc.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String("jubla-forms-responses"),
		Item:      data,
	})

	if err != nil {
		log.Printf("I am an error: %s", err)
		// create the input configuration instance
		input := &dynamodb.ListTablesInput{}
		result, err2 := svc.ListTables(context.TODO(), input)
		if err2 != nil {
			log.Printf("I am an error in list tables: %s", err2)
		} else {
			log.Printf("This is the tables: %s", result)
		}
		return fmt.Errorf("PutItem: %v\n", err)
	}

	return nil
}

type Response events.APIGatewayProxyResponse

type Form map[string]interface{}

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {

	ctx, seg := xray.BeginSubsegment(ctx, "MY-CUSTOM-SEGMENT")
	seg.AddMetadata("BODY", request.Body)

	log.Printf("REQUEST BODY: %s", request.Body)

	var form Form
	json.Unmarshal([]byte(request.Body), &form)

	form["type"] = time.Now()

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
			"Content-Type": "application/json",
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
