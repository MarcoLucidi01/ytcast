// See license file for copyright and license details.

// Package dial implements a basic DIAL (DIscovery And Launch) client.
// See http://www.dial-multiscreen.org/
package dial

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	dialSearchTarget = "urn:dial-multiscreen-org:service:dial:1"

	contentType = "text/plain; charset=utf-8"

	wakeupBroadcastAddr = "255.255.255.255:9"
	wakeupMinTimeout    = 10 * time.Second
	wakeupMaxTimeout    = 2 * time.Minute
)

var (
	httpClient    = &http.Client{Timeout: 5 * time.Second}
	wakeupParseRe = regexp.MustCompile(`MAC=(.+);Timeout=(\d+)`)

	errNoAppUrl = errors.New("missing Application-URL header")
	errNoMac    = errors.New("missing device MAC address")
	errNoWakeup = errors.New("unable to wakeup device")
)

// Device is a DIAL server device discovered on the network.
type Device struct {
	UniqueServiceName string // UniqueServiceName from the ssdpService.
	Location          string // Location from the ssdpService.
	ApplicationUrl    string // base DIAL REST service url.
	FriendlyName      string // UPnP friendlyName field of the device description.
	Wakeup            Wakeup // WAKEUP header values from the ssdpService (if available).
}

// Wakeup contains values of WAKEUP header from the ssdpService that can be used
// to WoL or WoWLAN the device.
type Wakeup struct {
	Mac     string        // MAC address of the device's wired or wireless network interface.
	Timeout time.Duration // estimated upper bound of the duration needed to wake the device and start its DIAL server.
}

// AppInfo contains information about an application on a specific Device.
type AppInfo struct {
	Name string `xml:"name"`

	// State valid values are:
	// - running: the application is installed and either starting or running;
	// - stopped: the application is installed and not running;
	// - installable=<URL>: the application is not installed but is
	//   available for installation by sending an HTTP GET request to the
	//   provided URL;
	// - hidden: the application is running but is not visible to the user;
	//
	// any other value is invalid and should be ignored.
	State string `xml:"state"`

	Options struct {
		// AllowStop true indicates that the application can be stopped
		// (if running) sending an HTTP DELETE request to Link.Href.
		AllowStop bool `xml:"allowStop,attr"`
	} `xml:"options"`

	// Link is included when the application is running and can be stopped
	// using a DELETE request.
	Link struct {
		// Rel is always "run".
		Rel string `xml:"rel,attr"`

		// Href contains instance URL of the running application.
		Href string `xml:"href,attr"`
	} `xml:"link"`

	Additional struct {
		// Additional.Data contains zero or more (dynamic) XML elements
		// specific to the application.
		Data string `xml:",innerxml"`
	} `xml:"additionalData"`
}

// Discover discovers (unique) DIAL server devices on the network.
func Discover(done chan struct{}, timeout time.Duration) (chan *Device, error) {
	ssdpCh, err := mSearch(dialSearchTarget, done, timeout)
	if err != nil {
		return nil, err
	}

	devCh := make(chan *Device)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		seen := make(map[string]bool)
		for service := range ssdpCh {
			if service.searchTarget != dialSearchTarget || seen[service.uniqueServiceName] {
				continue
			}
			seen[service.uniqueServiceName] = true
			wg.Add(1)
			go func(service *ssdpService) {
				defer wg.Done()
				respBody, headers, err := doReq("GET", service.location, "", "")
				if err != nil {
					log.Println(err)
					return
				}
				dev, err := parseDevice(service, respBody, headers)
				if err != nil {
					log.Printf("%s: parseDevice: %s", service.location, err)
					return
				}
				log.Printf("discovered DIAL device %q", dev.FriendlyName)
				select {
				case devCh <- dev:
				case <-done:
				}
			}(service)
		}
	}()

	go func() {
		wg.Wait()
		close(devCh)
	}()

	return devCh, nil
}

func doReq(method, url string, origin, body string) ([]byte, http.Header, error) {
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if body != "" {
		req.Header.Set("Content-Type", contentType)
	}

	log.Printf("%s %s", method, url)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err == nil && (resp.StatusCode < 200 || resp.StatusCode > 299) {
		err = fmt.Errorf("%s %s: %s: %w", method, url, resp.Status, errBadHttpStatus)
	}
	return respBody, resp.Header, err
}

