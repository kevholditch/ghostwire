package controlplane

import (
	"errors"
	"fmt"
	"math/big"
	"net/netip"
	"sync"
)

var ErrCIDRExhausted = errors.New("cidr exhausted")

type IPAM struct {
	mu     sync.Mutex
	prefix netip.Prefix
	next   netip.Addr
	last   netip.Addr
	leases map[string]string
}

func NewIPAM(cidr string) (*IPAM, error) {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, fmt.Errorf("parse cidr: %w", err)
	}
	if !prefix.Addr().Is4() {
		return nil, fmt.Errorf("only ipv4 cidrs are supported: %s", cidr)
	}

	network := prefix.Masked().Addr()
	broadcast, err := lastAddr(prefix)
	if err != nil {
		return nil, err
	}

	firstUsable := network.Next()
	lastUsable := prevAddr(broadcast)
	return &IPAM{
		prefix: prefix,
		next:   firstUsable,
		last:   lastUsable,
		leases: map[string]string{},
	}, nil
}

func (i *IPAM) Allocate(agentID string) (string, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if ip, ok := i.leases[agentID]; ok {
		return ip, nil
	}
	if !i.next.IsValid() || compareAddr(i.next, i.last) > 0 {
		return "", ErrCIDRExhausted
	}

	ip := i.next.String()
	i.leases[agentID] = ip
	i.next = i.next.Next()
	return ip, nil
}

func (i *IPAM) CIDR() string {
	return i.prefix.String()
}

func lastAddr(prefix netip.Prefix) (netip.Addr, error) {
	addr := prefix.Masked().Addr()
	bits := prefix.Bits()
	if bits < 0 || bits > 32 {
		return netip.Addr{}, fmt.Errorf("invalid ipv4 prefix bits: %d", bits)
	}

	base := addr.As4()
	baseInt := new(big.Int).SetBytes(base[:])
	size := new(big.Int).Lsh(big.NewInt(1), uint(32-bits))
	last := new(big.Int).Add(baseInt, new(big.Int).Sub(size, big.NewInt(1)))
	bytes := last.FillBytes(make([]byte, 4))
	return netip.AddrFrom4([4]byte{bytes[0], bytes[1], bytes[2], bytes[3]}), nil
}

func prevAddr(addr netip.Addr) netip.Addr {
	v := addr.As4()
	for idx := len(v) - 1; idx >= 0; idx-- {
		if v[idx] > 0 {
			v[idx]--
			break
		}
		v[idx] = 255
	}
	return netip.AddrFrom4(v)
}

func compareAddr(a, b netip.Addr) int {
	av := a.As4()
	bv := b.As4()
	for idx := range av {
		if av[idx] < bv[idx] {
			return -1
		}
		if av[idx] > bv[idx] {
			return 1
		}
	}
	return 0
}
