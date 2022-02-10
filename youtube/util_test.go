// See license file for copyright and license details.

package youtube

import (
	"testing"
	"time"
)

func TestExtractScreenId(t *testing.T) {
	tests := []string{
		"<screenId>foo-bar-baz</screenId>",
	}

	const want = "foo-bar-baz"
	for i, test := range tests {
		screenId, err := ExtractScreenId(test)
		if err != nil {
			t.Fatalf("tests[%d]: unexpected error: %q", i, err)
		}
		if screenId != want {
			t.Fatalf("tests[%d]: screenId: want %q got %q", i, want, screenId)
		}
	}
}

func TestExtractVideoInfo(t *testing.T) {
	// most examples are from https://gist.github.com/rodrigoborgesdeoliveira/987683cfbfcc8d800192da1e73adc486
	tests := []struct {
		u         string
		id        string
		startTime time.Duration
	}{
		{u: "jNQXAC9IVRw", id: "jNQXAC9IVRw"},
		{u: "jNQXAC9IVRw&t=25", id: "jNQXAC9IVRw", startTime: 25 * time.Second},
		{u: "v=jNQXAC9IVRw&t=25", id: "jNQXAC9IVRw", startTime: 25 * time.Second},
		{u: "t=25&v=jNQXAC9IVRw", id: "jNQXAC9IVRw", startTime: 25 * time.Second},

		{u: "youtube.com/watch?v=jNQXAC9IVRw", id: "jNQXAC9IVRw"},
		{u: "www.youtube.com/watch?v=jNQXAC9IVRw", id: "jNQXAC9IVRw"},
		{u: "m.youtube.com/watch?v=jNQXAC9IVRw", id: "jNQXAC9IVRw"},
		{u: "http://www.youtube.com/watch?v=jNQXAC9IVRw", id: "jNQXAC9IVRw"},
		{u: "https://www.youtube.com/watch?v=jNQXAC9IVRw", id: "jNQXAC9IVRw"},
		{u: "https://m.youtube.com/watch?v=jNQXAC9IVRw", id: "jNQXAC9IVRw"},
		{u: "https://youtu.be/jNQXAC9IVRw", id: "jNQXAC9IVRw"},

		{u: "https://www.youtube-nocookie.com/embed/jNQXAC9IVRw?rel=0", id: "jNQXAC9IVRw"},
		{u: "https://www.youtube-nocookie.com/v/jNQXAC9IVRw?version=3&hl=en_US&rel=0", id: "jNQXAC9IVRw"},
		{u: "https://www.youtube.com/?feature=player_embedded&v=jNQXAC9IVRw", id: "jNQXAC9IVRw"},
		{u: "https://www.youtube.com/?v=jNQXAC9IVRw", id: "jNQXAC9IVRw"},
		{u: "https://www.youtube.com/e/jNQXAC9IVRw", id: "jNQXAC9IVRw"},
		{u: "https://www.youtube.com/embed/jNQXAC9IVRw", id: "jNQXAC9IVRw"},
		{u: "https://www.youtube.com/embed/jNQXAC9IVRw?rel=0", id: "jNQXAC9IVRw"},
		{u: "https://www.youtube.com/v/jNQXAC9IVRw", id: "jNQXAC9IVRw"},
		{u: "https://www.youtube.com/v/jNQXAC9IVRw?fs=1&amp;hl=en_US&amp;rel=0", id: "jNQXAC9IVRw"},
		{u: "https://www.youtube.com/v/jNQXAC9IVRw?version=3&autohide=1", id: "jNQXAC9IVRw"},
		{u: "https://www.youtube.com/watch?feature=player_embedded&v=jNQXAC9IVRw", id: "jNQXAC9IVRw"},
		{u: "https://www.youtube.com/watch?v=jNQXAC9IVRw&feature=em-uploademail", id: "jNQXAC9IVRw"},
		{u: "https://www.youtube.com/watch?v=jNQXAC9IVRw&feature=youtu.be", id: "jNQXAC9IVRw"},
		{u: "https://www.youtube.com/watch?v=jNQXAC9IVRw&list=PLBGH6psvCLx46lC91XTNSwi5RPryOhhde&index=106&shuffle=2655", id: "jNQXAC9IVRw"},
		{u: "https://www.youtube.com/watch?v=jNQXAC9IVRw&playnext_from=TL&videos=osPknwzXEas&feature=sub", id: "jNQXAC9IVRw"},
		{u: "https://www.youtube.com/ytscreeningroom?v=jNQXAC9IVRw", id: "jNQXAC9IVRw"},
		{u: "https://youtu.be/jNQXAC9IVRw&feature=channel", id: "jNQXAC9IVRw"},
		{u: "https://youtu.be/jNQXAC9IVRw?feature=youtube_gdata_player", id: "jNQXAC9IVRw"},
		{u: "https://youtu.be/jNQXAC9IVRw?list=PLBGH6psvCLx46lC91XTNSwi5RPryOhhde", id: "jNQXAC9IVRw"},
		{u: "https://youtube.com/?feature=channel&v=jNQXAC9IVRw", id: "jNQXAC9IVRw"},
		{u: "https://youtube.com/?v=jNQXAC9IVRw&feature=youtube_gdata_player", id: "jNQXAC9IVRw"},
		{u: "https://youtube.com/v/jNQXAC9IVRw?feature=youtube_gdata_player", id: "jNQXAC9IVRw"},
		{u: "https://youtube.com/watch?v=jNQXAC9IVRw&feature=channel", id: "jNQXAC9IVRw"},

		{u: "https://youtu.be/k8vpB7GCYPE?t=110", id: "k8vpB7GCYPE", startTime: 110 * time.Second},
		{u: "https://www.youtube.com/watch?v=k8vpB7GCYPE&t=0", id: "k8vpB7GCYPE", startTime: 0},
		{u: "https://www.youtube.com/watch?v=k8vpB7GCYPE&t=1", id: "k8vpB7GCYPE", startTime: 1 * time.Second},
		{u: "https://www.youtube.com/watch?v=k8vpB7GCYPE&t=0s", id: "k8vpB7GCYPE", startTime: 0},
		{u: "https://www.youtube.com/watch?v=k8vpB7GCYPE&t=110s", id: "k8vpB7GCYPE", startTime: 110 * time.Second},
		{u: "https://www.youtube.com/watch?v=k8vpB7GCYPE&t=1m50s", id: "k8vpB7GCYPE", startTime: 1*time.Minute + 50*time.Second},
		{u: "https://www.youtube.com/watch?v=k8vpB7GCYPE&t=45m", id: "k8vpB7GCYPE", startTime: 45 * time.Minute},
		{u: "https://www.youtube.com/watch?v=k8vpB7GCYPE&t=1h14m33s", id: "k8vpB7GCYPE", startTime: 1*time.Hour + 14*time.Minute + 33*time.Second},
		{u: "https://www.youtube.com/watch?v=k8vpB7GCYPE&t=-10", id: "k8vpB7GCYPE", startTime: 0},
	}

	for i, test := range tests {
		id, startTime := extractVideoInfo(test.u)
		if test.id != id {
			t.Fatalf("tests[%d]: %q: id: want %q got %q", i, test.u, test.id, id)
		}
		if test.startTime != startTime {
			t.Fatalf("tests[%d]: %q: startTime: want %q got %q", i, test.u, test.startTime, startTime)
		}
	}
}
