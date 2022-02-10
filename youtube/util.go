// See license file for copyright and license details.

package youtube

import (
	"encoding/xml"
	"fmt"
	"math/rand"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var (
	// taken from this awesome answer https://webapps.stackexchange.com/a/101153
	videoIdRe = regexp.MustCompile(`^[0-9A-Za-z_-]{10}[048AEIMQUYcgkosw]$`)
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randDelay(min, max time.Duration) {
	time.Sleep(min + time.Duration(rand.Int63n(int64(max-min))))
}

// ExtractScreenId extracts the screen id of a YouTube TV app from the xml tag
// <additionalData> fetched with a GET request on the Application-URL (see DIAL
// protocol and dial.GetAppInfo()).
func ExtractScreenId(data string) (string, error) {
	// TODO dial.AppInfo.Additional.Data it's not wrapped in a root element,
	// I add a dummy root here but I think data should already be wrapped in
	// a root element.
	data = fmt.Sprintf("<dummy>%s</dummy>", data)
	var v struct {
		ScreenId string `xml:"screenId"`
	}
	if err := xml.Unmarshal([]byte(data), &v); err != nil {
		return "", err
	}
	return strings.TrimSpace(v.ScreenId), nil
}

// extractVideoInfo extracts video information from a video url or query string
// (see util_test.go for examples). It's not very smart.
func extractVideoInfo(v string) (string, time.Duration) {
	v = strings.ReplaceAll(strings.TrimSpace(v), "?", "&") // treat urls as query strings.
	q, _ := url.ParseQuery(v)
	id := extractVideoId(v, q)
	if id == "" {
		return v, 0 // assume v is a videoId we weren't able to extract.
	}
	return id, extractStartTime(q)
}

func extractVideoId(p string, q url.Values) string {
	if id := q.Get("v"); id != "" {
		return id
	}
	bp := path.Base(p)
	if i := strings.IndexRune(bp, '&'); i > -1 {
		// strip "invalid" query parameters from base path, e.g.
		// https://youtu.be/jNQXAC9IVRw&feature=channel
		// this also works for query strings like jNQXAC9IVRw&t=25
		bp = bp[:i]
	}
	// YouTube makes no guarantee on videoId format (see https://webapps.stackexchange.com/questions/54443)
	// we use the regex only when we can't find it in the query parameters.
	if videoIdRe.MatchString(bp) {
		return bp
	}
	return ""
}

func extractStartTime(q url.Values) time.Duration {
	t := q.Get("t")
	if t == "" {
		return 0
	}
	if unicode.IsDigit(rune(t[len(t)-1])) {
		t += "s"
	}
	if d, err := time.ParseDuration(t); err == nil && d > 0 {
		return d
	}
	return 0
}

func removeSpaces(s string) string {
	m := func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}
	return strings.Map(m, s)
}
