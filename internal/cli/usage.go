package cli

const usageText = `qualithm — operator CLI for the Qualithm platform management API.

Usage:
  qualithm <resource> <verb> [flags]

Global flags (accepted by every verb; place before positional arguments):
  --url string     management API base URL (env QUALITHM_API_URL)
  --token string   member API token (env QUALITHM_API_TOKEN)
  --json           emit JSON instead of a human table
  --dry-run        report the planned change without applying it

Resources and verbs:
  authority  list | create | revoke <id>
  enrollment list | create | revoke <id>
  credential list | mint | cert | rotate | revoke
  device     list | get <id> | create | update <id> | delete <id>
  token      list | create | revoke <id>
  apply      <manifest.yaml>            idempotent device-as-code reconcile
  version                               print the CLI version

Examples:
  qualithm enrollment create --space spc_123 --label lab-floor-2
  qualithm credential mint --device dev_123 --json
  qualithm credential revoke --device dev_123 --credential cred_123 --dry-run
  qualithm device list --json
  qualithm apply fleet.yaml --dry-run

Exit codes:
  0 ok (incl. dry-run)  1 error  2 usage  3 auth  4 not-found  5 conflict
  6 rate-limited        7 api
`
