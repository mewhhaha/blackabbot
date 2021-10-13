package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	runtime "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	pollyT "github.com/aws/aws-sdk-go-v2/service/polly/types"
)

var bucket = os.Getenv("AUDIO_BUCKET")
var botName = os.Getenv("TELEGRAM_BOT_NAME")

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

type InlineQuery struct {
	Id    string `json:"id"`
	From  User   `json:"from"`
	Query string `json:"query"`
}

type User struct {
	Id        int32  `json:"id"`
	FirstName string `json:"first_name"`
}

type Update struct {
	UpdateId    int64        `json:"update_id"`
	Message     *Message     `json:"message"`
	InlineQuery *InlineQuery `json:"inline_query"`
}

type SendMessageWebhookResponse struct {
	Method string `json:"method"`
	ChatId int64  `json:"chat_id"`
	Text   string `json:"text"`
}

func main() {
	runtime.Start(handleRequest)
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	result := &Update{}
	err := json.Unmarshal([]byte(request.Body), result)
	if err != nil {
		return errorResponse(err), nil
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-west-1"))
	if err != nil {
		return errorResponse(err), nil
	}

	if err != nil {
		return errorResponse(err), nil
	}

	if result.Message != nil && strings.HasPrefix(result.Message.Text, "@BlackAbbot ") {
		return handleMessage(cfg, result), nil
	}

	return nopResponse(), nil

}

func handleMessage(cfg aws.Config, update *Update) events.APIGatewayProxyResponse {
	errorResponse := func(err error) events.APIGatewayProxyResponse {
		return jsonResponse(SendMessageWebhookResponse{
			Method: MethodSendMessage,
			ChatId: update.Message.Chat.Id,
			Text:   err.Error(),
		})
	}

	svc := polly.NewFromConfig(cfg)

	voices := []pollyT.VoiceId{
		pollyT.VoiceIdSalli,
		pollyT.VoiceIdJoanna,
		pollyT.VoiceIdIvy,
		pollyT.VoiceIdKendra,
		pollyT.VoiceIdKimberly,
		pollyT.VoiceIdKevin,
		pollyT.VoiceIdMatthew,
		pollyT.VoiceIdJustin,
		pollyT.VoiceIdJoey,
	}
	index := rand.Intn(len(voices))
	prefix := fmt.Sprintf("%d/", update.Message.Chat.Id)
	text := trimText(update.Message.Text)

	input := &polly.StartSpeechSynthesisTaskInput{
		OutputFormat:       pollyT.OutputFormatPcm,
		Text:               &text,
		SampleRate:         aws.String("16000"),
		Engine:             pollyT.EngineNeural,
		VoiceId:            voices[index],
		OutputS3BucketName: &bucket,
		OutputS3KeyPrefix:  aws.String(prefix),
	}

	_, err := svc.StartSpeechSynthesisTask(context.TODO(), input)
	if err != nil {
		return errorResponse(err)
	}

	return nopResponse()
}

func trimText(t string) string {
	const limit = 1000
	trim := strings.TrimPrefix(t, fmt.Sprintf("%s ", botName))

	if len(trim) > limit {
		return trim[0:limit]
	} else {
		return trim
	}
}

func jsonResponse(content interface{}) events.APIGatewayProxyResponse {
	body, err := json.Marshal(content)
	if err != nil {
		return errorResponse(err)
	}

	return events.APIGatewayProxyResponse{
		Body:       string(body),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200,
	}
}

func errorResponse(err error) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 200}
}

func nopResponse() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{StatusCode: 200}
}
