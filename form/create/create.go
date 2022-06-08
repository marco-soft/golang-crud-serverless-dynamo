package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/grokify/go-awslambda"
)

// aws config
var region = "us-east-1"
var formsTable = "jubla-forms-responses"

// Uploads a file to AWS S3 given an S3 session client, a bucket name and a file path
func uploadFileToS3(
	bucketName string,
	filePath string,
	body string,
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
	log.Print("s3Client: ", s3Client, bucketName, filePath)

	// Put the file object to s3 with the file name
	
	log, err := s3Client.PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(bucketName),
		Key:                  aws.String(filePath),
		Body:                 bytes.NewReader([]byte(body)),
	})
fmt.Printf("log: %v\n", log)
fmt.Printf("log: %v\n", err)
	return err
}

// Form to receive, is map because is dynamic
type Form map[string]interface{}

func extractFile(request events.APIGatewayProxyRequest) (*multipart.Part){
	log.Print("extractFile")
	contentType := request.Headers["content-type"]
	log.Print(contentType)
	mediaType, params, err := mime.ParseMediaType(contentType)
	log.Print(mediaType)
	if err != nil {
		log.Fatal(err)
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		log.Print(request)
		mr := multipart.NewReader(strings.NewReader(request.Body), params["boundary"])
		log.Print(mr)
		for {
			p, err := mr.NextPart()
			log.Print("PART: ", p)
			if err == io.EOF {
				return nil
			}
			if err != nil {
				log.Print("ERROR ON extractFile METHOD: ", err)
				log.Fatal(err)
			}
			// p.FormName() is the name of the element.
			// p.FileName() is the name of the file (if it's a file)
			// p is an io.Reader on the part

			// The following code prints the part for demonstration purposes.
			slurp, err := ioutil.ReadAll(p)
			log.Print(slurp)
			if err != nil {
				log.Fatal(err)
			}
			return p
		}
	}
	return nil
}

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

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	res := events.APIGatewayProxyResponse{}
	form := make(map[string]interface{})
	r, err := awslambda.NewReaderMultipart(request)
    if err != nil {
        return res, err
    }
	for {
		part, err_part := r.NextPart()
		if err_part == io.EOF {
		   break
		}
		log.Print("PART: ", part.FormName())
		if part.FormName() == "dni" || part.FormName() == "carta_pastoral" {
			content, err := io.ReadAll(part)
			if err != nil {
				return res, err
			}
			uploadFileToS3("jubla-forms-docs", part.FileName(), string(content))

		} else{
			buf := new(bytes.Buffer)
      		buf.ReadFrom(part)
			form[part.FormName()] = buf.String()
	 }
    }
	form["type"] = "response"
	form["createdAt"] = time.Now()

	log.Print("LLEGUE POR ACA", form)

	err = insertItem(form)

	if err != nil {
		log.Fatalf("LoadDefaultConfig: %v\n", err)
		return res, err
	}

	res = events.APIGatewayProxyResponse{
        StatusCode: 200,
        Headers: map[string]string{
            "Content-Type": "application/json"},
        Body: "SUCCESS!!"}
    return res, nil

}

type customStruct struct {
    Content       string
    FileName      string
    FileExtension string
}

func handleRequest(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    res := events.APIGatewayProxyResponse{}
    r, err := awslambda.NewReaderMultipart(req)
    if err != nil {
        return res, err
    }
    part, err := r.NextPart()
    if err != nil {
        return res, err
    }
    content, err := io.ReadAll(part)
    if err != nil {
        return res, err
    }
    custom := customStruct{
        Content:       string(content),
        FileName:      part.FileName(),
        FileExtension: filepath.Ext(part.FileName())}
		
	log.Print(part);
	uploadFileToS3("jubla-forms-docs", custom.FileName, custom.Content)

    customBytes, err := json.Marshal(custom)
    if err != nil {
        return res, err
    }

    res = events.APIGatewayProxyResponse{
        StatusCode: 200,
        Headers: map[string]string{
            "Content-Type": "application/json"},
        Body: string(customBytes)}
    return res, nil
}

func main() {
	lambda.Start(Handler)
}
