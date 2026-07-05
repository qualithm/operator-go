// Package operator is the shared client for the Qualithm platform management
// API. It authenticates with a member API token (Bearer) and exposes typed
// methods over the provisioning surface: authorities, enrollments, credentials,
// devices, and API tokens.
//
// The same client backs both the qualithm operator CLI and the MCP server, so
// the two surfaces never diverge. Mutating calls honour a client-level dry-run:
// when enabled, non-GET requests are not sent and a [*DryRunError] carrying the
// planned [Action] is returned instead.
package operator
