package hconnect

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"k8s.io/klog/v2"
)

// Registers this nodes IP-Adresses to all rules of all specified Firewalls.
// 'c' is the cloud instance from hconnect.NewCloud()
// 'server' is the representation of this node via hetzner cloud
// 'fwPointer' is a pointer to a map mapping the firewall ids to the firewall instances
func (f *Firewall) registerFirewallsTarget(c *Cloud, server *hcloud.Server, fwPointer *map[int]*hcloud.Firewall) error {
	const op = "hcloud-connect/registerFirewallsTarget"
	firewalls := *fwPointer
	ip4 := server.PublicNet.IPv4.IP
	ip6 := server.PublicNet.IPv6.IP

	ip4Net := net.IPNet{
		IP:   ip4,
		Mask: net.CIDRMask(32, 32),
	}

	ip6Net := net.IPNet{
		IP:   ip6,
		Mask: net.CIDRMask(64, 128),
	}

	for _, id := range f.firewallTargetsIDs {
		firewall, ok := firewalls[id]

		if !ok {
			return fmt.Errorf("Firewall %d not found", id)
		}

		if err := f.applyServerToRules(op, c, &ip4, &ip4Net, &ip6, &ip6Net, firewall); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	return nil
}

// Applies the IP-Adresses of this server to all rules of the specified firewall
// If the api rate-limit is full or a conflict is returned this function will
// wait 5 seconds before trying again.
// 'op' the current operation
// 'ip4' the IPv4 address of this server
// 'ip4Net' the IPv4 of this server as /32 network
// 'ip6' the IPv6 address of this server
// 'ip6Net' the IPv6 of this server as /64 network
// 'firewall' the firewall that contains the rules the ip addresses will get addded to
func (f *Firewall) applyServerToRules(op string, c *Cloud, ip4 *net.IP, ip4Net *net.IPNet, ip6 *net.IP, ip6Net *net.IPNet, firewall *hcloud.Firewall) error {
	var newRules []hcloud.FirewallRule
	for _, rule := range firewall.Rules {
		source := rule.SourceIPs
		dest := rule.DestinationIPs
		if rule.Direction == hcloud.FirewallRuleDirectionIn {
			if !addressInNetSlice(&source, ip4) {
				source = append(source, *ip4Net)
			}

			if f.firewallTargetsIP6 && !addressInNetSlice(&source, ip6) {
				source = append(source, *ip6Net)
			}
		} else {
			if !addressInNetSlice(&dest, ip4) {
				dest = append(source, *ip4Net)
			}

			if f.firewallTargetsIP6 && !addressInNetSlice(&dest, ip6) {
				dest = append(source, *ip6Net)
			}
		}

		newRules = append(newRules, hcloud.FirewallRule{
			Direction:      rule.Direction,
			SourceIPs:      source,
			DestinationIPs: dest,
			Protocol:       rule.Protocol,
			Port:           rule.Port,
		})
	}

	action, _, err := c.Client.Firewall.SetRules(context.Background(), firewall, hcloud.FirewallSetRulesOpts{
		Rules: newRules,
	})

	if err != nil {
		if checkConflictOrLockBackoff(err) {
			errBackoff(fmt.Sprintf("%s/%d", op, firewall.ID), "conflict or lock", time.Second*5, err)
			return f.applyServerToRules(op, c, ip4, ip4Net, ip6, ip6Net, firewall)
		}

		return err
	}

	return WatchAction(context.Background(), &c.Client.Action, action[0])
}

// Deregisters this nodes IP-Adresses from all rules of all specified firewalls.
// 'c' is the cloud instance from hconnect.NewCloud()
// 'server' is the representation of this node via hetzner cloud
// 'fwPointer' is a pointer to a map mapping the firewall ids to the firewall instances
func (f *Firewall) deregisterFirewallsTarget(c *Cloud, server *hcloud.Server, fwPointer *map[int]*hcloud.Firewall) error {
	const op = "hcloud-connect/deregisterFirewallsTarget"
	firewalls := *fwPointer
	ip4 := server.PublicNet.IPv4.IP
	ip6 := server.PublicNet.IPv6.IP

	ip4Net := net.IPNet{
		IP:   ip4,
		Mask: net.CIDRMask(32, 32),
	}

	ip6Net := net.IPNet{
		IP:   ip6,
		Mask: net.CIDRMask(64, 128),
	}

	for _, id := range f.firewallTargetsIDs {
		firewall, ok := firewalls[id]

		if !ok {
			klog.Error("Firewall not found! Skipping ...", " op: ", fmt.Sprintf("%s/%d", op, firewall.ID))
			continue
		}

		if err := f.removeServerFromRules(op, c, &ip4, &ip4Net, &ip6, &ip6Net, firewall); err != nil {
			klog.Error("Error while updating Firewall rules! Skipping ...",
				" op: ", fmt.Sprintf("%s/%d", op, firewall.ID),
				" err: ", fmt.Sprintf("%v", err),
			)
			continue
		}
	}

	return nil
}

// Removes the IP-Adresses of this server from all rules of the specified firewall
// If the api rate-limit is full or a conflict is returned this function will
// wait 5 seconds before trying again.
// 'op' the current operation
// 'ip4' the IPv4 address of this server
// 'ip4Net' the IPv4 of this server as /32 network
// 'ip6' the IPv6 address of this server
// 'ip6Net' the IPv6 of this server as /64 network
// 'firewall' the firewall that contains the rules the ip addresses will get removed from
func (f *Firewall) removeServerFromRules(op string, c *Cloud, ip4 *net.IP, ip4Net *net.IPNet, ip6 *net.IP, ip6Net *net.IPNet, firewall *hcloud.Firewall) error {
	var newRules []hcloud.FirewallRule
	for _, rule := range firewall.Rules {
		source := rule.SourceIPs
		dest := rule.DestinationIPs
		if rule.Direction == hcloud.FirewallRuleDirectionIn {
			if addressInNetSlice(&source, ip4) {
				source = removeNetFromSlice(source, ip4Net)
			}

			if f.firewallTargetsIP6 && addressInNetSlice(&source, ip6) {
				source = removeNetFromSlice(source, ip6Net)
			}
		} else {
			if addressInNetSlice(&dest, ip4) {
				dest = removeNetFromSlice(dest, ip4Net)
			}

			if f.firewallTargetsIP6 && addressInNetSlice(&dest, ip6) {
				dest = removeNetFromSlice(dest, ip6Net)
			}
		}

		newRules = append(newRules, hcloud.FirewallRule{
			Direction:      rule.Direction,
			SourceIPs:      source,
			DestinationIPs: dest,
			Protocol:       rule.Protocol,
			Port:           rule.Port,
		})
	}

	action, _, err := c.Client.Firewall.SetRules(context.Background(), firewall, hcloud.FirewallSetRulesOpts{
		Rules: newRules,
	})

	if err != nil {
		if checkConflictOrLockBackoff(err) {
			errBackoff(fmt.Sprintf("%s/%d", op, firewall.ID), "conflict or lock", time.Second*2, err)
			return f.removeServerFromRules(op, c, ip4, ip4Net, ip6, ip6Net, firewall)
		}

		return err
	}

	return WatchAction(context.Background(), &c.Client.Action, action[0])
}

// Checks whether the provided 'needle' is part of the 'haystack'
// Uses 'IPNet.Contains' to do the acutal check.
func addressInNetSlice(haystack *[]net.IPNet, needle *net.IP) bool {
	for _, net := range *haystack {
		if net.Contains(*needle) {
			return true
		}
	}

	return false
}

// Removes a 'needle' (network) from the 'haystack' (slice). For a 'needle'
// to be removed it has to match an entry of 'haystack' exactly.
// This function returns a new slice that doesn't contain the needle
// This function is ignoring the order of the slice
func removeNetFromSlice(haystack []net.IPNet, needle *net.IPNet) []net.IPNet {
	nA, nB := needle.Mask.Size()
	for index, val := range haystack {
		vA, vB := val.Mask.Size()

		if !val.IP.Equal(needle.IP) || nA != vA || nB != vB {
			continue
		}

		haystack[index] = haystack[len(haystack)-1]
		return haystack[:len(haystack)-1]
	}

	return haystack
}
