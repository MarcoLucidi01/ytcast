package youtube

import (
	"strconv"
	"testing"
)

func TestConnectAndPlay(t *testing.T) {
	screenId := "" // put your screenId here
	videoIds := []string{"dQw4w9WgXcQ", "7BqJ8dzygtU", "EY6q5dv_B-o"}

	if len(screenId) == 0 {
		t.SkipNow()
	}
	r, err := Connect(screenId, "ytcast")
	failIfNotNil(t, err)
	err = r.Play(videoIds...)
	failIfNotNil(t, err)
}

func TestExtractLoungeToken(t *testing.T) {
	loungeToken := "lounge-token-foo-bar-baz"
	expiration := int64(1637512182177)
	data := []byte(`
{
  "screens": [
    {
      "screenId": "screen-id-foo-bar-baz",
      "refreshIntervalInMillis": 1123200000,
      "remoteRefreshIntervalMs": 79200000,
      "refreshIntervalMs": 1123200000,
      "loungeTokenLifespanMs": 1209600000,
      "loungeToken": "` + loungeToken + `",
      "remoteRefreshIntervalInMillis": 79200000,
      "expiration": ` + strconv.FormatInt(expiration, 10) + `
    }
  ]
}`)
	tok, exp, err := extractLoungeToken(data)
	failIfNotNil(t, err)
	failIfNotEqualS(t, "loungeToken", loungeToken, tok)
	failIfNotEqualI64(t, "expiration", expiration, exp)
}

func TestExtractSessionIds(t *testing.T) {
	sId := "sid-foo-bar-baz"
	gsessionId := "gsessionid-foo-bar-baz"
	data := []byte(`
270
[[0,["c","` + sId + `","",8]]
,[1,["S","` + gsessionId + `"]]
,[2,["loungeStatus",{}]]
,[3,["playlistModified",{}]]
,[4,["onAutoplayModeChanged",{"autoplayMode":"UNSUPPORTED"}]]
,[5,["onPlaylistModeChanged",{"shuffleEnabled":"false","loopEnabled":"false"}]]
]`)
	gotSId, gotGsessionId, err := extractSessionIds(data)
	failIfNotNil(t, err)
	failIfNotEqualS(t, "sId", sId, gotSId)
	failIfNotEqualS(t, "gsessionId", gsessionId, gotGsessionId)
}

func failIfNotNil(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("got unexpected error: %s", err)
	}
}

func failIfNotEqualS(t *testing.T, prefix, want, got string) {
	if want != got {
		t.Fatalf("%s: want %q got %q", prefix, want, got)
	}
}

func failIfNotEqualI64(t *testing.T, prefix string, want, got int64) {
	if want != got {
		t.Fatalf("%s: want %d got %d", prefix, want, got)
	}
}
