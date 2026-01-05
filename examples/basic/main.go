package main

import (
	"fmt"
	"log"

	"github.com/seaung/htcrawl-go"
)

func main() {
	targetURL := "https://example.com"

	options := htcrawl.DefaultOptions()
	options.Verbose = true
	options.HeadlessChrome = true

	crawler, err := htcrawl.Launch(targetURL, options)
	if err != nil {
		log.Fatalf("Failed to launch crawler: %v", err)
	}
	defer crawler.Close()

	crawler.On("xhr", func(event *htcrawl.Event, crawler *htcrawl.Crawler) (interface{}, error) {
		if req, ok := event.Params["request"].(*htcrawl.Request); ok {
			fmt.Printf("XHR to %s\n", req.URL)
		}
		return nil, nil
	})

	crawler.On("newdom", func(event *htcrawl.Event, crawler *htcrawl.Crawler) (interface{}, error) {
		if element, ok := event.Params["element"].(string); ok {
			fmt.Printf("New DOM element created: %s\n", element)
		}
		return nil, nil
	})

	crawler.On("triggerevent", func(event *htcrawl.Event, crawler *htcrawl.Crawler) (interface{}, error) {
		if element, ok := event.Params["element"].(string); ok {
			if eventType, ok := event.Params["event"].(string); ok {
				fmt.Printf("Triggered %s on '%s'\n", eventType, element)
			}
		}
		return nil, nil
	})

	if err := crawler.Start(); err != nil {
		log.Fatalf("Failed to start crawling: %v", err)
	}

	fmt.Println("Crawling completed!")
}
