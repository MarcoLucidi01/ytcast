// See license file for copyright and license details.

// Package youtube implements a minimal client for the YouTube Lounge API which
// allows to connect and play videos on a remote "screen" (YouTube on TV app).
// The API is not public so this code CAN BREAK AT ANY TIME.
//
// The implementation derives from the work of various people I found on the web
// that saved me hours of reverse engineering. I'd like to list and thank them
// here:
//   https://0x41.cf/automation/2021/03/02/google-assistant-youtube-smart-tvs.html
//   https://github.com/thedroidgeek/youtube-cast-automation-api
//   https://github.com/mutantmonkey/youtube-remote
//   https://bugs.xdavidhu.me/google/2021/04/05/i-built-a-tv-that-plays-all-of-your-private-youtube-videos
//   https://github.com/aykevl/plaincast
//   https://github.com/ur1katz/casttube
package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	apiBase           = "https://www.youtube.com/api/lounge"
	apiGetLoungeToken = apiBase + "/pairing/get_lounge_token_batch"
	apiBind           = apiBase + "/bc/bind"

	paramApp              = "youtube-desktop"
	paramCver             = "1"
	paramDevice           = "REMOTE_CONTROL"
	paramId               = "remote"
	paramRidGetSessionIds = "1"
	paramRidPlay          = "2"
	paramVer              = "8"

	reqMinDelay = 2 * time.Second
	reqMaxDelay = reqMinDelay + 3*time.Second

	contentType = "application/x-www-form-urlencoded"
	userAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.45 Safari/537.36"

	// Origin header value for HTTP requests to YouTube services.
	Origin = "https://www.youtube.com"

	// YouTube application name registered in the DIAL register.
	// See http://www.dial-multiscreen.org/dial-registry/namespace-database
	DialAppName = "YouTube"
)

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}

	errBadHttpStatus = errors.New("bad HTTP response status")
	errNoScreens     = errors.New("missing screens array")
	errNoToken       = errors.New("missing loungeToken")
	errNoSessionIds  = errors.New("missing session ids")
)

// Remote holds Lounge session tokens of a connected screen (tv app) and allows
// to play videos on it until Expiration.
type Remote struct {
	ScreenId    string // id of the screen (tv app) we are connected (or connecting) to.
	Name        string // name displayed on the screen at connection time.
	LoungeToken string // token for Lounge API requests.
	Expiration  int64  // LoungeToken expiration timestamp in milliseconds.
	SId         string // session id? it can expire very often so we fetch it at each Play() or Add().
	GSessionId  string // another session id? google session id? we fetch it along with SId.
}

// Connect connects to a screen (tv app) identified by screenId through the
// Lounge API. name will be displayed on the screen at connection time. Returns
// a Remote that can be used to play video on that screen.
func Connect(screenId, name string) (*Remote, error) {
	r := &Remote{ScreenId: screenId, Name: name}
	if err := r.RefreshToken(); err != nil {
		return nil, fmt.Errorf("RefreshToken: %w", err)
	}
	return r, nil
}

// RefreshToken gets a new LoungeToken for the screenId. Should be used when the
// token has Expired().
func (r *Remote) RefreshToken() error {
	b := url.Values{}
	b.Set("screen_ids", r.ScreenId)
	respBody, err := doReq("POST", apiGetLoungeToken, nil, b)
	if err != nil {
		return err
	}
	tok, exp, err := extractLoungeToken(respBody)
	if err != nil {
		return err
	}
	r.LoungeToken, r.Expiration = tok, exp
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
		return "", 0, errNoScreens
	}
	if len(v.Screens[0].LoungeToken) == 0 {
		return "", 0, errNoToken
	}
	return v.Screens[0].LoungeToken, v.Screens[0].Expiration, nil
}

// Expired returns true if the LoungeToken has expired.
func (r *Remote) Expired() bool {
	exp := time.Unix(0, r.Expiration*int64(time.Millisecond))
	return time.Now().After(exp)
}

