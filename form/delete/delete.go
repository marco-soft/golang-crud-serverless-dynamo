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

type Response events.APIGatewayProxyResponse

type Form map[string]interface{}

func deleteItem(responseForm string) (*dynamodb.DeleteItemOutput, error) {

	cfg, err := config.LoadDefaultConfig(context.TODO(), func(opts *config.LoadOptions) error {
		opts.Region = "us-east-1"
		return nil
	})

	if err != nil {
		panic(err)
	}

	svc := dynamodb.NewFromConfig(cfg)

	selectedKeys := map[string]string{
		"form": responseForm,
		"type": "2022-05-30T14:49:59.104317389Z",
	}

	key, err := attributevalue.MarshalMap(selectedKeys)

	log.Printf("DYNAMODB KEY: %s", key)

	//cond := expression.Equal(
	//	expression.Name("anyIntField"),
	//	expression.Value(1))

	//expr, err := expression.NewBuilder().WithCondition(cond).Build()

	out, err := svc.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: aws.String("jubla-forms-responses"),
		Key:       key,
		//ExpressionAttributeNames:  expr.Names(),
		//ExpressionAttributeValues: expr.Values(),
		//ConditionExpression:       expr.Condition(),
	})

	if err != nil {
		panic(err)
	}

	return out, nil
}

func HandlerDelete(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {

	ctx, seg := xray.BeginSubsegment(ctx, "MY-CUSTOM-SEGMENT")
	seg.AddMetadata("BODY", request.Body)

	log.Printf("REQUEST BODY: %s", request.Body)

	var form Form
	json.Unmarshal([]byte(request.Body), &form)

	_, err := deleteItem(form["id"].(string))

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
