package main

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/seaung/htcrawl-go"
)

type PayloadMap struct {
	Payload  string
	Element  string
}

func main() {
	targetURL := "https://example.com"

	options := htcrawl.DefaultOptions()
	options.Verbose = false
	options.HeadlessChrome = true

	payloads := []string{
		";window.___xssSink({0});",
		"<img src='a' onerror=window.___xssSink({0})>",
	}

	for _, payload := range payloads {
		crawlAndFuzz(targetURL, options, payload)
	}
}

func crawlAndFuzz(targetURL string, options *htcrawl.Options, payload string) {
	pmap := make(map[string]*PayloadMap)
	hashSet := false

	crawler, err := htcrawl.Launch(targetURL, options)
	if err != nil {
		log.Printf("Failed to launch crawler: %v", err)
		return
	}
	defer crawler.Close()

	crawler.Page().Eval(`
		() => {
			window.___xssSink = function(key) {
				console.log("XSS Sink called with key: " + key);
				return true;
			};
		}
	`)

	crawler.On("fillinput", func(event *htcrawl.Event, crawler *htcrawl.Crawler) (interface{}, error) {
		if element, ok := event.Params["element"].(string); ok {
			p := getNewPayload(payload, element, pmap)
			crawler.Page().Eval(fmt.Sprintf(`
				() => {
					var el = document.querySelector("%s");
					if (el) {
						el.value = "%s";
					}
				}
			`, element, escapeJS(p)))
			return false, nil
		}
		return nil, nil
	})

	crawler.On("triggerevent", func(event *htcrawl.Event, crawler *htcrawl.Crawler) (interface{}, error) {
		if !hashSet {
			p := getNewPayload(payload, "hash", pmap)
			crawler.Page().Eval(fmt.Sprintf(`
				() => {
					document.location.hash = "%s";
				}
			`, escapeJS(p)))
			hashSet = true
		}
		return nil, nil
	})

	if err := crawler.Start(); err != nil {
		log.Printf("Error during crawling: %v", err)
	}

	fmt.Println("Fuzzing completed!")
}

func getNewPayload(payload, element string, pmap map[string]*PayloadMap) string {
	k := strconv.Itoa(rand.Intn(4000000000))
	p := strings.ReplaceAll(payload, "{0}", k)
	pmap[k] = &PayloadMap{
		Payload: payload,
		Element: element,
	}
	return p
}

func escapeJS(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
