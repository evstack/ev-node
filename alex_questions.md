# direct TX

- What format can we expect for the direct TX on the DA? some wrapper type would be useful to unpack and distinct from
  random bytes
- Should we always include direct TX although they may be duplicates to mempool TX? spike: yes
- Should we fill the block space with directTX if possible or reserve space for mempool TX
- What should we do when a sequencer was not able to add a direct-TX within the time window?
- Do we need to trace the TX source chanel? There is currently no way to find out if a sequencer adds a direct TX to
  early
- What is a good max size for a block? Should this be configurable or genesis?
-

## Smarter sequencer

- build blocks by max bytes from the request rather than returning what was added as a batch before from the syncer.
- Executor.GetTxs method should have a max byte size to limit the response size or pass via context
