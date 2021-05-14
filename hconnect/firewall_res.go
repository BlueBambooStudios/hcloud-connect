package hconnect

import (
	"context"
	"fmt"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"k8s.io/klog/v2"
)

// Registers this node to all specified firewalls as resource
// 'f' the firewall instance from cloud.Firewall
// 'server' is the representation of this node via hetzner cloud
// 'fwPointer' is a pointer to a map mapping the firewall ids to
// the firewall instances from hetzner cloud
func (f *Firewall) registerFirewallsRes(c *Cloud, server *hcloud.Server, fwPointer *map[int]*hcloud.Firewall) error {
	const op = "hcloud-connect/registerFirewallsRes"
	firewalls := *fwPointer

	for _, id := range f.firwallResIDs {
		firewall, ok := firewalls[id]

		if !ok {
			return fmt.Errorf("Firewall %d not found", id)
		}

		if firewallHasResource(server, firewall) {
			continue
		}

		if err := applyFirewallRes(op, c, server, firewall); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	return nil
}

// Applies this node as a resource to a specific firewall
// If the api rate-limit is full or a conflict is returned this function will
// wait 5 seconds before trying again.
// 'op' is the current operation name
// 'c' is the cloud instance from hconnect.NewCloud()
// 'server' is the representation of this node via hetzner cloud
// 'firewall' is the firewall instance that this node shall be added to as resource
func applyFirewallRes(op string, c *Cloud, server *hcloud.Server, firewall *hcloud.Firewall) error {
	actions, _, err := c.Client.Firewall.ApplyResources(context.Background(), firewall, []hcloud.FirewallResource{
		{
			Type: hcloud.FirewallResourceTypeServer,
			Server: &hcloud.FirewallResourceServer{
				ID: server.ID,
			},
		},
	})

	if err != nil {
		if checkConflictOrLockBackoff(err) {
			errBackoff(fmt.Sprintf("%s/%d", op, firewall.ID), "conflict or lock", time.Second*5, err)
			return applyFirewallRes(op, c, server, firewall)
		}

		return err
	}

	return WatchAction(context.Background(), &c.Client.Action, actions[0])
}

// Removes this node as resource from all specified firewalls
// 'f' the firewall instance from cloud.Firewall
// 'server' is the representation of this node via hetzner cloud
// 'fwPointer' is a pointer to a map mapping the firewall ids to
// the firewall instances from hetzner cloud
func (f *Firewall) deregisterFirewallsRes(c *Cloud, server *hcloud.Server, fwPointer *map[int]*hcloud.Firewall) error {
	const op = "hcloud-connect/deregisterFirewallsRes"
	firewalls := *fwPointer

	for _, id := range f.firwallResIDs {
		firewall, ok := firewalls[id]

		if !ok {
			klog.Error("Firewall not found! Skipping ...", " op: ", fmt.Sprintf("%s/%d", op, firewall.ID))
			continue
		}

		if !firewallHasResource(server, firewall) {
			continue
		}

		if err := removeFirewallRes(op, c, server, firewall); err != nil {
			klog.Error("Error while updating Firewall! Skipping ...",
				" op: ", fmt.Sprintf("%s/%d", op, firewall.ID),
				" err: ", fmt.Sprintf("%v", err),
			)
			continue
		}
	}

	return nil
}

// Removes this node as a resource from a specific firewall
// If the api rate-limit is full or a conflict is returned this function will
// wait 2 seconds before trying again.
// 'op' is the current operation name
// 'c' is the cloud instance from hconnect.NewCloud()
// 'server' is the representation of this node via hetzner cloud
// 'firewall' is the firewall instance that this node shall be removed from as resource
func removeFirewallRes(op string, c *Cloud, server *hcloud.Server, firewall *hcloud.Firewall) error {
	actions, _, err := c.Client.Firewall.RemoveResources(context.Background(), firewall, []hcloud.FirewallResource{
		{
			Type: hcloud.FirewallResourceTypeServer,
			Server: &hcloud.FirewallResourceServer{
				ID: server.ID,
			},
		},
	})

	if err != nil {
		if checkConflictOrLockBackoff(err) {
			errBackoff(fmt.Sprintf("%s/%d", op, firewall.ID), "conflict or lock", time.Second*2, err)
			return removeFirewallRes(op, c, server, firewall)
		}

		return err
	}

	return WatchAction(context.Background(), &c.Client.Action, actions[0])
}
