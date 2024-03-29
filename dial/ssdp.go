// See license file for copyright and license details.

// This file implements the SSDP (Simple Service Discovery Protocol) portion
// used by the DIAL protocol (i.e. the M-SEARCH request).

package dial

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	ssdpMulticastAddr = "239.255.255.250:1900"

	mSearchMan = "ssdp:discover"
	mSearchMx  = 3

	MSearchMinTimeout  = time.Duration(mSearchMx)*time.Second + 1*time.Second
	MSearchMaxTimeout  = 2 * time.Minute
	mSearchMaxRespSize = 4096
)

var (
	errBadHttpStatus = errors.New("bad HTTP response status")
	errNoUSN         = errors.New("missing USN header")
	errNoLocation    = errors.New("missing LOCATION header")
	errNoST          = errors.New("missing ST header")
)

// ssdpService is a network service discovered with an SSDP M-SEARCH request.
type ssdpService struct {
	uniqueServiceName string      // composite unique service identifier.
	location          string      // URL to the UPnP description of the root device.
	searchTarget      string      // single URI, depends on the ST header sent in the M-SEARCH request.
	headers           http.Header // all headers contained in the M-SEARCH response.
}

// mSearch discovers network services sending an SSDP M-SEARCH request.
func mSearch(localAddr, searchTarget string, done chan struct{}, timeout time.Duration) (chan *ssdpService, error) {
	timeout = clamp(timeout, MSearchMinTimeout, MSearchMaxTimeout)

	maddr, err := net.ResolveUDPAddr("udp4", ssdpMulticastAddr)
	if err != nil {
		return nil, err
	}

	var laddr *net.UDPAddr
	if localAddr != "" {
		if laddr, err = net.ResolveUDPAddr("udp4", localAddr); err != nil {
			return nil, err
		}
	}

	conn, err := net.ListenUDP("udp4", laddr)
	if err != nil {
		return nil, err
	}
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		conn.Close() // can't defer before goroutine.
		return nil, err
	}

	req := bytes.NewBufferString("M-SEARCH * HTTP/1.1\r\n")
	fmt.Fprintf(req, "HOST: %s\r\n", ssdpMulticastAddr)
	fmt.Fprintf(req, "MAN: %q\r\n", mSearchMan) // must be quoted
	fmt.Fprintf(req, "ST: %s\r\n", searchTarget)
	fmt.Fprintf(req, "MX: %d\r\n", mSearchMx)
	req.WriteString("\r\n")
	log.Printf("M-SEARCH udp %s ST %q MX %d timeout %s", ssdpMulticastAddr, searchTarget, mSearchMx, timeout)
	if _, err := conn.WriteTo(req.Bytes(), maddr); err != nil {
		conn.Close() // can't defer before goroutine.
		return nil, err
	}

	ch := make(chan *ssdpService)
	go func() {
		defer conn.Close()
		defer close(ch)

		buf := make([]byte, mSearchMaxRespSize)
		for {
			_, raddr, err := conn.ReadFrom(buf)
			if err != nil {
				log.Println(err)
				return
			}
			service, err := parseMSearchResp(buf)
			if err != nil {
				log.Printf("parseMSearchResp udp %s: %s", raddr, err)
				continue
			}
			log.Printf("discovered service %s", service.location)
			select {
			case ch <- service:
			case <-done:
				return
			}
		}
	}()
	return ch, nil
}

func clamp(d, min, max time.Duration) time.Duration {
	if d < min {
		return min
	}
	if d > max {
		return max
	}
	return d
}

func parseMSearchResp(data []byte) (*ssdpService, error) {
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(data)), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s: %w", resp.Status, errBadHttpStatus)
	}

	service := &ssdpService{headers: resp.Header}

	if service.uniqueServiceName = strings.TrimSpace(service.headers.Get("USN")); service.uniqueServiceName == "" {
		return nil, errNoUSN
	}
	if service.location = strings.TrimSpace(service.headers.Get("LOCATION")); service.location == "" {
		return nil, errNoLocation
	}
	if service.searchTarget = strings.TrimSpace(service.headers.Get("ST")); service.searchTarget == "" {
		return nil, errNoST
	}
	return service, nil
}
