package common

import "time"

// MaxRetriesBeforeHalt is the maximum number of retries before halting.
const MaxRetriesBeforeHalt = 3

// MaxRetriesTimeout is the maximum time to wait before halting.
const MaxRetriesTimeout = 10 * time.Second
