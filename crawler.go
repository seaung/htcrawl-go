package htcrawl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type EventCallback func(event *Event, crawler *Crawler) (interface{}, error)

type Event struct {
	Name   string
	Params map[string]interface{}
}

type PendingRequest struct {
	RequestID string
	URL       string
	Type      string
}

type Crawler struct {
	targetUrl          string
	options            *Options
	browser            *rod.Browser
	page               *rod.Page
	trigger            *Trigger
	pendingRequests    []*PendingRequest
	sentRequests       map[string]bool
	domDeduplicator    *DOMDeduplicator
	cookies            []Cookie
	errors             [][2]string
	redirect           string
	loaded             bool
	allowNavigation    bool
	allowNewWindows    bool
	firstRun           bool
	stop               bool
	probeEvents        map[string]EventCallback
	uiEvents           map[string]EventCallback
	mu                 sync.RWMutex
	documentElement    *rod.Element
	status             struct {
		layer      string
		curElement string
	}
}

func Launch(targetURL string, options *Options) (*Crawler, error) {
	if options == nil {
		options = DefaultOptions()
	}

	targetURL = NormalizeURL(targetURL)

	if options.ShowUI {
		options.OpenChromeDevtools = true
	}

	if options.OpenChromeDevtools {
		options.HeadlessChrome = false
	}

	chromeArgs := []string{
		"--no-sandbox",
		"--disable-setuid-sandbox",
		"--disable-gpu",
		"--mute-audio",
		"--ignore-certificate-errors",
		"--ignore-certificate-errors-spki-list",
		"--ssl-version-max=tls1.3",
		"--ssl-version-min=tls1",
		"--disable-web-security",
		"--allow-running-insecure-content",
		"--proxy-bypass-list=<-loopback>",
		fmt.Sprintf("--window-size=%d,%d", options.WindowSize[0], options.WindowSize[1]),
	}

	if options.IncludeAllOrigins {
		chromeArgs = append(chromeArgs, "--disable-features=OutOfBlinkCors,IsolateOrigins,SitePerProcess")
	}

	if options.Proxy != "" {
		chromeArgs = append(chromeArgs, "--proxy-server="+options.Proxy)
	}

	launcherPath := launcher.New()
	if !options.HeadlessChrome {
		launcherPath = launcherPath.Headless(false)
	} else {
		launcherPath = launcherPath.Headless(true)
	}

	if options.OpenChromeDevtools {
		launcherPath = launcherPath.Devtools(true)
	}

	browserURL, err := launcherPath.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(browserURL)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	crawler := &Crawler{
		targetUrl:       targetURL,
		options:         options,
		browser:         browser,
		pendingRequests: make([]*PendingRequest, 0),
		sentRequests:    make(map[string]bool),
		domDeduplicator: NewDOMDeduplicator(),
		cookies:         make([]Cookie, 0),
		errors:          make([][2]string, 0),
		loaded:          false,
		allowNavigation: false,
		allowNewWindows: false,
		firstRun:        true,
		stop:            false,
		probeEvents:     make(map[string]EventCallback),
		uiEvents:        make(map[string]EventCallback),
	}

	if err := crawler.bootstrapPage(); err != nil {
		browser.Close()
		return nil, fmt.Errorf("failed to bootstrap page: %w", err)
	}

	go crawler.requestLoop()

	browser.EachEvent(func(e *proto.TargetTargetCreated) {
		if !crawler.allowNewWindows && e.TargetInfo.Type == "page" {
			page, err := crawler.browser.Page(proto.TargetCreateTarget{})
			if err == nil {
				page.Close()
			}
		}
	})

	return crawler, nil
}

func (c *Crawler) Browser() *rod.Browser {
	return c.browser
}

func (c *Crawler) Page() *rod.Page {
	return c.page
}

func (c *Crawler) Cookies() ([]Cookie, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cookies, err := c.page.Cookies([]string{})
	if err != nil {
		return c.cookies, err
	}

	for _, cookie := range cookies {
		c.cookies = append(c.cookies, Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Expires:  int64(cookie.Expires),
			HttpOnly: cookie.HTTPOnly,
			Secure:   cookie.Secure,
		})
	}

	return c.cookies, nil
}

