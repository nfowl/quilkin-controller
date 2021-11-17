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

package store

import (
	"sync"

	"go.uber.org/zap"
)

// SotwStore stores the State of the World as the controller sees it.
// It contains channels that can be used to communicate with the xds cache on changes.
// This is thread safe as every modification is done behind a mutex.
type SotwStore struct {
	mu          sync.Mutex //FIXME Consider using smarter method of updating this to avoid mutex abuse
	Nodes       map[string]*NodeConfig
	nodeUpdates chan NodeConfig
	nodeDeletes chan string
	logger      *zap.SugaredLogger
}

func NewSotWStore(updates chan NodeConfig, deletes chan string, logger *zap.SugaredLogger) *SotwStore {
	nodes := make(map[string]*NodeConfig)
	return &SotwStore{Nodes: nodes, nodeUpdates: updates, nodeDeletes: deletes, logger: logger}
}

type NodeConfig struct {
	Endpoints map[string]*Endpoint
	ProxyName string
	senders   map[string]struct{}
}

type Endpoint struct {
	Address string
	Port    int
}

func (s *SotwStore) AddReceiver(proxyName string, port int, address string, podName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.Nodes[proxyName]
	if !ok {
		//Making new NodeConfiguration
		endpoints := make(map[string]*Endpoint)
		senders := make(map[string]struct{})
		endpoints[podName] = &Endpoint{Address: address, Port: port}
		value = &NodeConfig{Endpoints: endpoints, ProxyName: proxyName, senders: senders}
		s.Nodes[proxyName] = value
	} else {
		value.Endpoints[podName] = &Endpoint{Address: address, Port: port}
	}
	s.logger.Infow("Added receiver endpoint", "node", proxyName, "endpoints", value.Endpoints)
	s.nodeUpdates <- *value
}

func (s *SotwStore) AddSender(proxyName string, podName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.Nodes[proxyName]
	if !ok {
		endpoints := make(map[string]*Endpoint)
		senders := make(map[string]struct{})
		value = &NodeConfig{ProxyName: proxyName, Endpoints: endpoints, senders: senders}
		s.Nodes[proxyName] = value
	}
	value.senders[podName] = struct{}{}
	s.logger.Infow("Added sender", "name", proxyName, "remaining", len(value.senders))
	s.nodeUpdates <- *value
}

// RemoveReceiver deletes a receiver from a node if it exists.
// The xds server is notified of the change if one occurs
func (s *SotwStore) RemoveReceiver(proxyName string, podName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.Nodes[proxyName]
	if ok {
		delete(value.Endpoints, podName)
		s.logger.Infow("Deleting receiver endpoint", "proxyName", proxyName, "receiver", podName)
		if len(value.senders) == 0 && len(value.Endpoints) == 0 {
			delete(s.Nodes, proxyName)
			return
		}
		s.nodeUpdates <- *s.Nodes[proxyName]
	}
}

// RemoveSender removes a quilkin proxy node/sender and returns whether or not its the last instance
// using that proxyName. If a node is removed the xds server is notified of the change.
func (s *SotwStore) RemoveSender(proxyName string, podName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	node, ok := s.Nodes[proxyName]
	if ok {
		delete(node.senders, podName)
		s.logger.Infow("removed sender", "name", proxyName, "remaining", len(node.senders))
		if len(node.senders) <= 0 {
			// Only delete the nodeconfig if all receivers are also empty
			if len(node.Endpoints) == 0 {
				delete(s.Nodes, proxyName)
			}
			// s.nodeDeletes <- proxyName
			return true
		}
	}
	return false
}
