package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	setupMainEngine().Run("0.0.0.0:" + os.Getenv("PORT"))
}

// setupMainEngine return *gin.Engine, split to use by testing
func setupMainEngine() *gin.Engine {
	r := gin.Default()

	// healthz
	healthz := &healthz{}
	r.GET("/healthz", healthz.getHealthz)
	news := &news{}
	r.GET("/", news.getNews)
	r.POST("/", news.getNews)

	return r
}

type healthz struct {
	*gin.HandlerFunc
}

func (h healthz) getHealthz(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}

type news struct {
	*gin.HandlerFunc
}

var sources = []string{
	//"google-news",
	"the-wall-street-journal",
	"the-washington-post",
	"the-new-york-times",
	"cnn",
	"bbc-news",
}

type NewsResponse struct {
	Status   string
	Source   string
	Articles []Articles
}

type Articles struct {
	Author      string
	Title       string
	Description string
	URL         string `json:"url"`
	URLToImage  string `json:"urlToImage"`
	PublishedAt string
}

type SlackResponse struct {
	ResponseType string       `json:"response_type"`
	Text         string       `json:"text"`
	Attachments  []Attachment `json:"attachments"`
}

type Attachment struct {
	AuthorName string `json:"author_name"`
	Title      string `json:"title"`
	TitleLink  string `json:"title_link"`
	Text       string `json:"text"`
	ImageURL   string `json:"image_url"`
}

func (n news) getNews(c *gin.Context) {
	source := sources[rand.Intn(len(sources))]
	apiKey := os.Getenv("NEWS_API_KEY")
	url := fmt.Sprintf("https://newsapi.org/v1/articles?source=%s&sortBy=top&apiKey=%s", source, apiKey)
	fmt.Println(url)

	resp, err := http.Get(url)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
	defer resp.Body.Close()

	// Fill the record with the data from the JSON
	var jsonResp NewsResponse
	// Use json.Decode for reading streams of JSON data
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	if jsonResp.Status == "error" || len(jsonResp.Articles) < 5 {
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	article := jsonResp.Articles[rand.Intn(5)]
	slackResp := SlackResponse{ResponseType: "in_channel", Text: article.Title, Attachments: []Attachment{Attachment{AuthorName: article.Author, Title: article.Title, TitleLink: article.URL, Text: article.Description, ImageURL: article.URLToImage}}}
	//slackResp := SlackResponse{Text: article.Title, Attachements: []Attachment{Attachment{AuthorName: article.Author, Title: article.Title, TitleLink: article.Url, Text: article.Description}}}
	c.JSON(http.StatusOK, slackResp)
}
