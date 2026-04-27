package main

import "runtime"

// runtimeYield is a thin wrapper to keep the loader file free of stdlib
// noise. Splitting it out makes future replacement (e.g. with a backoff
// strategy) a one-file change.
func runtimeYield() { runtime.Gosched() }
