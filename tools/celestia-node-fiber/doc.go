// Package celestianodefiber implements the ev-node fiber.DA interface by
// delegating to a celestia-node api/client.Client.
//
// Upload and Download run locally against a Celestia consensus node (over
// gRPC) and Fibre Storage Providers (over Fibre gRPC), without a bridge-node
// hop, using the self-sufficient client introduced in celestia-node #4961.
// Listen subscribes to a bridge node's blob stream (JSON-RPC/HTTP) and
// forwards only share-version-2 blobs — the ones settled on-chain through
// MsgPayForFibre.
//
// This package is a separate Go sub-module from the parent ev-node repository
// so that ev-node core does not pick up the celestia-app / cosmos-sdk
// replace-directive soup that celestia-node requires.
package celestianodefiber
