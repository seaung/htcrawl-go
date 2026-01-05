package main

import (
	"fmt"
	"log"
	"regexp"

	"github.com/seaung/htcrawl-go"
)

func main() {
	targetURL := "https://example.com"

	options := htcrawl.DefaultOptions()
	options.Verbose = false
	options.HeadlessChrome = true

	crawler, err := htcrawl.Launch(targetURL, options)
	if err != nil {
		log.Fatalf("Failed to launch crawler: %v", err)
	}
	defer crawler.Close()

	emailRegex := regexp.MustCompile(`[a-z0-9._-]+@[a-z0-9._-]+\.[a-z]+`)

	printEmails := func(text string) {
		emails := emailRegex.FindAllString(text, -1)
		for _, email := range emails {
			fmt.Printf("Found email: %s\n", email)
		}
	}

	crawler.On("domcontentloaded", func(event *htcrawl.Event, crawler *htcrawl.Crawler) (interface{}, error) {
		html, err := crawler.Page().Element("body")
		if err != nil {
			return nil, err
		}
		text, err := html.Text()
		if err != nil {
			return nil, err
		}
		printEmails(text)
		return nil, nil
	})

	crawler.On("newdom", func(event *htcrawl.Event, crawler *htcrawl.Crawler) (interface{}, error) {
		if rootNode, ok := event.Params["rootNode"].(string); ok {
			el, err := crawler.Page().Element(rootNode)
			if err != nil {
				return nil, err
			}
			text, err := el.Text()
			if err != nil {
				return nil, err
			}
			printEmails(text)
		}
		return nil, nil
	})

	if err := crawler.Start(); err != nil {
		log.Fatalf("Failed to start crawling: %v", err)
	}

	fmt.Println("Content scraping completed!")
}
