package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"k8s.io/klog/v2"
)

const (
	hcloudTokenENVVar           = "HCLOUD_TOKEN"
	hcloudEndpointENVVar        = "HCLOUD_ENDPOINT"
	hcloudDebugENVVar           = "HCLOUD_DEBUG"
	hcloudLoadBalancerENVVar    = "HCLOUD_LOAD_BALANCER"
	hcloudPrivateNetworksENVVar = "HCLOUD_USE_PRIVATE_NETWORK"
	nodeNameENVVar              = "NODE_NAME"
	providerVersion             = "v1.0.0"
)

type cloud struct {
	client         *hcloud.Client
	loadBalancerID int
	nodeName       string
	privateNetwork bool
}

func newCloud() (*cloud, error) {
	const op = "hcloud-connect/newCloud"

	token := os.Getenv(hcloudTokenENVVar)
	if token == "" {
		return nil, fmt.Errorf("environment variable %q is required", hcloudTokenENVVar)
	}
	if len(token) != 64 {
		return nil, fmt.Errorf("entered token is invalid (must be exactly 64 characters long)")
	}

	nodeName := os.Getenv(nodeNameENVVar)
	if nodeName == "" {
		return nil, fmt.Errorf("environment variable %q is required", nodeNameENVVar)
	}

	opts := []hcloud.ClientOption{
		hcloud.WithToken(token),
		hcloud.WithApplication("hcloud-connect", providerVersion),
	}
	if os.Getenv(hcloudDebugENVVar) == "true" {
		opts = append(opts, hcloud.WithDebugWriter(os.Stderr))
	}
	if endpoint := os.Getenv(hcloudEndpointENVVar); endpoint != "" {
		opts = append(opts, hcloud.WithEndpoint(endpoint))
	}
	client := hcloud.NewClient(opts...)

	var loadBalancerID int
	if v, ok := os.LookupEnv(hcloudLoadBalancerENVVar); ok {
		n, _, err := client.LoadBalancer.Get(context.Background(), v)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		if n == nil {
			return nil, fmt.Errorf("%s: Load Balancer %s not found", op, v)
		}
		loadBalancerID = n.ID
	}
	if loadBalancerID == 0 {
		klog.InfoS("%s: %s empty", op, hcloudLoadBalancerENVVar)
	}

	_, _, err := client.Server.List(context.Background(), hcloud.ServerListOpts{})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	fmt.Printf("Hetzner Cloud k8s connect %s started\n", providerVersion)

	return &cloud{
		client:         client,
		loadBalancerID: loadBalancerID,
		nodeName:       nodeName,
		privateNetwork: os.Getenv(hcloudPrivateNetworksENVVar) != "",
	}, nil
}

func (c *cloud) Register() error {
	const op = "hcloud-connect/register"

	server, _, err := c.client.Server.GetByName(context.Background(), c.nodeName)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if server == nil {
		return fmt.Errorf("instance not found")
	}

	lb, _, err := c.client.LoadBalancer.GetByID(context.Background(), c.loadBalancerID)
	if err != nil {
		return err
	}

	opts := hcloud.LoadBalancerAddServerTargetOpts{
		Server:       server,
		UsePrivateIP: hcloud.Bool(c.privateNetwork),
	}

	_, _, err = c.client.LoadBalancer.AddServerTarget(context.Background(), lb, opts)
	if err != nil {
		return err
	}

	return nil
}

func (c *cloud) Deregister() error {
	const op = "hcloud-connect/deregister"

	server, _, err := c.client.Server.GetByName(context.Background(), c.nodeName)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if server == nil {
		return fmt.Errorf("instance not found")
	}

	lb, _, err := c.client.LoadBalancer.GetByID(context.Background(), c.loadBalancerID)
	if err != nil {
		return err
	}

	_, _, err = c.client.LoadBalancer.RemoveServerTarget(context.Background(), lb, server)
	if err != nil {
		return err
	}

	return nil
}
