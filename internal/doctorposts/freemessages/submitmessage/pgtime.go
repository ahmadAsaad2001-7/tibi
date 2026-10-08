package submitmessage

import (
	"net/netip"
)

// pgInet converts a string IP to a *netip.Addr suitable for the INET column.
// Returns nil if the input is empty or malformed (stored as NULL).
func pgInet(ip string) *netip.Addr {
	if ip == "" {
		return nil
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return nil
	}
	a := addr
	return &a
}
