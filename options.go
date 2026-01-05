package htcrawl

type Options struct {
	Verbose               bool
	CheckAjax             bool
	FillValues            bool
	TriggerEvents         bool
	CheckWebsockets       bool
	SearchUrls            bool
	JsonOutput            bool
	MaxExecTime           int
	AjaxTimeout           int
	PrintAjaxPostData     bool
	LoadImages            bool
	GetCookies            bool
	MapEvents             bool
	CheckScriptInsertion  bool
	CheckFetch            bool
	HttpAuth              []string
	TriggerAllMappedEvents bool
	OutputMappedEvents    bool
	OverrideTimeoutFunctions bool
	Referer               string
	UserAgent             string
	AllEvents             []string
	MouseEvents           []string
	KeyboardEvents        []string
	SetCookies            []Cookie
	ExcludedUrls          []string
	MaximumRecursion      int
	MaximumAjaxChain      int
	RandomSeed            string
	InputNameMatchValue   []InputMatch
	EventsMap             map[string][]string
	Proxy                 string
	LoadWithPost          bool
	PostData              string
	HeadlessChrome        bool
	ExtraHeaders          map[string]string
	OpenChromeDevtools    bool
	ExceptionOnRedirect   bool
	NavigationTimeout    int
	BypassCSP             bool
	SimulateRealEvents    bool
	CrawlMode             string
	BrowserLocalstorage   []LocalstorageItem
	SkipDuplicateContent  bool
	WindowSize            []int
	ShowUI                bool
	CustomUI              *CustomUI
	OverridePostMessage   bool
	IncludeAllOrigins     bool
}

type Cookie struct {
	Name     string
	Value    string
	Domain   string
	Path     string
	Expires  int64
	HttpOnly bool
	Secure   bool
	URL      string
}

type InputMatch struct {
	Name  string
	Value string
}

type LocalstorageItem struct {
	Key   string
	Value string
}

type CustomUI struct {
	ExtensionPath string
}

type Trigger struct {
	Element string
	Event   string
}

type Request struct {
	Type         string
	Method       string
	URL          string
	Data         string
	Trigger      *Trigger
	ExtraHeaders map[string]string
	Timestamp    int64
}

func DefaultOptions() *Options {
	return &Options{
		Verbose:               false,
		CheckAjax:             true,
		FillValues:            true,
		TriggerEvents:         true,
		CheckWebsockets:       true,
		SearchUrls:            true,
		JsonOutput:            true,
		MaxExecTime:           300000,
		AjaxTimeout:           3000,
		PrintAjaxPostData:     true,
		LoadImages:            false,
		GetCookies:            true,
		MapEvents:             true,
		CheckScriptInsertion:  true,
		CheckFetch:            true,
		HttpAuth:              nil,
		TriggerAllMappedEvents: true,
		OutputMappedEvents:    false,
		OverrideTimeoutFunctions: false,
		Referer:               "",
		UserAgent:             "",
		AllEvents: []string{
			"abort", "autocomplete", "autocompleteerror", "beforecopy", "beforecut", "beforepaste",
			"blur", "cancel", "canplay", "canplaythrough", "change", "close", "contextmenu",
			"copy", "cuechange", "cut", "dblclick", "drag", "dragend", "dragenter", "dragleave",
			"dragover", "dragstart", "drop", "durationchange", "emptied", "ended", "error",
			"focus", "input", "invalid", "keydown", "keypress", "keyup", "load", "loadeddata",
			"loadedmetadata", "loadstart", "mousedown", "mouseenter", "mouseleave", "mousemove",
			"mouseout", "mouseover", "mouseup", "mousewheel", "paste", "pause", "play", "playing",
			"progress", "ratechange", "reset", "resize", "scroll", "search", "seeked", "seeking",
			"select", "selectstart", "show", "stalled", "submit", "suspend", "timeupdate",
			"toggle", "volumechange", "waiting", "webkitfullscreenchange", "webkitfullscreenerror",
			"wheel",
		},
		MouseEvents:    []string{"click"},
		KeyboardEvents: []string{},
		SetCookies:     []Cookie{},
		ExcludedUrls:   []string{},
		MaximumRecursion: 15,
		MaximumAjaxChain: 30,
		RandomSeed:     "IsHOulDb34RaNd0MsTR1ngbUt1mN0t",
		InputNameMatchValue: []InputMatch{
			{Name: "mail", Value: "email"},
			{Name: "((number)|(phone))|(^tel)", Value: "number"},
			{Name: "(date)|(birth)", Value: "humandate"},
			{Name: "((month)|(day))|(^mon$)", Value: "month"},
			{Name: "year", Value: "year"},
			{Name: "url", Value: "url"},
			{Name: "firstname", Value: "firstname"},
			{Name: "(surname)|(lastname)", Value: "surname"},
		},
		EventsMap: map[string][]string{
			"button":  {"click", "dblclick", "keydown", "keyup", "mouseup", "mousedown"},
			"select":  {"change", "click", "dblclick", "keydown", "keyup", "mouseup", "mousedown"},
			"input":   {"change", "click", "dblclick", "blur", "focus", "keydown", "keyup", "mouseup", "mousedown"},
			"a":       {"click", "dblclick", "keydown", "keyup", "mouseup", "mousedown"},
			"textarea": {"change", "click", "dblclick", "blur", "focus", "keydown", "keyup", "mouseup", "mousedown"},
			"span":    {"click", "dblclick", "mouseup", "mousedown"},
			"td":      {"click", "dblclick", "mouseup", "mousedown"},
			"tr":      {"click", "dblclick", "mouseup", "mousedown"},
			"div":     {"click", "dblclick", "mouseup", "mousedown"},
		},
		Proxy:                 "",
		LoadWithPost:          false,
		PostData:              "",
		HeadlessChrome:        true,
		ExtraHeaders:          map[string]string{},
		OpenChromeDevtools:    false,
		ExceptionOnRedirect:   false,
		NavigationTimeout:     20000,
		BypassCSP:             true,
		SimulateRealEvents:    true,
		CrawlMode:             "linear",
		BrowserLocalstorage:   []LocalstorageItem{},
		SkipDuplicateContent:  false,
		WindowSize:            []int{1600, 1000},
		ShowUI:                false,
		CustomUI:              nil,
		OverridePostMessage:   false,
		IncludeAllOrigins:     false,
	}
}

func (r *Request) Key() string {
	triggerKey := ""
	if r.Trigger != nil {
		triggerKey = r.Trigger.Element + r.Trigger.Event
	}
	return r.Type + r.Method + r.URL + r.Data + triggerKey
}
