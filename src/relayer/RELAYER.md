### relayers

this package defines some interfaces for a simple `DefaultMessageRelayer` and
a `PriorityMessageRelayer`. The `DefaultMessageRelayer` exists as a sanity check
for the priority relayer.

There are a handful of `TODO` for this package:

- [ ] fully implement dependecy injection in the tests so that each message relayer implementation can share a set of common tests without `copy pasta`.
- [ ] allow better stopping of goroutines via `done` signaling
