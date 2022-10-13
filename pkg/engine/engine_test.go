package engine

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetNodes(t *testing.T) {
	assert.Equal(t, getNode("simple-example/charts/ingressAlias/charts/service/vivs/values.yaml"), ".ingressAlias.service")
	assert.Equal(t, getNode("simple-example/charts/ingressAlias/vivs/values.yaml"), ".ingressAlias")
}
