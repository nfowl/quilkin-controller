package store

import (
	"sync"

	"go.uber.org/zap"
)

type SoTWStore struct {
	mu          sync.Mutex
	Nodes       map[string]*NodeConfig
	nodeUpdates chan NodeConfig
	nodeDeletes chan string
	logger      *zap.SugaredLogger
}

func NewSoTWStore(updates chan NodeConfig, deletes chan string, logger *zap.SugaredLogger) SoTWStore {
	nodes := make(map[string]*NodeConfig)
	return SoTWStore{Nodes: nodes, nodeUpdates: updates, nodeDeletes: deletes, logger: logger}
}

type NodeConfig struct {
	Endpoints map[string]*Endpoint
	ProxyName string
	count     int
}

type Endpoint struct {
	Address string
	Port    int
}

func (s *SoTWStore) AddReceiver(proxyName string, port int, address string, podName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.Nodes[proxyName]
	if !ok {
		//Making new NodeConfiguration
		endpoints := make(map[string]*Endpoint)
		endpoints[podName] = &Endpoint{Address: address, Port: port}
		value = &NodeConfig{Endpoints: endpoints, ProxyName: proxyName, count: 0}
		s.Nodes[proxyName] = value
	} else {
		value.Endpoints[podName] = &Endpoint{Address: address, Port: port}
	}
	s.logger.Infow("Added receiver endpoint", "node", proxyName, "endpoints", value.Endpoints)
	s.nodeUpdates <- *value
}

func (s *SoTWStore) AddSender(proxyName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.Nodes[proxyName]
	if !ok {
		endpoints := make(map[string]*Endpoint)
		value = &NodeConfig{ProxyName: proxyName, Endpoints: endpoints, count: 0}
		s.Nodes[proxyName] = value
	}
	value.count++
	s.nodeUpdates <- *value
}

func (s *SoTWStore) RemoveReceiver(proxyName string, podName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.Nodes[proxyName]
	if ok {
		delete(value.Endpoints, podName)
		s.logger.Infow("Deleting receiver endpoint", "proxyName", proxyName, "receiver", podName)
		s.nodeUpdates <- *s.Nodes[proxyName]
	}
}

/// RemoveSender removes a quilkin proxy node/sender and returns whether or not its the last instance
/// using that proxyName
func (s *SoTWStore) RemoveSender(proxyName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	node, ok := s.Nodes[proxyName]
	if ok {
		nodes := node.count - 1
		s.logger.Infow("removed sender", "name", proxyName, "remaining", nodes)
		if nodes == 0 {
			delete(s.Nodes, proxyName)
			s.nodeDeletes <- proxyName
			return true
		}
	}
	return false
}
