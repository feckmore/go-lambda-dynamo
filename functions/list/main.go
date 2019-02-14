package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var db *dynamodb.DynamoDB

type Response events.APIGatewayProxyResponse

// Handler is our lambda handler invoked by the `lambda.Start` function call in main()
func Handler(ctx context.Context) (Response, error) {
	results, err := db.Scan(&dynamodb.ScanInput{
		TableName: aws.String("todos"),
	})
	if err != nil {
		return Response{StatusCode: 500}, err // TODO: decide what's the correct status
	}

	var castTo interface{} // TODO: Consider replacing interface{}?
	if err := dynamodbattribute.UnmarshalListOfMaps(results.Items, &castTo); err != nil {
		return Response{StatusCode: 404}, err // TODO: decide what's the correct status
	}
	body, err := json.Marshal(castTo)
	if err != nil {
		return Response{StatusCode: 404}, err // TODO: decide what's the correct status
	}

	var buf bytes.Buffer
	json.HTMLEscape(&buf, body)

	resp := Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            buf.String(),
		Headers: map[string]string{
			"Content-Type":                     "application/json",
			"Access-Control-Allow-Origin":      "*",
			"Access-Control-Allow-Credentials": "true",
		},
	}

	return resp, nil
}

func main() {
	session, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")}) // TODO: REMOVE HARD-CODED REGION string
	if err != nil {
		log.Println(fmt.Sprintf("Failed to connect to AWS: %s", err.Error()))
	} else {
		db = dynamodb.New(session)
	}

	lambda.Start(Handler)
}
