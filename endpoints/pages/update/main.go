package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type Response events.APIGatewayProxyResponse
type Request events.APIGatewayProxyRequest

// Page defines the fields of the page model
type Page struct {
	ID          string    `json:"id"`
	Version     string    `json:"version"`
	Path        string    `json:"path"`
	Type        string    `json:"type"`
	Name        *string   `json:"name,omitempty"`
	Description *string   `json:"description,omitempty"`
	Keywords    *string   `json:"keywords,omitempty"`
	Author      *string   `json:"author,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
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
	// Get existing page from datbase
	var original Page
	pageid := aws.String(request.PathParameters["pageid"])
	version := aws.String(request.QueryStringParameters["version"]) //TODO: get latest version, or latest for changeset

	//TODO: look for path also for different query

	key := map[string]*dynamodb.AttributeValue{
		"id":      {S: pageid},
		"version": {S: version},
	}

	result, err := db.GetItem(&dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(table),
	})
	if err != nil {
		log.Println("Error getting page from dynamodb")
		return Response{StatusCode: http.StatusInternalServerError}, err
	}

	err = dynamodbattribute.UnmarshalMap(result.Item, &original)
	if err != nil {
		log.Println("Error unmarshalling into page")
		return Response{StatusCode: http.StatusInternalServerError}, err
	}

	// make sure there's a valid page returned from the database
	if original.ID == "" {
		log.Println("No page returned from database query")
		return Response{StatusCode: http.StatusNotFound}, err
	}

	// Get page changes from request body
	var changes Page
	err = json.Unmarshal([]byte(request.Body), &changes)
	if err != nil {
		log.Println("Error unmarshalling request body into page")
		return Response{StatusCode: http.StatusBadRequest}, err
	}

	// combine original page with requested changes
	changes.ID = original.ID
	changes.Version = original.Version //TODO: new version
	changes.Type = "page"
	if len(changes.Path) == 0 {
		changes.Path = original.Path
	}
	changes.CreatedAt = original.CreatedAt
	changes.UpdatedAt = time.Now()
	updated, err := mergePages(&original, &changes)
	if err != nil {
		log.Println("Error merging page attributes")
		return Response{StatusCode: http.StatusBadRequest}, err
	}

	av, err := dynamodbattribute.MarshalMap(updated)
	if err != nil {
		log.Println("Error marshalling page into dynamodb attribute")
		return Response{StatusCode: http.StatusInternalServerError}, err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(table),
	}

	_, err = db.PutItem(input)
	if err != nil {
		log.Println("Error putting item into DyanmoDB")
		return Response{StatusCode: http.StatusBadRequest}, err
	}

	body, err := json.Marshal(updated)
	if err != nil {
		log.Println("Error marshalling page into json for response body")
		return Response{StatusCode: http.StatusInternalServerError}, err
	}

	response := Response{
		StatusCode: http.StatusOK,
		Body:       string(body),
		Headers: map[string]string{
			"Content-Type":                     "application/json",
			"Access-Control-Allow-Origin":      "*",
			"Access-Control-Allow-Credentials": "true",
		},
	}

	return response, nil

}

// mergePages merges two structs by serializing the struct with the changes to JSON, then
// deserializes the changes into the original
func mergePages(original, changes *Page) (*Page, error) {
	// serialize changes to JSON
	changeJSON, err := json.Marshal(changes)
	if err != nil {
		return nil, err
	}

	// deserialize the "changes" page struct into the original
	err = json.Unmarshal(changeJSON, &original)
	if err != nil {
		return nil, err
	}

	return original, nil
}
