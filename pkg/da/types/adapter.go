package datypes

import coreda "github.com/evstack/ev-node/core/da"

// DA is the shared DA interface using datypes aliases.
// It currently aliases the core DA interface to ease migration.
type DA = coreda.DA

// WrapCoreDA adapts a core DA implementation to the datypes.DA interface.
func WrapCoreDA(da coreda.DA) DA {
	return da
}
