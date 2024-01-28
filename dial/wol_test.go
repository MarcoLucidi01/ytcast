// See license file for copyright and license details.

package dial

import (
	"net"
	"testing"
)

func TestWakeOnLan(t *testing.T) {
	mac := "" // put your target MAC here
	baddr := "255.255.255.255:9"

	if mac == "" {
		t.SkipNow()
	}
	if err := wakeOnLan(mac, "", baddr); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
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

	for i, mac := range macs {
		addr, err := net.ParseMAC(mac)
		if err != nil {
			t.Fatalf("%d: unexpected error: %s", i, err)
		}
		magic := makeMagicPacket(addr)
		var j int
		for j = 0; j < 6; j++ {
			if magic[j] != 0xff {
				t.Fatalf("%d: magic[%d]: want 0xff got 0x%02x", i, j, magic[j])
			}
		}
		for k := 0; k < 16; k++ {
			for z := 0; z < len(addr); z, j = z+1, j+1 {
				if magic[j] != addr[z] {
					f := "%d: %q: magic[%d] != addr[%d]: want 0x%02x got 0x%02x"
					t.Fatalf(f, i, mac, j, z, addr[z], magic[j])
				}
			}
		}
	}
}
