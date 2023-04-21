package caveats

import (
	"testing"

	"github.com/stretchr/testify/require"

	"spicedb/pkg/caveats/types"
)

func TestAddVariable(t *testing.T) {
	req := require.New(t)
	env := NewEnvironment()
	err := env.AddVariable("foobar", types.IntType)
	req.NoError(err)
	err = env.AddVariable("foobar", types.IntType)
	req.Error(err)
}
