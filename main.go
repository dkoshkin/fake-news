package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

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
	r.GET("/", news.getArticleUrl)
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
	//"the-wall-street-journal",
	"the-washington-post",
	"the-new-york-times",
	"cnn",
	"bbc-news",
}

type NewsResponse struct {
	Status   string
	Source   string
	Articles []Article
}

type Article struct {
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
	article, err := fetchArticle()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	slackResp := SlackResponse{ResponseType: "in_channel", Text: article.Title, Attachments: []Attachment{Attachment{AuthorName: article.Author, Title: article.Title, TitleLink: article.URL, Text: article.Description, ImageURL: article.URLToImage}}}
	//slackResp := SlackResponse{Text: article.Title, Attachements: []Attachment{Attachment{AuthorName: article.Author, Title: article.Title, TitleLink: article.Url, Text: article.Description}}}
	c.JSON(http.StatusOK, slackResp)
}

func (n news) getArticleUrl(c *gin.Context) {
	article, err := fetchArticle()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	c.Redirect(http.StatusFound, article.URL)
}

func fetchArticle() (*Article, error) {
	rand.Seed(time.Now().Unix())
	source := sources[rand.Intn(len(sources))]
	apiKey := os.Getenv("NEWS_API_KEY")
	url := fmt.Sprintf("https://newsapi.org/v1/articles?source=%s&sortBy=top&apiKey=%s", source, apiKey)
	fmt.Println(url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Fill the record with the data from the JSON
	var newsResp NewsResponse
	// Use json.Decode for reading streams of JSON data
	if err := json.NewDecoder(resp.Body).Decode(&newsResp); err != nil {
		return nil, err
	}

	if newsResp.Status == "error" {
		return nil, fmt.Errorf("Error response from news API")
	}

	numberOfArticles := 5
	if len(newsResp.Articles) < 5 {
		numberOfArticles = len(newsResp.Articles)
	}

	rand.Seed(time.Now().Unix())
	article := newsResp.Articles[rand.Intn(numberOfArticles)]

	return &article, nil
}
