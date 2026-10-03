# Limitations

- Synthetic names and events only. There is no bank integration.
- The product does not freeze accounts, move money, or declare that fraud occurred.
- Rules are configured thresholds, version `2026.1`. They are not a trained detector and have no measured precision or recall.
- Shared device and country change are observations. They do not establish identity.
- The local stub restates supplied evidence ids. It is not Gemini. A live call requires `GEMINI_API_KEY` and is still advisory.
- Session auth is local. OIDC is a seam, not a connected provider.
- Rate limits, metrics, and the AI call cap are per process.
- Terraform for Cloud Run and Cloud SQL is a sketch. It has not been applied, so there is no cloud URL.
- Logs from the local demo were inspected by the tests and the secret scan, not by a third-party audit.
