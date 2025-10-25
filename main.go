package main

import (
	"encoding/json"
	"itunes-xml-parser/itunes"
	"log"
)

func main() {
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

	if err2 != nil {
		log.Fatalf("something went wrong %v", err)
	}

	jsonBytes2, err := json.MarshalIndent(res2, "", "  ")
	if err != nil {
		log.Fatalf("error marshaling: %v", err)
	}

	log.Println(string(jsonBytes2))

}
