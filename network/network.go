package network

import (
	"encoding/binary"
	"net"
)

type ByNetwork []*net.IPNet

func (a ByNetwork) Len() int           { return len(a) }
func (a ByNetwork) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByNetwork) Less(i, j int) bool { return IP4Less(a[i].IP, a[j].IP) }

func IP4Less(a, b net.IP) bool {
	return binary.LittleEndian.Uint32(a.To4()) < binary.LittleEndian.Uint32(b.To4())
}

func FirstLast(network *net.IPNet) (net.IP, net.IP) {
	first := binary.LittleEndian.Uint32(network.IP)
	mask := binary.LittleEndian.Uint32(network.Mask)
	last := (first & mask) | (0xffffffff - mask)
	l := make([]byte, 4)
	binary.LittleEndian.PutUint32(l, last)
	return network.IP, net.IP(l)
}
