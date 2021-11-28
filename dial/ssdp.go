// Package ssdp implements M-SEARCH method of Simple Service Discovery Protocol
// to discover services on the network.
package dial

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

const (
	multicastAddr = "239.255.255.250:1900" // multicast address and port reserved for SSDP by IANA

	minTimeout = 1 * time.Second // min wait time in seconds for M-SEARCH responses
	maxTimeout = 5 * time.Second // max wait time in seconds for M-SEARCH responses

	maxRespSize = 4096 // max size of M-SEARCH response
	chanBufSize = 10   // buffer size of the channel used to return discovered services
)

// Service represents a service discovered on the network through an M-SEARCH.
type ssdpService struct {
	UniqueServiceName string      // composite unique service identifier
	Location          string      // URL to the UPnP description of the root device
	SearchTarget      string      // single URI, depends on the ST header sent in the M-SEARCH request
	Headers           http.Header // all headers from the M-SEARCH response
}

// Search discovers services on the network. It sends an M-SEARCH request and
// waits in a goroutine for responses.
func Search(searchTarget string, timeout time.Duration) (chan *ssdpService, error) {
	timeout = clamp(timeout, minTimeout, maxTimeout)

	laddr, err := sendMSearchReq(searchTarget, timeout)
	if err != nil {
		return nil, err
	}

	logVerbosef("listening on udp %s for M-SEARCH responses, timeout %ds", laddr, timeout)
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return nil, err
	}

	conn.SetReadDeadline(time.Now().Add(timeout))
	ch := make(chan *ssdpService, chanBufSize)
	go listenMSearchResp(conn, ch)
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

func sendMSearchReq(searchTarget string, timeout time.Duration) (*net.UDPAddr, error) {
	conn, err := net.Dial("udp", multicastAddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	buf := bytes.NewBufferString("M-SEARCH * HTTP/1.1\r\n")
	fmt.Fprintf(buf, "HOST: %s\r\n", multicastAddr)
	fmt.Fprintf(buf, "MAN: %q\r\n", "ssdp:discover") // must be quoted
	fmt.Fprintf(buf, "MX: %d\r\n", timeout/time.Second)
	fmt.Fprintf(buf, "ST: %s\r\n", searchTarget)
	// TODO user agent header
	buf.WriteString("\r\n")

	logVerbosef("sending M-SEARCH request to udp %s with ST %q", multicastAddr, searchTarget)
	if _, err := conn.Write(buf.Bytes()); err != nil {
		return nil, err
	}

	return conn.LocalAddr().(*net.UDPAddr), nil
}

func listenMSearchResp(conn *net.UDPConn, ch chan *ssdpService) {
	defer conn.Close()
	defer close(ch)

	buf := make([]byte, maxRespSize)
	for {
		n, raddr, err := conn.ReadFrom(buf)
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				return
			}
			log.Printf("error receiving udp message: %s", err) // always log errors != timeout
			return
		}

		logVerbosef("received message from udp %s of size %d", raddr, n)
		service, err := parseMSearchResp(buf)
		if err != nil {
			logVerbosef("error parsing M-SEARCH response: %s\n%s\n", err, string(buf))
			continue
		}
		logVerbosef("discovered network service %q at %s", service.UniqueServiceName, service.Location)
		ch <- service
	}
}

func parseMSearchResp(buf []byte) (*ssdpService, error) {
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(buf)), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // useless?

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("invalid M-SEARCH response line: status code != 200")
	}

	service := &ssdpService{Headers: resp.Header}

	if service.UniqueServiceName = service.Headers.Get("USN"); service.UniqueServiceName == "" {
		return nil, fmt.Errorf("missing USN header")
	}
	if service.Location = service.Headers.Get("LOCATION"); service.Location == "" {
		return nil, fmt.Errorf("missing LOCATION header")
	}
	if service.SearchTarget = service.Headers.Get("ST"); service.SearchTarget == "" {
		return nil, fmt.Errorf("missing ST header")
	}

	return service, nil
}
