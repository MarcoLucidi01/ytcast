package dial

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func TestParseDevice(t *testing.T) {
	tests := []struct {
		resp    []byte
		mustErr bool
		service *ssdpService
		device  *Device
	}{
		{
			resp: []byte("HTTP/1.1 200 OK\r\n" +
				"Connection: Close\r\n" +
				"Application-URL: http://192.168.1.1:12345/apps\r\n" +
				"Content-Type: text/plain; charset=US-ASCII\r\n" +
				"\r\n" +
				`<?xml version="1.0" encoding="utf-8" standalone="yes"?>
				<root xmlns="urn:schemas-upnp-org:device-1-0">
				  <specVersion>
				    <major>1</major>
				    <minor>0</minor>
				  </specVersion>
				  <device>
				    <deviceType>urn:dial-multiscreen-org:device:dial:1</deviceType>
				    <friendlyName>Friendly FOO BAR</friendlyName>
				    <manufacturer>FOO</manufacturer>
				    <modelName>BAR</modelName>
				    <UDN>device-UUID</UDN>
				    <serviceList>
				      <service>
					<serviceType>urn:dial-multiscreen-org:service:dial:1</serviceType>
					<serviceId>urn:dial-multiscreen-org:serviceId:dial</serviceId>
					<SCPDURL>/upnp/dev/device-UUID/svc/dial-multiscreen-org/dial/desc</SCPDURL>
					<controlURL>/upnp/dev/device-UUID/svc/dial-multiscreen-org/dial/action</controlURL>
					<eventSubURL>/upnp/dev/device-UUID/svc/dial-multiscreen-org/dial/event</eventSubURL>
				      </service>
				    </serviceList>
				  </device>
				</root>`),
			mustErr: false,
			service: &ssdpService{
				uniqueServiceName: "device-UUID",
				location:          "http://192.168.1.1:52235/dd.xml",
				searchTarget:      "urn:dial-multiscreen-org:service:dial:1",
				headers: map[string][]string{
					"Server": []string{"OS/version UPnP/1.1 product/version"},
					"Wakeup": []string{"MAC=10:dd:b1:c9:00:e4;Timeout=60"},
				},
			},
			device: &Device{
				UniqueServiceName: "device-UUID",
				Location:          "http://192.168.1.1:52235/dd.xml",
				ApplicationUrl:    "http://192.168.1.1:12345/apps",
				FriendlyName:      "Friendly FOO BAR",
				Wakeup: Wakeup{
					Mac:     "10:dd:b1:c9:00:e4",
					Timeout: 60 * time.Second,
				},
			},
		}, {
			resp: []byte("HTTP/1.1 200 OK\r\n" +
				"Connection: Close\r\n" +
				"Content-Type: text/plain; charset=US-ASCII\r\n" +
				"\r\n" +
				`
				<?xml version="1.0" encoding="utf-8" standalone="yes"?>
				<root xmlns="urn:schemas-upnp-org:device-1-0">
				</root>`),
			mustErr: true,
		},
	}

	for i, test := range tests {
		respBody, headers := makeResp(t, test.resp)
		device, err := parseDevice(test.service, respBody, headers)
		if err == nil {
			if test.mustErr {
				t.Fatalf("tests[%d]: was expecting error but got nil", i)
			}
		} else {
			if !test.mustErr {
				t.Fatalf("tests[%d]: unexpected error: %s", i, err)
			}
			continue
		}
		if test.device.UniqueServiceName != device.UniqueServiceName {
			t.Fatalf("tests[%d]: device.UniqueServiceName: want %q got %q", i, test.device.UniqueServiceName, device.UniqueServiceName)
		}
		if test.device.Location != device.Location {
			t.Fatalf("tests[%d]: device.Location: want %q got %q", i, test.device.Location, device.Location)
		}
		if test.device.ApplicationUrl != device.ApplicationUrl {
			t.Fatalf("tests[%d]: device.ApplicationUrl: want %q got %q", i, test.device.ApplicationUrl, device.ApplicationUrl)
		}
		if test.device.FriendlyName != device.FriendlyName {
			t.Fatalf("tests[%d]: device.FriendlyName: want %q got %q", i, test.device.FriendlyName, device.FriendlyName)
		}
		if test.device.Wakeup.Mac != device.Wakeup.Mac {
			t.Fatalf("tests[%d]: device.Wakeup.Mac: want %q got %q", i, test.device.Wakeup.Mac, device.Wakeup.Mac)
		}
		if test.device.Wakeup.Timeout != device.Wakeup.Timeout {
			t.Fatalf("tests[%d]: device.Wakeup.Timeout: want %d got %d", i, test.device.Wakeup.Timeout, device.Wakeup.Timeout)
		}
	}
}

