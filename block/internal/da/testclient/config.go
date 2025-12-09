package testclient

import (
	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// Config defines the minimal settings required to build a test DA client adapter.
type Config struct {
	DA                       datypes.DA
	Namespace                string
	DataNamespace            string
	ForcedInclusionNamespace string
}
