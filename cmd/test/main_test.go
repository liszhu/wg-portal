package main

import (
	"testing"

	"github.com/h44z/wg-portal/internal/model"
)

func BenchmarkAppendingStructs(b *testing.B) {
	var s []model.Interface = make([]model.Interface, 0, b.N)

	for i := 0; i < b.N; i++ {
		s = append(s, model.Interface{})
	}
}

func BenchmarkAppendingPointers(b *testing.B) {
	var s []*model.Interface = make([]*model.Interface, 0, b.N)

	for i := 0; i < b.N; i++ {
		s = append(s, &model.Interface{})
	}
}

func BenchmarkAppendingStructsPeer(b *testing.B) {
	var s []model.Peer = make([]model.Peer, 0, b.N)

	for i := 0; i < b.N; i++ {
		s = append(s, model.Peer{Interface: &model.PeerInterfaceConfig{}})
	}
}

func BenchmarkAppendingPointersPeer(b *testing.B) {
	var s []*model.Peer = make([]*model.Peer, 0, b.N)

	for i := 0; i < b.N; i++ {
		s = append(s, &model.Peer{Interface: &model.PeerInterfaceConfig{}})
	}
}
