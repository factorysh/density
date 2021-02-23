package compose

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
)

var subnetPattern *regexp.Regexp

// Subnet is a class B somewhere between 172.18.0.0 and 172.31.255.255 with a /24
//
type Subnet [2]byte

func (s Subnet) Subnet() *net.IPNet {
	return &net.IPNet{
		IP:   net.IPv4(172, s[0], s[1], 0),
		Mask: net.CIDRMask(24, 32),
	}
}

func (s Subnet) Next() (Subnet, error) {
	r := Subnet{}
	if s[1] < 255 {
		r[1] = s[1] + 1
		r[0] = s[0]
	} else {
		if s[0] > 31 {
			return r, errors.New("Too large")
		}
		r[1] = 0
		r[0] = s[0] + 1
	}
	return r, nil
}

func (s Subnet) String() string {
	return fmt.Sprintf("172.%d.%d.0/24", s[0], s[1])
}

func ParseSubnet(txt string) (Subnet, error) {
	if subnetPattern == nil {
		subnetPattern = regexp.MustCompile("172\\.(\\d+)\\.(\\d+)\\.\\d+/24")
	}
	m := subnetPattern.FindStringSubmatch(txt)
	if m == nil {
		return Subnet{}, fmt.Errorf("Can't parse %s", txt)
	}
	a1, err := strconv.Atoi(m[1])
	if err != nil {
		return Subnet{}, err
	}
	a2, err := strconv.Atoi(m[2])
	if err != nil {
		return Subnet{}, err
	}
	for _, a := range []int{a1, a2} {
		if a < 0 || a > 255 {
			return Subnet{}, fmt.Errorf("Not a byte %v", a)
		}
	}
	return Subnet{byte(a1), byte(a2)}, nil
}

type BySubnet []Subnet

func (b BySubnet) Len() int      { return len(b) }
func (b BySubnet) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b BySubnet) Less(i, j int) bool {
	if b[i][0] != b[j][0] {
		return b[i][0] < b[j][0]
	}
	return b[i][1] < b[j][1]
}
