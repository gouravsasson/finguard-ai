# ADR 0002 — Database jobs instead of Kafka

AI report work is recorded in `jobs` in the same transaction as the case update. A worker claims rows with `FOR UPDATE SKIP LOCKED`, retries with bounded backoff, and can dead-letter. Kafka and Redis are deferred until a measured bottleneck exists.

Status: accepted.
