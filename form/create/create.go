package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/grokify/go-awslambda"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// aws config
var region = "us-east-1"
var formsTable = "jubla-forms-responses"

// Uploads a file to AWS S3 given an S3 session client, a bucket name and a file path
func uploadFileToS3(
	bucketName string,
	filePath string,
) error {
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		log.Fatal(err)
	}
	if err != nil {
		log.Fatalf("could not initialize new aws session: %v", err)
	}

	// Initialize an s3 client from the session created
	s3Client := s3.New(session)
	// Get the fileName from Path
	fileName := filepath.Base(filePath)

	// Open the file from the file path
	upFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("could not open local filepath [%v]: %+v", filePath, err)
	}
	defer upFile.Close()

	// Get the file info
	upFileInfo, _ := upFile.Stat()
	var fileSize int64 = upFileInfo.Size()
	fileBuffer := make([]byte, fileSize)
	upFile.Read(fileBuffer)

	// Put the file object to s3 with the file name
	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(bucketName),
		Key:                  aws.String(fileName),
		ACL:                  aws.String("private"),
		Body:                 bytes.NewReader(fileBuffer),
		ContentLength:        aws.Int64(fileSize),
		ContentType:          aws.String(http.DetectContentType(fileBuffer)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})
	return err
}

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
	r, err := awslambda.NewReaderMultipart(request)

	if err != nil {
		panic(err)
	}
	part, err := r.NextPart()
	if err != nil {
		panic(err)
	}
	content, err := io.ReadAll(part)
	if err != nil {
		panic(err)
	}

	filename := extractFile(request)

	uploadFileToS3("", filename)
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
