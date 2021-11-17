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
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestEmptyStore(t *testing.T) {
	t.Parallel()
	updates := make(chan NodeConfig)
	deletes := make(chan string)
	store := NewSotWStore(updates, deletes, zap.L().Sugar())

	//Delete sender from empty store
	if store.RemoveSender("fail", "fail") {
		t.Error("Shouldn't return true")
	}

	//Delete receiver from empty store
	timer := time.NewTimer(time.Second / 2)
	go store.RemoveReceiver("fail", "error")
	select {
	case <-updates:
		t.Error("Shouldn't get data")
	case <-timer.C:
		break
	}
}

func TestSenderInteractions(t *testing.T) {
	t.Parallel()
	updates := make(chan NodeConfig)
	deletes := make(chan string)
	store := NewSotWStore(updates, deletes, zap.L().Sugar())
	//Add Sender
	go store.AddSender("test", "pod-10")
	timer := time.NewTimer(time.Second / 2)
	select {
	case data := <-updates:
		if len(data.senders) != 1 || len(data.Endpoints) != 0 || data.ProxyName != "test" {
			t.Error("mismatch")
		}
		break
	case <-timer.C:
		t.Error("Should return update")
	}

	go store.AddSender("test", "pod-11")
	timer = time.NewTimer(time.Second / 2)
	select {
	case data := <-updates:
		if len(data.senders) != 2 || len(data.Endpoints) != 0 || data.ProxyName != "test" {
			t.Error("mismatch")
		}
		break
	case <-timer.C:
		t.Error("Should return update")
	}

	// Delete multi sender + no receiver
	go func() {
		if store.RemoveSender("test", "pod-11") {
			t.Error("SHould not delete with multiple")
		}
	}()
	timer = time.NewTimer(time.Second / 2)
	select {
	case <-deletes:
		t.Error("Should not delete node")
	case <-timer.C:
		break
	}

	// Delete single sender + no receiver
	go func() {
		if !store.RemoveSender("test", "pod-10") {
			t.Error("Should delete")
		}
	}()
	timer = time.NewTimer(time.Second / 2)
	select {
	case data := <-deletes:
		if data != "test" {
			t.Error("mismatch")
		}
		break
	case <-timer.C:
		// t.Error("Should return update")
	}

	// Delete already deleted sender + no receiver
	go func() {
		if store.RemoveSender("test", "pod-10") {
			t.Error("Shouldn't delete")
		}
	}()
	timer = time.NewTimer(time.Second / 2)
	select {
	case <-deletes:
		t.Error("Shouldn't return update")
	case <-timer.C:
		if len(store.Nodes) != 0 {
			t.Error("node list should be empty")
		}
	}
}

func TestReceiverInteractions(t *testing.T) {
	t.Parallel()
	updates := make(chan NodeConfig)
	deletes := make(chan string)
	store := NewSotWStore(updates, deletes, zap.L().Sugar())
	//Add Sender
	go store.AddReceiver("test", 1000, "10.0.0.1", "pod-1")
	timer := time.NewTimer(time.Second / 2)
	select {
	case data := <-updates:
		if len(data.Endpoints) != 1 || data.ProxyName != "test" {
			t.Error("mismatch")
		}
		break
	case <-timer.C:
		t.Error("Should return update")
	}

	go store.AddReceiver("test", 1000, "10.0.0.0", "pod-2")
	timer = time.NewTimer(time.Second / 2)
	select {
	case data := <-updates:
		if len(data.Endpoints) != 2 || data.ProxyName != "test" {
			t.Error("mismatch")
		}
		break
	case <-timer.C:
		t.Error("Should return update")
	}

	// Delete multi sender + no receiver
	go store.RemoveReceiver("test", "pod-1")
	timer = time.NewTimer(time.Second / 2)
	select {
	case update := <-updates:
		if _, ok := update.Endpoints["pod-2"]; !ok {
			t.Error("missing data")
		}
	case <-timer.C:
		break
	}

	// Delete single sender + no receiver
	go store.RemoveReceiver("test", "pod-2")
	timer = time.NewTimer(time.Second / 2)
	select {
	case <-updates:
		t.Error("No updates due to empty list")
	case <-timer.C:
		if len(store.Nodes) != 0 {
			t.Error("Nodelist should be empty")
		}
	}
}

func TestMixedStore(t *testing.T) {
	t.Parallel()
	updates := make(chan NodeConfig)
	deletes := make(chan string)
	store := NewSotWStore(updates, deletes, zap.L().Sugar())

	//Add Sender
	go store.AddSender("test", "pod-10")
	timer := time.NewTimer(time.Second / 2)
	select {
	case data := <-updates:
		if len(data.Endpoints) != 0 || data.ProxyName != "test" {
			t.Error("mismatch")
		}
		break
	case <-timer.C:
		t.Error("Should return update")
	}

	//Add receiver
	go store.AddReceiver("test", 1000, "10.0.0.1", "pod-1")
	timer = time.NewTimer(time.Second / 2)
	select {
	case data := <-updates:
		if len(data.Endpoints) != 1 || data.ProxyName != "test" {
			t.Error("mismatch")
		}
		break
	case <-timer.C:
		t.Error("Should return update")
	}

	// Remove receiver
	go store.RemoveReceiver("test", "pod-1")
	timer = time.NewTimer(time.Second / 2)
	select {
	case data := <-updates:
		if len(data.Endpoints) != 0 || data.ProxyName != "test" {
			t.Error("mismatch")
		}
		if len(store.Nodes) != 1 {
			t.Error("Node should still exist")
		}
	case <-timer.C:
		t.Error("Should return update")
	}

	// Re-add receiver
	go store.AddReceiver("test", 1000, "10.0.0.1", "pod-1")
	timer = time.NewTimer(time.Second / 2)
	select {
	case data := <-updates:
		if len(data.Endpoints) != 1 || data.ProxyName != "test" {
			t.Error("mismatch")
		}
		break
	case <-timer.C:
		t.Error("Should return update")
	}

	//Remove sender
	go func() {
		if !store.RemoveSender("test", "pod-10") {
			t.Error("Should delete")
		}
	}()
	timer = time.NewTimer(time.Second / 2)
	select {
	case <-deletes:
		if len(store.Nodes) != 1 {
			t.Error("Node should still exist")
		}
	case <-timer.C:
		// t.Error("Should send delete")
	}

	//Remove receiver
	go store.RemoveReceiver("test", "pod-1")
	timer = time.NewTimer(time.Second / 2)
	select {
	case <-updates:
		t.Error("Shouldn't return update")
	case <-timer.C:
		if len(store.Nodes) != 0 {
			t.Error("Node should be deleted")
		}
	}
}
