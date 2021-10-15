package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	runtime "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	pollyT "github.com/aws/aws-sdk-go-v2/service/polly/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3T "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

const syncTaskLimit = 140

var bucket = os.Getenv("AUDIO_BUCKET")
var botName = os.Getenv("TELEGRAM_BOT_NAME")

const (
	MethodSendMessage = "sendMessage"
)

type MessageChat struct {
	ID int64 `json:"id"`
}

type MessageFrom struct {
	LastName  string `json:"last_name"`
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type Message struct {
	Date      int64       `json:"date"`
	Chat      MessageChat `json:"chat"`
	MessageID int64       `json:"message_id"`
	From      MessageFrom `json:"from"`
	Text      string      `json:"text"`
}

type InlineQuery struct {
	ID    string `json:"id"`
	From  User   `json:"from"`
	Query string `json:"query"`
}

type User struct {
	ID        int32  `json:"id"`
	FirstName string `json:"first_name"`
}

type Update struct {
	UpdateID    int64        `json:"update_id"`
	Message     *Message     `json:"message"`
	InlineQuery *InlineQuery `json:"inline_query"`
}

type SendMessageWebhookResponse struct {
	Method string `json:"method"`
	ChatID int64  `json:"chat_id"`
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
			ChatID: update.Message.Chat.ID,
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
	prefix := fmt.Sprintf("%d/voice", update.Message.Chat.ID)
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

	if len(text) > syncTaskLimit {
		_, err := svc.StartSpeechSynthesisTask(context.TODO(), input)
		if err != nil {
			return errorResponse(err)
		}

	} else {
		output, err := svc.SynthesizeSpeech(context.TODO(), toSyncTask(input))
		if err != nil {
			return errorResponse(err)
		}

		err = saveToStorage(cfg, output.AudioStream, input)
		if err != nil {
			return errorResponse(err)
		}
	}

	return nopResponse()

}

func trimText(t string) string {
	const limit = 1000

	trim := strings.TrimPrefix(t, fmt.Sprintf("%s ", botName))

	if len(trim) > limit {
		return trim[0:limit]
	}

	return trim
}

func jsonResponse(content interface{}) events.APIGatewayProxyResponse {
	body, err := json.Marshal(content)
	if err != nil {
		return errorResponse(err)
	}

	return events.APIGatewayProxyResponse{
		Body:       string(body),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: http.StatusOK,
	}
}

func errorResponse(err error) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: http.StatusOK}
}

func nopResponse() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}
}

func toSyncTask(i *polly.StartSpeechSynthesisTaskInput) *polly.SynthesizeSpeechInput {
	sync := polly.SynthesizeSpeechInput{
		OutputFormat: i.OutputFormat,
		Text:         i.Text,
		SampleRate:   i.SampleRate,
		Engine:       i.Engine,
		VoiceId:      i.VoiceId,
	}
	return &sync
}

func saveToStorage(cfg aws.Config, audio io.ReadCloser, input *polly.StartSpeechSynthesisTaskInput) error {
	svc := s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(svc)
	filename := fmt.Sprintf("%s.%s.pcm", *input.OutputS3KeyPrefix, uuid.New().String())

	_, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(filename),
		Body:        audio,
		ContentType: aws.String("audio/pcm"),
		ACL:         s3T.ObjectCannedACLPublicRead,
	})
	return err
}
