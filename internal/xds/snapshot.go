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

package xds

import (
	"strconv"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/nfowl/quilkin-controller/internal/store"
)

const (
	ClusterName  = ""
	UpstreamHost = "127.0.0.1"
	UpstreamPort = 3000
)

var (
	nodeVersions map[string]int
)

func makeCluster(clusterName string, node store.NodeConfig) *cluster.Cluster {
	return &cluster.Cluster{
		Name:                 clusterName,
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_STATIC},
		LoadAssignment:       makeClusterLoadAssignment(clusterName, node),
	}
}

func makeClusterLoadAssignment(clusterName string, node store.NodeConfig) *endpoint.ClusterLoadAssignment {
	endpoints := make([]*endpoint.LbEndpoint, 0)
	for _, receiver := range node.Endpoints {
		endpoints = append(endpoints, &endpoint.LbEndpoint{HostIdentifier: &endpoint.LbEndpoint_Endpoint{Endpoint: makeEndpoint(receiver.Address, uint32(receiver.Port))}})
	}
	return &endpoint.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			LbEndpoints: endpoints,
		}},
	}
}

func makeEndpoint(host string, port uint32) *endpoint.Endpoint {
	return &endpoint.Endpoint{
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_UDP,
					Address:  host,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: port,
					},
				},
			},
		},
	}
}

func generateNodeSnapshot(node store.NodeConfig) cache.Snapshot {
	val, ok := nodeVersions[node.ProxyName]
	if !ok {
		val = 1
	}
	// resources := cache.SnapshotResources{}
	clusterResources := make([]types.Resource, 1)
	clusterResources = append(clusterResources, makeCluster("", node))
	snapshot := cache.NewSnapshot(strconv.Itoa(val),
		[]types.Resource{}, // endpoints
		clusterResources,
		[]types.Resource{}, // routes
		[]types.Resource{}, // listeners
		[]types.Resource{}, // runtimes
		[]types.Resource{}, // secrets
	)
	val++
	nodeVersions[node.ProxyName] = val
	return snapshot
}
