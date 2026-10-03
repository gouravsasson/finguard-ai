# Judge FAQ

**Does the model decide the case?** No. It drafts text. A senior investigator escalates or resolves, and resolution means no adverse action.

**What if Gemini is down?** The case, signals, evidence, and graph still load. The report row is `failed` or the stub is used only when no key is configured, and the screen says the call was not live.

**How do I know a claim is grounded?** Each observed fact lists evidence ids. Ids that are not in the case are removed or marked unverified. The evidence panel is beside the draft.

**Can a memo jailbreak the rules?** No. The rule engine does not receive memo text. The prompt wraps evidence as data and attaches no tools. “Freeze the account” is refused.

**Is the shared device one person?** No. The signal and the graph say that explicitly.

**What was measured?** Local tests passed, including tenant isolation, replay, catalog rules, and migration rollback. One demo-path log showed event posts around 1–9 ms on a laptop and Docker Postgres. That is not a production throughput or accuracy number.

**What should I not hear in the demo?** Regulator approval, proven fraud prevention, real-world accuracy, or revenue impact.
