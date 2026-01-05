package htcrawl

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
var numbers = "0123456789"
var symbols = "!#&^;.,?%$*"
var months = []string{"01", "02", "03", "04", "05", "06", "07", "08", "09", "10", "11", "12"}
var years = []string{"1982", "1989", "1990", "1994", "1995", "1996"}
var names = []string{"james", "john", "robert", "michael", "william", "david", "richard", "charles", "joseph", "thomas", "christopher", "daniel", "paul", "mark", "donald", "george", "kenneth"}
var surnames = []string{"anderson", "thomas", "jackson", "white", "harris", "martin", "thompson", "garcia", "martinez", "robinson", "clark", "rodriguez", "lewis", "lee", "walker", "hall"}
var domains = []string{".com", ".org", ".net", ".it", ".tv", ".de", ".fr"}

type RandomGenerator struct {
	seed    []int
	current int
}

func NewRandomGenerator(seed string) *RandomGenerator {
	rg := &RandomGenerator{}
	for _, c := range seed {
		rg.seed = append(rg.seed, int(c))
	}
	return rg
}

func (rg *RandomGenerator) rand(max int) int {
	if len(rg.seed) == 0 {
		return 0
	}
	val := rg.seed[rg.current] % max
	rg.current = (rg.current + 1) % len(rg.seed)
	return val
}

func (rg *RandomGenerator) randArr(arr []string) string {
	if len(arr) == 0 {
		return ""
	}
	return arr[rg.rand(len(arr))]
}

func (rg *RandomGenerator) randString(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = letters[rg.rand(len(letters))]
	}
	return string(result)
}

func GenerateRandomValues(seed string) map[string]string {
	rg := NewRandomGenerator(seed)
	values := make(map[string]string)

	values["string"] = rg.randString(8)
	values["number"] = rg.randString(3)
	values["month"] = rg.randArr(months)
	values["year"] = rg.randArr(years)
	values["date"] = rg.randArr(years) + "-" + rg.randArr(months) + "-" + rg.randArr(months)
	values["color"] = "#" + rg.randString(6)
	values["week"] = rg.randArr(years) + "-W" + rg.randArr(months[:6])
	values["time"] = rg.randArr(months) + ":" + rg.randArr(months)
	values["datetimeLocal"] = values["date"] + "T" + values["time"]
	values["domain"] = strings.ToLower(rg.randString(12)) + rg.randArr(domains)
	values["surname"] = rg.randArr(surnames)
	values["firstname"] = rg.randArr(names)
	values["email"] = values["firstname"] + "." + values["surname"] + "@" + values["domain"]
	values["url"] = "http://www." + values["domain"]
	values["humandate"] = rg.randArr(months) + "/" + rg.randArr(months) + "/" + rg.randArr(years)
	values["password"] = rg.randString(3) + rg.randArr([]string{symbols}) + rg.randString(2) + rg.randString(3) + rg.randString(2)
	values["lastname"] = values["surname"]
	values["tel"] = "+" + rg.randString(1) + " " + rg.randString(10)

	return values
}

func ParseCookiesFromHeaders(headers map[string][]string, targetURL string) []Cookie {
	var cookies []Cookie

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return cookies
	}

	domain := parsedURL.Hostname()

	for key, values := range headers {
		if strings.ToLower(key) == "set-cookie" {
			for _, cookieStr := range values {
				parts := strings.Split(cookieStr, ";")
				cookie := Cookie{
					Domain:   domain,
					Path:     "/",
					Secure:   false,
					HttpOnly: false,
				}

				for i, part := range parts {
					part = strings.TrimSpace(part)
					if i == 0 {
						kv := strings.SplitN(part, "=", 2)
						if len(kv) == 2 {
							cookie.Name = kv[0]
							cookie.Value = kv[1]
						}
						continue
					}

					kv := strings.SplitN(strings.ToLower(part), "=", 2)
					switch kv[0] {
					case "expires":
						if len(kv) > 1 {
							if t, err := time.Parse(time.RFC1123, kv[1]); err == nil {
								cookie.Expires = t.Unix()
							}
						}
					case "max-age":
						if len(kv) > 1 {
							if seconds, err := strconv.ParseInt(kv[1], 10, 64); err == nil {
								cookie.Expires = time.Now().Unix() + seconds
							}
						}
					case "domain":
						if len(kv) > 1 {
							cookie.Domain = kv[1]
						}
					case "path":
						if len(kv) > 1 {
							cookie.Path = kv[1]
						}
					case "httponly":
						cookie.HttpOnly = true
					case "secure":
						cookie.Secure = true
					}
				}

				if cookie.Expires == 0 {
					cookie.Expires = time.Now().Unix() + 60*60*24*365
				}

				cookies = append(cookies, cookie)
			}
		}
	}

	return cookies
}

