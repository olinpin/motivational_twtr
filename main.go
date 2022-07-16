package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/gocolly/colly"
	"github.com/joho/godotenv"
)

var url = "https://www.brainyquote.com/quote_of_the_day"

type Quote struct {
	Text   string
	Author string
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
	if err != nil {
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
	c.OnHTML("body main div:nth-of-type(1).qotd-wrapper-cntr", func(e *colly.HTMLElement) {
		text := e.ChildText("div div.grid-item a:nth-of-type(1) div")
		author := e.ChildText("div div.grid-item a:nth-of-type(2)")
		quotes = append(quotes, Quote{Text: text, Author: author})
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
	text := quote.Text + "\n\n - " + quote.Author
	_, _, err = client.Statuses.Update(text, nil)
	if err != nil {
		fmt.Println(err)
		if string(err.Error()) == "twitter: 187 Status is a duplicate." {
			run()
		}
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
	user, _, err := client.Accounts.VerifyCredentials(verifyParams)
	if err != nil {
		return nil, err
	}
	fmt.Println("user:", user)
	return client, nil
}
