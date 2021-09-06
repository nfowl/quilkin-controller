package quilkin

type ProxyConfig struct {
	Id   string `yaml:"id"`
	Port int    `yaml:"port"`
}

type DynamicConfig struct {
	ManagementServers []*ManagementServer `yaml:"management_servers"`
}

type ManagementServer struct {
	Address string `yaml:"address"`
}

type QuilkinConfig struct {
	Version string        `yaml:"version"`
	Proxy   ProxyConfig   `yaml:"proxy"`
	Dynamic DynamicConfig `yaml:"dynamic"`
}

func NewQuilkinConfig(proxyName string) QuilkinConfig {
	return QuilkinConfig{
		Version: "v1alpha1",
		Proxy:   ProxyConfig{Id: proxyName, Port: 7000},
		Dynamic: DynamicConfig{ManagementServers: []*ManagementServer{{Address: "http://quilkin-controller.quilkin.svc.cluster.local"}}},
	}
}
