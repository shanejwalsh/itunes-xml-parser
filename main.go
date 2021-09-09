package main

import (
	"itunes-xml-parser/feeds"
	"itunes-xml-parser/itunes"
	"log"
)


func main()  {
	ias := itunes.NewItunesApiServices()

	res, err := ias.Search("Random Trek")

	if (err != nil)  {
		log.Fatalf("something went wrong %v", err)
	}

	for _, item := range res.Results {
		log.Println("----------------")
		log.Println("Podcast Name:", item.TrackName )
		log.Println("Feed URL:", item.FeedURL )


		feed, err := feeds.GetFeed(item.FeedURL)

		if err != nil {
			log.Fatalf("error: %s", err)
		}


		for _, pod := range feed.Channel.Item {

			log.Println("------------------------")
			log.Printf("title: %s", pod.Title)
			log.Printf("URL: %s", pod.Enclosure.URL)

		}
		log.Println("rest:", item )
	}
};