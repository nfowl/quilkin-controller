package store

import (
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type SoTWStore struct {
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
	count     atomic.Int32
}

type Endpoint struct {
	Address string
	Port    int
}

func (s *SoTWStore) AddReceiver(proxyName string, port int, address string, podName string) {
	value, ok := s.Nodes[proxyName]
	if !ok {
		//Making new NodeConfiguration
		endpoints := make(map[string]*Endpoint)
		endpoints[podName] = &Endpoint{Address: address, Port: port}
		value = &NodeConfig{Endpoints: endpoints, ProxyName: proxyName}
		s.Nodes[proxyName] = value
	} else {
		value.Endpoints[podName] = &Endpoint{Address: address, Port: port}
	}
	s.nodeUpdates <- *value
}

func (s *SoTWStore) AddSender(proxyName string) {
	value, ok := s.Nodes[proxyName]
	if !ok {
		endpoints := make(map[string]*Endpoint)
		value := &NodeConfig{ProxyName: proxyName, Endpoints: endpoints, count: *atomic.NewInt32(0)}
		s.Nodes[proxyName] = value
	}
	value.count.Inc()
	s.nodeUpdates <- *value
}

func (s *SoTWStore) RemoveReceiver(proxyName string, podName string) {
	delete(s.Nodes[proxyName].Endpoints, podName)
	s.nodeUpdates <- *s.Nodes[proxyName]
}

/// RemoveSender removes a quilkin proxy node/sender and returns whether or not its the last instance
/// using that proxyName
func (s *SoTWStore) RemoveSender(proxyName string) bool {
	node := s.Nodes[proxyName]
	nodes := node.count.Dec()
	s.logger.Infow("removed sender", "name", proxyName, "remaining", nodes)
	if nodes == 0 {
		delete(s.Nodes, proxyName)
		s.nodeDeletes <- proxyName
		return true
	}
	return false
}