func (c *Crawler) Redirect() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.redirect
}

func (c *Crawler) Errors() [][2]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.errors
}

func (c *Crawler) On(eventName string, handler EventCallback) error {
	eventName = strings.ToLower(eventName)
	validEvents := map[string]bool{
		"start": true, "xhr": true, "xhrcompleted": true, "fetch": true,
		"fetchcompleted": true, "jsonp": true, "jsonpcompleted": true,
		"websocket": true, "websocketmessage": true, "websocketsend": true,
		"formsubmit": true, "fillinput": true, "newdom": true,
		"navigation": true, "domcontentloaded": true, "redirect": true,
		"earlydetach": true, "triggerevent": true, "eventtriggered": true,
		"pageinitialized": true, "crawlelement": true, "postmessage": true,
	}

	if !validEvents[eventName] {
		return fmt.Errorf("unknown event name: %s", eventName)
	}

	if eventName == "postmessage" && !c.options.OverridePostMessage {
		return fmt.Errorf("overridePostMessage option must be true to use 'postmessage'")
	}

	c.mu.Lock()
	c.probeEvents[eventName] = handler
	c.mu.Unlock()
	return nil
}

func (c *Crawler) RemoveEvent(eventName string) error {
	eventName = strings.ToLower(eventName)
	validEvents := map[string]bool{
		"start": true, "xhr": true, "xhrcompleted": true, "fetch": true,
		"fetchcompleted": true, "jsonp": true, "jsonpcompleted": true,
		"websocket": true, "websocketmessage": true, "websocketsend": true,
		"formsubmit": true, "fillinput": true, "newdom": true,
		"navigation": true, "domcontentloaded": true, "redirect": true,
		"earlydetach": true, "triggerevent": true, "eventtriggered": true,
		"pageinitialized": true, "crawlelement": true, "postmessage": true,
	}

	if !validEvents[eventName] {
		return fmt.Errorf("unknown event name: %s", eventName)
	}

	c.mu.Lock()
	delete(c.probeEvents, eventName)
	c.mu.Unlock()
	return nil
}

func (c *Crawler) Load() error {
	_, err := c.navigateTo(c.targetUrl)
	if err != nil {
		return err
	}
	return c.afterNavigation(nil)
}

func (c *Crawler) navigateTo(url string) (*proto.NetworkResponse, error) {
	if c.options.Verbose {
		fmt.Println("LOAD")
	}

	c.mu.Lock()
	c.allowNavigation = true
	c.mu.Unlock()

	var err error

	wait := c.page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	err = c.page.Navigate(url)
	if err == nil {
		wait()
	}

	c.mu.Lock()
	c.allowNavigation = false
	c.mu.Unlock()

	return nil, err
}

func (c *Crawler) afterNavigation(resp *proto.NetworkResponse) error {
	c.documentElement, _ = c.page.Element("html")
	c.domDeduplicator.Reset()
	c.loaded = true

	if c.isEventRegistered("domcontentloaded") {
		c.dispatchProbeEvent("domcontentloaded", map[string]interface{}{})
	}

	c.waitForRequestsCompletion()

	if c.isEventRegistered("pageinitialized") {
		c.dispatchProbeEvent("pageinitialized", map[string]interface{}{})
	}

	if err := c.startMutationObserver(); err != nil {
		return err
	}

	return nil
}

func (c *Crawler) waitForRequestsCompletion() {
	c.waitForRequests()
	c.page.Eval(`() => {
		return Promise.all([
			window.__PROBE__ ? window.__PROBE__.waitJsonp() : Promise.resolve(),
			window.__PROBE__ ? window.__PROBE__.waitWebsocket() : Promise.resolve()
		]);
	}`)
}

func (c *Crawler) waitForRequests() {
	timeout := time.Duration(c.options.AjaxTimeout) * time.Millisecond
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.mu.RLock()
			pendingCount := len(c.pendingRequests)
			c.mu.RUnlock()
			if pendingCount == 0 {
				return
			}
		}
	}
}