func parseDevice(service *ssdpService, desc []byte, descHeaders http.Header) (*Device, error) {
	appUrl := strings.TrimSpace(descHeaders.Get("Application-URL"))
	if appUrl == "" {
		return nil, errNoAppUrl
	}

	var v struct {
		FriendlyName string `xml:"device>friendlyName"`
	}
	if err := xml.Unmarshal(desc, &v); err != nil {
		return nil, err
	}

	dev := &Device{
		UniqueServiceName: service.uniqueServiceName,
		Location:          service.location,
		ApplicationUrl:    appUrl,
		FriendlyName:      v.FriendlyName,
		Wakeup:            parseWakeup(service.headers.Get("WAKEUP")),
	}
	return dev, nil
}

func parseWakeup(v string) Wakeup {
	if v == "" {
		return Wakeup{}
	}
	fields := wakeupParseRe.FindStringSubmatch(v)
	if len(fields) != 3 {
		return Wakeup{}
	}
	mac := fields[1]
	timeout, err := strconv.Atoi(fields[2])
	if err != nil || timeout < 0 {
		return Wakeup{}
	}
	return Wakeup{Mac: mac, Timeout: time.Duration(timeout) * time.Second}
}

// GetAppInfo returns information about an application on the Device.
// appName should be an application name registered in the DIAL Registry.
// origin (if present) will be passed as Origin HTTP header.
func (d *Device) GetAppInfo(appName, origin string) (*AppInfo, error) {
	u, err := urlJoin(d.ApplicationUrl, appName)
	if err != nil {
		return nil, err
	}
	respBody, _, err := doReq("GET", u, origin, "")
	if err != nil {
		return nil, err
	}
	var appInfo AppInfo
	if err := xml.Unmarshal(respBody, &appInfo); err != nil {
		return nil, err
	}
	return &appInfo, nil
}

func urlJoin(base, end string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, end)
	return u.String(), nil
}

// Launch launches (starts) an application on the Device and returns its
// instance url (if available).
// appName should be an application name registered in the DIAL Registry.
// origin (if present) will be passed as Origin HTTP header.
// payload (if present) will be passed as HTTP message body with
// Content-Type: text/plain; charset=utf-8 header.
func (d *Device) Launch(appName, origin, payload string) (string, error) {
	u, err := urlJoin(d.ApplicationUrl, appName)
	if err != nil {
		return "", err
	}
	_, headers, err := doReq("POST", u, origin, payload)
	if err != nil {
		return "", err
	}
	return headers.Get("Location"), nil
}

// TryWakeup tries to Wake-On-Lan the Device sending magic packets to its MAC
// address and waiting for it to become available. It eventually updates
// Location and ApplicationUrl (re-Discover) because the Device may have changed
// ip address and/or service ports.
// Returns nil if it successfully wakes up the Device.
func (d *Device) TryWakeup() error {
	if d.Wakeup.Mac == "" {
		return errNoMac
	}
	done := make(chan struct{})
	defer close(done)
	timeout := clamp(d.Wakeup.Timeout*2, wakeupMinTimeout, wakeupMaxTimeout)
	for start := time.Now(); time.Since(start) < timeout; {
		if err := wakeOnLan(d.Wakeup.Mac, wakeupBroadcastAddr); err != nil {
			return err
		}
		if d.Ping() {
			return nil
		}
		// Ping() may have failed because the device changed ip or port.
		devCh, err := Discover(done, MSearchMinTimeout+1*time.Second)
		if err != nil {
			return fmt.Errorf("Discover: %w", err)
		}
		for updatedDev := range devCh {
			if updatedDev.UniqueServiceName == d.UniqueServiceName {
				*d = *updatedDev
				return nil
			}
		}
	}
	return errNoWakeup
}

// Ping returns true if the Device is up i.e. if it responds to requests.
func (d *Device) Ping() bool {
	_, _, err := doReq("GET", d.ApplicationUrl, "", "")
	if err != nil && errors.Is(err, errBadHttpStatus) {
		return true
	}
	return err == nil
}

// Hostname returns the Device's hostname extracted from ApplicationUrl.
func (d *Device) Hostname() string {
	u, err := url.Parse(d.ApplicationUrl)
	if err != nil {
		return ""
	}
	return u.Hostname()
}