func NormalizeURL(targetURL string) string {
	targetURL = strings.TrimSpace(targetURL)
	if len(targetURL) < 4 || !strings.HasPrefix(strings.ToLower(targetURL), "http") {
		return "http://" + targetURL
	}
	return targetURL
}

func MatchesExcludedURL(urlStr string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, urlStr)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func GetInputValueForName(name string, matches []InputMatch, values map[string]string) string {
	for _, match := range matches {
		matched, err := regexp.MatchString(match.Name, name)
		if err == nil && matched {
			if val, ok := values[match.Value]; ok {
				return val
			}
			return values["string"]
		}
	}
	return values["string"]
}

func CRC32Table() []uint32 {
	table := make([]uint32, 256)
	for i := range table {
		crc := uint32(i)
		for j := 0; j < 8; j++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xEDB88320
			} else {
				crc >>= 1
			}
		}
		table[i] = crc
	}
	return table
}

var crc32Table = CRC32Table()

func CRC32(str string) uint32 {
	crc := uint32(0xFFFFFFFF)
	for _, c := range str {
		crc = (crc >> 8) ^ crc32Table[(crc^uint32(c))&0xFF]
	}
	return (crc ^ 0xFFFFFFFF)
}

func HammingWeight(n uint32) uint32 {
	count := uint32(0)
	for n != 0 {
		n &= n - 1
		count++
	}
	return count
}

func ShingleW(arr []string, w int) []string {
	if len(arr) < w {
		return arr
	}
	result := make([]string, 0, len(arr)-w+1)
	for i := 0; i < len(arr)-w+1; i++ {
		result = append(result, strings.Join(arr[i:i+w], " "))
	}
	return result
}

func HashTokens(tokens []string) []uint32 {
	hashes := make([]uint32, len(tokens))
	for i, token := range tokens {
		hashes[i] = CRC32(token)
	}
	return hashes
}

func SimHash(arr []string) uint32 {
	features := HashTokens(ShingleW(arr, 2))
	hashVector := make([]int, 32)
	m := uint32(0x00000001)

	for _, hash := range features {
		for i := 0; i < 32; i++ {
			if hash&(m<<i) == 0 {
				hashVector[i]--
			} else {
				hashVector[i]++
			}
		}
	}

	var sh uint32
	for i := 0; i < 32; i++ {
		if hashVector[i] > 0 {
			sh |= m << i
		}
	}

	return sh
}

func Similarity(x, y uint32) float64 {
	and := x & y
	or := x | y
	return float64(HammingWeight(and)) / float64(HammingWeight(or))
}

func FormatTimestamp(ts int64) string {
	return time.Unix(ts/1000, (ts%1000)*1000000).Format("2006-01-02 15:04:05.000")
}

func StringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func Uint16ToBytes(n uint16) []byte {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, n)
	return buf
}

func BytesToUint16(buf []byte) uint16 {
	return binary.BigEndian.Uint16(buf)
}

func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func SanitizeURL(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}
	u.Fragment = ""
	return u.String()
}

func GetContentType(headers map[string][]string) string {
	if ct, ok := headers["Content-Type"]; ok && len(ct) > 0 {
		return strings.Split(ct[0], ";")[0]
	}
	return ""
}

func IsHTMLContentType(headers map[string][]string) bool {
	ct := GetContentType(headers)
	return strings.ToLower(ct) == "text/html"
}

func FormatRequest(r *Request) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s] %s %s", r.Type, r.Method, r.URL))
	if r.Data != "" {
		sb.WriteString(fmt.Sprintf("\nData: %s", r.Data))
	}
	if r.Trigger != nil {
		sb.WriteString(fmt.Sprintf("\nTrigger: %s on %s", r.Trigger.Event, r.Trigger.Element))
	}
	return sb.String()
}

func ParseTimeRange(duration string) (time.Duration, error) {
	return time.ParseDuration(duration)
}

func SafeString(s interface{}) string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("%v", s)
}

func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func RemoveDuplicateStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

func MergeStringMaps(m1, m2 map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range m1 {
		result[k] = v
	}
	for k, v := range m2 {
		result[k] = v
	}
	return result
}

func NewUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0F) | 0x40
	b[8] = (b[8] & 0x3F) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
