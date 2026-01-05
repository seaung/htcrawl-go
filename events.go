package htcrawl

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type EventHandler struct {
	mu      sync.RWMutex
	handlers map[string][]EventCallback
}

func NewEventHandler() *EventHandler {
	return &EventHandler{
		handlers: make(map[string][]EventCallback),
	}
}

func (eh *EventHandler) Add(eventName string, handler EventCallback) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	eventName = strings.ToLower(eventName)
	eh.handlers[eventName] = append(eh.handlers[eventName], handler)
}

func (eh *EventHandler) Remove(eventName string, handler EventCallback) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	eventName = strings.ToLower(eventName)
	handlers, ok := eh.handlers[eventName]
	if !ok {
		return
	}

	for i, h := range handlers {
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", handler) {
			eh.handlers[eventName] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

func (eh *EventHandler) Dispatch(eventName string, event *Event, crawler *Crawler) ([]interface{}, error) {
	eh.mu.RLock()
	handlers, ok := eh.handlers[strings.ToLower(eventName)]
	eh.mu.RUnlock()

	if !ok || len(handlers) == 0 {
		return nil, nil
	}

	results := make([]interface{}, 0, len(handlers))
	for _, handler := range handlers {
		result, err := handler(event, crawler)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

func (eh *EventHandler) HasHandler(eventName string) bool {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	_, ok := eh.handlers[strings.ToLower(eventName)]
	return ok
}

func (eh *EventHandler) Clear(eventName string) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	if eventName == "" {
		eh.handlers = make(map[string][]EventCallback)
	} else {
		delete(eh.handlers, strings.ToLower(eventName))
	}
}

type Logger struct {
	verbose bool
	logger  *log.Logger
}

func NewLogger(verbose bool) *Logger {
	return &Logger{
		verbose: verbose,
		logger:  log.Default(),
	}
}

func (l *Logger) Log(format string, v ...interface{}) {
	if l.verbose {
		l.logger.Printf(format, v...)
	}
}

func (l *Logger) LogEvent(event *Event) {
	if !l.verbose {
		return
	}

	switch event.Name {
	case "xhr":
		if req, ok := event.Params["request"].(*Request); ok {
			l.logger.Printf("[XHR] %s %s", req.Method, req.URL)
		}
	case "fetch":
		if req, ok := event.Params["request"].(*Request); ok {
			l.logger.Printf("[FETCH] %s %s", req.Method, req.URL)
		}
	case "newdom":
		if element, ok := event.Params["element"].(string); ok {
			l.logger.Printf("[NEW DOM] Element created: %s", element)
		}
	case "triggerevent":
		if element, ok := event.Params["element"].(string); ok {
			if eventType, ok := event.Params["event"].(string); ok {
				l.logger.Printf("[TRIGGER] %s on '%s'", eventType, element)
			}
		}
	case "navigation":
		if req, ok := event.Params["request"].(*Request); ok {
			l.logger.Printf("[NAVIGATION] %s %s", req.Method, req.URL)
		}
	case "formsubmit":
		if req, ok := event.Params["request"].(*Request); ok {
			l.logger.Printf("[FORM SUBMIT] %s %s", req.Method, req.URL)
		}
	case "websocket":
		if req, ok := event.Params["request"].(*Request); ok {
			l.logger.Printf("[WEBSOCKET] %s", req.URL)
		}
	case "jsonp":
		if req, ok := event.Params["request"].(*Request); ok {
			l.logger.Printf("[JSONP] %s", req.URL)
		}
	case "postmessage":
		if message, ok := event.Params["message"]; ok {
			l.logger.Printf("[POSTMESSAGE] %v", message)
		}
	default:
		l.logger.Printf("[%s] %v", strings.ToUpper(event.Name), event.Params)
	}
}

type RequestCollector struct {
	mu       sync.RWMutex
	requests []*Request
}

func NewRequestCollector() *RequestCollector {
	return &RequestCollector{
		requests: make([]*Request, 0),
	}
}

func (rc *RequestCollector) Add(req *Request) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.requests = append(rc.requests, req)
}

func (rc *RequestCollector) GetAll() []*Request {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	result := make([]*Request, len(rc.requests))
	copy(result, rc.requests)
	return result
}

func (rc *RequestCollector) GetByType(reqType string) []*Request {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	result := make([]*Request, 0)
	for _, req := range rc.requests {
		if req.Type == reqType {
			result = append(result, req)
		}
	}
	return result
}

func (rc *RequestCollector) GetByURL(urlPattern string) []*Request {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	result := make([]*Request, 0)
	for _, req := range rc.requests {
		if strings.Contains(req.URL, urlPattern) {
			result = append(result, req)
		}
	}
	return result
}

func (rc *RequestCollector) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.requests = make([]*Request, 0)
}

