package main

import (
	"github.com/glassechidna/lastkeypair/cmd"
	"github.com/eawsy/aws-lambda-go-core/service/lambda/runtime"
	"encoding/json"
	"github.com/glassechidna/lastkeypair/pkg/lastkeypair"
)

func main() {
	cmd.Execute()
}

func Handle(evt json.RawMessage, ctx *runtime.Context) (interface{}, error) {
	return lastkeypair.LambdaHandle(evt, ctx)
}
