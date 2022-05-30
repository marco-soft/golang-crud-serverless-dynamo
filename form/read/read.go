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
)

// aws config
var region = "us-east-1"
var formsTable = "jubla-forms-responses"

// Form to receive, is map because is dynamic
type Form map[string]interface{}

// method to get all items from dynamodb forms table
func getAllItems() ([]Form, error) {
	var items []Form

	cfg, err := config.LoadDefaultConfig(context.TODO(), func(opts *config.LoadOptions) error {
		opts.Region = region
		return nil
	})

	if err != nil {
		log.Printf("ERROR ON getAllItems METHOD: %s", err)
		panic(err)
	}
	filter := expression.Name("type").Equal(expression.Value("response"))
	expr, err := expression.NewBuilder().WithFilter(filter).Build()
	if err != nil {
		log.Printf("ERROR ON getAllItems METHOD: %s", err)
		fmt.Println("Got error building expression:")
		fmt.Println(err.Error())
		return nil, nil
	}

	svc := dynamodb.NewFromConfig(cfg)

	data, err := svc.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName:                 aws.String(formsTable),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
	})

	err = attributevalue.UnmarshalListOfMaps(data.Items, &items)

	if err != nil {
		return items, fmt.Errorf("UnmarshalListOfMaps: %v\n", err)
	}

	return items, nil
}

type Response events.APIGatewayProxyResponse

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {

	ctx, seg := xray.BeginSubsegment(ctx, "READ-FORMS-SEGMENT")
	seg.AddMetadata("BODY", request)

	var form Form
	json.Unmarshal([]byte(request.Body), &form)

	data, err := getAllItems()

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
