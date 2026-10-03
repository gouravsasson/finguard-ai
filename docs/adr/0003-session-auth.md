# ADR 0003 — Local session with an OIDC-shaped principal

There is no existing identity provider in this repo. Login issues an HttpOnly session cookie. The principal carries user, tenant, and role. A production deployment can replace `auth.Service.Login` with an OIDC code flow that creates the same principal. The client cannot choose the tenant.

Status: accepted for the demonstrator.
