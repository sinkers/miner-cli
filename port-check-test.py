#!/usr/bin/env python3
"""
SNMP Interface Lister for Cisco 2960 Switch
Polls a Cisco switch via SNMP and lists interface names with MAC addresses
"""

import subprocess
import sys
import re
from collections import defaultdict

# Configuration
SWITCH_IP = '10.110.101.6'
SNMP_COMMUNITY = 'public'  # Change this to your SNMP community string
SNMP_VERSION = '2c'  # SNMP version

# OID for interface names (ifName from IF-MIB)
IF_NAME_OID = '1.3.6.1.2.1.31.1.1.1.1'  # ifName - standard interface name
IF_DESCR_OID = '1.3.6.1.2.1.2.2.1.2'    # ifDescr - interface description (fallback)

# OID for MAC address table (dot1dTpFdbPort from BRIDGE-MIB)
MAC_FDB_PORT_OID = '1.3.6.1.2.1.17.4.3.1.2'  # dot1dTpFdbPort - bridge port for each MAC
# OID for bridge port to interface mapping (dot1dBasePortIfIndex)
BRIDGE_PORT_IF_INDEX_OID = '1.3.6.1.2.1.17.1.4.1.2'  # Maps bridge port to ifIndex


def get_interfaces(target_ip, community, use_descr=False):
    """
    Query SNMP to get interface names from the switch using snmpwalk

    Args:
        target_ip: IP address of the switch
        community: SNMP community string
        use_descr: Use ifDescr instead of ifName (for older devices)

    Returns:
        List of tuples (interface_index, interface_name)
    """
    interfaces = []
    oid = IF_DESCR_OID if use_descr else IF_NAME_OID

    try:
        # Run snmpwalk command
        cmd = [
            'snmpwalk',
            '-v', SNMP_VERSION,
            '-c', community,
            target_ip,
            oid
        ]

        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=10
        )

        if result.returncode != 0:
            print(f"snmpwalk failed: {result.stderr}", file=sys.stderr)
            return []

        # Parse output
        # Format: IF-MIB::ifName.1 = STRING: Fa0/1
        # or:     iso.3.6.1.2.1.31.1.1.1.1.1 = STRING: Fa0/1
        for line in result.stdout.strip().split('\n'):
            if not line:
                continue

            # Match the OID index and value
            # Handle both named MIB format and numeric OID format
            match = re.search(r'\.(\d+)\s*=\s*(?:STRING:\s*)?(.+)$', line)
            if match:
                if_index = match.group(1)
                interface_name = match.group(2).strip('"').strip()
                interfaces.append((if_index, interface_name))

        return interfaces

    except subprocess.TimeoutExpired:
        print(f"Timeout querying {target_ip}", file=sys.stderr)
        return []
    except Exception as e:
        print(f"Exception occurred: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        return []


def get_vlans(target_ip, community):
    """
    Get list of VLANs from the switch

    Args:
        target_ip: IP address of the switch
        community: SNMP community string

    Returns:
        List of VLAN IDs
    """
    vlans = []
    # OID for VLAN IDs (vtpVlanState from CISCO-VTP-MIB)
    vlan_oid = '1.3.6.1.4.1.9.9.46.1.3.1.1.2'

    try:
        cmd = [
            'snmpwalk',
            '-v', SNMP_VERSION,
            '-c', community,
            target_ip,
            vlan_oid
        ]

        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=10
        )

        if result.returncode == 0:
            for line in result.stdout.strip().split('\n'):
                if not line:
                    continue
                # Extract VLAN ID from OID
                match = re.search(r'\.1\.(\d+)\s*=', line)
                if match:
                    vlans.append(match.group(1))

        # If no VLANs found, try common defaults
        if not vlans:
            vlans = ['1', '101', '103']  # Defaults based on interfaces we saw

        return vlans

    except Exception as e:
        print(f"Could not get VLANs, using defaults: {e}", file=sys.stderr)
        return ['1', '101', '103']


def get_mac_addresses(target_ip, community, vlans):
    """
    Query SNMP to get MAC address table from the switch
    For Cisco switches, must query per-VLAN using community@vlan syntax

    Args:
        target_ip: IP address of the switch
        community: SNMP community string
        vlans: List of VLAN IDs to query

    Returns:
        Dict mapping bridge port to list of (MAC address, VLAN) tuples
    """
    mac_table = defaultdict(list)

    for vlan in vlans:
        try:
            # Cisco-specific: append VLAN to community string
            vlan_community = f"{community}@{vlan}"

            # Get MAC address forwarding database
            cmd = [
                'snmpwalk',
                '-v', SNMP_VERSION,
                '-c', vlan_community,
                target_ip,
                MAC_FDB_PORT_OID
            ]

            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=10
            )

            if result.returncode != 0:
                continue

            # Parse output
            # Format: BRIDGE-MIB::dot1dTpFdbPort.0.12.144.12.34.56 = INTEGER: 23
            # The MAC address is encoded in the OID itself
            for line in result.stdout.strip().split('\n'):
                if not line or 'No Such' in line:
                    continue

                # Extract MAC from OID and bridge port number
                # OID format: ...17.4.3.1.2.MAC_OCTETS = INTEGER: BRIDGE_PORT
                match = re.search(r'\.(\d+)\.(\d+)\.(\d+)\.(\d+)\.(\d+)\.(\d+)\s*=\s*(?:INTEGER:\s*)?(\d+)', line)
                if match:
                    mac_octets = [int(match.group(i)) for i in range(1, 7)]
                    mac_address = ':'.join(f'{octet:02x}' for octet in mac_octets)
                    bridge_port = match.group(7)
                    mac_table[bridge_port].append((mac_address, vlan))

        except subprocess.TimeoutExpired:
            print(f"Timeout querying VLAN {vlan}", file=sys.stderr)
            continue
        except Exception as e:
            print(f"Exception querying VLAN {vlan}: {e}", file=sys.stderr)
            continue

    return dict(mac_table)


