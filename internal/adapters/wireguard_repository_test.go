package adapters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_wgRepository_GetInterfaces(t *testing.T) {
	repo := NewWireGuardRepository()

	interfaces, err := repo.GetInterfaces(context.Background())
	assert.NoError(t, err)
	t.Log(interfaces)
	for _, addr := range interfaces[0].Addresses {
		t.Log(addr.String())
	}
}
