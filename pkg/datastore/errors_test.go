package datastore

import (
	"fmt"
	"testing"

	"spicedb/internal/logging"
)

func TestError(_ *testing.T) {
	logging.Info().Err(ErrNamespaceNotFound{
		error:         fmt.Errorf("test"),
		namespaceName: "test/test",
	},
	).Msg("test")
}
