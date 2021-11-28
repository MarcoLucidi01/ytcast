package dial

import (
	"testing"
)

func TestParseMSearchRespGood(t *testing.T) {
	location := "http://192.168.1.1:52235/dd.xml"
	usn := "device UUID"
	st := "urn:dial-multiscreen-org:service:dial:1"
	resp := []byte("HTTP/1.1 200 OK\r\n" +
		"LOCATION: " + location + "\r\n" +
		"CACHE-CONTROL: max-age=1800\r\n" +
		"EXT:\r\n" +
		"BOOTID.UPNP.ORG: 1\r\n" +
		"SERVER: OS/version UPnP/1.1 product/version\r\n" +
		"USN: " + usn + "\r\n" +
		"ST: " + st + "\r\n" +
		"WAKEUP: MAC=10:dd:b1:c9:00:e4;Timeout=10\r\n" +
		"\r\n")

	service, err := parseMSearchResp(resp)
	failIfNotNil(t, err)
	failIfNotEqual(t, "service.Location", location, service.Location)
	failIfNotEqual(t, "service.UniqueServiceName", usn, service.UniqueServiceName)
	failIfNotEqual(t, "service.SearchTarget", st, service.SearchTarget)
}

func TestParseMSearchRespMalformed(t *testing.T) {
	resp := []byte("HTTP/1.1 200 OK\r\n" +
		"FOO\r\n" +
		"BAR\r\n" +
		"\r\n")

	_, err := parseMSearchResp(resp)
	failIfNil(t, err)
}

func TestParseMSearchRespInvalidStatus(t *testing.T) {
	resp := []byte("HTTP/1.1 500 Internal Server Error\r\n\r\n")

	_, err := parseMSearchResp(resp)
	failIfNil(t, err)
}

func TestParseMSearchRespMissingUsn(t *testing.T) {
	resp := []byte("HTTP/1.1 200 OK\r\n" +
		"LOCATION: http://192.168.1.1:52235/dd.xml\r\n" +
		"ST: urn:dial-multiscreen-org:service:dial:1\r\n" +
		"\r\n")

	_, err := parseMSearchResp(resp)
	failIfNil(t, err)
}

func TestParseMSearchMissingLocation(t *testing.T) {
	resp := []byte("HTTP/1.1 200 OK\r\n" +
		"USN: device UUID\r\n" +
		"ST: urn:dial-multiscreen-org:service:dial:1\r\n" +
		"\r\n")

	_, err := parseMSearchResp(resp)
	failIfNil(t, err)
}

func TestParseMSearchRespMissingSt(t *testing.T) {
	resp := []byte("HTTP/1.1 200 OK\r\n" +
		"LOCATION: http://192.168.1.1:52235/dd.xml\r\n" +
		"USN: device UUID\r\n" +
		"\r\n")

	_, err := parseMSearchResp(resp)
	failIfNil(t, err)
}

func failIfNotEqual(t *testing.T, prefix, want, got string) {
	if want != got {
		t.Fatalf("%s: want %q got %q", prefix, want, got)
	}
}
