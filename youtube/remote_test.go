// See license file for copyright and license details.

package youtube

import (
	"testing"
)

func TestConnectAndPlay(t *testing.T) {
	screenId := "" // put your screenId here
	videoIds := []string{"dQw4w9WgXcQ", "7BqJ8dzygtU", "EY6q5dv_B-o"}

	if screenId == "" {
		t.SkipNow()
	}
	r, err := Connect(screenId, "TestConnectAndPlay")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := r.Play(videoIds); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestConnectAndPlayAndAdd(t *testing.T) {
	screenId := "" // put your screenId here
	play1 := []string{"Opqgwn8TdlM", "0MLaYe3y0BU"}
	add1 := []string{"RzWB5jL5RX0", "fPU7Uq4TtNU"}
	add2 := []string{"BK5x7IUTIyU"}
	add3 := []string{"ci1PJexnfNE"}

	if screenId == "" {
		t.SkipNow()
	}
	r, err := Connect(screenId, "TestConnectAndPlayAndAdd")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := r.Play(play1); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := r.Add(add1); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := r.Add(add2); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := r.Add(add3); err != nil {
		t.Fatalf("unexpected error: %s", err)
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
