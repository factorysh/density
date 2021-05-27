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
	ib := binary.BigEndian.Uint32(b)
	ia := binary.BigEndian.Uint32(a)
	if ib > ia {
		return int(ib - ia)
	}
	return -int(ia - ib)
}

func NetDistance(a, b *net.IPNet) (int, error) {
	af, al := FirstLast(a)
	bf, bl := FirstLast(b)
	d := Distance(al, bf)
	if d > 0 { // a then b
		return d, nil
	}
	return Distance(af, bl), nil
	// FIXME collision handling
}

func doesItCollide(a, b *net.IPNet) bool {
	if a.IP.Equal(b.IP) {
		return true
	}
	_, l := FirstLast(a)
	f, _ := FirstLast(b)
	return Distance(l, f) <= 0
}

func nextNet(a *net.IPNet, mask net.IPMask) *net.IPNet {
	_, l := FirstLast(a)
	start := make([]byte, 4)
	binary.BigEndian.PutUint32(start, binary.BigEndian.Uint32(l)+1)
	return &net.IPNet{
		IP:   start,
		Mask: mask}
}

func NextAvailableNetwork(networks []*net.IPNet, mini *net.IPNet, max *net.IPNet, mask net.IPMask) (*net.IPNet, error) {
	if len(networks) == 0 { // no other networks
		return mini, nil
	}
	sort.Sort(ByNetwork(networks))
	_, l := FirstLast(networks[len(networks)-1])
	fMax, _ := FirstLast(max)
	d := Distance(l, fMax)
	ones, _ := mask.Size()
	maskSize := intPow(2, (32 - ones))
	if d > maskSize { // there is room in the queue
		return nextNet(networks[len(networks)-1], mask), nil
	} else {
		poz := 0
		n := mini
		for { // FIXME don't loop for ever
			if doesItCollide(n, networks[poz]) {
				n = nextNet(n, mask)
				poz++
				continue
			}
			d, err := NetDistance(n, networks[poz+1])
			if err != nil {
				return nil, err
			}
			if d < 0 {
				d = -d
			}
			if d >= maskSize {
				return n, nil
			}
			n = nextNet(n, mask)
			poz++
		}
	}
	return nil, nil
}
