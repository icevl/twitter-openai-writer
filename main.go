package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type SaveData struct {
	Stream *Stream
	Next   int64
}

const ConfigFile = "config.json"

func init() {
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	log.SetOutput(os.Stdout)
}

func main() {
	aiToken := os.Getenv("OPENAI_TOKEN")

	if aiToken == "" {
		log.Panic("No open ai token provided")
	}

	openAI := NewOpenAI(aiToken)
	streams := loadConfig()
	saveData := make(chan SaveData)

	for _, stream := range streams {
		go scheduler(openAI, stream, saveData)
	}

	go func() {
		for data := range saveData {
			saveStreamNextTime(data.Stream, data.Next)
		}
	}()

	<-make(chan bool)
}

func scheduler(openAI *OpenAI, stream Stream, saveNext chan<- SaveData) {
	log.Printf("Starting scheduler for: '%s'", stream.Title)

	for {
		unixTime := time.Now().Unix()

		if unixTime < stream.NextTime {
			time.Sleep(1 * time.Minute)
			continue
		}

		if stream.Auth.AccessToken == "" {
			time.Sleep(1 * time.Minute)
			continue
		}

		randomMinutes := rand.Intn(stream.MaxMins-stream.MinMins+1) + stream.MinMins
		randomDuration := time.Duration(randomMinutes) * time.Minute
		prompt := fmt.Sprintf("%s. Must be no more than 280 characters", stream.Prompt)

		gptText, ok := openAI.GetAnswer(prompt)
		if !ok {
			time.Sleep(10 * time.Second)
			continue
		}

		data := strings.Split(gptText, "|")
		text := gptText
		imageFile := ""

		if len(data) == 3 {
			emoji, title, body := data[2], sanitizeString(data[1]), sanitizeString(data[0])
			text = fmt.Sprintf("%s %s\n\n%s", title, emoji, body)
		}

		if stream.Image != "" {
			imageFile = getImage(*openAI, stream, text)
		}

		stream.Send(text, imageFile)
		_ = os.Remove(imageFile)

		stream.NextTime = time.Now().Add(randomDuration).Unix()

		log.Printf("Next message for '%s' will be sent in %d minutes", stream.Title, randomMinutes)

		saveData := SaveData{
			Stream: &stream,
			Next:   stream.NextTime,
		}

		saveNext <- saveData
	}
}

func loadConfig() []Stream {
	data := []Stream{}
	file, err := os.ReadFile(ConfigFile)

	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal([]byte(file), &data)
	if err != nil {
		log.Panic(err)
	}

	return data
}

func saveStreamNextTime(stream *Stream, next int64) {
	streams := loadConfig()

	for i, ch := range streams {
		if ch.Prompt == stream.Prompt {
			streams[i].NextTime = next
		}
	}

	file, _ := json.MarshalIndent(streams, "", " ")
	_ = os.WriteFile(ConfigFile, file, 0644)
}

func getImage(openAI OpenAI, stream Stream, prompt string) string {
	picturePrompt := stream.Image

	if stream.Image == "from_prompt_result" {
		picturePrompt = prompt
	}

	if picturePrompt == "" {
		return ""
	}

	file, ok := openAI.GetImage(picturePrompt)
	if !ok {
		return ""
	}

	return file
}
