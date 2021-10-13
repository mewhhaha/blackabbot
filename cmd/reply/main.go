package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	runtime "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3T "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/digital-dream-labs/opus-go/opus"
)

var errSilence = errors.New("exclusively empty bytes in pcm")
var botToken = os.Getenv("TELEGRAM_BOT_TOKEN")

type SendVoiceMethodResponse struct {
	ChatID int64  `json:"chat_id"`
	Voice  string `json:"voice"`
}

type SendMessageMethodResponse struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

func main() {
	runtime.Start(handleRequest)
}

func handleRequest(ctx context.Context, request events.S3Event) error {
	record := request.Records[0]
	bucket := record.S3.Bucket.Name
	key := record.S3.Object.Key

	chatID, err := strconv.ParseInt(strings.Split(key, "/")[0], 10, 64)
	if err != nil {
		return err
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-west-1"))
	if err != nil {
		return sendErrorResponse(chatID, err)
	}

	pcm, err := getFromStorage(cfg, bucket, key)
	if err != nil {
		return sendErrorResponse(chatID, err)
	}

	audio, err := convertToOpus(pcm)
	if err != nil {
		return sendErrorResponse(chatID, err)
	}

	uri, err := saveToStorage(cfg, bucket, key, audio)
	if err != nil {
		return sendErrorResponse(chatID, err)
	}

	return sendVoiceResponse(chatID, *uri)
}

func getFromStorage(cfg aws.Config, bucket string, key string) ([]byte, error) {
	svc := s3.NewFromConfig(cfg)

	output, err := svc.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(output.Body)
}

func saveToStorage(cfg aws.Config, bucket string, key string, audio []byte) (*string, error) {
	svc := s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(svc)

	output, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(strings.Replace(key, ".pcm", ".ogg", 1)),
		Body:        io.NopCloser(bytes.NewReader(audio)),
		ContentType: aws.String("audio/ogg"),
		ACL:         s3T.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return nil, err
	}

	return &output.Location, nil
}

func convertToOpus(pcm []byte) ([]byte, error) {
	if isSilence(pcm) {
		return nil, errSilence
	}

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

func isSilence(audio []byte) bool {
	for _, b := range audio {
		if b != 0 {
			return false
		}
	}

	return true
}

func sendVoiceResponse(chatID int64, uri string) error {
	voiceResponse := SendVoiceMethodResponse{
		ChatID: chatID,
		Voice:  uri,
	}

	body, err := json.Marshal(voiceResponse)
	if err != nil {
		return err
	}

	res, err := http.Post(
		fmt.Sprintf("https://api.telegram.org/bot%s/sendVoice", botToken),
		"application/json",
		bytes.NewReader(body))
	if err != nil {
		return err
	}

	res.Body.Close()

	return nil
}

func sendErrorResponse(chatID int64, err error) error {
	messageResponse := SendMessageMethodResponse{
		ChatID: chatID,
		Text:   err.Error(),
	}

	body, err := json.Marshal(messageResponse)
	if err != nil {
		return err
	}

	res, err := http.Post(
		fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken),
		"application/json",
		bytes.NewReader(body))
	if err != nil {
		return err
	}

	res.Body.Close()

	return nil
}
