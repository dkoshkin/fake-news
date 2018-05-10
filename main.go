package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

var defaultSources = []string{
	"the-washington-post",
	"the-new-york-times",
	"cnn",
	"bbc-news",
}

func main() {
	setupMainEngine().Run("0.0.0.0:" + os.Getenv("PORT"))
}

// setupMainEngine return *gin.Engine, split to use by testing
func setupMainEngine() *gin.Engine {
	r := gin.Default()

	// healthz
	healthz := &healthz{}
	r.GET("/healthz", healthz.getHealthz)

	sources := defaultSources
	sourcesFromEnv := strings.Split(os.Getenv("NEWS_SOURCES"), ",")
	if len(sourcesFromEnv) != 0 {
		sources = sourcesFromEnv
	}
	news := &news{
		apiKey:  os.Getenv("NEWS_API_KEY"),
		url:     "https://newsapi.org/v2",
		sources: sources,
	}
	r.GET("/", news.articleURL)
	r.GET("/slack", news.articleForSlack)

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
	apiKey  string
	url     string
	sources []string
}

type NewsResponse struct {
	Status       string
	TotalResults int
	Articles     []Article
}

type Article struct {
	Author      string
	Title       string
	Description string
	URL         string `json:"url"`
	URLToImage  string `json:"urlToImage"`
	PublishedAt string
}

type SlackRequest struct {
	Text string `form:"text"`
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

func (n news) articleURL(c *gin.Context) {
	var req SlackRequest
	if err := c.ShouldBindWith(&req, binding.Form); err != nil {
		fmt.Printf("Error parsing request data, continuing: %v", err)
	}
	article, err := fetchArticle(req.Text, n.sources, n.url, n.apiKey)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
	c.Redirect(http.StatusFound, article.URL)
}

func (n news) articleForSlack(c *gin.Context) {
	var req SlackRequest
	if err := c.ShouldBindWith(&req, binding.Form); err != nil {
		fmt.Printf("Error parsing request data, continuing: %v", err)
	}
	article, err := fetchArticle(req.Text, n.sources, n.url, n.apiKey)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	slackResp := SlackResponse{ResponseType: "in_channel", Text: article.Title, Attachments: []Attachment{Attachment{AuthorName: article.Author, Title: article.Title, TitleLink: article.URL, Text: article.Description, ImageURL: article.URLToImage}}}
	c.JSON(http.StatusOK, slackResp)
}

func fetchArticle(query string, sources []string, baseURL string, apiKey string) (*Article, error) {
	fullURL := fmt.Sprintf("%s/top-headlines?sources=%s&q=%s&apiKey=%s", baseURL, strings.Join(sources, ","), url.QueryEscape(query), apiKey)
	fmt.Println(fullURL)

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Fill the record with the data from the JSON
	var news NewsResponse
	// Use json.Decode for reading streams of JSON data
	if err := json.NewDecoder(resp.Body).Decode(&news); err != nil {
		return nil, err
	}

	if news.Status == "error" {
		return nil, fmt.Errorf("Error response from news API")
	}
	if news.TotalResults == 0 {
		return nil, fmt.Errorf("Did not get any articles from news API")
	}

	// Random article from top 5
	// If less than 5 articles fetched,return random from all articles
	topArticles := 5
	if news.TotalResults < topArticles {
		topArticles = news.TotalResults
	}

	rand.Seed(time.Now().Unix())
	article := news.Articles[rand.Intn(topArticles)]

	return &article, nil
}
