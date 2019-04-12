// package main

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"log"

// 	"github.com/aws/aws-lambda-go/events"
// 	"github.com/aws/aws-lambda-go/lambda"
// 	"github.com/aws/aws-sdk-go/aws"
// 	"github.com/aws/aws-sdk-go/aws/session"
// 	"github.com/aws/aws-sdk-go/service/dynamodb"
// 	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
// )

// var db *dynamodb.DynamoDB

// type Response events.APIGatewayProxyResponse
// type Request events.APIGatewayProxyRequest

// // Handler is our lambda handler invoked by the `lambda.Start` function call in main()
// func Handler(ctx context.Context, req Request) (Response, error) {
// 	id := aws.String(req.PathParameters["id"])
// 	key := map[string]*dynamodb.AttributeValue{"id": {S: id}}

// 	result, err := db.GetItem(&dynamodb.GetItemInput{
// 		Key:       key,
// 		TableName: aws.String("todos"),
// 	})

// 	if err != nil {
// 		return Response{StatusCode: 500}, err // TODO: decide what's the correct status
// 	}

// 	var castTo interface{} // TODO: Consider replacing interface{}?
// 	if err := dynamodbattribute.UnmarshalMap(result.Item, &castTo); err != nil {
// 		return Response{StatusCode: 404}, err // TODO: decide what's the correct status
// 	}
// 	body, err := json.Marshal(castTo)
// 	if err != nil {
// 		return Response{StatusCode: 404}, err // TODO: decide what's the correct status
// 	}

// 	var buf bytes.Buffer
// 	json.HTMLEscape(&buf, body)

// 	resp := Response{
// 		StatusCode:      200,
// 		IsBase64Encoded: false,
// 		Body:            buf.String(),
// 		Headers: map[string]string{
// 			"Content-Type":                     "application/json",
// 			"Access-Control-Allow-Origin":      "*",
// 			"Access-Control-Allow-Credentials": "true",
// 		},
// 	}

// 	return resp, nil
// }

// func main() {
// 	session, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")}) // TODO: REMOVE HARD-CODED REGION string
// 	if err != nil {
// 		log.Println(fmt.Sprintf("Failed to connect to AWS: %s", err.Error()))
// 	} else {
// 		db = dynamodb.New(session)
// 	}

// 	lambda.Start(Handler)
// }

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
	var site Site

	id := aws.String(request.PathParameters["id"])
	version := aws.String(request.QueryStringParameters["version"])

	// key := expression.Key("type").Equal(expression.Value("site"))
	// builder := expression.NewBuilder().WithKeyCondition(key)
	// expr, err := builder.Build()
	// if err != nil {
	// 	log.Println("Error building dynamodb expression")
	// 	return Response{StatusCode: http.StatusInternalServerError}, err
	// }

	// queryInput := dynamodb.QueryInput{
	// 	KeyConditionExpression:    expr.KeyCondition(),
	// 	ExpressionAttributeNames:  expr.Names(),
	// 	ExpressionAttributeValues: expr.Values(),
	// 	IndexName:                 aws.String("type-path-index"),
	// 	TableName:                 aws.String(table),
	// }
	// results, err := db.Query(&queryInput)
	// if err != nil {
	// 	log.Println("Error querying sites in dynamodb")
	// 	return Response{StatusCode: http.StatusInternalServerError}, err
	// }

	key := map[string]*dynamodb.AttributeValue{
		"id":      {S: id},
		"version": {S: version},
	}

	result, err := db.GetItem(&dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(table),
	})
	if err != nil {
		log.Println("Error getting item from dynamodb")
		return Response{StatusCode: http.StatusInternalServerError}, err
	}

	err = dynamodbattribute.UnmarshalMap(result.Item, &site)
	if err != nil {
		log.Println("Error unmarshalling into site")
		return Response{StatusCode: http.StatusInternalServerError}, err
	}

	body, err := json.Marshal(site)
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
