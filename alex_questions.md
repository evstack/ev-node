# direct TX

- What format can we expect for the direct TX on the DA? some wrapper type would be useful to unpack and distinct from
  random bytes
- Should we always include direct TX although they may be duplicates to mempool TX? spike: yes
- Should we fill the block space with directTX if possible or reserve space for mempool TX

## Smarter sequencer
build blocks by max bytes from the request rather than returning what was added as a batch before from the syncer.
