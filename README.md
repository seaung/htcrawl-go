# htcrawl-go

htcrawl 的 Go 实现 - 一个使用无头 Chrome 的强大单页应用（SPA）爬虫。此版本使用 [go-rod](https://github.com/go-rod/rod) 库进行浏览器自动化。

## 功能特性

- **递归爬取**：触发元素上附加的所有事件，等待请求，检测 DOM 变化
- **启发式内容去重**：避免爬取重复内容
- **请求拦截**：拦截所有请求，包括 websocket、JSONP 和表单
- **PostMessage 拦截**：监控 postMessage 通信
- **Iframe 处理**：透明地将 iframe 作为同一 DOM 的一部分进行爬取
- **自定义选择器**：使用自定义 CSS 选择器选择 iframe 内的元素
- **事件驱动架构**：注册爬取过程中各种事件的回调

## 安装

```bash
go get github.com/seaung/htcrawl-go
```

## 基本用法

```go
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
    
    // 打印 ajax 调用的 URL
    crawler.On("xhr", func(event *htcrawl.Event, crawler *htcrawl.Crawler) (interface{}, error) {
        if req, ok := event.Params["request"].(*htcrawl.Request); ok {
            fmt.Printf("XHR to %s\n", req.URL)
        }
        return nil, nil
    })
    
    // 打印新创建的DOM元素的选择器
    crawler.On("newdom", func(event *htcrawl.Event, crawler *htcrawl.Crawler) (interface{}, error) {
        if element, ok := event.Params["element"].(string); ok {
            fmt.Printf("New DOM element created: %s\n", element)
        }
        return nil, nil
    })
    
    // 打印爬虫触发的所有事件
    crawler.On("triggerevent", func(event *htcrawl.Event, crawler *htcrawl.Crawler) (interface{}, error) {
        if element, ok := event.Params["element"].(string); ok {
            if eventType, ok := event.Params["event"].(string); ok {
                fmt.Printf("Triggered %s on '%s'\n", eventType, element)
            }
        }
        return nil, nil
    })
    
    // 开始爬取！
    if err := crawler.Start(); err != nil {
        log.Fatalf("Failed to start crawling: %v", err)
    }
    
    fmt.Println("Crawling completed!")
}
```

## 配置选项

爬虫可以通过各种选项进行自定义：

```go
options := htcrawl.DefaultOptions()

// 基本选项
options.Verbose = true               // 启用详细日志
options.HeadlessChrome = true        // 在无头模式下运行 Chrome
options.UserAgent = "Custom UA"      // 设置自定义用户代理
options.Proxy = "http://proxy:8080"  // 设置代理服务器

// 爬取行为
options.MaximumRecursion = 15         // 最大递归深度
options.MaximumAjaxChain = 30         // 最大 AJAX 链长度
options.AjaxTimeout = 3000           // AJAX 超时时间（毫秒）
options.NavigationTimeout = 20000     // 导航超时时间（毫秒）

// 事件处理
options.TriggerEvents = true         // 触发元素上的事件
options.CheckAjax = true             // 拦截 AJAX 请求
options.CheckFetch = true            // 拦截 fetch 请求
options.CheckWebsockets = true       // 拦截 WebSocket 连接
options.CheckScriptInsertion = true  // 监控脚本插入

// 内容处理
options.FillValues = true            // 用随机值填充输入字段
options.SkipDuplicateContent = false // 跳过重复内容
options.LoadImages = false           // 爬取时加载图片

// 安全
options.BypassCSP = true             // 绕过内容安全策略
options.OverridePostMessage = false  // 覆盖 postMessage
```

## 事件

您可以注册以下事件的回调：

- `start`: 爬取开始
- `xhr`: XHR 请求已发起
- `xhrcompleted`: XHR 请求已完成
- `fetch`: Fetch 请求已发起
- `fetchcompleted`: Fetch 请求已完成
- `jsonp`: JSONP 请求
- `jsonpcompleted`: JSONP 请求已完成
- `websocket`: WebSocket 连接
- `websocketmessage`: 收到 WebSocket 消息
- `websocketsend`: 发送 WebSocket 消息
- `formsubmit`: 表单已提交
- `fillinput`: 输入字段已填充
- `newdom`: 新 DOM 元素已创建
- `navigation`: 导航已发生
- `domcontentloaded`: DOM 内容已加载
- `redirect`: 重定向已发生
- `triggerevent`: 元素上触发的事件
- `postmessage`: 收到 PostMessage
- `pageinitialized`: 页面已初始化

## 示例

### 高级内容抓取器

```go
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
```

### DOM XSS 扫描器

```go
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
```

## 运行测试

```bash
cd htcrawl-go
go test -v
```

## 项目结构

```
htcrawl-go/
├── options.go              # 配置选项
├── utils.go                # 工具函数
├── domdeduplicator.go      # DOM 去重
├── probe.js                # JavaScript 探针脚本
├── crawler.go              # 主要爬虫实现
├── events.go               # 事件处理和工具
├── htcrawl_test.go         # 单元测试
├── examples/
│   ├── basic/
│   │   └── main.go         # 基本用法示例
│   ├── content-scraper/
│   │   └── main.go         # 内容抓取示例
│   └── dom-xss-scanner/
│       └── main.go         # DOM XSS 扫描器示例
└── README.md               # 本文件
```

## 许可证

本程序是自由软件；您可以根据自由软件基金会发布的 GNU 通用公共许可证版本 2 或（根据您的选择）任何更高版本重新分发和/或修改它。

## 致谢

这是Filippo Cavallarin的原始 [htcrawl](https://github.com/fcavallarin/htcrawl) 项目的Go移植版本。

## 贡献

欢迎贡献！请随时提交Pull Request。
