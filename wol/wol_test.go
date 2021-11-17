package wol

import (
	"fmt"
	"net"
	"testing"
)

func TestWakeup(t *testing.T) {
	mac := "" // put your target MAC here
	baddr := "255.255.255.255:9"

	if len(mac) == 0 {
		t.SkipNow()
	}
	failIfNotNil(t, Wakeup(mac, baddr))
}

func TestMakeMagicPacket(t *testing.T) {
	macs := []string{
		"71:5f:9f:60:0c:30",
		"e5:c0:7f:91:99:c8",
		"4f:6f:de:d5:72:20",
		"db:e9:15:e9:f5:9d",
		"ea:79:93:77:db:cd",
		"ff:e3:50:90:81:00",
		"96:10:d6:62:14:a5",
		"0b:b7:ad:92:02:a4",
		"ec:e7:e3:c2:1d:6e",
		"56:53:57:28:6f:6c",
	}
	for _, mac := range macs {
		addr, err := net.ParseMAC(mac)
		failIfNotNil(t, err)

		magic := makeMagicPacket(addr)
		var i int
		for i = 0; i < 6; i++ {
			failIfNotEqualByte(t, fmt.Sprintf("magic[%d]", i), 0xff, magic[i])
		}
		for j := 0; j < 16; j++ {
			for k := 0; k < len(addr); k, i = k+1, i+1 {
				prefix := fmt.Sprintf("magic[%d] != addr[%d]", i, k)
				failIfNotEqualByte(t, prefix, addr[k], magic[i])
			}
		}
	}
}

func failIfNotNil(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("got unexpected error: %s", err)
	}
}

func failIfNotEqualByte(t *testing.T, prefix string, want, got byte) {
	if want != got {
		t.Fatalf("%s: want 0x%02x got 0x%02x", prefix, want, got)
	}
}