def get_bridge_port_mapping(target_ip, community):
    """
    Query SNMP to get bridge port to interface index mapping

    Args:
        target_ip: IP address of the switch
        community: SNMP community string

    Returns:
        Dict mapping bridge port to interface index
    """
    bridge_map = {}

    try:
        cmd = [
            'snmpwalk',
            '-v', SNMP_VERSION,
            '-c', community,
            target_ip,
            BRIDGE_PORT_IF_INDEX_OID
        ]

        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=10
        )

        if result.returncode != 0:
            print(f"snmpwalk bridge port mapping failed: {result.stderr}", file=sys.stderr)
            return {}

        # Parse output
        # Format: BRIDGE-MIB::dot1dBasePortIfIndex.23 = INTEGER: 10023
        for line in result.stdout.strip().split('\n'):
            if not line:
                continue

            match = re.search(r'\.(\d+)\s*=\s*(?:INTEGER:\s*)?(\d+)', line)
            if match:
                bridge_port = match.group(1)
                if_index = match.group(2)
                bridge_map[bridge_port] = if_index

        return bridge_map

    except subprocess.TimeoutExpired:
        print(f"Timeout querying bridge port mapping from {target_ip}", file=sys.stderr)
        return {}
    except Exception as e:
        print(f"Exception occurred getting bridge port mapping: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        return {}


def main():
    """Main function to query and display interfaces with MAC addresses"""
    print(f"Querying switch at {SWITCH_IP}...")
    print(f"Using SNMP community: {SNMP_COMMUNITY}")
    print("-" * 80)

    # Try ifName first (newer MIB, better naming)
    interfaces = get_interfaces(SWITCH_IP, SNMP_COMMUNITY, use_descr=False)

    # Fallback to ifDescr if ifName returns nothing
    if not interfaces:
        print("No results from ifName, trying ifDescr...")
        interfaces = get_interfaces(SWITCH_IP, SNMP_COMMUNITY, use_descr=True)

    if not interfaces:
        print("\nNo interfaces found. Check:")
        print("  1. SNMP community string is correct")
        print("  2. Switch is reachable (try: ping 10.110.101.6)")
        print("  3. SNMP is enabled on the switch")
        print("  4. Network firewall allows SNMP (UDP port 161)")
        return 1

    print("Querying VLANs...")
    vlans = get_vlans(SWITCH_IP, SNMP_COMMUNITY)
    print(f"Found VLANs: {', '.join(vlans)}")

    print("Querying MAC address table...")
    mac_table = get_mac_addresses(SWITCH_IP, SNMP_COMMUNITY, vlans)

    print("Querying bridge port mappings...")
    bridge_map = get_bridge_port_mapping(SWITCH_IP, SNMP_COMMUNITY)

    # Map MAC addresses to interface indices
    if_index_to_macs = defaultdict(list)
    for bridge_port, mac_vlan_list in mac_table.items():
        if_index = bridge_map.get(bridge_port)
        if if_index:
            if_index_to_macs[if_index].extend(mac_vlan_list)

    # Display results
    print(f"\nFound {len(interfaces)} interface(s):\n")
    print(f"{'Index':<8} {'Interface':<15} {'MAC Address':<20} {'VLAN'}")
    print("-" * 80)

    for if_index, if_name in sorted(interfaces, key=lambda x: int(x[0])):
        mac_vlan_list = if_index_to_macs.get(if_index, [])
        if mac_vlan_list:
            # Print first MAC on same line as interface
            mac, vlan = mac_vlan_list[0]
            print(f"{if_index:<8} {if_name:<15} {mac:<20} {vlan}")
            # Print additional MACs indented
            for mac, vlan in mac_vlan_list[1:]:
                print(f"{'':<24} {mac:<20} {vlan}")
        else:
            print(f"{if_index:<8} {if_name:<15} (no MACs)")

    # Summary
    total_macs = sum(len(macs) for macs in if_index_to_macs.values())
    ports_with_macs = sum(1 for macs in if_index_to_macs.values() if macs)
    print(f"\nTotal MAC addresses: {total_macs}")
    print(f"Ports with MACs: {ports_with_macs}/{len(interfaces)}")

    return 0


if __name__ == '__main__':
    sys.exit(main())
