package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
)

var db *dynamodb.DynamoDB

type Response events.APIGatewayProxyResponse
type Request events.APIGatewayProxyRequest

// Handler is our lambda handler invoked by the `lambda.Start` function call in main()
func Handler(ctx context.Context, req Request) (Response, error) {
	var reqBody interface{}
	err := json.Unmarshal([]byte(req.Body), &reqBody) // TODO: extract http request transformation out
	if err != nil {
		return Response{StatusCode: 404}, err // TODO: decide what's the correct status
	}

	item, err := dynamodbattribute.MarshalMap(reqBody)
	if err != nil {
		return Response{StatusCode: 404}, err // TODO: decide what's the correct status
	}
	id := uuid.New().String()
	item["id"] = &dynamodb.AttributeValue{S: &id}

	t := time.Now().Format(time.RFC3339)
	item["createdAt"] = &dynamodb.AttributeValue{S: &t}
	item["updatedAt"] = &dynamodb.AttributeValue{S: &t}

	input := &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String("todos"),
	}
	_, err = db.PutItem(input)
	if err != nil {
		return Response{StatusCode: 404}, err // TODO: decide what's the correct status
	}

	var buf bytes.Buffer
	var r interface{}
	err = dynamodbattribute.UnmarshalMap(item, &r)
	if err != nil {
		return Response{StatusCode: 404}, err // TODO: decide what's the correct status
	}

	respBody, _ := r.(string)
	json.HTMLEscape(&buf, []byte(respBody))

	resp := Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            req.Body, //buf.String(),
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
