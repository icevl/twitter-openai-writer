package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"

	openai "github.com/sashabaranov/go-openai"
)

type OpenAI struct {
	Client *openai.Client
}

func NewOpenAI(token string) *OpenAI {
	return &OpenAI{
		Client: openai.NewClient(token),
	}
}

func (o *OpenAI) GetAnswer(message string) (string, bool) {
	resp, err := o.Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: message,
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return "", false
	}

	resultText := resp.Choices[0].Message.Content
	response := sanitizeString(resultText)

	return response, true
}

func (o *OpenAI) GetImage(prompt string) (string, bool) {
	ctx := context.Background()
	uuid := uuid.New().String()
	fileName := fmt.Sprintf(".%s.png", uuid)

	req := openai.ImageRequest{
		Prompt:         prompt,
		Size:           openai.CreateImageSize1792x1024,
		ResponseFormat: openai.CreateImageResponseFormatB64JSON,
		Model:          openai.CreateImageModelDallE3,
		N:              1,
	}

	respBase64, err := o.Client.CreateImage(ctx, req)
	if err != nil {
		fmt.Printf("Image creation error: %v\n", err)
		return "", false
	}

	imgBytes, err := base64.StdEncoding.DecodeString(respBase64.Data[0].B64JSON)
	if err != nil {
		fmt.Printf("Base64 decode error: %v\n", err)
		return "", false
	}

	r := bytes.NewReader(imgBytes)
	imgData, err := png.Decode(r)
	if err != nil {
		fmt.Printf("PNG decode error: %v\n", err)
		return "", false
	}

	file, err := os.Create(fileName)
	if err != nil {
		fmt.Printf("File creation error: %v\n", err)
		return "", false
	}
	defer file.Close()

	if err := png.Encode(file, imgData); err != nil {
		fmt.Printf("PNG encode error: %v\n", err)
		return "", false
	}

	return fileName, true
}

func sanitizeString(text string) string {
	re := regexp.MustCompile(`^"(.*)"$`)
	trimedText := strings.TrimSpace(text)
	return re.ReplaceAllString(trimedText, "$1")
}
