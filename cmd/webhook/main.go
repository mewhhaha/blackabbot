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
	"github.com/google/uuid"
	"github.com/oov/audio/converter"
	"gopkg.in/hraban/opus.v2"
)

var bucket = os.Getenv("AUDIO_BUCKET")
var botName = os.Getenv("TELEGRAM_BOT_NAME")

const (
	MethodSendVoice         = "sendVoice"
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
	Method        string
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

type SendVoiceMethodResponse struct {
	Method string `json:"method"`
	ChatId int64  `json:"chat_id"`
	Voice  string `json:"voice"`
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

	if result.InlineQuery != nil {
		return handleInlineQuery(cfg, result), nil
	}

	if result.Message != nil && strings.HasPrefix(result.Message.Text, "@BlackAbbot") {
		return handleMessage(cfg, result), nil
	}

	return nopResponse(), nil

}

func handleInlineQuery(cfg aws.Config, update *Update) events.APIGatewayProxyResponse {
	method := AnswerInlineQuery{
		Method:        MethodAnswerInlineQuery,
		InlineQueryId: update.InlineQuery.Id,
		Results:       []InlineQueryResult{},
	}

	return jsonResponse(method)
}

func handleMessage(cfg aws.Config, update *Update) events.APIGatewayProxyResponse {
	text := trimText(update.Message.Text)

	pcm, err := textToSpeech(cfg, text, pollyT.OutputFormatPcm)
	if err != nil {
		return errorResponse(err, 500)
	}

	audio, err := convertToOpus(pcm)
	if err != nil {
		return errorResponse(err, 500)
	}

	uri, err := saveToStorage(cfg, audio)
	if err != nil {
		return errorResponse(err, 500)
	}

	method := SendVoiceMethodResponse{
		Method: MethodSendVoice,
		ChatId: update.Message.Chat.Id,
		Voice:  *uri,
	}

	return jsonResponse(method)
}

func textToSpeech(cfg aws.Config, text string, format pollyT.OutputFormat) (io.ReadCloser, error) {
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
		ContentType: aws.String("audio/ogg"),
		ACL:         s3T.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return nil, fmt.Errorf("decompress %v: %w", "S3 FAILED", err)
	}

	return &output.Location, nil
}

func convertToOpus(audio io.ReadCloser) (io.ReadCloser, error) {
	bs, err := ioutil.ReadAll(audio)
	if err != nil {
		return nil, err
	}

	const sampleRate = 16000
	const channels = 1

	enc, err := opus.NewEncoder(sampleRate, channels, opus.AppVoIP)
	if err != nil {
		return nil, err
	}
	// stream := &opus.OggStream{
	// 	SampleRate: 16000,
	// 	Channels:   1,
	// 	Bitrate:    24000,
	// 	FrameSize:  20,
	// 	Complexity: 1,
	// }

	pcm := make([]int16, len(bs)/2)
	for i := 0; i < len(bs)/2; i++ {
		pcm = append(pcm, converter.ByteToInt16(bs[i*2], bs[i*2+1]))
	}

	frameSize := len(pcm) // must be interleaved if stereo
	frameSizeMs := float32(frameSize) / channels * 1000 / sampleRate
	switch frameSizeMs {
	case 2.5, 5, 10, 20, 40, 60:
		// Good.
	default:
		return nil, fmt.Errorf("Illegal frame size: %d bytes (%f ms)", frameSize, frameSizeMs)
	}

	data := make([]byte, 1000)

	n, err := enc.Encode(pcm, data)
	if err != nil {
		return nil, err
	}

	data = data[:n]

	return io.NopCloser(bytes.NewReader(data)), nil
}

func trimText(t string) string {
	trim := strings.TrimPrefix(t, fmt.Sprintf("%s ", botName))

	if len(trim) > 140 {
		return trim[0:140]
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
	return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: statusCode}
}

func nopResponse() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{StatusCode: 200}
}
