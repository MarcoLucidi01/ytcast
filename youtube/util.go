// See license file for copyright and license details.

package youtube

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"path"
	"strings"
)

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

// ExtractVideoId extracts the video id from a video url. If video is already a
// video id it is returned unchanged.
// Supports various url formats (see util_test.go for examples).
func ExtractVideoId(video string) string {
	video = strings.TrimSpace(video)
	u, err := url.Parse(video)
	if err != nil {
		return video
	}
	vid := u.Query().Get("v")
	if vid != "" {
		return vid
	}
	if vid = path.Base(u.Path); vid != "" && vid != "." && vid != "/" {
		return vid
	}
	return video
}
