# Pre-flight findings (task-063)

## CREATEDB on db-credentials user
- Result: PASS 2026-05-08
- Action taken (if FAIL): n/a

### Evidence
Query against `postgres.home` as the user from `atlas/db-credentials`:

```
 rolname | rolcreatedb
---------+-------------
 atlas   | t
(1 row)
```

### Note on secret hygiene
`kubectl get secret -n atlas db-credentials -o jsonpath='{.data.DB_USER}' | base64 -d` returns `atlas \r\n` (a literal trailing space plus CR+LF). The same applies to `DB_PASSWORD`. Authentication only succeeds after stripping the trailing whitespace (`tr -d ' \r\n'`); the literal-space form fails with `password authentication failed for user "atlas "`. Atlas services in-cluster appear to tolerate this today, but the per-PR overlay tooling introduced in later phases should either strip whitespace defensively or the secret should be re-issued without the trailing whitespace. Tracking this here so a downstream phase can address it if needed.
