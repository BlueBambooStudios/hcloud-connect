package hconnect

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"k8s.io/klog/v2"
)

const (
	hcloudFirewallResENVVar = "HCLOUD_FIREWALL_RESOURCE"
)

type Firewall struct {
	// IDs of firewalls that the current node will be
	// added to as a resource
	firwallResIDs []int
}

// Fetches firewall ids from the user provided name/ids
// in the environment variable
// 'c' is the hetzner cloud client
func newFirewall(c *hcloud.Client) (*Firewall, error) {
	const op = "hcloud-connect/newFirewall"
	var err error
	firewalls, err := getFirewalls(c)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var fwResIDs []int = nil
	if v, ok := os.LookupEnv(hcloudFirewallResENVVar); ok {
		fwResIDs, err = inputToIDs(v, firewalls)

		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
	}

	return &Firewall{
		firwallResIDs: fwResIDs,
	}, nil
}

// Registers this node with firewalls
// 'f' the firewall instance from cloud.Firewall
// 'c' is the cloud instance from hconnect.NewCloud()
// 'server' is the representation of this node via hetzner cloud
func (f *Firewall) Register(c *Cloud, server *hcloud.Server) error {
	const op = "hcloud-connect/registerFirewall"
	firewalls, err := getFirewalls(c.Client)
	if err != nil {
		return err
	}

	if f.firwallResIDs != nil {
		if err := f.registerFirewallsRes(c, server, firewalls); err != nil {
			return err
		}
	}

	return nil
}

// Deregisters all firewall related things regarding this node that were
// registered during 'Register'. This function will fail softly by only
// printing errors to console withot aborting execution. This method will
// clean up the most stuff even if there are errors
// 'f' the firewall instance from cloud.Firewall
// 'c' is the cloud instance from hconnect.NewCloud()
// 'server' is the representation of this node via hetzner cloud
func (f *Firewall) Deregister(c *Cloud, server *hcloud.Server) error {
	const op = "hcloud-connect/registerFirewall"
	firewalls, err := getFirewalls(c.Client)
	if err != nil {
		return err
	}

	if f.firwallResIDs != nil {
		if err := f.deregisterFirewallsRes(c, server, firewalls); err != nil {
			return err
		}
	}

	return nil
}

// Gets all available firewalls via the hetzner cloud api
// and maps them by their id.
// 'c' is the cloud instance from hconnect.NewCloud()
// This returns a pointer to a map
func getFirewalls(c *hcloud.Client) (*map[int]*hcloud.Firewall, error) {
	firewalls, err := c.Firewall.All(context.Background())

	if err != nil {
		return nil, err
	}

	out := make(map[int]*hcloud.Firewall)

	for _, firewall := range firewalls {
		out[firewall.ID] = firewall
	}

	return &out, nil
}

// Converts the comma separated input string to an array and tries to
// Find the firewalls that match to the provided values. This function will
// Fail when a firewall cannot be found!
// 'input' the input string with comma separated values
// 'firewalls' the pointer to the firewall map from 'getFirewalls'
// This will return an array of firewall ids as integers
func inputToIDs(input string, firewalls *map[int]*hcloud.Firewall) (ids []int, err error) {
	splitted := strings.Split(strings.ToLower(input), ",")

OUTER:
	for _, split := range splitted {
		for key, value := range *firewalls {
			lowerName := strings.ToLower(value.Name)
			idString := strconv.Itoa(key)

			if split == lowerName || split == idString {
				ids = append(ids, key)
				continue OUTER
			}
		}

		return nil, fmt.Errorf("Firewall %s not found", split)
	}

	return ids, nil
}

// Checks whether a backoff should occur because of a conflict or lock returned
// by the api.
// 'err' the error to check
// Returns true when a backoff is required
func checkConflictOrLockBackoff(err error) bool {
	if !hcloud.IsError(err, hcloud.ErrorCodeLocked) && !hcloud.IsError(err, hcloud.ErrorCodeConflict) {
		return false
	}

	return true
}

// Backoff function that logs information about the backoff to console and waits
// a specified amount of time.
// 'op' the current operation
// 'cause' the cause of why a backoff is required
// 'delay' the time to wait
// 'err' the error that caused the backoff
func errBackoff(op string, cause string, delay time.Duration, err error) {
	klog.InfoS("retry due to ", cause, " op: ", op, " delay: ", fmt.Sprintf("%v", delay), " err: ", fmt.Sprintf("%v", err))
	time.Sleep(delay)
}

// Checks if a 'firewall' already has 'server' as a resource
// Returns true if the 'server' is already a resource of 'firewall'
func firewallHasResource(server *hcloud.Server, firewall *hcloud.Firewall) bool {
	for _, res := range firewall.AppliedTo {
		if res.Server.ID == server.ID {
			return true
		}
	}

	return false
}
