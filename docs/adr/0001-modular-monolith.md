# ADR 0001 — Modular monolith

The workspace had no existing FinGuard service. A Go monolith with package boundaries is enough for the demonstrator and keeps one transaction around ingest, signals, evidence, and the outbox row. Splitting services would add failure modes without a measured need.

Status: accepted.
