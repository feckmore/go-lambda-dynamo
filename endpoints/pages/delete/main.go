package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type Response events.APIGatewayProxyResponse
type Request events.APIGatewayProxyRequest
type SiteStatus int

const (
	Unpublished SiteStatus = iota
	Published
)

// Site defines the fields of the site model
type Site struct {
	ID           string     `json:"id"`
	Version      string     `json:"version"`
	Path         string     `json:"path"`
	Type         string     `json:"type"`
	Status       SiteStatus `json:"status,omitempty"`
	Name         *string    `json:"name,omitempty"`
	Description  *string    `json:"description,omitempty"`
	Keywords     *string    `json:"keywords,omitempty"`
	URL          *string    `json:"url,omitempty"`
	TagManagerID *string    `json:"tagManagerId,omitempty"`
	CardImageURL *string    `json:"cardImageUrl,omitempty"`
	CreatedAt    time.Time  `json:"createdAt,omitempty"`
	UpdatedAt    time.Time  `json:"updatedAt,omitempty"`
}

var db *dynamodb.DynamoDB
var region, stage, table string
var currentTime time.Time

func init() {
	// Enable line numbers in log output, but remove date/time
	log.SetFlags(log.Llongfile)

	region = strings.TrimSpace(os.Getenv("AWS_REGION"))
	stage = os.Getenv("STAGE")
	table = os.Getenv("TABLE_NAME")

	log.Println("AWS_REGION:", region)
	log.Println("STAGE:", stage)
	log.Println("TABLE_NAME:", table)

	// TODO: validate env vars
}

// main starts the session, news up the db & invokes lambda handler
func main() {
	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		log.Println("Failed to connect to AWS:", err)
	} else {
		db = dynamodb.New(session)
	}

	lambda.Start(Handler)
}

// Handler is our lambda handler invoked by the `lambda.Start` function call in main()
func Handler(ctx context.Context, request Request) (Response, error) {
	// siteid := aws.String(request.PathParameters["siteid"])
	pageid := aws.String(request.PathParameters["pageid"])
	version := aws.String(request.QueryStringParameters["version"])

	// TODO: also option to delete all versions?

	key := map[string]*dynamodb.AttributeValue{
		"id":      {S: pageid},
		"version": {S: version},
	}

	log.Println(key)

	_, err := db.DeleteItem(&dynamodb.DeleteItemInput{
		Key:       key,
		TableName: aws.String(table),
	})
	if err != nil {
		return Response{StatusCode: 500}, err // TODO: decide what's the correct status
	}

	// TODO: consider returning body with status

	response := Response{
		StatusCode: 200,
		Headers: map[string]string{
			"Access-Control-Allow-Origin":      "*",
			"Access-Control-Allow-Credentials": "true",
		},
	}

	return response, nil
}
