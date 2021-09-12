# quilkin-controller

simple kubernetes control plane for [quilkin](https://github.com/googleforgames/quilkin) udp proxy that uses pod annotations to add endpoints and sidecar containers where specified.

:warning: **USAGE WARNING** :warning:
This project is not production ready use at own risk. Breaking changes may be introduced at a moments notice. it is also lacking any meaningful tests at this point in time.

## Usage

To configure the client pods add the following annotations to the pod annotation spec in your deployment.
see the [examples](examples) folder for deployments you can use to test it.

`nfowler.dev/quilkin.receiver: "proxy:4000"`: Indicates the pod wants to receive data from the node name provided at the port specified.
`nfowler.dev/quilkin.sender: "proxy"`: Injects a quilkin proxy with a name corresponding to the value provided.

## Installation
