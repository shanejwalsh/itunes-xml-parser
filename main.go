package main

import (
	"encoding/json"
	"log"

	"github.com/shanejwalsh/itunes-xml-parser/feeds"
	"github.com/shanejwalsh/itunes-xml-parser/itunes"
)

func main() {

	// For demo purposes only, packages should be imported directly to your project

	ias := itunes.NewItunesApiServices()

	res, err := ias.Search("My therapist ghosted me")

	if err != nil {
		log.Fatalf("something went wrong %v", err)
	}

	jsonBytes, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		log.Fatalf("error marshaling: %v", err)
	}

	log.Println(string(jsonBytes))

	firstId := res.Results[0].CollectionID

	log.Printf("Results: %d", len(res.Results))
	log.Printf("Id Result: %d", res.Results[0].CollectionID)

	res2, err2 := ias.FindById(firstId)
	feedUrl := res2.Results[0].FeedURL

	log.Printf("Id Result: %s", feedUrl)

	if err2 != nil {
		log.Fatalf("something went wrong %v", err)
	}

	feedRes, err := feeds.GetFeed(feedUrl)

	jsonBytes2, err := json.MarshalIndent(feedRes, "", "  ")
	if err != nil {
		log.Fatalf("error marshaling: %v", err)
	}

	log.Println(string(jsonBytes2))

}
