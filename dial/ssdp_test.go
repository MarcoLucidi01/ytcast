// See license file for copyright and license details.

package dial

import (
	"testing"
)

func TestParseMSearchResp(t *testing.T) {
	tests := []struct {
		resp    []byte
		mustErr bool
		service *ssdpService
	}{
		{
			resp: []byte("HTTP/1.1 200 OK\r\n" +
				"LOCATION: http://192.168.1.1:52235/dd.xml\r\n" +
				"CACHE-CONTROL: max-age=1800\r\n" +
				"EXT:\r\n" +
				"BOOTID.UPNP.ORG: 1\r\n" +
				"SERVER: OS/version UPnP/1.1 product/version\r\n" +
				"USN: uuid-foo-bar-baz\r\n" +
				"ST: urn:dial-multiscreen-org:service:dial:1\r\n" +
				"WAKEUP: MAC=10:dd:b1:c9:00:e4;Timeout=10\r\n" +
				"\r\n"),
			mustErr: false,
			service: &ssdpService{
				uniqueServiceName: "uuid-foo-bar-baz",
				location:          "http://192.168.1.1:52235/dd.xml",
				searchTarget:      dialSearchTarget,
			},
		}, {
			resp: []byte("HTTP/1.1 200 OK\r\n" +
				"FOO\r\n" +
				"BAR\r\n" +
				"\r\n"),
			mustErr: true,
		}, {
			resp:    []byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"),
			mustErr: true,
		}, {

			resp: []byte("HTTP/1.1 200 OK\r\n" +
				"LOCATION: http://192.168.1.1:52235/dd.xml\r\n" +
				"ST: urn:dial-multiscreen-org:service:dial:1\r\n" +
				"\r\n"),
			mustErr: true,
		}, {

			resp: []byte("HTTP/1.1 200 OK\r\n" +
				"USN: device UUID\r\n" +
				"ST: urn:dial-multiscreen-org:service:dial:1\r\n" +
				"\r\n"),
			mustErr: true,
		}, {
			resp: []byte("HTTP/1.1 200 OK\r\n" +
				"LOCATION: http://192.168.1.1:52235/dd.xml\r\n" +
				"USN: device UUID\r\n" +
				"\r\n"),
			mustErr: true,
		},
	}

	for i, test := range tests {
		service, err := parseMSearchResp(test.resp)
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
		if test.service.uniqueServiceName != service.uniqueServiceName {
			t.Fatalf("tests[%d]: service.uniqueServiceName: want %q got %q", i, test.service.uniqueServiceName, service.uniqueServiceName)
		}
		if test.service.location != service.location {
			t.Fatalf("tests[%d]: service.location: want %q got %q", i, test.service.location, service.location)
		}
		if test.service.searchTarget != service.searchTarget {
			t.Fatalf("tests[%d]: service.searchTarget: want %q got %q", i, test.service.searchTarget, service.searchTarget)
		}
	}
}
