package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	runtime "github.com/aws/aws-lambda-go/lambda"
	telegram "github.com/go-telegram-bot-api/telegram-bot-api"
)

type MethodResponse struct {
	Method string `json:"method"`
}

type SendMessageMethodResponse struct {
	MethodResponse
	telegram.MessageConfig
}

func main() {
	runtime.Start(HandleRequest)
}

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var result *telegram.Update
	err := json.Unmarshal([]byte(request.Body), result)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 400}, nil
	}

	response := SendMessageMethodResponse{
		MethodResponse: MethodResponse{Method: "sendMessage"},
		MessageConfig: telegram.MessageConfig{
			BaseChat: telegram.BaseChat{
				ChatID: result.Message.Chat.ID,
			},
			Text: "Hello World!",
		},
	}

	body, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	return events.APIGatewayProxyResponse{Body: string(body), StatusCode: 200}, nil
}
