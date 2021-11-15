package dial

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/MarcoLucidi01/ytcast/ssdp"
)

func TestParseDeviceGood(t *testing.T) {
	mac := "10:dd:b1:c9:00:e4"
	timeout := 60
	service := &ssdp.Service{
		UniqueServiceName: "device-UUID",
		Location:          "http://192.168.1.1:52235/dd.xml",
		SearchTarget:      "urn:dial-multiscreen-org:service:dial:1",
		Headers: map[string][]string{
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
	failIfNotEqualS(t, "dev.UniqueServiceName", service.UniqueServiceName, dev.UniqueServiceName)
	failIfNotEqualS(t, "dev.Location", service.Location, dev.Location)
	failIfNotEqualS(t, "dev.ApplicationUrl", appUrl, dev.ApplicationUrl)
	failIfNotEqualS(t, "dev.FriendlyName", friendlyName, dev.FriendlyName)
	failIfNotEqualS(t, "dev.Wakeup.Mac", mac, dev.Wakeup.Mac)
	failIfNotEqualI(t, "dev.Wakeup.Timeout", timeout, dev.Wakeup.Timeout)
}

func TestParseDeviceInvalidStatus(t *testing.T) {
	resp := makeResp(t, []byte("HTTP/1.1 404 Not Found\r\nConnection: Close\r\n\r\n"))
	defer resp.Body.Close()
	_, err := parseDevice(&ssdp.Service{}, resp)
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
	_, err := parseDevice(&ssdp.Service{}, resp)
	failIfNil(t, err)
}

func makeResp(t *testing.T, raw []byte) *http.Response {
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(raw)), nil)
	failIfNotNil(t, err)
	return resp
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
