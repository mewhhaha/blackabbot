package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
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
	"github.com/digital-dream-labs/opus-go/opus"
	"github.com/google/uuid"
)

var bucket = os.Getenv("AUDIO_BUCKET")
var botName = os.Getenv("TELEGRAM_BOT_NAME")

const (
	MethodSendVoice   = "sendVoice"
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

type SendVoiceMethodResponse struct {
	Method string `json:"method"`
	ChatId int64  `json:"chat_id"`
	Voice  string `json:"voice"`
}

type SendMessageMethodResponse struct {
	Method string `json:"method"`
	ChatId int64  `json:"chat_id"`
	Text   string `json:"text"`
}

func main() {
	runtime.Start(handleRequest)
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (resp events.APIGatewayProxyResponse, err error) {
	result := &Update{}
	err = json.Unmarshal([]byte(request.Body), result)
	if err != nil {
		return errorResponse(err, 400), nil
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-west-1"))
	if err != nil {
		return errorResponse(err, 500), nil
	}

	if err != nil {
		return errorResponse(err, 500), nil
	}

	if result.Message != nil && strings.HasPrefix(result.Message.Text, "@BlackAbbot ") {
		return handleMessage(cfg, result), nil
	}

	return nopResponse(), nil

}

func handleMessage(cfg aws.Config, update *Update) events.APIGatewayProxyResponse {
	text := trimText(update.Message.Text)

	errorResponse := func(err error) events.APIGatewayProxyResponse {
		return jsonResponse(SendMessageMethodResponse{
			Method: MethodSendMessage,
			ChatId: update.Message.Chat.Id,
			Text:   err.Error(),
		})
	}

	pcm, err := textToSpeech(cfg, text, pollyT.OutputFormatPcm)
	if err != nil {
		return errorResponse(err)
	}

	audio, err := convertToOpus(pcm)
	if err != nil {
		return errorResponse(err)
	}

	uri, err := saveToStorage(cfg, audio)
	if err != nil {
		return errorResponse(err)
	}

	voiceResponse := SendVoiceMethodResponse{
		Method: MethodSendVoice,
		ChatId: update.Message.Chat.Id,
		Voice:  *uri,
	}

	return jsonResponse(voiceResponse)
}

func textToSpeech(cfg aws.Config, text string, format pollyT.OutputFormat) ([]byte, error) {
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

	input := &polly.SynthesizeSpeechInput{
		OutputFormat: format,
		Text:         &text,
		SampleRate:   aws.String("16000"),
		Engine:       pollyT.EngineNeural,
		VoiceId:      voices[index],
	}

	output, err := svc.SynthesizeSpeech(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("decompress %v: %w", "POLLY FAILED", err)
	}

	if output.RequestCharacters == 0 {
		return []byte{}, nil
	}

	pcm, err := ioutil.ReadAll(output.AudioStream)
	if err != nil {
		return nil, err
	}

	return pcm, nil
}

func saveToStorage(cfg aws.Config, audio []byte) (*string, error) {
	svc := s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(svc)

	filename := uuid.New().String()

	output, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(filename),
		Body:        io.NopCloser(bytes.NewReader(audio)),
		ContentType: aws.String("audio/ogg"),
		ACL:         s3T.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return nil, fmt.Errorf("decompress %v: %w", "S3 FAILED", err)
	}

	return &output.Location, nil
}

func convertToOpus(pcm []byte) ([]byte, error) {
	stream := &opus.OggStream{
		SampleRate: 16000,
		Channels:   1,
		Bitrate:    192000,
		FrameSize:  2.5,
		Complexity: 10,
	}

	data, err := stream.EncodeBytes(pcm)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func trimText(t string) string {
	const limit = 140
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
		return errorResponse(err, 500)
	}

	return events.APIGatewayProxyResponse{
		Body:       string(body),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200,
	}
}

func errorResponse(err error, statusCode int) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 200}
}

func nopResponse() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{StatusCode: 200}
}
