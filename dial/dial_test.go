package dial

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"testing"
)

func TestParseDeviceGood(t *testing.T) {
	mac := "10:dd:b1:c9:00:e4"
	timeout := 60
	service := &ssdpService{
		uniqueServiceName: "device-UUID",
		location:          "http://192.168.1.1:52235/dd.xml",
		searchTarget:      "urn:dial-multiscreen-org:service:dial:1",
		headers: map[string][]string{
			"Server": []string{"OS/version UPnP/1.1 product/version"},
			"Wakeup": []string{"MAC=" + mac + ";Timeout=" + strconv.Itoa(timeout)},
		},
	}

	appUrl := "http://192.168.1.1:12345/apps"
	friendlyName := "Friendly FOO BAR"
	resp := makeResp(t, []byte("HTTP/1.1 200 OK\r\n"+
		"Connection: Close\r\n"+
		"Application-URL: "+appUrl+"\r\n"+
		"Content-Type: text/plain; charset=US-ASCII\r\n"+
		"\r\n"+
		`
		<?xml version="1.0" encoding="utf-8" standalone="yes"?>
		<root xmlns="urn:schemas-upnp-org:device-1-0">
		  <specVersion>
		    <major>1</major>
		    <minor>0</minor>
		  </specVersion>
		  <device>
		    <deviceType>urn:dial-multiscreen-org:device:dial:1</deviceType>
		    <friendlyName>`+friendlyName+`</friendlyName>
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
		</root>
		`))
	defer resp.Body.Close()

	dev, err := parseDevice(service, resp)
	failIfNotNil(t, err)
	failIfNotEqualS(t, "dev.UniqueServiceName", service.uniqueServiceName, dev.UniqueServiceName)
	failIfNotEqualS(t, "dev.Location", service.location, dev.Location)
	failIfNotEqualS(t, "dev.ApplicationUrl", appUrl, dev.ApplicationUrl)
	failIfNotEqualS(t, "dev.FriendlyName", friendlyName, dev.FriendlyName)
	failIfNotEqualS(t, "dev.Wakeup.Mac", mac, dev.Wakeup.Mac)
	failIfNotEqualI(t, "dev.Wakeup.Timeout", timeout, dev.Wakeup.Timeout)
}

func TestParseDeviceInvalidStatus(t *testing.T) {
	resp := makeResp(t, []byte("HTTP/1.1 404 Not Found\r\nConnection: Close\r\n\r\n"))
	defer resp.Body.Close()
	_, err := parseDevice(&ssdpService{}, resp)
	failIfNil(t, err)
}

func TestParseDeviceMissingApplicationUrl(t *testing.T) {
	resp := makeResp(t, []byte("HTTP/1.1 200 OK\r\n"+
		"Connection: Close\r\n"+
		"Content-Type: text/plain; charset=US-ASCII\r\n"+
		"\r\n"+
		`
		<?xml version="1.0" encoding="utf-8" standalone="yes"?>
		<root xmlns="urn:schemas-upnp-org:device-1-0">
		</root>
		`))
	defer resp.Body.Close()
	_, err := parseDevice(&ssdpService{}, resp)
	failIfNil(t, err)
}

func makeResp(t *testing.T, raw []byte) *http.Response {
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(raw)), nil)
	failIfNotNil(t, err)
	return resp
}

func TestUnmarshalAppInfo(t *testing.T) {
	name := "YouTube"
	allowStop := true
	state := "running"
	rel := "run"
	href := "run"
	resp := []byte(`
<?xml version="1.0" encoding="UTF-8"?>
<service xmlns="urn:dial-multiscreen-org:schemas:dial" dialVer="1.7">
	<name>` + name + `</name>
	<options allowStop="` + strconv.FormatBool(allowStop) + `"/>
	<state>` + state + `</state>
	<link rel="` + rel + `" href="` + href + `"/>
</service>`)

	var appInfo AppInfo
	err := xml.Unmarshal(resp, &appInfo)
	failIfNotNil(t, err)
	failIfNotEqualS(t, "appInfo.Name", name, appInfo.Name)
	failIfNotEqualS(t, "appInfo.State", state, appInfo.State)
	failIfNotEqualS(t, "appInfo.Link.Rel", rel, appInfo.Link.Rel)
	failIfNotEqualS(t, "appInfo.Link.Href", href, appInfo.Link.Href)
	failIfNotEqualB(t, "appInfo.Options.AllowStop", allowStop, appInfo.Options.AllowStop)
	failIfNotEqualS(t, "appInfo.Additional.Data", "", appInfo.Additional.Data)
}

func TestUnmarshalAppInfoAdditionalData(t *testing.T) {
	additionalData := "\n<screenId>screen123</screenId>\n<sessionId>token123</sessionId>\n"
	resp := []byte(`
<?xml version="1.0" encoding="UTF-8"?>
<service xmlns="urn:dial-multiscreen-org:schemas:dial" dialVer="1.7">
	<name>YouTube</name>
	<options allowStop="true"/>
	<state>running</state>
	<link rel="run" href="run"/>
	<additionalData>` + additionalData + `</additionalData>
</service>
`)
	var appInfo AppInfo
	err := xml.Unmarshal(resp, &appInfo)
	failIfNotNil(t, err)
	failIfNotEqualS(t, "appInfo.Additional.Data", additionalData, appInfo.Additional.Data)
}

func TestParseWakeupGood(t *testing.T) {
	mac := "10:dd:b1:c9:00:e4"
	timeout := 10
	wakeup := parseWakeup(fmt.Sprintf("  MAC = %s ; Timeout  =   %d", mac, timeout))
	failIfNotEqualS(t, "wakeup.Mac", mac, wakeup.Mac)
	failIfNotEqualI(t, "wakeup.Timeout", timeout, wakeup.Timeout)
}

func TestParseWakeupMalformed(t *testing.T) {
	wakeup := parseWakeup("MAC=foo Timeout=bar")
	failIfNotEqualS(t, "wakeup.Mac", "", wakeup.Mac)
	failIfNotEqualI(t, "wakeup.Timeout", 0, wakeup.Timeout)
}

func TestParseWakeupEmpty(t *testing.T) {
	wakeup := parseWakeup("")
	failIfNotEqualS(t, "wakeup.Mac", "", wakeup.Mac)
	failIfNotEqualI(t, "wakeup.Timeout", 0, wakeup.Timeout)
}

func failIfNil(t *testing.T, err error) {
	if err == nil {
		t.Fatalf("want error but got nil")
	}
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

func failIfNotEqualI(t *testing.T, prefix string, want, got int) {
	if want != got {
		t.Fatalf("%s: want %d got %d", prefix, want, got)
	}
}

func failIfNotEqualB(t *testing.T, prefix string, want, got bool) {
	if want != got {
		t.Fatalf("%s: want %t got %t", prefix, want, got)
	}
}
