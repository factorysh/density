package network

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sort"
)

type ByNetwork []*net.IPNet

func (a ByNetwork) Len() int      { return len(a) }
func (a ByNetwork) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByNetwork) Less(i, j int) bool {
	return ip4Less(a[i].IP, a[j].IP)
}

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
	ib := binary.BigEndian.Uint32(b)
	ia := binary.BigEndian.Uint32(a)
	if ib > ia {
		return int(ib - ia)
	}
	return -int(ia - ib)
}

// NetDistance is absolute distance between 2 networks, last to last, if < 0 it's an overlap
/*
|--a--|  |--b--|
      |<------>|

|--a--|
   |---b--|
   |<->|
*/
func NetDistance(a, b *net.IPNet) int {
	af, al := FirstLast(a)
	iaf := binary.BigEndian.Uint32(af)
	ial := binary.BigEndian.Uint32(al)
	bf, bl := FirstLast(b)
	ibf := binary.BigEndian.Uint32(bf)
	ibl := binary.BigEndian.Uint32(bl)
	if iaf == ibf && ial == ibl { // same network
		return 0
	}
	if iaf >= ibf && iaf <= ibl { // a start after b
		// overlap
		return -int(ibl - iaf)
	}
	if ibf >= iaf && ibf <= ial { // b start after b
		// overlap
		return -int(ial - ibf)
	}
	if iaf > ibl { // a is after b
		return int(ial - ibl)
	}
	// b is after a
	return int(ibl - ial)
}

func nextNet(a *net.IPNet, mask net.IPMask) *net.IPNet {
	_, l := FirstLast(a)
	start := make([]byte, 4)
	binary.BigEndian.PutUint32(start, binary.BigEndian.Uint32(l)+1)
	return &net.IPNet{
		IP:   start,
		Mask: mask}
}

func NextAvailableNetwork(networks []*net.IPNet,
	mini *net.IPNet,
	maxi *net.IPNet,
	mask net.IPMask) (*net.IPNet, error) {
	if len(networks) == 0 { // no other networks
		return mini, nil
	}
	sort.Sort(ByNetwork(networks))
	_, l := FirstLast(networks[len(networks)-1])
	fMax, _ := FirstLast(maxi)
	d := Distance(l, fMax)
	ones, _ := mask.Size()
	maskSize := intPow(2, (32 - ones))
	if d > maskSize { // there is room in the queue
		return nextNet(networks[len(networks)-1], mask), nil
	}
	poz := 0
	n := mini
	for {
		d := NetDistance(n, networks[poz])
		if d < 0 {
			n = nextNet(n, mask)
			poz++
			continue
		}
		if d >= maskSize {
			return n, nil
		}
		n = nextNet(n, mask)
		f, _ := FirstLast(n)
		if Distance(fMax, f) > 0 {
			return nil, fmt.Errorf("beyond max net : %v", maxi)
		}
		poz++
		if poz == len(networks) {
			return nil, errors.New("full network")
		}
	}
}
