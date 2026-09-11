# ADR 0067: Select topology-aware TorchTitan parallelism

Status: accepted

## Decision

TorchTitan training defaults to automatic model placement. WALDO compares the
estimated FP32 model, gradient, and AdamW state (16 bytes per parameter) with
the smallest visible GPU and reserves 40% of memory for activations and runtime
workspace.

When the complete state fits, every GPU holds a complete model and trains on a
different slice of the global batch. When it does not fit but a host-local
shard fits, each host holds one complete model divided across its local GPUs.
Otherwise one model is divided across the full world. NCCL selects NVLink or
other peer transport locally and the configured RDMA or TCP path between hosts.

A stage may explicitly request any supported strategy. WALDO rejects an
impossible explicit placement during preflight. The run BOM pins the request,
resolved strategy, physical topology, communication paths, estimated state
size, and GPU memory. User-facing progress describes complete models and
divided models directly rather than relying on the ambiguous term “replica.”
