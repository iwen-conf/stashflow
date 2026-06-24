# StashFlow Edge

Private Cloudflare Worker for turning one upstream subscription into QX and Stash subscription URLs.

Secrets required:

- `ADMIN_PASSWORD`: password for the management page.
- `SESSION_TOKEN`: random cookie token used after login.
- `PUBLIC_TOKEN`: random token embedded in generated QX/Stash subscription URLs.

KV binding required:

- `STASHFLOW_KV`: stores the upstream subscription URL and display name.

Routes:

- `/` private management UI.
- `/api/login`, `/api/logout`, `/api/state`, `/api/subscription` private APIs.
- `/sub/qx?token=...` public unguessable QX output.
- `/sub/stash?token=...` public unguessable Stash output.
