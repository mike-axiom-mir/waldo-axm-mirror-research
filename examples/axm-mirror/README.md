# WALDO Witness Mirror synthetic examples

These files are public-safe synthetic inputs. They do not describe a real
trained model or a real independent evaluation.

The checked-in corpus, run, release, comparison, and behavior drafts exercise
Wave 1. The complete gated-clone foundation flow is exercised by
`TestGatedCloneFoundationCLIFlow` and the focused tests under
`internal/axmmirror`.

The separate experimental CLI now supports:

```text
waldo-axm-mirror lens-corpus
waldo-axm-mirror witness-run
waldo-axm-mirror profile-contract
waldo-axm-mirror anchor
waldo-axm-mirror lock
waldo-axm-mirror contamination
waldo-axm-mirror context
waldo-axm-mirror gate-claims
waldo-axm-mirror seal-evaluation
waldo-axm-mirror seal-gated
waldo-axm-mirror verify
```

`seal-evaluation` is a pre-execution operation in a real experiment. The test
constructs all deterministic inputs in one process for replay convenience, but
the final gated seal still requires the output and claim assessment to bind the
exact precommitted protocol digest.

HOLD and REFUSED receipts are written before the CLI returns a nonzero status,
so unresolved evidence remains inspectable. Output paths are atomic no-replace
writes and are never silently overwritten.
