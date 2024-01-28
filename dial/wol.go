// See license file for copyright and license details.

package dial

import "net"

// wakeOnLan sends a magic packet to wake-on-lan a computer on the network, see
// https://en.wikipedia.org/wiki/Wake-on-LAN
// The magic packet is composed by 6 times 0xff followed by 16 times the MAC
// address and it's sent using UDP.
// baddr is UDP's destination address, should be a broadcast address, usually
// "255.255.255.255:9" is a good choice (limited broadcast address and
// discard port).
func wakeOnLan(mac, laddr, baddr string) error {
	addr, err := net.ParseMAC(mac)
	if err != nil {
		return err
	}
	var d net.Dialer
	if laddr != "" {
		if d.LocalAddr, err = net.ResolveUDPAddr("udp", laddr); err != nil {
			return err
		}
	}
	magic := makeMagicPacket(addr)
	conn, err := d.Dial("udp", baddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write(magic)
	return err
}

func makeMagicPacket(addr net.HardwareAddr) []byte {
	var magic []byte
	for i := 0; i < 6; i++ {
		magic = append(magic, 0xff)
	}
	for i := 0; i < 16; i++ {
		magic = append(magic, addr...)
	}
	return magic
}
