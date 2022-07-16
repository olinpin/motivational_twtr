package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/gocolly/colly"
	"github.com/joho/godotenv"
)

var url = "https://www.brainyquote.com/quote_of_the_day"

type Quote struct {
	Text     string
	Author   string
	Hashtags string
}

type Credentials struct {
	api_key             string
	api_secret          string
	access_token        string
	access_token_secret string
}

func main() {
	if time.Now().Hour()%4 == 2 {
		fmt.Print("running")
		run()
	} else {
		fmt.Print("Not running rn")
	}
}

func run() {
	var quotes []Quote

	jsonFile, err := os.Open("quotes.json")
	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &quotes)
	if err != nil || len(quotes) == 0 {
		fmt.Println("No json file or length of quotes is 0")
		quotes = scrape(url)
	}
	// assign quote to a variable
	length := len(quotes)
	quote := quotes[0]
	// delete quote from quotes
	quotes[0] = quotes[length-1]
	quotes = quotes[:length-1]
	file, _ := json.MarshalIndent(quotes, "", " ")
	_ = ioutil.WriteFile("quotes.json", file, 0644)
	tweet(quote)
}

func scrape(url string) []Quote {
	var quotes []Quote
	c := colly.NewCollector()

	c.OnRequest(func(request *colly.Request) {
		fmt.Println("Visiting", request.URL)
	})
	// /html/body/main/div[1]/div[3]/div/div/a[1]/div
	hashtags := chooseHashtags(3)
	c.OnHTML("body main div:nth-of-type(1).qotd-wrapper-cntr", func(e *colly.HTMLElement) {
		text := e.ChildText("div div.grid-item a:nth-of-type(1) div")
		author := e.ChildText("div div.grid-item a:nth-of-type(2)")
		hashtag_str := hashtags[0] + " " + hashtags[1] + " " + hashtags[2]
		quotes = append(quotes, Quote{Text: text, Author: author, Hashtags: hashtag_str})
	})

	c.Visit(url)

	return quotes
}

func tweet(quote Quote) {
	if os.Getenv("DYNO") == "" {
		envErr := godotenv.Load(".env")
		if envErr != nil {
			fmt.Println("Could not load .env file")
			os.Exit(1)
		}
	}
	config := Credentials{
		api_key:             os.Getenv("API_KEY"),
		api_secret:          os.Getenv("API_SECRET"),
		access_token:        os.Getenv("ACCESS_TOKEN"),
		access_token_secret: os.Getenv("ACCESS_TOKEN_SECRET"),
	}
	client, err := Twitter(&config)
	if err != nil {
		fmt.Println(err)
	}
	// get last 10 tweets
	old_tweets, _, _ := client.Timelines.UserTimeline(&twitter.UserTimelineParams{
		ScreenName: "QuoteMotiv1",
		Count:      10,
	})
	text := quote.Text + "\n\n - " + quote.Author
	// check if tweet already exists in the last 10 tweets
	for _, tweet := range old_tweets {
		Time, _ := time.Parse("Mon Jan 2 15:04:05 +0000 2006", old_tweets[0].CreatedAt)
		if Time.Hour() == time.Now().Hour()+4 {
			if strings.Contains(tweet.Text, text) {
				fmt.Println("Already tweeted")
				run()
				return
			}
		} else {
			return
		}
	}
	hashs := "\n" + quote.Hashtags
	text = text + hashs
	_, _, err = client.Statuses.Update(text, nil)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(text)
}

func Twitter(creds *Credentials) (*twitter.Client, error) {
	config := oauth1.NewConfig(creds.api_key, creds.api_secret)
	token := oauth1.NewToken(os.Getenv("ACCESS_TOKEN"), os.Getenv("ACCESS_TOKEN_SECRET"))

	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)

	verifyParams := &twitter.AccountVerifyParams{
		SkipStatus:   twitter.Bool(true),
		IncludeEmail: twitter.Bool(true),
	}
	_, _, err := client.Accounts.VerifyCredentials(verifyParams)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func chooseHashtags(length int) []string {
	var hashtags = []string{
		"#MondayMotivation",
		"#Motivation",
		"#inspiration",
		"#quote",
		"#life",
		"#quoteoftheday",
		"#quotes",
		"#quotesoftheday",
		"#quotesdaily",
		"#quotestoliveby",
	}
	hastag_result := []string{}
	iterators := []int{}
	for i := 0; i < length; i++ {
		// first_hashtag = hashtags[rand.Intn(len(hashtags))]
		iterator1 := rand.Intn(len(hashtags) - 1)
		for _, b := range iterators {
			if iterator1 == b {
				iterator1 = rand.Intn(len(hashtags) - 1)
			}
		}
		iterators = append(iterators, iterator1)
		hastag_result = append(hastag_result, hashtags[iterator1])
	}
	return hastag_result
}