func (rc *RequestCollector) Count() int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return len(rc.requests)
}

type DOMCollector struct {
	mu     sync.RWMutex
	elements []string
}

func NewDOMCollector() *DOMCollector {
	return &DOMCollector{
		elements: make([]string, 0),
	}
}

func (dc *DOMCollector) Add(selector string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.elements = append(dc.elements, selector)
}

func (dc *DOMCollector) GetAll() []string {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	result := make([]string, len(dc.elements))
	copy(result, dc.elements)
	return result
}

func (dc *DOMCollector) Clear() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.elements = make([]string, 0)
}

func (dc *DOMCollector) Count() int {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return len(dc.elements)
}

type EventRecorder struct {
	mu     sync.RWMutex
	events []*Event
}

func NewEventRecorder() *EventRecorder {
	return &EventRecorder{
		events: make([]*Event, 0),
	}
}

func (er *EventRecorder) Record(event *Event) {
	er.mu.Lock()
	defer er.mu.Unlock()
	er.events = append(er.events, event)
}

func (er *EventRecorder) GetAll() []*Event {
	er.mu.RLock()
	defer er.mu.RUnlock()
	result := make([]*Event, len(er.events))
	copy(result, er.events)
	return result
}

func (er *EventRecorder) GetByName(eventName string) []*Event {
	er.mu.RLock()
	defer er.mu.RUnlock()

	result := make([]*Event, 0)
	for _, e := range er.events {
		if e.Name == eventName {
			result = append(result, e)
		}
	}
	return result
}

func (er *EventRecorder) Clear() {
	er.mu.Lock()
	defer er.mu.Unlock()
	er.events = make([]*Event, 0)
}

func (er *EventRecorder) Count() int {
	er.mu.RLock()
	defer er.mu.RUnlock()
	return len(er.events)
}

type Stats struct {
	mu                sync.RWMutex
	startTime         time.Time
	endTime           time.Time
	totalRequests     int
	xhrRequests       int
	fetchRequests     int
	websockets        int
	jsonpRequests     int
	formSubmits       int
	navigations       int
	domMutations      int
	triggeredEvents   int
	errors            int
}

func NewStats() *Stats {
	return &Stats{
		startTime: time.Now(),
	}
}

func (s *Stats) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startTime = time.Now()
}

func (s *Stats) End() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.endTime = time.Now()
}

func (s *Stats) RecordRequest(reqType string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.totalRequests++
	switch strings.ToLower(reqType) {
	case "xhr":
		s.xhrRequests++
	case "fetch":
		s.fetchRequests++
	case "websocket":
		s.websockets++
	case "jsonp":
		s.jsonpRequests++
	case "form":
		s.formSubmits++
	case "navigation":
		s.navigations++
	}
}

func (s *Stats) RecordDOMMutation() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.domMutations++
}

func (s *Stats) RecordTriggeredEvent() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.triggeredEvents++
}

func (s *Stats) RecordError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors++
}

func (s *Stats) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	duration := time.Duration(0)
	if !s.endTime.IsZero() {
		duration = s.endTime.Sub(s.startTime)
	}

	return map[string]interface{}{
		"duration":           duration.String(),
		"total_requests":     s.totalRequests,
		"xhr_requests":       s.xhrRequests,
		"fetch_requests":     s.fetchRequests,
		"websockets":         s.websockets,
		"jsonp_requests":     s.jsonpRequests,
		"form_submits":       s.formSubmits,
		"navigations":        s.navigations,
		"dom_mutations":      s.domMutations,
		"triggered_events":   s.triggeredEvents,
		"errors":             s.errors,
	}
}

