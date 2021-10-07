package main

import (
	"context"
	"encoding/json"
	"fmt"

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

type AnswerInlineQueryMethodResponse struct {
	MethodResponse
	telegram.InlineConfig
}

func main() {
	runtime.Start(handleRequest)
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	result := &telegram.Update{}
	err := json.Unmarshal([]byte(request.Body), result)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: fmt.Sprintf("%v", err), StatusCode: 400}, nil
	}

	if result.Message != nil {
		return handleMessage(result)
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func handleMessage(update *telegram.Update) (events.APIGatewayProxyResponse, error) {
	response := SendMessageMethodResponse{
		MethodResponse: MethodResponse{Method: "sendMessage"},
		MessageConfig: telegram.MessageConfig{
			BaseChat: telegram.BaseChat{
				ChatID: update.Message.Chat.ID,
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
