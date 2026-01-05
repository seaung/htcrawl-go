package htcrawl

import (
	"testing"
	"time"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts == nil {
		t.Fatal("DefaultOptions returned nil")
	}

	if !opts.CheckAjax {
		t.Error("Expected CheckAjax to be true")
	}

	if !opts.HeadlessChrome {
		t.Error("Expected HeadlessChrome to be true")
	}

	if len(opts.MouseEvents) == 0 {
		t.Error("Expected MouseEvents to have default values")
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com", "http://example.com"},
		{"https://example.com", "https://example.com"},
		{"http://example.com", "http://example.com"},
		{"  example.com  ", "http://example.com"},
	}

	for _, test := range tests {
		result := NormalizeURL(test.input)
		if result != test.expected {
			t.Errorf("NormalizeURL(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestGenerateRandomValues(t *testing.T) {
	seed := "testseed"
	values := GenerateRandomValues(seed)

	if values == nil {
		t.Fatal("GenerateRandomValues returned nil")
	}

	expectedTypes := []string{"string", "number", "email", "url", "password"}
	for _, typ := range expectedTypes {
		if _, ok := values[typ]; !ok {
			t.Errorf("Expected value type %s not found", typ)
		}
	}
}

func TestCRC32(t *testing.T) {
	tests := []struct {
		input    string
		expected uint32
	}{
		{"test", 0xD87F7E0C},
		{"hello", 0x3610A686},
	}

	for _, test := range tests {
		result := CRC32(test.input)
		if result != test.expected {
			t.Errorf("CRC32(%q) = %d, expected %d", test.input, result, test.expected)
		}
	}
}

func TestSimHash(t *testing.T) {
	arr := []string{"div", "span", "a", "button"}
	hash := SimHash(arr)

	if hash == 0 {
		t.Error("SimHash returned 0")
	}
}

func TestSimilarity(t *testing.T) {
	x := uint32(0xFFFFFFFF)
	y := uint32(0xFFFFFFFF)
	sim := Similarity(x, y)

	if sim != 1.0 {
		t.Errorf("Similarity of identical hashes should be 1.0, got %f", sim)
	}
}

func TestHammingWeight(t *testing.T) {
	tests := []struct {
		input    uint32
		expected uint32
	}{
		{0x00000000, 0},
		{0xFFFFFFFF, 32},
		{0x00000001, 1},
		{0x00000011, 2},
	}

	for _, test := range tests {
		result := HammingWeight(test.input)
		if result != test.expected {
			t.Errorf("HammingWeight(%d) = %d, expected %d", test.input, result, test.expected)
		}
	}
}

func TestDOMDeduplicator(t *testing.T) {
	dd := NewDOMDeduplicator()

	domArray := []string{"div", "span", "a", "button"}
	result := dd.AddNode(domArray, 10)

	if !result.Added {
		t.Error("Expected first node to be added")
	}

	result2 := dd.AddNode(domArray, 10)
	if result2.Added {
		t.Error("Expected duplicate node not to be added")
	}

	if result2.SeenCount != 2 {
		t.Errorf("Expected SeenCount to be 2, got %d", result2.SeenCount)
	}
}

func TestDOMDeduplicatorReset(t *testing.T) {
	dd := NewDOMDeduplicator()

	domArray := []string{"div", "span"}
	dd.AddNode(domArray, 5)

	dd.Reset()

	if dd.GetNodeCount() != 0 {
		t.Error("Expected node count to be 0 after reset")
	}
}

func TestRequestKey(t *testing.T) {
	req := &Request{
		Type:   "xhr",
		Method: "GET",
		URL:    "https://example.com/api",
		Data:   "test=data",
		Trigger: &Trigger{
			Element: "button#submit",
			Event:   "click",
		},
	}

	key := req.Key()
	if key == "" {
		t.Error("Request key should not be empty")
	}

	expected := "xhrGEThttps://example.com/apitest=databutton#submitclick"
	if key != expected {
		t.Errorf("Request key = %q, expected %q", key, expected)
	}
}

func TestMatchesExcludedURL(t *testing.T) {
	patterns := []string{`.*\.js$`, `.*\.css$`, `.*analytics.*`}

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com/script.js", true},
		{"https://example.com/style.css", true},
		{"https://example.com/analytics.js", true},
		{"https://example.com/page.html", false},
		{"https://example.com/image.png", false},
	}

	for _, test := range tests {
		result := MatchesExcludedURL(test.url, patterns)
		if result != test.expected {
			t.Errorf("MatchesExcludedURL(%q) = %v, expected %v", test.url, result, test.expected)
		}
	}
}

func TestStringSliceContains(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !StringSliceContains(slice, "b") {
		t.Error("Expected slice to contain 'b'")
	}

	if StringSliceContains(slice, "d") {
		t.Error("Expected slice not to contain 'd'")
	}
}

func TestRemoveDuplicateStrings(t *testing.T) {
	input := []string{"a", "b", "a", "c", "b", "d"}
	expected := []string{"a", "b", "c", "d"}

	result := RemoveDuplicateStrings(input)

	if len(result) != len(expected) {
		t.Errorf("Expected %d elements, got %d", len(expected), len(result))
	}

	for i, val := range expected {
		if result[i] != val {
			t.Errorf("Expected %q at index %d, got %q", val, i, result[i])
		}
	}
}

func TestStats(t *testing.T) {
	stats := NewStats()
	stats.Start()

	stats.RecordRequest("xhr")
	stats.RecordRequest("fetch")
	stats.RecordRequest("xhr")
	stats.RecordDOMMutation()
	stats.RecordTriggeredEvent()
	stats.RecordTriggeredEvent()
	stats.RecordTriggeredEvent()
	stats.End()

	s := stats.GetStats()

	if s["total_requests"] != 3 {
		t.Errorf("Expected total_requests to be 3, got %v", s["total_requests"])
	}

	if s["xhr_requests"] != 2 {
		t.Errorf("Expected xhr_requests to be 2, got %v", s["xhr_requests"])
	}

	if s["dom_mutations"] != 1 {
		t.Errorf("Expected dom_mutations to be 1, got %v", s["dom_mutations"])
	}

	if s["triggered_events"] != 3 {
		t.Errorf("Expected triggered_events to be 3, got %v", s["triggered_events"])
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(3, time.Second)

	if !rl.Allow() {
		t.Error("First request should be allowed")
	}

	if !rl.Allow() {
		t.Error("Second request should be allowed")
	}

	if !rl.Allow() {
		t.Error("Third request should be allowed")
	}

	if rl.Allow() {
		t.Error("Fourth request should be blocked")
	}

	time.Sleep(time.Second + 100*time.Millisecond)

	if !rl.Allow() {
		t.Error("Request should be allowed after window expires")
	}
}

func TestRequestCollector(t *testing.T) {
	rc := NewRequestCollector()

	req1 := &Request{
		Type:   "xhr",
		Method: "GET",
		URL:    "https://example.com/api1",
	}
	req2 := &Request{
		Type:   "fetch",
		Method: "POST",
		URL:    "https://example.com/api2",
	}

	rc.Add(req1)
	rc.Add(req2)

	if rc.Count() != 2 {
		t.Errorf("Expected count to be 2, got %d", rc.Count())
	}

	xhrReqs := rc.GetByType("xhr")
	if len(xhrReqs) != 1 {
		t.Errorf("Expected 1 XHR request, got %d", len(xhrReqs))
	}

	rc.Clear()

	if rc.Count() != 0 {
		t.Errorf("Expected count to be 0 after clear, got %d", rc.Count())
	}
}

func TestEventRecorder(t *testing.T) {
	er := NewEventRecorder()

	event1 := &Event{Name: "xhr", Params: map[string]interface{}{"url": "test"}}
	event2 := &Event{Name: "fetch", Params: map[string]interface{}{"url": "test2"}}

	er.Record(event1)
	er.Record(event2)

	if er.Count() != 2 {
		t.Errorf("Expected count to be 2, got %d", er.Count())
	}

	xhrEvents := er.GetByName("xhr")
	if len(xhrEvents) != 1 {
		t.Errorf("Expected 1 XHR event, got %d", len(xhrEvents))
	}

	er.Clear()

	if er.Count() != 0 {
		t.Errorf("Expected count to be 0 after clear, got %d", er.Count())
	}
}

func TestURLPatternFilter(t *testing.T) {
	upf := NewURLPatternFilter()

	upf.AddPattern("example.com")
	upf.AddPattern("test")

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com/page", true},
		{"https://test.com/page", true},
		{"https://other.com/page", false},
	}

	for _, test := range tests {
		result := upf.ShouldAllow(test.url)
		if result != test.expected {
			t.Errorf("ShouldAllow(%q) = %v, expected %v", test.url, result, test.expected)
		}
	}
}
