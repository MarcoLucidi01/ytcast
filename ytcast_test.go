// See license file for copyright and license details.

package main

import (
	"testing"
)

func TestExtractScreenId(t *testing.T) {
	tests := []string{
		"<screenId>foo-bar-baz</screenId>",
	}

	const want = "foo-bar-baz"
	for i, test := range tests {
		screenId, err := extractScreenId(test)
		if err != nil {
			t.Fatalf("tests[%d]: unexpected error: %q", i, err)
		}
		if screenId != want {
			t.Fatalf("tests[%d]: screenId: want %q got %q", i, want, screenId)
		}
	}
}

func TestExtractVideoId(t *testing.T) {
	tests := []string{
		"0zM3nApSvMg",
		"https://www.youtube.com/watch?v=0zM3nApSvMg&feature=feedrec_grec_index",
		"https://www.youtube.com/v/0zM3nApSvMg?fs=1&amp;hl=en_US&amp;rel=0",
		"https://www.youtube.com/watch?v=0zM3nApSvMg#t=0m10s",
		"https://www.youtube.com/embed/0zM3nApSvMg?rel=0",
		"https://www.youtube.com/watch?v=0zM3nApSvMg",
		"https://youtu.be/0zM3nApSvMg",
	}

	const want = "0zM3nApSvMg"
	for i, test := range tests {
		videoId := extractVideoId(test)
		if videoId != want {
			t.Fatalf("tests[%d]: videoId: want %q got %q", i, want, videoId)
		}
	}
}
