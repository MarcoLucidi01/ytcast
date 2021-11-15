// Package dial implements basic functionality of the DIscovery And Launch
// protocol http://www.dial-multiscreen.org/
package dial

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/MarcoLucidi01/ytcast/ssdp"
)

const (
	// SSDP search target for DIAL devices
	dialSearchTarget = "urn:dial-multiscreen-org:service:dial:1"

	// buffer size of the channel used to return discovered devices
	devChanBufSize = 10
)

var (
	LogVerbose = false // enable verbose logging
)

// Device represents a DIAL server device discovered on the network. Contains
// information from both ssdp.Service and device description response from the
// service Location.
type Device struct {
	UniqueServiceName string // UniqueServiceName from the ssdp.Service
	Location          string // Location from the ssdp.Service
	ApplicationUrl    string // absolute HTTP URL, identifies the base DIAL REST service
	FriendlyName      string // UPnP friendlyName field of the device description response
	Wakeup            Wakeup // WAKEUP header values from the ssdp.Service (optional)
}

// Wakeup contains values of WAKEUP header from the ssdp.Service (i.e. an
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
func Discover(timeout int) (chan *Device, error) {
	ssdpCh, err := ssdp.Search(dialSearchTarget, timeout)
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

func getDeviceDesc(service *ssdp.Service, wg *sync.WaitGroup, ch chan *Device) {
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
func parseDevice(service *ssdp.Service, resp *http.Response) (*Device, error) {
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
		XMLName      xml.Name `xml:"root"`
		FriendlyName string   `xml:"device>friendlyName"`
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
