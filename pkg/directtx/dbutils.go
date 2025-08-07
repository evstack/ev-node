package directtx

import (
	"crypto/sha256"
	"encoding/binary"
	"hash"

	ds "github.com/ipfs/go-datastore"
)

func buildStoreKey(d DirectTX) ds.Key {
	hasher := sha256.New()
	hashWriteInt(hasher, len(d.ID))
	hasher.Write(d.ID)
	hashWriteInt(hasher, len(d.TX))
	hasher.Write(d.TX)
	return ds.NewKey(string(hasher.Sum(nil)))
}

func hashWriteInt(hasher hash.Hash, data int) {
	txLen := make([]byte, 8) // 8 bytes for uint64
	binary.BigEndian.PutUint64(txLen, uint64(data))
	hasher.Write(txLen)
}
