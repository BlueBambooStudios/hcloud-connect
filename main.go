package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bluebamboostudios/hcloud-connect/hconnect"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"

	_ "k8s.io/component-base/metrics/prometheus/clientgo" // load all the prometheus client-go plugins
	_ "k8s.io/component-base/metrics/prometheus/version"  // for version metric registration
)

func main() {
	rand.Seed(time.Now().UnixNano())

	logs.InitLogs()
	defer logs.FlushLogs()

	cloud, err := hconnect.NewCloud()
	if err != nil {
		klog.ErrorS(err, "hcloud-connect failed")
		os.Exit(1)
	}

	// on start register with the load balancer
	if err := register(cloud); err != nil {
		klog.ErrorS(err, "hcloud-connect failed")
		os.Exit(1)
	}

	fmt.Printf("Hetzner Cloud k8s connect %s started\n", hconnect.ProviderVersion)
	klog.InfoS("hcloud-connect waiting for shutdown...")

	// wait for stop signal then unregister
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c

	klog.InfoS("Shutting down ...")

	if err := deregister(cloud); err != nil {
		klog.ErrorS(err, "hcloud-connect failed")
		os.Exit(1)
	}

	klog.InfoS("Shutdown successful!")
}

// Registers everything (startup)
func register(c *hconnect.Cloud) error {
	var err error

	server, err := getServer(c)
	if err != nil {
		return err
	}

	err = c.LoadBalancer.Register(c, server)
	if err != nil && !strings.Contains(err.Error(), "target_already_defined") {
		return err
	}

	err = c.Firewall.Register(c, server)
	if err != nil {
		return err
	}

	return nil
}

// Deregisters everything (prepares for shutdown)
func deregister(c *hconnect.Cloud) error {
	var err error

	server, err := getServer(c)
	if err != nil {
		return err
	}

	err = c.LoadBalancer.Deregister(c, server)
	if err != nil {
		return err
	}

	err = c.Firewall.Deregister(c, server)
	if err != nil {
		return err
	}

	return nil
}

// Gets the hetzner api representation of this node
// 'c' is the cloud instance from hconnect.NewCloud()
func getServer(c *hconnect.Cloud) (server *hcloud.Server, err error) {
	server, _, err = c.Client.Server.GetByName(context.Background(), c.NodeName)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, fmt.Errorf("instance not found")
	}

	return server, nil
}
