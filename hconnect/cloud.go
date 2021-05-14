package hconnect

import (
	"context"
	"fmt"
	"os"

	"github.com/hetznercloud/hcloud-go/hcloud"
)

const (
	ProviderVersion      = "v1.0.0"
	hcloudTokenENVVar    = "HCLOUD_TOKEN"
	hcloudEndpointENVVar = "HCLOUD_ENDPOINT"
	hcloudDebugENVVar    = "HCLOUD_DEBUG"
	nodeNameENVVar       = "NODE_NAME"
)

type Cloud struct {
	Client       *hcloud.Client
	NodeName     string
	LoadBalancer *LoadBalancer
	Firewall     *Firewall
}

func NewCloud() (*Cloud, error) {
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
		hcloud.WithApplication("hcloud-connect", ProviderVersion),
	}
	if os.Getenv(hcloudDebugENVVar) == "true" {
		opts = append(opts, hcloud.WithDebugWriter(os.Stderr))
	}
	if endpoint := os.Getenv(hcloudEndpointENVVar); endpoint != "" {
		opts = append(opts, hcloud.WithEndpoint(endpoint))
	}
	client := hcloud.NewClient(opts...)
	var err error

	loadBalancer, err := newLoadBalancer(client)

	if err != nil {
		return nil, err
	}

	firewall, err := newFirewall(client)

	if err != nil {
		return nil, err
	}

	_, _, err = client.Server.List(context.Background(), hcloud.ServerListOpts{})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Cloud{
		Client:       client,
		NodeName:     nodeName,
		LoadBalancer: loadBalancer,
		Firewall:     firewall,
	}, nil
}
