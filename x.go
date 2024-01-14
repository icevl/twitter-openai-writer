package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/dghubble/oauth1"
	twitter "github.com/g8rswimmer/go-twitter/v2"
)

type authorize struct{}

type MediaUpload struct {
	MediaId int `json:"media_id"`
}

type Stream struct {
	Title    string    `json:"title"`
	Prompt   string    `json:"prompt"`
	Image    string    `json:"image"`
	MinMins  int       `json:"min_mins"`
	MaxMins  int       `json:"max_mins"`
	NextTime int64     `json:"next_time"`
	Tags     *[]string `json:"tags"`
	Auth     Auth      `json:"auth"`
}

type Auth struct {
	ApiKey            string `json:"api_key"`
	ApiKeySecret      string `json:"api_key_secret"`
	AccessToken       string `json:"access_token"`
	AccessTokenSecret string `json:"access_token_secret"`
}

func (a authorize) Add(req *http.Request) {}

func (s Stream) Send(text string, image string) {
	message := fmt.Sprintf("%s%s", text, s.getHashTags())
	oauth1Config := oauth1.NewConfig(s.Auth.ApiKey, s.Auth.ApiKeySecret)
	twitterHttpClient := oauth1Config.Client(oauth1.NoContext, &oauth1.Token{
		Token:       s.Auth.AccessToken,
		TokenSecret: s.Auth.AccessTokenSecret,
	})

	var mediaId string
	var mediaErr error

	if image != "" {
		mediaId, mediaErr = uploadMedia(twitterHttpClient, image)
		if mediaErr != nil {
			fmt.Printf("Error: %s\n", mediaErr)
			return
		}
	}

	client := &twitter.Client{
		Authorizer: authorize{},
		Client:     twitterHttpClient,
		Host:       "https://api.twitter.com",
	}

	req := twitter.CreateTweetRequest{
		Text: message,
	}

	if image != "" {
		req.Media = &twitter.CreateTweetMedia{IDs: []string{mediaId}}
	}

	_, err := client.CreateTweet(context.Background(), req)
	if err != nil {
		log.Panicf("create tweet error: %v", err)
	}
}

func (s Stream) getHashTags() string {
	if s.Tags == nil {
		return ""
	}

	var tags []string

	for _, tag := range *s.Tags {
		tags = append(tags, fmt.Sprintf("#%s", tag))
	}

	return fmt.Sprintf("\n\n%s", strings.Join(tags, " "))
}

func uploadMedia(httpClient *http.Client, file string) (string, error) {
	b := &bytes.Buffer{}
	form := multipart.NewWriter(b)

	fw, err := form.CreateFormFile("media", file)
	if err != nil {
		return "", err
	}

	opened, err := os.Open(file)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(fw, opened)
	if err != nil {
		return "", err
	}

	form.Close()

	resp, err := httpClient.Post("https://upload.twitter.com/1.1/media/upload.json?media_category=tweet_image", form.FormDataContentType(), bytes.NewReader(b.Bytes()))
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	media := &MediaUpload{}
	_ = json.NewDecoder(resp.Body).Decode(media)

	mediaId := strconv.Itoa(media.MediaId)

	if mediaId == "0" {
		return "", errors.New("file does not uploaded")
	}

	return mediaId, nil
}