func (r *Remote) getSessionIds() error {
	q := url.Values{}
	q.Set("CVER", paramCver)
	q.Set("RID", paramRidGetSessionIds)
	q.Set("VER", paramVer)
	q.Set("app", paramApp)
	q.Set("device", paramDevice)
	q.Set("id", paramId)
	q.Set("loungeIdToken", r.LoungeToken)
	q.Set("name", r.Name)
	respBody, err := doReq("POST", apiBind, q, nil)
	if err != nil {
		return err
	}
	sId, gSessionId, err := extractSessionIds(respBody)
	if err != nil {
		return err
	}
	r.SId, r.GSessionId = sId, gSessionId
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
	// that's why interface{}. See remote_test.go for an example.
	var v [][]interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return "", "", err
	}
	var sId string
	var gsessionId string
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
		default:
			continue
		}

		if sId != "" && gsessionId != "" {
			return sId, gsessionId, nil
		}
	}
	return "", "", errNoSessionIds
}

// Play requests the Lounge API to play immediately the first video on the
// tv app and to enqueue the others. Accepts both video urls and video ids.
func (r *Remote) Play(videos []string) error {
	if len(videos) == 0 {
		return nil
	}
	if err := r.getSessionIds(); err != nil {
		return fmt.Errorf("getSessionIds: %w", err)
	}
	q := url.Values{}
	q.Set("CVER", paramCver)
	q.Set("RID", paramRidPlay)
	q.Set("SID", r.SId)
	q.Set("VER", paramVer)
	q.Set("gsessionid", r.GSessionId)
	q.Set("loungeIdToken", r.LoungeToken)
	b := url.Values{}
	b.Set("count", "1")
	b.Set("req0__sc", "setPlaylist")
	// start time can be set only for the first video.
	first := extractVideoInfo(videos[0])
	b.Set("req0_videoId", first.id)
	b.Set("req0_currentTime", strconv.FormatInt(int64(first.startTime.Seconds()), 10))
	b.Set("req0_currentIndex", "0")
	var videoIds []string
	for _, v := range videos {
		videoIds = append(videoIds, extractVideoInfo(v).id)
	}
	b.Set("req0_videoIds", strings.Join(videoIds, ","))
	_, err := doReq("POST", apiBind, q, b)
	return err
}

// Add requests the Lounge API to add videos to the queue without changing
// what's currently playing on the tv app. Accepts both video urls and video ids.
func (r *Remote) Add(videos []string) error {
	if len(videos) == 0 {
		return nil
	}
	if err := r.getSessionIds(); err != nil {
		return fmt.Errorf("getSessionIds: %w", err)
	}
	q := url.Values{}
	q.Set("CVER", paramCver)
	q.Set("RID", paramRidPlay)
	q.Set("SID", r.SId)
	q.Set("VER", paramVer)
	q.Set("gsessionid", r.GSessionId)
	q.Set("loungeIdToken", r.LoungeToken)
	for i, v := range videos {
		// addVideo doesn't have reqX_videoIds parameter so we send a
		// request for each video, but without this random delay the
		// queue may get messed up and some video may get "lost". also,
		// each reqX_ needs to have its own index for the same reason.
		randDelay(reqMinDelay, reqMaxDelay)
		b := url.Values{}
		b.Set("count", "1")
		b.Set(fmt.Sprintf("req%d__sc", i), "addVideo")
		b.Set(fmt.Sprintf("req%d_videoId", i), extractVideoInfo(v).id)
		if _, err := doReq("POST", apiBind, q, b); err != nil {
			return err
		}
	}
	return nil
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
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Origin", Origin) // doesn't hurt
	req.Header.Set("User-Agent", userAgent)

	log.Printf("%s %s", method, url)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err == nil && resp.StatusCode != 200 {
		err = fmt.Errorf("%s %s: %s: %w", method, url, resp.Status, errBadHttpStatus)
	}
	return respBody, err
}
