# quilkin-controller

simple kubernetes control plane for quilkin udp proxy that uses pod annotations to add endpoints and sidecar containers where specified.

## Configuration

`nfowler.dev/quilkin.receiver: "proxy:4000"`: Indicates the pod wants to receive data from the node name provided at the port specified.
`nfowler.dev/quilkin.sender: "proxy"`: Injects a quilkin proxy with a name corresponding to the value provided.

## Notes

This is a bit of a POC/Hack atm and doesn't cleanup after itself very well.