func (c *Crawler) Start() error {
	if !c.loaded {
		if err := c.Load(); err != nil {
			return err
		}
	}

	c.mu.Lock()
	c.stop = false
	c.mu.Unlock()

	if err := c.fillInputValues(nil); err != nil {
		return err
	}

	if err := c.crawlDOM(nil); err != nil {
		return err
	}

	return nil
}

func (c *Crawler) Stop() {
	c.mu.Lock()
	c.stop = true
	c.mu.Unlock()
}

func (c *Crawler) Close() error {
	return c.browser.Close()
}

func (c *Crawler) isEventRegistered(event string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.probeEvents[event]
	return ok
}

func (c *Crawler) dispatchProbeEvent(name string, params map[string]interface{}) (interface{}, error) {
	name = strings.ToLower(name)
	evt := &Event{
		Name:   name,
		Params: params,
	}

	c.mu.RLock()
	handler, ok := c.probeEvents[name]
	c.mu.RUnlock()

	if !ok {
		return true, nil
	}

	ret, err := handler(evt, c)
	if err != nil {
		return nil, err
	}

	if ret == false {
		return false, nil
	}

	return ret, nil
}

func (c *Crawler) requestLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		for i := len(c.pendingRequests) - 1; i >= 0; i-- {
			c.pendingRequests = append(c.pendingRequests[:i], c.pendingRequests[i+1:]...)
		}
		c.mu.Unlock()
	}
}

func parseContentLength(s string) (int64, error) {
	var size int64
	_, err := fmt.Sscanf(s, "%d", &size)
	return size, err
}

func (c *Crawler) readProbeScript() (string, error) {
	content, err := os.ReadFile("probe.js")
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (c *Crawler) bootstrapPage() error {
	page, err := c.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}

	c.page = page

	if err := c.setupRequestInterception(); err != nil {
		return fmt.Errorf("failed to setup request interception: %w", err)
	}

	if err := c.setupDialogHandler(); err != nil {
		return fmt.Errorf("failed to setup dialog handler: %w", err)
	}

	if err := c.setupProbeScript(); err != nil {
		return fmt.Errorf("failed to setup probe script: %w", err)
	}

	if err := c.setupHeadersAndCookies(); err != nil {
		return fmt.Errorf("failed to setup headers and cookies: %w", err)
	}

	if c.options.Referer != "" {
		if _, err := c.page.SetExtraHeaders([]string{"Referer:" + c.options.Referer}); err != nil {
			return err
		}
	}

	if len(c.options.ExtraHeaders) > 0 {
		headers := make([]string, 0)
		for k, v := range c.options.ExtraHeaders {
			headers = append(headers, k+":"+v)
		}
		if _, err := c.page.SetExtraHeaders(headers); err != nil {
			return err
		}
	}

	if c.options.UserAgent != "" {
		return nil
	}

	return nil
}

func (c *Crawler) setupRequestInterception() error {
	return nil
}

func (c *Crawler) setupDialogHandler() error {
	return nil
}

func (c *Crawler) setupProbeScript() error {
	probeScript, err := c.readProbeScript()
	if err != nil {
		return err
	}

	optionsJSON, err := json.Marshal(c.options)
	if err != nil {
		return err
	}

	inputValues := GenerateRandomValues(c.options.RandomSeed)
	inputValuesJSON, err := json.Marshal(inputValues)
	if err != nil {
		return err
	}

	script := fmt.Sprintf(`
		%s
		window.__htcrawl_probe_event__ = async function(name, params) {
			return window.__htcrawl_go_bridge__(name, params);
		};
	`, probeScript)

	if _, err := c.page.EvalOnNewDocument(script); err != nil {
		return err
	}

	initScript := fmt.Sprintf(`
		(function() {
			var options = %s;
			var inputValues = %s;
			%s
		})();
	`, string(optionsJSON), string(inputValuesJSON), probeScript)

	_, err = c.page.Eval(initScript)
	return err
}

