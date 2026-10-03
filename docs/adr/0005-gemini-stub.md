# ADR 0005 — Gemini behind a stub

`google.golang.org/genai` is the live adapter. It sets no tools. Tests and the default demo use a local stub that can only cite supplied evidence ids and sets `live: false`. Missing credentials are a blocker for a live call, not a reason to relabel the stub as Gemini.

Status: accepted.
