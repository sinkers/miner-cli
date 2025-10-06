package snmp

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
)

// SwitchClient handles SNMP queries to network switches
type SwitchClient struct {
	host      string
	community string
	timeout   time.Duration
	retries   int
}

// PortMACMap represents the MAC address to port mapping
type PortMACMap struct {
	mu       sync.RWMutex
	macToPort map[string]string // MAC address -> interface name (e.g., "Fa0/17")
}

// VLANInfo represents VLAN information
type VLANInfo struct {
	ID   string
	Name string
}

const (
	// OIDs for SNMP queries
	oidIfName           = "1.3.6.1.2.1.31.1.1.1.1"     // Interface names
	oidIfOperStatus     = "1.3.6.1.2.1.2.2.1.8"        // Interface operational status
	oidVLANs            = "1.3.6.1.4.1.9.9.46.1.3.1.1.2" // Cisco VTP VLAN state
	oidMACFDB           = "1.3.6.1.2.1.17.4.3.1.2"     // MAC forwarding database
	oidBridgePortIfIndex = "1.3.6.1.2.1.17.1.4.1.2"    // Bridge port to ifIndex mapping
)

// NewSwitchClient creates a new SNMP client for querying switches
func NewSwitchClient(host, community string) *SwitchClient {
	return &SwitchClient{
		host:      host,
		community: community,
		timeout:   5 * time.Second,
		retries:   1,
	}
}

// NewPortMACMap creates a new port MAC mapping
func NewPortMACMap() *PortMACMap {
	return &PortMACMap{
		macToPort: make(map[string]string),
	}
}

// Set stores a MAC to port mapping
func (pm *PortMACMap) Set(mac, port string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	// Normalize MAC address to lowercase with colons
	normalizedMAC := normalizeMACAddress(mac)
	pm.macToPort[normalizedMAC] = port
}

// Get retrieves the port for a given MAC address
func (pm *PortMACMap) Get(mac string) (string, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	normalizedMAC := normalizeMACAddress(mac)
	port, ok := pm.macToPort[normalizedMAC]
	return port, ok
}

// Size returns the number of MAC mappings
func (pm *PortMACMap) Size() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.macToPort)
}

// GetAllMappings returns a copy of all mappings
func (pm *PortMACMap) GetAllMappings() map[string]string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	result := make(map[string]string, len(pm.macToPort))
	for k, v := range pm.macToPort {
		result[k] = v
	}
	return result
}

// normalizeMACAddress converts MAC address to lowercase with colons
func normalizeMACAddress(mac string) string {
	// Remove common separators and spaces
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, ".", "")
	mac = strings.ReplaceAll(mac, " ", "")
	mac = strings.ToLower(mac)

	// Add colons every 2 characters
	if len(mac) == 12 {
		result := make([]byte, 17)
		for i := 0; i < 6; i++ {
			result[i*3] = mac[i*2]
			result[i*3+1] = mac[i*2+1]
			if i < 5 {
				result[i*3+2] = ':'
			}
		}
		return string(result)
	}

	return strings.ToLower(mac)
}

// connectSNMP creates an SNMP connection
func (sc *SwitchClient) connectSNMP(community string) (*gosnmp.GoSNMP, error) {
	params := &gosnmp.GoSNMP{
		Target:    sc.host,
		Port:      161,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   sc.timeout,
		Retries:   sc.retries,
	}

	err := params.Connect()
	if err != nil {
		return nil, fmt.Errorf("SNMP connect failed: %w", err)
	}

	return params, nil
}

// GetVLANs retrieves the list of VLANs from the switch
func (sc *SwitchClient) GetVLANs() ([]string, error) {
	conn, err := sc.connectSNMP(sc.community)
	if err != nil {
		return nil, err
	}
	defer conn.Conn.Close()

	vlans := []string{}

	// Walk the VLAN OID tree
	err = conn.Walk(oidVLANs, func(pdu gosnmp.SnmpPDU) error {
		// Extract VLAN ID from OID
		// OID format: 1.3.6.1.4.1.9.9.46.1.3.1.1.2.1.VLANID
		parts := strings.Split(pdu.Name, ".")
		if len(parts) > 0 {
			vlanID := parts[len(parts)-1]
			vlans = append(vlans, vlanID)
		}
		return nil
	})

	if err != nil || len(vlans) == 0 {
		// Return common default VLANs if query fails
		return []string{"1", "101", "103"}, nil
	}

	return vlans, nil
}

