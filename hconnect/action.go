/*
Inspired by action.go in hcloud-cloud-controller-manager
https://github.com/hetznercloud/hcloud-cloud-controller-manager/blob/b54847b9163a3fc39dd902ab2f5f0494003b7230/internal/hcops/action.go
*/
package hconnect

import (
	"context"

	"github.com/hetznercloud/hcloud-go/hcloud"
)

type HCloudActionClient interface {
	WatchProgress(ctx context.Context, a *hcloud.Action) (<-chan int, <-chan error)
}

func WatchAction(ctx context.Context, ac HCloudActionClient, a *hcloud.Action) error {
	_, errCh := ac.WatchProgress(ctx, a)
	return <-errCh
}
