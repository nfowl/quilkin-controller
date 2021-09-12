package quilkin

import "os"

type ProxyConfig struct {
	Id   string `yaml:"id"`
	Port int    `yaml:"port"`
}

type DynamicConfig struct {
	ManagementServers []*Address `yaml:"management_servers"`
}

type Address struct {
	Address string `yaml:"address"`
}

type AdminConfig struct {
	Address string `yaml:"address"`
}

type QuilkinConfig struct {
	Version string        `yaml:"version"`
	Proxy   ProxyConfig   `yaml:"proxy"`
	Admin   AdminConfig   `yaml:"admin"`
	Dynamic DynamicConfig `yaml:"dynamic"`
}

func NewQuilkinConfig(proxyName string) QuilkinConfig {
	return QuilkinConfig{
		Version: "v1alpha1",
		Proxy:   ProxyConfig{Id: proxyName, Port: 7000},
		Admin:   AdminConfig{Address: "[::]:9091"},
		Dynamic: DynamicConfig{ManagementServers: []*Address{{Address: "http://" + os.Getenv("SVC_NAME") + "." + os.Getenv("POD_NAMESPACE") + ".svc.cluster.local:18000"}}},
	}
}
