package network

import (
	"encoding/binary"
	"net"
	"sort"
)

type ByNetwork []*net.IPNet

func (a ByNetwork) Len() int           { return len(a) }
func (a ByNetwork) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByNetwork) Less(i, j int) bool { return ip4Less(a[i].IP, a[j].IP) }

func ip4Less(a, b net.IP) bool {
	return binary.BigEndian.Uint32(a.To4()) < binary.BigEndian.Uint32(b.To4())
}

func intPow(n, m int) int {
	if m == 0 {
		return 1
	}
	result := n
	for i := 2; i <= m; i++ {
		result *= n
	}
	return result
}

// FirstLast returns the first and last IP of a network
func FirstLast(network *net.IPNet) (net.IP, net.IP) {
	first := binary.BigEndian.Uint32(network.IP)
	mask := binary.BigEndian.Uint32(network.Mask)
	last := (first & mask) | (0xffffffff - mask)
	l := make([]byte, 4)
	binary.BigEndian.PutUint32(l, last)
	return network.IP, net.IP(l)
}

func Distance(a, b net.IP) int {
	return int(binary.BigEndian.Uint32(b) - binary.BigEndian.Uint32(a))
}

func NextAvailableNetwork(networks []*net.IPNet, mini *net.IPNet, max *net.IPNet, mask net.IPMask) (*net.IPNet, error) {
	if len(networks) == 0 { // no other networks
		return mini, nil
	}
	sort.Sort(ByNetwork(networks))
	_, l := FirstLast(networks[len(networks)-1])
	fMax, _ := FirstLast(max)
	ones, _ := mask.Size()
	d := Distance(l, fMax)
	if d > intPow(2, (32-ones)) { // there is room in the queue
		start := make([]byte, 4)
		binary.BigEndian.PutUint32(start, binary.BigEndian.Uint32(l)+1)
		return &net.IPNet{
			IP:   start,
			Mask: mask}, nil
	}
	return nil, nil
}
