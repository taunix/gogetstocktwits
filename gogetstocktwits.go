package gogetstocktwits

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// Stock defines a stock result from db
type Stock struct {
	CompanyName string `bson:"companyname" json:"companyname"`
	Symbol      string `bson:"symbol" json:"symbol"`
}

// List defines a list result from the db
type List struct {
	Name        string  `bson:"name" json:"name"`
	Description string  `bson:"description" json:"description"`
	User        string  `bson:"user" json:"user"`
	Stocks      []Stock `bson:"stocks" json:"stocks"`
}

// structs for stocktwits return
type StocktwitUser struct {
	ID             float64  `bson:"id" json:"id"`
	Username       string   `bson:"username" json:"username"`
	Name           string   `bson:"name" json:"name"`
	AvatarURL      string   `bson:"avatar_url" json:"avatar_url"`
	AvatarURLSsl   string   `bson:"avatar_url_ssl" json:"avatar_url_ssl"`
	Identity       string   `bson:"identity" json:"identity"`
	Classification []string `bson:"classification" json:"classification"`
}

type StocktwitSource struct {
	ID    float64 `bson:"id" json:"id"`
	Title string  `bson:"title" json:"title"`
	URL   string  `bson:"url" json:"url"`
}

type Symbol struct {
	ID             float64  `bson:"id" json:"id"`
	Symbol         string   `bson:"symbol" json:"symbol"`
	Title          string   `bson:"title" json:"title"`
	Aliases        []string `bson:"aliases" json:"aliases"`
	IsFollowing    bool     `bson:"is_following" json:"is_following"`
	WatchlistCount float64  ` bson:"watchlist_count" json:"watchlist_count"`
}

type Response struct {
	Status int `bson:"status" json:"status"`
}

type StocktwitMessage struct {
	ID        float64         `bson:"id" json:"id"`
	Body      string          `bson:"body" json:"body"`
	CreatedAt time.Time       `bson:"created_at" json:"created_at"`
	User      StocktwitUser   `bson:"user" json:"user"`
	Source    StocktwitSource `bson:"source" json:"source"`
	Symbols   []Symbol        `bson:"symbols" json:"symbols"`
}

type Cursor struct {
	More  bool    `bson:"more" json:"more"`
	Since float64 `bson:"since" json:"since"`
	Max   float64 `bson:"max" json:"max"`
}

type StocktwitResponse struct {
	Response Response           `bson:"response" json:"response"`
	Symbol   Symbol             `bson:"symbol" json:"symbol"`
	Cursor   Cursor             `bson:"cursor" json:"cursor"`
	Messages []StocktwitMessage `bson:"messages" json:"messages"`
}

type Temperature struct {
	last10Minutes int `bson:"last10Minutes" json:"last10Minutes"`
	last1Hour     int `bson:"last1Hour" json:"last1Hour"`
	last3Hours    int `bson:"last3Hours" json:"last3Hours"`
}

type StocktwitProfile struct {
	Response    Response    `bson:"response" json:"response"`
	Symbol      Symbol      `bson:"symbol" json:"symbol"`
	Cursor      Cursor      `bson:"cursor" json:"cursor"`
	Temperature Temperature `bson:"temperature" json:"temperature"`
}

type StocktwitMessages struct {
	Symbol    string          `bson:"symbol" json:"symbol"`
	Body      string          `bson:"body" json:"body"`
	CreatedAt time.Time       `bson:"created_at" json:"created_at"`
	User      StocktwitUser   `bson:"user" json:"user"`
	Source    StocktwitSource `bson:"source" json:"source"`
	Symbols   []Symbol        `bson:"symbols" json:"symbols"`
}

// StocktwitsCallAPI handles calling the stocktwits API to return the response for
// an individual stock.  The response contains extra metadata which is parsed for the
// return.  It returns
func StocktwitsCallAPI(symbol string) (StocktwitsMessages, StocktwitProfile) {
	// call api to get all stocks, then loop over each stock symbol and call quote url to get
	// current and previous close prices.  Compare for % change and print qualifying stocks
	stocktwitURL := "https://api.stocktwits.com/api/2/streams/symbol/" + symbol + ".json"

	response := StocktwitResponse{}
	stocktwitsAPICall(stocktwitURL, &response)

	// Process Messages first:
	// loop over new messages, see if already in database and add if not
	// build temperature
	currentTime := time.Now()
	last10Minutes := 0
	last1Hour := 0
	last3Hours := 0
	//newMessages := 0
	for _, message := range response.Messages {
		var messageTime = message.CreatedAt
		var elapsed = currentTime.Sub(messageTime)
		var minutes = int(elapsed.Minutes())
		var hours = int(elapsed.Hours())
		if minutes <= 10 {
			last10Minutes++
		} else if hours <= 1 {
			last1Hour++
		} else if hours <= 3 {
			last3Hours++
		}
	}

	// build the Temperature to add to profile
	stocktwitsTemperature := Temperature{
		last10Minutes: last10Minutes,
		last1Hour:     last1Hour,
		last3Hours:    last3Hours,
	}

	// build the StocktwitsProfile to return
	stocktwitsProfile := StocktwitProfile{
		"response":    response.Response,
		"symbol":      response.Symbol,
		"cursor":      response.Cursor,
		"temperature": stocktwitsTemperature,
	}

	// Return the profile and the message
	return response.Messages, stocktwitsProfile
}

// stocktwitsAPICall makes the actual http call to the stocktwits API and
// returns the response.
func stocktwitsAPICall(url string, response *StocktwitResponse) *StocktwitResponse {
	body, readErr := returnURL(url)
	if readErr != nil {
		fmt.Println(readErr.Error())
		writeFile(logPath, "ERROR: Stocktwits Api error - "+readErr.Error())
	}

	jsonErr := json.Unmarshal(body, response)
	if jsonErr != nil {
		fmt.Println(jsonErr.Error())
		writeFile(logPath, "ERROR: Json Unmarshal error - "+jsonErr.Error())
	}

	return response
}

func returnURL(url string) ([]byte, error) {
	client := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println(err.Error())
	}

	req.Header.Set("User-Agent", "Not Firefox")

	res, getErr := client.Do(req)
	if getErr != nil {
		fmt.Println(err.Error())
	}

	return ioutil.ReadAll(res.Body)
}
