# direct TX

- What format can we expect for the direct TX on the DA? some wrapper type would be useful to unpack and distinct from
  random bytes
- Should we always include direct TX although they may be duplicates to mempool TX? spike: yes, first one wins.
- Should we fill the block space with directTX if possible or reserve space for mempool TX
- What should we do when a sequencer was not able to add a direct-TX within the time window?
- Do we need to trace the TX source chanel? There is currently no way to find out if a sequencer adds a direct TX "to
  early". Can be a duplicate from mempool as well. 
- What is a good max size for a sequencer/fallback block? Should this be configurable or genesis?
- Do we need to track the sequencer DA activity? When it goes down with no direct TX on DA there are no state changes.
  When it goes down and direct TX missed their time window, fullnodes switch to fallback mode anyway.
- How do we restore the chain from recovery mode? 
    => socially coordinated fork which nominates a new sequencer to continue to “normal” functionality

## Smarter sequencer

- Flatten the batches from mempool to limit memory more efficiently
- `Executor.GetTxs` method should have a max byte size to limit the response size or pass via context
