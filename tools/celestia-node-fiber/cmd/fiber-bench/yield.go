package main

import "time"

// loaderBackoff is what each worker waits when InjectTx returns false
// because the mempool channel is full. Using a real sleep (rather than
// runtime.Gosched) caps the per-worker drop rate, which keeps the
// load generator's allocation pressure proportional to actual drain
// throughput. Without this, full-mempool workers spin a tight
// allocate-then-drop loop at ~200k iter/s/worker — millions of
// short-lived 200 B slices per second across the pool, which drives GC
// hard and drove the OOM kills observed at sustained load.
//
// 100 µs caps a single worker to ~10k drops/s when the mempool is
// permanently full. Total drop rate scales with --workers and serves
// as a bounded backpressure signal in the stats line.
func runtimeYield() { time.Sleep(100 * time.Microsecond) }
