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
	// TODO: if we pulled the Get Site & Create Site out from the handler, just passing in the Site object,
	// we could call Get Site, then merge with body & call Create/Put Site functions.
	// Keeping all of the code in-line for now

	// Get existing site from datbase
	var original Site
	id := aws.String(request.PathParameters["siteid"])
	version := aws.String(request.QueryStringParameters["version"]) //TODO: get latest version, or latest for changeset

	//TODO: look for path also for different query

	key := map[string]*dynamodb.AttributeValue{
		"id":      {S: id},
		"version": {S: version},
	}

	result, err := db.GetItem(&dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(table),
	})
	if err != nil {
		log.Println("Error getting site from dynamodb")
		return Response{StatusCode: http.StatusInternalServerError}, err
	}

	err = dynamodbattribute.UnmarshalMap(result.Item, &original)
	if err != nil {
		log.Println("Error unmarshalling into site")
		return Response{StatusCode: http.StatusInternalServerError}, err
	}

	// body, err := json.Marshal(site)
	// if err != nil {
	// 	log.Println("Error marshalling site into json for response body")
	// 	return Response{StatusCode: http.StatusInternalServerError}, err
	// }

	// make sure there's a valid site returned
	if original.ID == "" {
		log.Println("No site returned from database query")
		return Response{StatusCode: http.StatusNotFound}, err
	}

	// Get site changes from request body
	var changes Site
	err = json.Unmarshal([]byte(request.Body), &changes)
	if err != nil {
		log.Println("Error unmarshalling request body into site")
		return Response{StatusCode: http.StatusBadRequest}, err
	}

	// combine original site with requested changes
	changes.ID = original.ID
	changes.Version = original.Version //TODO: new version
	changes.Type = "site"
	if len(changes.Path) == 0 {
		changes.Path = original.Path
	}
	changes.CreatedAt = original.CreatedAt
	changes.UpdatedAt = time.Now()
	updated, err := mergeSites(&original, &changes)
	if err != nil {
		log.Println("Error merging site attributes")
		return Response{StatusCode: http.StatusBadRequest}, err
	}

	av, err := dynamodbattribute.MarshalMap(updated)
	if err != nil {
		log.Println("Error marshalling site into dynamodb attribute")
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
		log.Println("Error marshalling site into json for response body")
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

// mergeSites merges two structs by serializing the struct with the changes to JSON, then
// deserializes the changes into the original
func mergeSites(original, changes *Site) (*Site, error) {
	// serialize changes to JSON
	changeJSON, err := json.Marshal(changes)
	if err != nil {
		return nil, err
	}

	// deserialize the "changes" site struct into the original
	err = json.Unmarshal(changeJSON, &original)
	if err != nil {
		return nil, err
	}

	return original, nil
}
