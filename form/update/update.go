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
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
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

// method to update a item of dynamodb forms table
func updateItem(item Form) (*dynamodb.UpdateItemOutput, error) {

	cfg, err := config.LoadDefaultConfig(context.TODO(), func(opts *config.LoadOptions) error {
		opts.Region = region
		return nil
	})

	if err != nil {
		log.Printf("ERROR ON updateItem METHOD: %s", err)
		panic(err)
	}

	svc := dynamodb.NewFromConfig(cfg)

	primaryKey := map[string]string{
		"form": item["form"].(string),
		"type": item["type"].(string),
	}

	pk, err := attributevalue.MarshalMap(primaryKey)

	upd := expression.Set(expression.Name("updatedAt"), expression.Value(time.Now()))
	for key, element := range item {
		if key != "form" && key != "type" {
			upd.Set(expression.Name(key), expression.Value(element))
		}
	}
	expr, err := expression.NewBuilder().WithUpdate(upd).Build()

	out, err := svc.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		Key:                       pk,
		TableName:                 aws.String(formsTable),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})

	if err != nil {
		log.Printf("ERROR ON updateItem METHOD: %s", err)
		return nil, fmt.Errorf("TrasnacitonWrite: %v\n", err)
	}

	return out, nil
}

type Response events.APIGatewayProxyResponse

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {

	ctx, seg := xray.BeginSubsegment(ctx, "UPDATE-FORM-SEGMENT")
	seg.AddMetadata("BODY", request.Body)

	log.Printf("REQUEST BODY: %s", request.Body)

	var form Form
	json.Unmarshal([]byte(request.Body), &form)

	out, err := updateItem(form)
	log.Printf("UPDATE ITEM OUTPUT: %s", out)

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
		Body:            "Successfully updated!",
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
