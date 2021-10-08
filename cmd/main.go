package main

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/aws/aws-lambda-go/events"
	runtime "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/polly"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"
)

var AudioBucket = os.Getenv("AUDIO_BUCKET")

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

	sess, err := session.NewSession()
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}, nil
	}

	if result.Message != nil {
		return handleMessage(sess, result)
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func handleMessage(sess *session.Session, update *Update) (events.APIGatewayProxyResponse, error) {
	text := "Hello World!"

	audio, err := textToSpeech(sess, text)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}, nil
	}

	uri, err := saveToStorage(sess, audio)
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

func textToSpeech(sess *session.Session, text string) (io.ReadCloser, error) {
	svc := polly.New(sess)

	input := &polly.SynthesizeSpeechInput{
		OutputFormat: aws.String("ogg_vorbis"),
		Text:         &text,
		VoiceId:      aws.String("Kevin"),
	}

	output, err := svc.SynthesizeSpeech(input)
	if err != nil {

		return nil, err
	}

	return output.AudioStream, nil
}

func saveToStorage(sess *session.Session, audio io.ReadCloser) (*string, error) {
	svc := s3manager.NewUploader(sess)

	filename := uuid.New()

	output, err := svc.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(AudioBucket),
		Key:         aws.String(filename.String()),
		Body:        audio,
		ContentType: aws.String("audio/mpeg"),
		ACL:         aws.String("public-read"),
	})

	if err != nil {
		return nil, err
	}

	return &output.Location, nil
}
