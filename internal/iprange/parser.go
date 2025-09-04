package iprange

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type IPRange struct {
	IPs []net.IP
}

func ParseIPRange(input string) (*IPRange, error) {
	input = strings.TrimSpace(input)

	if strings.Contains(input, "/") {
		return parseCIDR(input)
	}

	if strings.Contains(input, "-") {
		return parseRange(input)
	}

	ip := net.ParseIP(input)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", input)
	}
	return &IPRange{IPs: []net.IP{ip}}, nil
}

func parseCIDR(cidr string) (*IPRange, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR notation: %w", err)
	}

	var ips []net.IP
	for ip := ipNet.IP.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
		ips = append(ips, duplicateIP(ip))
	}

	if len(ips) > 0 && ips[0].IsUnspecified() {
		ips = ips[1:]
	}
	if len(ips) > 0 {
		last := ips[len(ips)-1]
		if isBroadcast(last, ipNet) {
			ips = ips[:len(ips)-1]
		}
	}

	return &IPRange{IPs: ips}, nil
}

func parseRange(rangeStr string) (*IPRange, error) {
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid range format, expected start-end: %s", rangeStr)
	}

	startIP := net.ParseIP(strings.TrimSpace(parts[0]))
	endIP := net.ParseIP(strings.TrimSpace(parts[1]))

	if startIP == nil || endIP == nil {
		return nil, fmt.Errorf("invalid IP addresses in range: %s", rangeStr)
	}

	start4 := startIP.To4()
	end4 := endIP.To4()

	if start4 == nil || end4 == nil {
		return nil, fmt.Errorf("only IPv4 ranges are supported")
	}

	startInt := ipToUint32(start4)
	endInt := ipToUint32(end4)

	if startInt > endInt {
		return nil, fmt.Errorf("start IP must be less than or equal to end IP")
	}

	var ips []net.IP
	for i := startInt; i <= endInt; i++ {
		ips = append(ips, uint32ToIP(i))
	}

	return &IPRange{IPs: ips}, nil
}

func ParseMultipleRanges(inputs []string) (*IPRange, error) {
	var allIPs []net.IP
	seenIPs := make(map[string]bool)

	for _, input := range inputs {
		r, err := ParseIPRange(input)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %w", input, err)
		}

		for _, ip := range r.IPs {
			ipStr := ip.String()
			if !seenIPs[ipStr] {
				seenIPs[ipStr] = true
				allIPs = append(allIPs, ip)
			}
		}
	}

	return &IPRange{IPs: allIPs}, nil
}

func (r *IPRange) GetIPs() []string {
	result := make([]string, len(r.IPs))
	for i, ip := range r.IPs {
		result[i] = ip.String()
	}
	return result
}

func (r *IPRange) Count() int {
	return len(r.IPs)
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func duplicateIP(ip net.IP) net.IP {
	dup := make(net.IP, len(ip))
	copy(dup, ip)
	return dup
}

func isBroadcast(ip net.IP, ipNet *net.IPNet) bool {
	if len(ip) != net.IPv4len {
		return false
	}

	ones, bits := ipNet.Mask.Size()
	if ones == bits {
		return false
	}

	broadcast := make(net.IP, len(ipNet.IP))
	copy(broadcast, ipNet.IP)

	for i := 0; i < len(broadcast); i++ {
		broadcast[i] |= ^ipNet.Mask[i]
	}

	return ip.Equal(broadcast)
}

func ipToUint32(ip net.IP) uint32 {
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func uint32ToIP(n uint32) net.IP {
	return net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}

func ParsePort(portStr string) (int, error) {
	if portStr == "" {
		return 4028, nil
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("invalid port: %w", err)
	}

	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port must be between 1 and 65535")
	}

	return port, nil
}
