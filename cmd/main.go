package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

var bucket = os.Getenv("AUDIO_BUCKET")
var botName = os.Getenv("BOT_NAME")

const (
	MethodSendAudio         = "sendAudio"
	MethodAnswerInlineQuery = "answerInlineQuery"
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

type InlineQueryResult struct {
}

type AnswerInlineQuery struct {
	InlineQueryId string              `json:"inline_query_id"`
	Results       []InlineQueryResult `json:"results"`
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

type SendAudioMethodResponse struct {
	Method    string `json:"method"`
	ChatId    int64  `json:"chat_id"`
	Audio     string `json:"audio"`
	Performer string `json:"performer"`
	Title     string `json:"title"`
	Caption   string `json:"caption"`
}

func main() {
	runtime.Start(handleRequest)
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	result := &Update{}
	err := json.Unmarshal([]byte(request.Body), result)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 400}, nil
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-west-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}, nil
	}

	if result.InlineQuery != nil {
		return handleInlineQuery(cfg, result), nil
	}

	if result.Message != nil && strings.HasPrefix(result.Message.Text, botName) {
		return handleMessage(cfg, result), nil
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil

}

func handleInlineQuery(cfg aws.Config, update *Update) events.APIGatewayProxyResponse {
	method := AnswerInlineQuery{
		InlineQueryId: update.InlineQuery.Id,
		Results:       []InlineQueryResult{},
	}

	return jsonResponse(method)
}

func handleMessage(cfg aws.Config, update *Update) events.APIGatewayProxyResponse {
	text := trimText(update.Message.Text)

	audio, err := textToSpeech(cfg, text)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}
	}

	uri, err := saveToStorage(cfg, audio)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}
	}

	name := update.Message.From.FirstName
	method := SendAudioMethodResponse{
		Method:    MethodSendAudio,
		Performer: name,
		Title:     fmt.Sprintf("%s said", name),
		Caption:   text,
		ChatId:    update.Message.Chat.Id,
		Audio:     *uri,
	}

	return jsonResponse(method)
}

func textToSpeech(cfg aws.Config, text string) (io.ReadCloser, error) {
	svc := polly.NewFromConfig(cfg)

	input := &polly.SynthesizeSpeechInput{
		OutputFormat: pollyT.OutputFormatMp3,
		Text:         &text,
		Engine:       pollyT.EngineNeural,
		VoiceId:      pollyT.VoiceIdKevin,
	}

	output, err := svc.SynthesizeSpeech(context.TODO(), input)
	if err != nil {

		return nil, fmt.Errorf("decompress %v: %w", "POLLY FAILED", err)
	}

	return output.AudioStream, nil
}

func saveToStorage(cfg aws.Config, audio io.ReadCloser) (*string, error) {
	svc := s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(svc)

	filename := uuid.New().String()

	output, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(filename),
		Body:        audio,
		ContentType: aws.String("audio/mpeg"),
		ACL:         s3T.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return nil, fmt.Errorf("decompress %v: %w", "S3 FAILED", err)
	}

	return &output.Location, nil
}

func trimText(t string) string {
	t0 := strings.TrimPrefix(t, "@BlackAbbot")
	t1 := strings.TrimPrefix(t0, " ")

	var text string
	if len(t1) > 140 {
		text = t1[0:140]
	} else {
		text = t1
	}

	return text
}

func jsonResponse(content interface{}) events.APIGatewayProxyResponse {
	body, err := json.Marshal(content)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}
	}

	return events.APIGatewayProxyResponse{
		Body:       string(body),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200,
	}
}