// GetInterfaceNames retrieves interface names and their indices
func (sc *SwitchClient) GetInterfaceNames() (map[string]string, error) {
	conn, err := sc.connectSNMP(sc.community)
	if err != nil {
		return nil, err
	}
	defer conn.Conn.Close()

	interfaces := make(map[string]string) // ifIndex -> ifName

	err = conn.Walk(oidIfName, func(pdu gosnmp.SnmpPDU) error {
		// Extract interface index from OID
		parts := strings.Split(pdu.Name, ".")
		if len(parts) > 0 {
			ifIndex := parts[len(parts)-1]
			ifName := string(pdu.Value.([]byte))
			interfaces[ifIndex] = ifName
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get interface names: %w", err)
	}

	return interfaces, nil
}

// GetInterfaceStatus retrieves operational status for all interfaces
func (sc *SwitchClient) GetInterfaceStatus() (map[string]int, error) {
	conn, err := sc.connectSNMP(sc.community)
	if err != nil {
		return nil, err
	}
	defer conn.Conn.Close()

	status := make(map[string]int) // ifIndex -> status (1=up, 2=down)

	err = conn.Walk(oidIfOperStatus, func(pdu gosnmp.SnmpPDU) error {
		// Extract interface index from OID
		parts := strings.Split(pdu.Name, ".")
		if len(parts) > 0 {
			ifIndex := parts[len(parts)-1]
			status[ifIndex] = pdu.Value.(int)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get interface status: %w", err)
	}

	return status, nil
}

// isPhysicalPort checks if an interface is a physical port
func isPhysicalPort(ifName string) bool {
	lower := strings.ToLower(ifName)
	// Physical ports: FastEthernet (Fa), GigabitEthernet (Gi), TenGigabitEthernet (Te)
	// Exclude: VLANs (Vl), Port-channels (Po), Null (Nu)
	return !strings.HasPrefix(lower, "vl") &&
		!strings.HasPrefix(lower, "po") &&
		!strings.HasPrefix(lower, "nu")
}

// GetMACAddressTable retrieves MAC address to port mappings for all VLANs
func (sc *SwitchClient) GetMACAddressTable() (*PortMACMap, error) {
	portMap := NewPortMACMap()

	// Get VLANs
	vlans, err := sc.GetVLANs()
	if err != nil {
		return nil, fmt.Errorf("failed to get VLANs: %w", err)
	}

	// Get interface names for mapping
	interfaces, err := sc.GetInterfaceNames()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface names: %w", err)
	}

	// Get interface status
	status, err := sc.GetInterfaceStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface status: %w", err)
	}

	// Query each VLAN
	for _, vlan := range vlans {
		err := sc.queryVLAN(vlan, interfaces, status, portMap)
		if err != nil {
			// Log but continue with other VLANs
			continue
		}
	}

	return portMap, nil
}

// queryVLAN queries a specific VLAN for MAC addresses
func (sc *SwitchClient) queryVLAN(vlan string, interfaces map[string]string, status map[string]int, portMap *PortMACMap) error {
	// Cisco-specific: use community@vlan for VLAN context
	vlanCommunity := fmt.Sprintf("%s@%s", sc.community, vlan)

	conn, err := sc.connectSNMP(vlanCommunity)
	if err != nil {
		return err
	}
	defer conn.Conn.Close()

	// Get bridge port to interface index mapping for this VLAN
	bridgeMap := make(map[string]string) // bridge port -> ifIndex

	err = conn.Walk(oidBridgePortIfIndex, func(pdu gosnmp.SnmpPDU) error {
		parts := strings.Split(pdu.Name, ".")
		if len(parts) > 0 {
			bridgePort := parts[len(parts)-1]
			ifIndex := fmt.Sprintf("%d", pdu.Value.(int))
			bridgeMap[bridgePort] = ifIndex
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Get MAC address forwarding database
	err = conn.Walk(oidMACFDB, func(pdu gosnmp.SnmpPDU) error {
		// Extract MAC address from OID
		// OID format: 1.3.6.1.2.1.17.4.3.1.2.MAC_OCTETS
		parts := strings.Split(pdu.Name, ".")
		if len(parts) >= 6 {
			// Last 6 parts are MAC octets
			macOctets := parts[len(parts)-6:]
			mac := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
				mustAtoi(macOctets[0]),
				mustAtoi(macOctets[1]),
				mustAtoi(macOctets[2]),
				mustAtoi(macOctets[3]),
				mustAtoi(macOctets[4]),
				mustAtoi(macOctets[5]))

			// Get bridge port
			bridgePort := fmt.Sprintf("%d", pdu.Value.(int))

			// Map bridge port to interface index
			if ifIndex, ok := bridgeMap[bridgePort]; ok {
				// Get interface name
				if ifName, ok := interfaces[ifIndex]; ok {
					// Only include physical ports that are UP
					if isPhysicalPort(ifName) && status[ifIndex] == 1 {
						portMap.Set(mac, ifName)
					}
				}
			}
		}
		return nil
	})

	return err
}

// mustAtoi converts string to int, returns 0 on error
func mustAtoi(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

// GetPortForMAC is a convenience method to get the port for a specific MAC
func (sc *SwitchClient) GetPortForMAC(mac string) (string, error) {
	portMap, err := sc.GetMACAddressTable()
	if err != nil {
		return "", err
	}

	port, ok := portMap.Get(mac)
	if !ok {
		return "", fmt.Errorf("MAC address %s not found on switch", mac)
	}

	return port, nil
}
