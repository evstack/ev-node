//go:build fibre

// Package cnfibertest wires a single-validator Celestia chain, an in-process
// Fibre server, a celestia-node bridge and the celestia-node-fiber adapter
// together so Upload → Listen → Download can be exercised end-to-end in a
// Go test.
//
// The chain is a celestia-app testnode built with -tags fibre. The Fibre
// server runs in the same process and its FSP endpoint is registered with
// the valaddr module so the client's host registry can find it. The
// underlying adapter talks directly to consensus gRPC and the Fibre server;
// only Listen goes through the bridge node's blob subscription.
//
// This is the "fast sanity" variant. A multi-validator showcase is planned
// as a Docker Compose follow-up that exercises real quorum collection.
package cnfibertest
