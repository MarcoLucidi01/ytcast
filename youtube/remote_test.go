// See license file for copyright and license details.

package youtube

import (
	"testing"
	"time"
)

func connectOrSkip(t *testing.T, name, screenId string) *Remote {
	if screenId == "" {
		t.SkipNow()
	}
	r, err := Connect("", screenId, name)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	return r
}

func TestPlay(t *testing.T) {
	r := connectOrSkip(t, "TestPlay", "") // put your screenId here
	if err := r.Play([]string{"dQw4w9WgXcQ", "7BqJ8dzygtU", "EY6q5dv_B-o"}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestPlayAndAdd(t *testing.T) {
	r := connectOrSkip(t, "TestPlayAndAdd", "") // put your screenId here
	if err := r.Play([]string{"Opqgwn8TdlM", "0MLaYe3y0BU"}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := r.Add([]string{"RzWB5jL5RX0", "fPU7Uq4TtNU"}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := r.Add([]string{"BK5x7IUTIyU"}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := r.Add([]string{"ci1PJexnfNE"}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestPlayFromTimestamp(t *testing.T) {
	r := connectOrSkip(t, "TestPlayFromTimestamp", "") // put your screenId here
	if err := r.Play([]string{"OgO1gpXSUzU&t=363"}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	time.Sleep(5 * time.Second)
	if err := r.Play([]string{"0JUN9aDxVmI&t=10m"}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestConnectWithCode(t *testing.T) {
	code := "" // put your TV code here
	if code == "" {
		t.SkipNow()
	}
	r, err := ConnectWithCode("", code, "TestConnectWithCode")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := r.Play([]string{"w3Wluvzoggg"}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestExtractScreenInfo(t *testing.T) {
	tests := []struct {
		data        []byte
		screenId    string
		loungeToken string
		expiration  int64
		deviceId    string
		screenName  string
	}{
		{
			data: []byte(`
{
  "screen": {
    "accessType": "permanent",
    "screenId": "screen-id-foo-bar-baz",
    "dialAdditionalDataSupportLevel": "unknown",
    "loungeTokenRefreshIntervalMs": 1123200000,
    "loungeToken": "lounge-token-foo-bar-baz",
    "clientName": "tvhtml5",
    "name": "YouTube on TV",
    "expiration": 1645614559007,
    "deviceId": "device-id-foo-bar-baz"
  }
}`),
			screenId:    "screen-id-foo-bar-baz",
			loungeToken: "lounge-token-foo-bar-baz",
			expiration:  int64(1645614559007),
			deviceId:    "device-id-foo-bar-baz",
			screenName:  "YouTube on TV",
		},
	}

	for i, test := range tests {
		screenId, loungeToken, expiration, deviceId, screenName, err := extractScreenInfo(test.data)
		if err != nil {
			t.Fatalf("tests[%d]: unexpected error: %s", i, err)
		}
		if test.screenId != screenId {
			t.Fatalf("tests[%d]: screenId: want %q got %q", i, test.screenId, screenId)
		}
		if test.loungeToken != loungeToken {
			t.Fatalf("tests[%d]: loungeToken: want %q got %q", i, test.loungeToken, loungeToken)
		}
		if test.expiration != expiration {
			t.Fatalf("tests[%d]: expiration: want %q got %q", i, test.expiration, expiration)
		}
		if test.deviceId != deviceId {
			t.Fatalf("tests[%d]: deviceId: want %q got %q", i, test.deviceId, deviceId)
		}
		if test.screenName != screenName {
			t.Fatalf("tests[%d]: screenName: want %q got %q", i, test.screenName, screenName)
		}
	}
}

func TestExtractLoungeToken(t *testing.T) {
	tests := []struct {
		data        []byte
		loungeToken string
		expiration  int64
	}{
		{
			data: []byte(`
{
  "screens": [
    {
      "screenId": "screen-id-foo-bar-baz",
      "refreshIntervalInMillis": 1123200000,
      "remoteRefreshIntervalMs": 79200000,
      "refreshIntervalMs": 1123200000,
      "loungeTokenLifespanMs": 1209600000,
      "loungeToken": "lounge-token-foo-bar-baz",
      "remoteRefreshIntervalInMillis": 79200000,
      "expiration": 1637512182177
    }
  ]
}`),
			loungeToken: "lounge-token-foo-bar-baz",
			expiration:  int64(1637512182177),
		},
	}

	for i, test := range tests {
		loungeToken, expiration, err := extractLoungeToken(test.data)
		if err != nil {
			t.Fatalf("tests[%d]: unexpected error: %s", i, err)
		}
		if test.loungeToken != loungeToken {
			t.Fatalf("tests[%d]: loungeToken: want %q got %q", i, test.loungeToken, loungeToken)
		}
		if test.expiration != expiration {
			t.Fatalf("tests[%d]: expiration: want %d got %d", i, test.expiration, expiration)
		}
	}
}

func TestExtractSessionIds(t *testing.T) {
	tests := []struct {
		data       []byte
		sId        string
		gSessionId string
	}{
		{
			data: []byte(`
270
[[0,["c","sid-foo-bar-baz","",8]]
,[1,["S","gsessionid-foo-bar-baz"]]
,[2,["loungeStatus",{}]]
,[3,["playlistModified",{}]]
,[4,["onAutoplayModeChanged",{"autoplayMode":"UNSUPPORTED"}]]
,[5,["onPlaylistModeChanged",{"shuffleEnabled":"false","loopEnabled":"false"}]]
]`),
			sId:        "sid-foo-bar-baz",
			gSessionId: "gsessionid-foo-bar-baz",
		},
	}

	for i, test := range tests {
		sId, gSessionId, err := extractSessionIds(test.data)
		if err != nil {
			t.Fatalf("tests[%d]: unexpected error: %s", i, err)
		}
		if test.sId != sId {
			t.Fatalf("tests[%d]: sId: want %q got %q", i, test.sId, sId)
		}
		if test.gSessionId != gSessionId {
			t.Fatalf("tests[%d]: gSessionId: want %q got %q", i, test.gSessionId, gSessionId)
		}
	}
}
