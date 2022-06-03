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
	"log"
)

// aws config
var region = "us-east-1"
var formsTable = "jubla-forms-responses"

type Form struct {
	Form      string
	Type      string
	Structure string
}

// method to get all items from dynamodb forms table
func getForm(formId string, typeForm string) (Form, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), func(opts *config.LoadOptions) error {
		opts.Region = region
		return nil
	})

	if err != nil {
		log.Printf("ERROR ON getAllItems METHOD: %s", err)
		panic(err)
	}
	log.Printf("MARCO Y SANTI: %s, %s", formId, typeForm)

	selectedKeys := map[string]string{
		"form": formId,
		"type": typeForm,
	}

	key, err := attributevalue.MarshalMap(selectedKeys)
	if err != nil {
		log.Printf("ERROR ON deleteItem METHOD: %s", err)
		panic(err)
	}
	svc := dynamodb.NewFromConfig(cfg)
	out, err := svc.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(formsTable),
		Key:       key,
	})

	var item Form

	err = attributevalue.UnmarshalMap(out.Item, &item)
	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal Record, %v", err))
	}

	return item, nil
}

type Response events.APIGatewayProxyResponse

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	log.Printf("MARCO Y SANTI: %s", "form")

	data, err := getForm(request.PathParameters["id"], "form")

	if err != nil {
		log.Fatalf("LoadDefaultConfig: %v\n", err)
		resp := Response{
			StatusCode:      500,
			IsBase64Encoded: false,
		}
		return resp, nil
	}
	bodyResponse, _ := json.Marshal(data)
	resp := Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            string(bodyResponse),
		Headers: map[string]string{
			"Content-Type":                 "application/json",
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Headers": "Content-Type,access-control-allow-origin, access-control-allow-headers",
		},
	}

	log.Printf("SUCCESS")

	return resp, nil
}

func main() {
	lambda.Start(Handler)
}
