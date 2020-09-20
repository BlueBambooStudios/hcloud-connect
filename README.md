# Hetzner Cloud Connect

Handles automatically adding servers to load balancers

## Usage

All configuration is passed with environment variables. We recommend storing these variables as secrets.

First create a secret containing your hetzner credentials:

```
---
apiVersion: v1
kind: Secret
metadata:
  name: hcloud
  namespace: kube-system
type: Opaque
stringData:
  token: "{HETZNER_API_TOKEN}"
  loadBalancer: "{LOAD_BALANCER_ID}"
```

Then deploy the daemonset to your cluster:

```
kubectl apply -f https://raw.githubusercontent.com/BlueBambooStudios/hcloud-connect/master/deployment/daemonset.yaml
```

### Environment variables

|          **Key**           |      **Type**      | **Default** |                **Description**                 |
| :------------------------: | :----------------: | :---------: | :--------------------------------------------: |
|        HCLOUD_TOKEN        |       String       |             |               Hetzner API token                |
|      HCLOUD_ENDPOINT       | String (Optional)  |             |    Optional endpoint URL for Hetzner Cloud     |
|        HCLOUD_DEBUG        | Boolean (Optional) |    FALSE    |              Enable debug loggin               |
|    HCLOUD_LOAD_BALANCER    |       String       |             |                Load balancer id                |
| HCLOUD_USE_PRIVATE_NETWORK | Boolean (Optional) |    FALSE    | Use the private network when attaching targets |
|         NODE_NAME          |       String       |             |  Node name as shown in Hetzner control panel   |

### N.B.

There are no tests, use at your own peril
