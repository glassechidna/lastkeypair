package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"encoding/json"
	"github.com/glassechidna/lastkeypair/pkg/lastkeypair"
)

func HandleLambdaEvent(event json.RawMessage) (interface{}, error) {
	return lastkeypair.LambdaHandle(event)
}

func main() {
	lambda.Start(HandleLambdaEvent)
}