func (s *Stats) Print() {
	stats := s.GetStats()
	fmt.Println("\n=== Crawler Statistics ===")
	fmt.Printf("Duration: %v\n", stats["duration"])
	fmt.Printf("Total Requests: %d\n", stats["total_requests"])
	fmt.Printf("  - XHR: %d\n", stats["xhr_requests"])
	fmt.Printf("  - Fetch: %d\n", stats["fetch_requests"])
	fmt.Printf("  - WebSockets: %d\n", stats["websockets"])
	fmt.Printf("  - JSONP: %d\n", stats["jsonp_requests"])
	fmt.Printf("  - Form Submits: %d\n", stats["form_submits"])
	fmt.Printf("  - Navigations: %d\n", stats["navigations"])
	fmt.Printf("DOM Mutations: %d\n", stats["dom_mutations"])
	fmt.Printf("Triggered Events: %d\n", stats["triggered_events"])
	fmt.Printf("Errors: %d\n", stats["errors"])
	fmt.Println("=========================")
}

type TimeoutManager struct {
	mu         sync.RWMutex
	timeouts   map[string]time.Time
	defaultTimeout time.Duration
}

func NewTimeoutManager(defaultTimeout time.Duration) *TimeoutManager {
	return &TimeoutManager{
		timeouts:       make(map[string]time.Time),
		defaultTimeout: defaultTimeout,
	}
}

func (tm *TimeoutManager) SetTimeout(key string, duration time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.timeouts[key] = time.Now().Add(duration)
}

func (tm *TimeoutManager) IsExpired(key string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	expiry, ok := tm.timeouts[key]
	if !ok {
		return false
	}
	return time.Now().After(expiry)
}

func (tm *TimeoutManager) Clear(key string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.timeouts, key)
}

func (tm *TimeoutManager) ClearAll() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.timeouts = make(map[string]time.Time)
}

type RateLimiter struct {
	mu       sync.Mutex
	requests []time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make([]time.Time, 0),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	var validRequests []time.Time
	for _, req := range rl.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	rl.requests = validRequests

	if len(rl.requests) >= rl.limit {
		return false
	}

	rl.requests = append(rl.requests, now)
	return true
}

func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.requests = make([]time.Time, 0)
}

type RequestFilter struct {
	mu       sync.RWMutex
	filters  []func(*Request) bool
}

func NewRequestFilter() *RequestFilter {
	return &RequestFilter{
		filters: make([]func(*Request) bool, 0),
	}
}

func (rf *RequestFilter) AddFilter(filter func(*Request) bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.filters = append(rf.filters, filter)
}

func (rf *RequestFilter) ShouldAllow(req *Request) bool {
	rf.mu.RLock()
	defer rf.mu.RUnlock()

	for _, filter := range rf.filters {
		if !filter(req) {
			return false
		}
	}
	return true
}

func (rf *RequestFilter) Clear() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.filters = make([]func(*Request) bool, 0)
}

type URLPatternFilter struct {
	mu       sync.RWMutex
	patterns []string
}

func NewURLPatternFilter() *URLPatternFilter {
	return &URLPatternFilter{
		patterns: make([]string, 0),
	}
}

func (upf *URLPatternFilter) AddPattern(pattern string) {
	upf.mu.Lock()
	defer upf.mu.Unlock()
	upf.patterns = append(upf.patterns, pattern)
}

func (upf *URLPatternFilter) ShouldAllow(url string) bool {
	upf.mu.RLock()
	defer upf.mu.RUnlock()

	for _, pattern := range upf.patterns {
		if strings.Contains(url, pattern) {
			return true
		}
	}
	return false
}

func (upf *URLPatternFilter) ShouldBlock(url string) bool {
	upf.mu.RLock()
	defer upf.mu.RUnlock()

	for _, pattern := range upf.patterns {
		if strings.Contains(url, pattern) {
			return false
		}
	}
	return true
}

func (upf *URLPatternFilter) Clear() {
	upf.mu.Lock()
	defer upf.mu.Unlock()
	upf.patterns = make([]string, 0)
}
