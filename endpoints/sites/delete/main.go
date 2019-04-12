package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var db *dynamodb.DynamoDB

type Response events.APIGatewayProxyResponse
type Request events.APIGatewayProxyRequest

// Handler is our lambda handler invoked by the `lambda.Start` function call in main()
func Handler(ctx context.Context, req Request) (Response, error) {
	id := aws.String(req.PathParameters["id"])
	key := map[string]*dynamodb.AttributeValue{"id": {S: id}}

	_, err := db.DeleteItem(&dynamodb.DeleteItemInput{
		Key:       key,
		TableName: aws.String("todos"),
	})
	if err != nil {
		return Response{StatusCode: 500}, err // TODO: decide what's the correct status
	}

	resp := Response{
		StatusCode: 200,
		Headers: map[string]string{
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