func (c *Crawler) setupHeadersAndCookies() error {
	for i := range c.options.SetCookies {
		cookie := c.options.SetCookies[i]
		if cookie.Expires == 0 {
			cookie.Expires = time.Now().Unix() + 60*60*24*365
		}
		if cookie.URL == "" || cookie.Domain == "" {
			parsedURL, err := url.Parse(c.targetUrl)
			if err == nil {
				cookie.Domain = parsedURL.Hostname()
				cookie.URL = c.targetUrl
			}
		}
		if cookie.Value == "" {
			cookie.Value = ""
		}
		c.options.SetCookies[i] = cookie
	}

	if len(c.options.SetCookies) > 0 {
		protoCookies := make([]*proto.NetworkCookieParam, len(c.options.SetCookies))
		for i, cookie := range c.options.SetCookies {
			protoCookies[i] = &proto.NetworkCookieParam{
				Name:     cookie.Name,
				Value:    cookie.Value,
				Domain:   cookie.Domain,
				Path:     cookie.Path,
				Expires:  proto.TimeSinceEpoch(cookie.Expires),
				Secure:   cookie.Secure,
				HTTPOnly: cookie.HttpOnly,
			}
		}
		if err := c.page.SetCookies(protoCookies); err != nil {
			return err
		}
	}

	return nil
}

func (c *Crawler) startMutationObserver() error {
	_, err := c.page.Eval(`
		() => {
			if (window.__PROBE__) {
				window.__PROBE__._newMutationObserver(document.documentElement);
			}
		}
	`)
	return err
}

func (c *Crawler) resetMutationObserver() error {
	_, err := c.page.Eval(`
		() => {
			if (window.__PROBE__) {
				window.__PROBE__.DOMMutations = [];
				window.__PROBE__.DOMMutationsToPop = [];
				window.__PROBE__.totalDOMMutations = 0;
			}
		}
	`)
	return err
}

func (c *Crawler) fillInputValues(element *rod.Element) error {
	selector := "html"
	if element != nil {
		selector = element.MustText()
	}

	_, err := c.page.Eval(fmt.Sprintf(`
		() => {
			if (window.__PROBE__) {
				return window.__PROBE__.fillInputValues(%s);
			}
		}
	`, selector))

	return err
}

func (c *Crawler) crawlDOM(element *rod.Element) error {
	elements, err := c.getDOMTreeAsArray(element)
	if err != nil {
		return err
	}

	for _, el := range elements {
		c.mu.RLock()
		stop := c.stop
		c.mu.RUnlock()

		if stop {
			break
		}

		if err := c.crawlElement(el); err != nil {
			continue
		}
	}

	return nil
}

func (c *Crawler) crawlElement(element *rod.Element) error {
	events, err := c.getEventsForElement(element)
	if err != nil {
		return err
	}

	for _, event := range events {
		c.mu.RLock()
		stop := c.stop
		c.mu.RUnlock()

		if stop {
			break
		}

		if err := c.triggerElementEvent(element, event); err != nil {
			continue
		}
	}

	return nil
}

func (c *Crawler) getDOMTreeAsArray(node *rod.Element) ([]*rod.Element, error) {
	var out []*rod.Element

	selector := "html"
	if node != nil {
		selector = node.MustText()
	}

	elements, err := c.page.Elements(selector + " > *:not([data-htcrawl_crawl_excluded_element])")
	if err != nil {
		return out, err
	}

	for _, el := range elements {
		out = append(out, el)
		children, err := c.getDOMTreeAsArray(el)
		if err != nil {
			continue
		}
		out = append(out, children...)
	}

	return out, nil
}

func (c *Crawler) getEventsForElement(el *rod.Element) ([]string, error) {
	var events []string

	selector := el.MustText()

	_, err := c.page.Eval(fmt.Sprintf(`
		() => {
			if (window.__PROBE__) {
				return window.__PROBE__.getEventsForElement(%s);
			}
			return [];
		}
	`, selector))

	return events, err
}

func (c *Crawler) triggerElementEvent(el *rod.Element, event string) error {
	selector := el.MustText()

	_, err := c.page.Eval(fmt.Sprintf(`
		() => {
			if (window.__PROBE__) {
				window.__PROBE__.triggerElementEvent(%s, "%s");
			}
		}
	`, selector, event))

	return err
}

