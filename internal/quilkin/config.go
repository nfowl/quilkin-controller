/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
