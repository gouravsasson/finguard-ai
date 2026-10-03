# ADR 0006 — Integer minor units

Amounts are `int64` minor units. Rule ratios are integers. That keeps comparisons deterministic and avoids binary floating point in the risk engine.

Status: accepted.
