package store

var InMemoryConfig = map[string]NodeConfig{}

type NodeConfig struct {
	Endpoints map[string]Endpoint
}

type Endpoint struct {
	Address string
	Port    int
}

func AddReceiver(proxyName string, port int, address string, podName string) {
	value, ok := InMemoryConfig[proxyName]
	if !ok {
		//Making new NodeConfiguration
		endpoints := make(map[string]Endpoint)
		endpoints[podName] = Endpoint{Address: address, Port: port}
		InMemoryConfig[proxyName] = NodeConfig{Endpoints: endpoints}
	}
	value.Endpoints[podName] = Endpoint{Address: address, Port: port}
}

func AddSender(proxyName string) {
	_, ok := InMemoryConfig[proxyName]
	if !ok {
		InMemoryConfig[proxyName] = NodeConfig{}
	}
}
