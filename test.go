package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Main functionality -
// Get the stocks from the database, getting lists and then the stocks
// on those list.  Pass the stocks to the stocktwits module to return
// the stocktwit data and store in the database.
func main() {
	// Setup the logger and custom file to write to
	// If the file doesn't exist, create it or append to the file
	customLogFile := "$HOME/stocktwitslogs.txt"
	file, err := os.OpenFile(customLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(file)

	// connect to the mongo db
	mongoURI := "mongodb+srv://rssfeeduser:FeEd__3467@cluster0.rxylk.mongodb.net/rssFeed?retryWrites=true&w=majority"
	client := connectMongo(mongoURI)

	/******************************
	 * GET STOCKS TO SEARCH FOR
	 * FROM DB
	 ******************************/
	// get stocks from db
	var lists []List
	stockCollection := client.Database("rssFeed").Collection("list")
	ctx := context.TODO()
	filter := bson.D{}
	var listResult List
	cursor, err := stockCollection.Find(context.TODO(), filter)
	if err != nil {
		fmt.Println("FAILED to find list collection: ", err.Error())
		log.Fatal(err)
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		err := cursor.Decode(&listResult)
		if err != nil {
			fmt.Println("FAILED to decode list: ", err.Error())
			log.Fatal(err)
		}
		lists = append(lists, listResult)
	}

	// Loop over lists and search for stocks
	for _, list := range lists {
		for _, stock := range list.Stocks {
			log.Println("Searching stockwtits for " + stock.Symbol)
			messages, profile := StocktwitsCallAPI(stock.Symbol)

			// Get the existing stocktwits messages to see which need to be
			// added to the database, add if new
			/*****************************************
			* GET CURRENTLY SAVED STOCKTWIT MESSAGES
			*****************************************/
			var currentMessages []StocktwitMessages
			var currentMessage StocktwitMessages
			messagesCollection := client.Database("rssFeed").Collection("stocktwitmessages")
			ctx := context.TODO()
			filter := bson.M{"symbol": symbol}
			findOptions := options.Find()
			cursor, err := messagesCollection.Find(context.TODO(), filter, findOptions)
			if err != nil {
				fmt.Println("FAILED to find stocktwit message collection: ", err.Error())
				log.Fatal(err)
			}
			defer cursor.Close(ctx)
			for cursor.Next(ctx) {
				err := cursor.Decode(&currentMessage)
				if err != nil {
					fmt.Println("FAILED to decode stocktwit message: ", err.Error())
					log.Fatal(err)
				}
				// append to currentMessages
				currentMessages = append(currentMessages, currentMessage)
			}
		
			fmt.Println("Current Messages Number: ", len(currentMessages))

			// loop over messages and check if already in database; insert if not
			for _, message := range messages {
				if checkForExistingMessage(message.Body, &currentMessages) {
					//fmt.Println("Already there.")
				} else {
					// insert the message
					_, insertErr := messagesCollection.InsertOne(ctx, bson.M{
						"symbol":     symbol,
						"body":       message.Body,
						"created_at": message.CreatedAt,
						"user":       message.User,
						"source":     message.Source,
						"symbols":    message.Symbols,
					})
					if insertErr != nil {
						fmt.Println("FAILED to insert: ", insertErr.Error())
						log.Println("ERROR: "+insertErr.Error())
					}
					//id := res.InsertedID
					//fmt.Println(id)
					//newMessages++
				}
			}
		}
	
		// insert the profile into the database
		opts := options.Update().SetUpsert(true)
		filter = bson.M{"symbol.symbol": symbol}
		/*update := bson.M{"$set": bson.M{
			"response":    response.Response,
			"symbol":      response.Symbol,
			"cursor":      response.Cursor,
			"temperature": temp,
		}}*/
		stocktwitProfileCollection := client.Database("rssFeed").Collection("stocktwitprofile")
		res, err := stocktwitProfileCollection.UpdateOne(ctx, filter, profile, opts)
		if err != nil {
			fmt.Println("FAILED to insert: ", err.Error())
			log.Println("ERROR: "+err.Error())
		}
		fmt.Println("Updated the profile!", res)
		}
	}
}

func connectMongo(mongoURI string) *mongo.client {
	/******************************
	 * MONGO CONNECTION SECTION
	 ******************************/
	// Set client options
	mongoConnectionURI := mongoURI
	clientOptions := options.Client().ApplyURI(mongoConnectionURI)

	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		fmt.Println("FAILED to conect to Mongo: ", err.Error())
		log.Fatal(err)
	}
	defer client.Disconnect(context.TODO())

	return &client
}

func checkForExistingMessage(body string, current *[]StocktwitMessages) bool {
	// from https://stackoverflow.com/questions/38654383/how-to-search-for-an-element-in-a-golang-slice
	/*idx := sort.Search(len(*current), func(i int) bool {
		fmt.Println("INDEX ", i)
		return string((*current)[i].Headline) >= headline
	})

	if (*current)[idx].Headline == headline {
		return true
	}
	NEED TO WORK ON THE BINARY SEARCH AND USE IT, NOT THIS MUCH SLOWER LINEAR...
	*/

	// LINEAR SEARCH
	for _, hl := range *current {
		if hl.Body == body {
			return true
		}
	}

	return false
}