func (c *Crawler) Navigate(url string) error {
	if !c.loaded {
		return fmt.Errorf("crawler must be loaded before navigate")
	}

	_, err := c.navigateTo(url)
	if err != nil {
		c.errors = append(c.errors, [2]string{"navigation", "navigation aborted"})
		return fmt.Errorf("navigation error: %w", err)
	}

	return nil
}

func (c *Crawler) Reload() error {
	if !c.loaded {
		return fmt.Errorf("crawler must be loaded before reload")
	}

	c.mu.Lock()
	c.allowNavigation = true
	c.mu.Unlock()

	var err error

	wait := c.page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	err = c.page.Reload()
	if err == nil {
		wait()
	}

	c.mu.Lock()
	c.allowNavigation = false
	c.mu.Unlock()

	if err != nil {
		c.errors = append(c.errors, [2]string{"navigation", "navigation aborted"})
		return fmt.Errorf("navigation error: %w", err)
	}

	return nil
}

func (c *Crawler) ClickToNavigate(selector string, timeout time.Duration, untilSelector string) error {
	if !c.loaded {
		return fmt.Errorf("crawler must be loaded before navigate")
	}

	if timeout == 0 {
		timeout = 5 * time.Second
	}

	el, err := c.page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %w", err)
	}

	c.mu.Lock()
	c.allowNavigation = true
	c.mu.Unlock()

	var navErr error
	var navigated bool

	for {
		navigated = false
		navErr = nil

		done := make(chan error, 1)

		go func() {
			err := el.Click(proto.InputMouseButtonLeft, 1)
			if err != nil {
				done <- err
				return
			}

			time.Sleep(timeout)
			done <- nil
		}()

		select {
		case err := <-done:
			if err == nil {
				navigated = true
			}
		case <-time.After(timeout):
			navErr = fmt.Errorf("navigation timeout")
		}

		if untilSelector != "" {
			_, err := c.page.Element(untilSelector)
			if err == nil {
				break
			}
		}

		if navigated || navErr != nil {
			break
		}
	}

	c.mu.Lock()
	c.allowNavigation = false
	c.mu.Unlock()

	if navErr != nil {
		c.errors = append(c.errors, [2]string{"navigation", "navigation aborted"})
		return navErr
	}

	return nil
}

func (c *Crawler) NewPage(url string) error {
	if url != "" {
		c.targetUrl = NormalizeURL(url)
	}
	c.firstRun = true
	return c.bootstrapPage()
}

func (c *Crawler) NewDetachedPage(url string) (*rod.Page, error) {
	c.mu.Lock()
	c.allowNewWindows = true
	c.mu.Unlock()

	page, err := c.browser.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		c.mu.Lock()
		c.allowNewWindows = false
		c.mu.Unlock()
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	c.mu.Lock()
	c.allowNewWindows = false
	c.mu.Unlock()

	return page, nil
}

func (c *Crawler) GetElementText(el *rod.Element) (string, error) {
	return el.Text()
}

func (c *Crawler) GetElementSelector(el *rod.Element) (string, error) {
	selector := el.MustText()

	_, err := c.page.Eval(fmt.Sprintf(`
		() => {
			if (window.__PROBE__) {
				return window.__PROBE__.getElementSelector(%s);
			}
			return "";
		}
	`, selector))

	return "", err
}

func (c *Crawler) GetTotalDomMutations() (int, error) {
	_, err := c.page.Eval(`
		() => {
			if (window.__PROBE__) {
				return window.__PROBE__.totalDOMMutations;
			}
			return 0;
		}
	`)

	return 0, err
}

func (c *Crawler) PopMutation() (string, error) {
	_, err := c.page.Eval(`
		() => {
			if (window.__PROBE__) {
				return window.__PROBE__.popMutation();
			}
			return null;
		}
	`)

	return "", err
}

func (c *Crawler) SetTrigger(trigger *Trigger) {
	c.mu.Lock()
	c.trigger = trigger
	c.mu.Unlock()
}

func (c *Crawler) DownloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
