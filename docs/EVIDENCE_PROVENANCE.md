# Evidence provenance

Evidence ids are `ev_` plus a stable hash of tenant and natural key, so a replay points at the same id.

| Kind | Stable key | Provenance |
| --- | --- | --- |
| `transaction_record` | `txn:{source_system}:{source_id}` | Ingestion of that source event |
| `rule_evaluation` | `rule:{source}:{source_id}:{code}:{version}` | Deterministic rule `2026.1` |
| `device_observation` | `device:{external_id}:{account}` | Rule observation. `identity_inference` is false |

Records store a content hash, classification `confidential`, and a retention timestamp (365 days in this demonstrator, which is a product setting, not a legal schedule). Updates and deletes are rejected by a trigger. Packages sent to a model are capped at 40 items and about 24,000 characters.
