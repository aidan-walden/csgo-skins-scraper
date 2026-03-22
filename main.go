package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	results, err := ScrapeWebsite(context.Background())
	if err != nil {
		fmt.Println("error scraping website:", err)
		return
	}

	marshalled, err := json.MarshalIndent(results, "", "    ")

	if err != nil {
		fmt.Println("error marshalling results:", err)
		return
	}

	err = os.WriteFile("skins.json", marshalled, 0644)

	if err != nil {
		fmt.Println("error writing to skins.json:", err)
		return
	}

	fmt.Println("done")

}
