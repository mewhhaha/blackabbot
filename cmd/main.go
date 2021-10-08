package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	runtime "github.com/aws/aws-lambda-go/lambda"
)

const (
	MethodSendMessage = "sendMessage"
)

type MessageChat struct {
	Id int64 `json:"id"`
}

type MessageFrom struct {
	LastName  string `json:"last_name"`
	Id        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type Message struct {
	Date      int64       `json:"date"`
	Chat      MessageChat `json:"chat"`
	MessageId int64       `json:"message_id"`
	From      MessageFrom `json:"from"`
	Text      string      `json:"text"`
}

type Update struct {
	UpdateId int64    `json:"update_id"`
	Message  *Message `json:"message"`
}

type SendMessageMethodResponse struct {
	Method                string       `json:"method"`
	ChatId                int64        `json:"chat_id"`
	ChannelUsername       *string      `json:"channel_username,omitempty"`
	ReplyToMessageID      *int         `json:"reply_to_message_id,omitempty"`
	ReplyMarkup           *interface{} `json:"reply_markup,omitempty"`
	DisableNotification   *bool        `json:"disable_notification,omitempty"`
	Text                  *string      `json:"text,omitempty"`
	ParseMode             *string      `json:"parse_mode,omitempty"`
	DisableWebPagePreview *bool        `json:"disable_web_page_preview,omitempty"`
}

func main() {
	runtime.Start(handleRequest)
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	result := &Update{}
	err := json.Unmarshal([]byte(request.Body), result)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: fmt.Sprintf("%v", err), StatusCode: 400}, nil
	}

	if result.Message != nil {
		return handleMessage(result)
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func handleMessage(update *Update) (events.APIGatewayProxyResponse, error) {
	text := "Hello World!"

	response := SendMessageMethodResponse{
		Method: MethodSendMessage,
		ChatId: update.Message.Chat.Id,
		Text:   &text,
	}

	body, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	return events.APIGatewayProxyResponse{
		Body:       string(body),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200,
	}, nil
}
