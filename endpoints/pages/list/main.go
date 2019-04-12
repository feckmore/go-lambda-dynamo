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
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

type Response events.APIGatewayProxyResponse
type Request events.APIGatewayProxyRequest

// Page defines the fields of the page model
type Page struct {
	ID          string    `json:"id"`
	Version     string    `json:"version"`
	Path        string    `json:"path"`
	Type        string    `json:"type,omitempty"`
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
	sitePath := request.PathParameters["siteid"]
	// TODO: validate site path

	var pages []Page

	// define key condition for sort to begin with
	sortCondition := expression.Key("path").BeginsWith(sitePath)
	// "And" the sort key with partition key
	key := expression.Key("type").Equal(expression.Value("page")).And(sortCondition)
	// projection represents the list of attribute names
	projection := expression.NamesList(expression.Name("id"), expression.Name("version"), expression.Name("path"), expression.Name("createdAt"), expression.Name("updatedAt"), expression.Name("name"))

	builder := expression.NewBuilder().WithKeyCondition(key).WithProjection(projection)
	expr, err := builder.Build()
	if err != nil {
		log.Println("Error building dynamodb expression")
		return Response{StatusCode: http.StatusInternalServerError}, err
	}

	queryInput := dynamodb.QueryInput{
		KeyConditionExpression:    expr.KeyCondition(),
		ProjectionExpression:      expr.Projection(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		IndexName:                 aws.String("type-path-index"),
		TableName:                 aws.String(table),
	}

	results, err := db.Query(&queryInput)
	if err != nil {
		log.Println("Error querying pages in dynamodb")
		return Response{StatusCode: http.StatusInternalServerError}, err
	}

	err = dynamodbattribute.UnmarshalListOfMaps(results.Items, &pages)
	if err != nil {
		log.Println("Error unmarshalling into pages slice")
		return Response{StatusCode: http.StatusInternalServerError}, err
	}

	body, err := json.Marshal(pages)
	if err != nil {
		log.Println("Error marshalling pages into json for response body")
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
