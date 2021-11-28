// Package dial implements basic functionality of the DIscovery And Launch
// protocol http://www.dial-multiscreen.org/
package dial

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// SSDP search target for DIAL devices
	dialSearchTarget = "urn:dial-multiscreen-org:service:dial:1"

	// buffer size of the channel used to return discovered devices
	devChanBufSize = 10

	wakeupBroadcastAddress = "255.255.255.255:9"
	wakeupMinTimeout       = 10 * time.Second
	wakeupMaxTimeout       = 150 * time.Second
	wakeupCheckInterval    = 2 * time.Second
)

var (
	LogVerbose = false // enable verbose logging

	errNoMac = errors.New("missing device MAC address")
)

// Device represents a DIAL server device discovered on the network. Contains
// information from both ssdpService and device description response from the
// service Location.
type Device struct {
	UniqueServiceName string // UniqueServiceName from the ssdpService
	Location          string // Location from the ssdpService
	ApplicationUrl    string // absolute HTTP URL, identifies the base DIAL REST service
	FriendlyName      string // UPnP friendlyName field of the device description response
	Wakeup            Wakeup // WAKEUP header values from the ssdpService (optional)
}

// Wakeup contains values of WAKEUP header from the ssdpService (i.e. an
// M-SEARCH response) that could be used to WoL or WoWLAN the Device.
type Wakeup struct {
	// MAC address of the first-screen device's wired or wireless network
	// interface that is currently in use.
	Mac string

	// estimated upper bound of the duration in seconds of the time needed
	// to wake the DIAL server device and then start its DIAL server.
	Timeout int
}

// Discover discovers unique DIAL server devices on the network. timeout is used
// to wait for the underlying SSDP M-SEARCH responses.
func Discover(timeout time.Duration) (chan *Device, error) {
	ssdpCh, err := Search(dialSearchTarget, timeout)
	if err != nil {
		return nil, err
	}

	devCh := make(chan *Device, devChanBufSize)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		seen := make(map[string]bool)
		for service := range ssdpCh {
			if service.SearchTarget != dialSearchTarget || seen[service.UniqueServiceName] {
				continue
			}
			seen[service.UniqueServiceName] = true
			wg.Add(1)
			go getDeviceDesc(service, &wg, devCh)
		}
	}()

	go func() {
		wg.Wait()
		close(devCh)
	}()

	return devCh, nil
}

func logVerbosef(format string, args ...interface{}) {
	if LogVerbose {
		log.Printf(format, args...)
	}
}

func getDeviceDesc(service *ssdpService, wg *sync.WaitGroup, ch chan *Device) {
	defer wg.Done()

	logVerbosef("sending GET %s", service.Location)
	resp, err := http.Get(service.Location)
	if err != nil {
		logVerbosef("GET %s: %s", service.Location, err)
		return
	}
	defer resp.Body.Close()

	dev, err := parseDevice(service, resp)
	if err != nil {
		logVerbosef("GET %s: %s", service.Location, err)
		return
	}
	logVerbosef("discovered device %#v", dev)
	ch <- dev
}

// parseDevice builds a Device struct joining values from service and device
// description response from service.Location.
func parseDevice(service *ssdpService, resp *http.Response) (*Device, error) {
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s", resp.Status)
	}

	appUrl := strings.TrimSpace(resp.Header.Get("Application-URL"))
	if len(appUrl) == 0 {
		return nil, fmt.Errorf("missing Application-URL header")
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var desc struct {
		FriendlyName string `xml:"device>friendlyName"`
	}
	if err := xml.Unmarshal(respBody, &desc); err != nil {
		return nil, err
	}

	dev := &Device{
		UniqueServiceName: service.UniqueServiceName,
		Location:          service.Location,
		ApplicationUrl:    appUrl,
		FriendlyName:      desc.FriendlyName,
		Wakeup:            parseWakeup(service.Headers.Get("WAKEUP")),
	}
	return dev, nil
}

// parseWakeup parses a WAKEUP SSDP header value e.g. MAC=10:dd:b1:c9:00:e4;Timeout=10
func parseWakeup(value string) Wakeup {
	fields := strings.FieldsFunc(value, func(r rune) bool { return r == ';' || r == '=' })
	if len(fields) != 4 {
		return Wakeup{}
	}

	for i := 0; i < len(fields); i++ {
		fields[i] = strings.TrimSpace(fields[i])
		if len(fields[i]) == 0 {
			return Wakeup{}
		}
	}

	if !strings.EqualFold(fields[0], "MAC") {
		return Wakeup{}
	}
	mac := fields[1]

	if !strings.EqualFold(fields[2], "Timeout") {
		return Wakeup{}
	}
	timeout, err := strconv.Atoi(fields[3])
	if err != nil || timeout < 0 {
		return Wakeup{}
	}

	return Wakeup{Mac: mac, Timeout: timeout}
}

