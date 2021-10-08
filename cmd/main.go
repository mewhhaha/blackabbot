package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

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

const (
	MethodSendVoice = "sendVoice"
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

type SendVoiceMethodResponse struct {
	Method string `json:"method"`
	ChatId int64  `json:"chat_id"`
	Voice  string `json:"voice"`
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

	if result.Message != nil {
		return handleMessage(cfg, result)
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func handleMessage(cfg aws.Config, update *Update) (events.APIGatewayProxyResponse, error) {
	text := "Hello World!"

	audio, err := textToSpeech(cfg, text)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}, nil
	}

	uri, err := saveToStorage(cfg, audio)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}, nil
	}

	response := SendVoiceMethodResponse{
		Method: MethodSendVoice,
		ChatId: update.Message.Chat.Id,
		Voice:  *uri,
	}

	body, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}, nil
	}

	return events.APIGatewayProxyResponse{
		Body:       string(body),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200,
	}, nil
}

func textToSpeech(cfg aws.Config, text string) (io.ReadCloser, error) {
	svc := polly.NewFromConfig(cfg)

	input := &polly.SynthesizeSpeechInput{
		OutputFormat: pollyT.OutputFormatOggVorbis,
		Text:         &text,

		Engine:  pollyT.EngineNeural,
		VoiceId: pollyT.VoiceIdKevin,
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
