// TODO add logging
// TODO add package description and links
package youtube

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// YouTube Lounge API base url and endpoints.
	apiBase           = "https://www.youtube.com/api/lounge"
	apiGetLoungeToken = apiBase + "/pairing/get_lounge_token_batch"
	apiBind           = apiBase + "/bc/bind"

	// Origin header value for api requests.
	Origin = "https://www.youtube.com"
	// userAgent header value for api requests.
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.45 Safari/537.36"
)

// Remote holds session state and tokens for a "connected" youtube tv app and
// allows to play videos on it until Expiration.
type Remote struct {
	ScreenId    string // id of the screen (tv app) we are connected (or connecting) to.
	Name        string // "our" name displayed on tv app at connection time.
	Expiration  int64  // loungeToken expiration timestamp in milliseconds.
	id          string // our (client) id? we use a randString() at the moment, an uuid would be better.
	loungeToken string // token for lounge api requests.
	aId         int    // don't know what it is, we pass 5.
	rId         int    // request id? random id? we take a random integer and increment it at each request.
	sId         string // session id? we fetch it after the token.
	gsessionId  string // another session id? google session id? we fetch it along with sId.
	ofs         int    // don't know what it is, we start at 0 and increment it on each "setPlaylist" request.
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Connect connects to a screen (tv app) identified by screenId. name will be
// displayed on the screen at connection time. Returns a *Remote that can be
// used to play video on that screen.
func Connect(screenId, name string) (*Remote, error) {
	r := &Remote{
		ScreenId: screenId,
		Name:     name,
		id:       randString(32),
		aId:      5,
		rId:      rand.Intn(99999),
	}
	if err := r.getLoungeToken(); err != nil {
		return nil, err
	}
	if err := r.getSessionIds(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Remote) getLoungeToken() error {
	b := url.Values{}
	b.Set("screen_ids", r.ScreenId)
	respBody, err := doReq("POST", apiGetLoungeToken, nil, b)
	if err != nil {
		return err
	}

	r.loungeToken, r.Expiration, err = extractLoungeToken(respBody)
	if err != nil {
		return err
	}
	return nil
}

func extractLoungeToken(data []byte) (string, int64, error) {
	var v struct {
		Screens []struct {
			LoungeToken string `json:"loungeToken"`
			Expiration  int64  `json:"expiration"`
		} `json:"screens"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return "", 0, err
	}
	if len(v.Screens) == 0 {
		return "", 0, errors.New("screens array empty")
	}
	if len(v.Screens[0].LoungeToken) == 0 {
		return "", 0, errors.New("loungeToken empty")
	}
	return v.Screens[0].LoungeToken, v.Screens[0].Expiration, nil
}

func (r *Remote) getSessionIds() error {
	q := url.Values{}
	q.Set("device", "REMOTE_CONTROL")
	q.Set("mdx-version", "3")
	q.Set("ui", "1")
	q.Set("v", "2")
	q.Set("VER", "8")
	q.Set("CVER", "1")
	q.Set("app", "youtube-desktop")
	q.Set("name", r.Name)
	q.Set("loungeIdToken", r.loungeToken)
	q.Set("id", r.id)
	q.Set("zx", randString(12))
	q.Set("RID", strconv.Itoa(r.nextRId()))
	b := url.Values{}
	b.Set("count", "0")
	respBody, err := doReq("POST", apiBind, q, b)
	if err != nil {
		return err
	}

	r.sId, r.gsessionId, err = extractSessionIds(respBody)
	if err != nil {
		return err
	}
	return nil
}

func extractSessionIds(data []byte) (string, string, error) {
	// first thing we get is a number that we can safely skip (I think it's
	// payload length).
	for i, c := range data {
		if c == '[' {
			data = data[i:]
			break
		}
	}

	// next we have a bunch of json arrays containing mixed type values,
	// that's why interface{}.
	var v [][]interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return "", "", err
	}
	sId := ""
	gsessionId := ""
	for _, a1 := range v {
		if len(a1) < 2 {
			continue
		}
		a2, ok := a1[1].([]interface{})
		if !ok || len(a2) < 2 {
			continue
		}

		var key, value string
		if key, ok = a2[0].(string); !ok {
			continue
		}
		if value, ok = a2[1].(string); !ok {
			continue
		}
		switch key {
		case "c":
			sId = value
		case "S":
			gsessionId = value
		}

		if len(sId) > 0 && len(gsessionId) > 0 {
			return sId, gsessionId, nil
		}
	}
	return "", "", errors.New("unable to extract session ids")
}

func (r *Remote) Play(videoIds ...string) error {
	q := url.Values{}
	q.Set("device", "REMOTE_CONTROL")
	q.Set("VER", "8")
	q.Set("loungeIdToken", r.loungeToken)
	q.Set("id", r.id)
	q.Set("zx", randString(12))
	q.Set("SID", r.sId)
	q.Set("gsessionid", r.gsessionId)
	q.Set("AID", strconv.Itoa(r.aId))
	q.Set("RID", strconv.Itoa(r.nextRId()))
	b := url.Values{}
	b.Set("count", "1")
	b.Set("req0__sc", "setPlaylist")
	b.Set("req0_videoId", videoIds[0])                  // play first video
	b.Set("req0_videoIds", strings.Join(videoIds, ",")) // enqueue first and the others
	b.Set("ofs", strconv.Itoa(r.nextOfs()))
	_, err := doReq("POST", apiBind, q, b)
	return err
}

func doReq(method, url string, query, body url.Values) ([]byte, error) {
	req, err := http.NewRequest(method, url, strings.NewReader(body.Encode()))
	if err != nil {
		return nil, err
	}
	if len(query) > 0 {
		req.URL.RawQuery = query.Encode()
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("Origin", Origin)
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err == nil && resp.StatusCode != 200 {
		err = errors.New(resp.Status)
	}
	return respBody, err
}

func (r *Remote) nextRId() int {
	rId := r.rId
	r.rId++
	return rId
}

func (r *Remote) nextOfs() int {
	ofs := r.ofs
	r.ofs++
	return ofs
}

func randString(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxdzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

	buf := make([]byte, n)
	for i := range buf {
		buf[i] = chars[rand.Intn(len(chars))]
	}
	return string(buf)
}
