# Hetzner Cloud Connect

Handles automatically adding servers to load balancers

## Usage

All configuration is passed with environment variables. We recommend storing these variables as secrets.

Launch as a daemon set, see `deployment/daemonset.yaml`.

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
