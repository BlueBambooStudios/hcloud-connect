package main

import (
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/bluebamboostudios/hcloud-connect/hconnect"
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

	klog.InfoS("hcloud-connect waiting for shutdown...")

	// wait for stop signal then unregister
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c

	if err := deregister(cloud); err != nil {
		klog.ErrorS(err, "hcloud-connect failed")
		os.Exit(1)
	}
}

func register(c *hconnect.Cloud) error {
	var err error
	err = c.LoadBalancer.Register(c)

	if err != nil && !strings.Contains(err.Error(), "target_already_defined") {
		return err
	}

	return nil
}

func deregister(c *hconnect.Cloud) error {
	var err error
	err = c.LoadBalancer.Register(c)

	if err != nil {
		return err
	}

	return nil
}
