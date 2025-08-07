package directtx

import (
	"fmt"
)

type DirectTX struct {
	TX              []byte
	ID              []byte
	FirstSeenHeight uint64
	// FirstSeenTime unix time
	FirstSeenTime int64
}

// ValidateBasic performs basic validation of DirectTX fields
func (d *DirectTX) ValidateBasic() error {
	if len(d.TX) == 0 {
		return fmt.Errorf("tx cannot be empty")
	}
	if len(d.ID) == 0 {
		return fmt.Errorf("id cannot be empty")
	}
	if d.FirstSeenHeight == 0 {
		return fmt.Errorf("first seen height cannot be zero")
	}
	if d.FirstSeenTime == 0 {
		return fmt.Errorf("first seen time cannot be zero")
	}
	return nil
}
