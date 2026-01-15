package common

import blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"

// BlobsFromSubscription returns non-empty blob data from a subscription response.
func BlobsFromSubscription(resp *blobrpc.SubscriptionResponse) [][]byte {
	if resp == nil || len(resp.Blobs) == 0 {
		return nil
	}

	blobs := make([][]byte, 0, len(resp.Blobs))
	for _, blob := range resp.Blobs {
		if blob == nil {
			continue
		}
		data := blob.Data()
		if len(data) == 0 {
			continue
		}
		blobs = append(blobs, data)
	}

	return blobs
}