func makeResp(t *testing.T, raw []byte) ([]byte, http.Header) {
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(raw)), nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	return respBody, resp.Header
}

func TestUnmarshalAppInfo(t *testing.T) {
	tests := []struct {
		resp    []byte
		appInfo *AppInfo
	}{
		{
			resp: []byte(`<?xml version="1.0" encoding="UTF-8"?>
				<service xmlns="urn:dial-multiscreen-org:schemas:dial" dialVer="1.7">
					<name>YouTube</name>
					<options allowStop="true"/>
					<state>running</state>
					<link rel="run" href="run"/>
				</service>`),
			appInfo: &AppInfo{
				Name:  "YouTube",
				State: "running",
				Options: struct {
					AllowStop bool `xml:"allowStop,attr"`
				}{AllowStop: true},
				Link: struct {
					Rel  string `xml:"rel,attr"`
					Href string `xml:"href,attr"`
				}{Rel: "run", Href: "run"},
			},
		}, {
			resp: []byte(`
				<?xml version="1.0" encoding="UTF-8"?>
				<service xmlns="urn:dial-multiscreen-org:schemas:dial" dialVer="1.7">
					<name>YouTube</name>
					<options allowStop="true"/>
					<state>running</state>
					<link rel="run" href="run"/>
					<additionalData><screenId>screen123</screenId><sessionId>token123</sessionId></additionalData>
				</service>`),
			appInfo: &AppInfo{
				Name:  "YouTube",
				State: "running",
				Options: struct {
					AllowStop bool `xml:"allowStop,attr"`
				}{AllowStop: true},
				Link: struct {
					Rel  string `xml:"rel,attr"`
					Href string `xml:"href,attr"`
				}{Rel: "run", Href: "run"},
				Additional: struct {
					Data string `xml:",innerxml"`
				}{Data: "<screenId>screen123</screenId><sessionId>token123</sessionId>"},
			},
		},
	}

	for i, test := range tests {
		var appInfo AppInfo
		err := xml.Unmarshal(test.resp, &appInfo)
		if err != nil {
			t.Fatalf("tests[%d]: unexpected error: %q", i, err)
		}
		if test.appInfo.Name != appInfo.Name {
			t.Fatalf("tests[%d]: appInfo.Name: want %q got %q", i, test.appInfo.Name, appInfo.Name)
		}
		if test.appInfo.State != appInfo.State {
			t.Fatalf("tests[%d]: appInfo.State: want %q got %q", i, test.appInfo.State, appInfo.State)
		}
		if test.appInfo.Options.AllowStop != appInfo.Options.AllowStop {
			t.Fatalf("tests[%d]: appInfo.Options.AllowStop: want %t got %t", i, test.appInfo.Options.AllowStop, appInfo.Options.AllowStop)
		}
		if test.appInfo.Link.Rel != appInfo.Link.Rel {
			t.Fatalf("tests[%d]: appInfo.Link.Rel: want %q got %q", i, test.appInfo.Link.Rel, appInfo.Link.Rel)
		}
		if test.appInfo.Link.Href != appInfo.Link.Href {
			t.Fatalf("tests[%d]: appInfo.Link.Href: want %q got %q", i, test.appInfo.Link.Href, appInfo.Link.Href)
		}
		if test.appInfo.Additional.Data != appInfo.Additional.Data {
			t.Fatalf("tests[%d]: appInfo.Additional.Data: want %q got %q", i, test.appInfo.Additional.Data, appInfo.Additional.Data)
		}
	}
}

func TestParseWakeup(t *testing.T) {
	tests := []struct {
		value  string
		wakeup Wakeup
	}{
		{value: "MAC=10:dd:b1:c9:00:e4;Timeout=10", wakeup: Wakeup{Mac: "10:dd:b1:c9:00:e4", Timeout: 10 * time.Second}},
		{value: "MAC=foo Timeout=bar", wakeup: Wakeup{Mac: "", Timeout: 0}},
		{value: "", wakeup: Wakeup{Mac: "", Timeout: 0}},
	}

	for i, test := range tests {
		wakeup := parseWakeup(test.value)
		if test.wakeup.Mac != wakeup.Mac {
			t.Fatalf("tests[%d]: wakeup.Mac: want %q got %q", i, test.wakeup.Mac, wakeup.Mac)
		}
		if test.wakeup.Timeout != wakeup.Timeout {
			t.Fatalf("tests[%d]: wakeup.Timeout: want %s got %s", i, test.wakeup.Timeout, wakeup.Timeout)
		}
	}
}
