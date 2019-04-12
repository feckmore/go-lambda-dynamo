package main

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/google/uuid"
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
	currentTime = time.Now()

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
	// TODO: what to do with the site id?
	// siteid := request.PathParameters["siteid"]

	var page *Page
	err := json.Unmarshal([]byte(request.Body), &page)
	if err != nil {
		log.Println("Error unmarshalling request body into page")
		return Response{StatusCode: http.StatusBadRequest}, err
	}
	log.Println(page)
	if strings.TrimSpace(page.Path) == "" {
		log.Println("Can't create page without path")
		return Response{StatusCode: http.StatusBadRequest}, errors.New("Can't create page without path")
	}

	page.ID = uuid.New().String()
	page.Version = uuid.New().String()
	page.Path = strings.ToLower(page.Path)
	page.Type = "page"
	page.CreatedAt = currentTime
	page.UpdatedAt = currentTime

	av, err := dynamodbattribute.MarshalMap(page)
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

	body, err := json.Marshal(page)
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