// AppInfo contains information about an application on a specific device.
type AppInfo struct {
	// Name is the application name
	Name string `xml:"name"`

	// State valid values are:
	// - running: the application is installed and either starting or running;
	// - stopped: the application is installed and not running;
	// - installable=<URL>: the application is not installed but is
	//   available for installation by sending an HTTP GET request to the
	//   provided URL;
	// - hidden: the application is running but is not visible to the user;
	//
	// any other value is invalid and should be ignored
	State string `xml:"state"`

	Options struct {
		// AllowStop true indicates that the application can be stopped
		// (if running) using an HTTP DELETE request
		AllowStop bool `xml:"allowStop,attr"`
	} `xml:"options"`

	// Link is included when the application is running and can be stopped
	// using a DELETE request.
	Link struct {
		// Rel is always "run".
		Rel string `xml:"rel,attr"`

		// Href contains the resource name of the running application
		// and should match the last portion of the name returned in the
		// 201 CREATED response.
		Href string `xml:"href,attr"`
	} `xml:"link"`

	Additional struct {
		// Additional.Data contains zero or more (dynamic) elements
		// specific to the application and are returned as unparsed XML.
		Data string `xml:",innerxml"`
	} `xml:"additionalData"`
}

// GetAppInfo obtains information about an application on a Device.
// appName should be an application name registered in the DIAL Registry.
// If origin is not empty, it will be passed as Origin HTTP header.
// Any response code != 200 from the server will be returned as an error.
func (d *Device) GetAppInfo(appName string, origin string) (*AppInfo, error) {
	req, err := makeReq("GET", d.ApplicationUrl, appName, origin, "")
	if err != nil {
		return nil, err
	}

	logVerbosef("%s %s", req.Method, req.URL)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var appInfo AppInfo
	if err := xml.Unmarshal(respBody, &appInfo); err != nil {
		return nil, err
	}
	logVerbosef("application %q info %#v", appName, appInfo)
	return &appInfo, nil
}

func makeReq(method, baseUrl, appName, origin, payload string) (*http.Request, error) {
	req, err := http.NewRequest(method, baseUrl, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.URL.Path = path.Join(req.URL.Path, appName)
	if len(origin) > 0 {
		req.Header.Set("Origin", origin)
	}
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	}
	return req, nil
}

// Launch launches an application on a Device.
// appName should be an application name registered in the DIAL Registry.
// If origin is not empty, it will be passed as Origin HTTP header.
// If payload is not empty, it will be passed as HTTP message body with
// Content-Type: text/plain; charset=utf-8.
// Any non-successful response code (< 200 or > 299) from the server will
// be returned as an error.
// If present, the value of the Location response header will be returned by
// this method, it represents the Application Instance URL.
func (d *Device) Launch(appName, origin, payload string) (string, error) {
	req, err := makeReq("POST", d.ApplicationUrl, appName, origin, payload)
	if err != nil {
		return "", err
	}

	logVerbosef("%s %s", req.Method, req.URL)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", errors.New(resp.Status)
	}

	appInstanceUrl := resp.Header.Get("Location")
	logVerbosef("application %q successfully launched, instance URL %s", appName, appInstanceUrl)
	return appInstanceUrl, nil
}

// TODO add logging
// TODO add doc
// TODO fix function name
// TODO use common request builder
func (d *Device) WakeupFunc() error {
	if _, err := http.Get(d.ApplicationUrl); err == nil {
		return nil // device is already up
	}

	if len(d.Wakeup.Mac) == 0 {
		return errNoMac
	}
	if err := wakeOnLan(d.Wakeup.Mac, wakeupBroadcastAddress); err != nil {
		return err
	}

	timeout := time.Duration(d.Wakeup.Timeout) * time.Second
	timeout = clamp(timeout, wakeupMinTimeout, wakeupMaxTimeout)
	for start := time.Now(); time.Since(start) < timeout; {
		time.Sleep(wakeupCheckInterval)
		if _, err := http.Get(d.ApplicationUrl); err == nil {
			return nil
		}
	}

	_, err := http.Get(d.ApplicationUrl)
	return err
}
