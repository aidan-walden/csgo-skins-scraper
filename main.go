package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	toStdout := flag.Bool("stdout", false, "write JSON to stdout instead of skins.json")
	flag.Parse()

	// Remaining positional args are an optional case-name allowlist.
	// Empty means scrape every case (back-compat).
	filter := flag.Args()

	results, err := ScrapeWebsite(context.Background(), filter)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error scraping website:", err)
		os.Exit(1)
	}

	marshalled, err := json.MarshalIndent(results, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error marshalling results:", err)
		os.Exit(1)
	}

	if *toStdout {
		os.Stdout.Write(marshalled)
		return
	}

	if err := os.WriteFile("skins.json", marshalled, 0644); err != nil {
		fmt.Fprintln(os.Stderr, "error writing to skins.json:", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "done")
}
