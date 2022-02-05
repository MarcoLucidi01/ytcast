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

type videoInfo struct {
	id        string
	startTime time.Duration
}

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
func extractVideoInfo(v string) videoInfo {
	v = strings.TrimSpace(v)
	// try simple query string first e.g. v=jNQXAC9IVRw&t=25
	q, _ := url.ParseQuery(v)
	info := videoInfo{
		id:        extractVideoId(v, q),
		startTime: extractStartTime(q),
	}
	if info.id == "" {
		// only later try parsing as url because for example
		// url.Parse("v=jNQXAC9IVRw&t=25") yields no error, but
		// extraction won't work.
		if u, err := url.Parse(v); err == nil {
			q := u.Query()
			info = videoInfo{
				id:        extractVideoId(u.Path, q),
				startTime: extractStartTime(q),
			}
		}
	}
	if info.id == "" {
		info = videoInfo{id: v} // assume v is already a videoId, probably won't work.
	}
	return info
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
	if d, err := time.ParseDuration(t); err == nil && d >= 0 {
		return d
	}
	return 0
}
