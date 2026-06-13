# Stellar Interceptor Model

Stellar manages HTTP server, HTTP client, gRPC server, and gRPC client interceptors through one ordered registry.

The interceptor pipeline is intentionally small and stable. Rate limit, circuit breaker, route policy, authentication, authorization, retry, and signing are governance policies. They should read dynamic rules from the governance rule store instead of becoming public interceptor stages.

## Core Concepts

| Concept | Responsibility |
| --- | --- |
| Pipeline stage | Stable execution point controlled by the framework. |
| Framework interceptor | Framework-owned implementation mounted on a stable stage. |
| Business interceptor | Application-owned extension mounted only on `business`, ordered by application code. |
| Governance rule store | Local atomic snapshot of rules delivered by the service governance server. |
| Policy evaluator | Reads the latest rule snapshot and evaluates a policy inside a framework interceptor. |

Rule updates should not rebuild the interceptor chain. A governance client should parse and validate delivered rules, then atomically replace the local snapshot. Framework interceptors read that snapshot at runtime.

```text
governance server
-> governance client watcher
-> parse and validate rules
-> governance.Store.Replace(snapshot)
-> framework interceptor reads governance.Store.Snapshot()
```

## Server Inbound Order

The server inbound chain is used by HTTP server and gRPC server.

| Order | Stage | Purpose |
| --- | --- | --- |
| 1 | `recovery` | Catch panics before they escape the transport. |
| 2 | `route_resolve` | Resolve route, service, method, and policy keys before expensive work. |
| 3 | `observe` | Attach request id and start lightweight trace/log/metric envelope. |
| 4 | `deadline` | Apply server deadline before downstream work. |
| 5 | `admission` | Fast reject overload, concurrency limit, rate limit, quota, and circuit-open requests. |
| 6 | `security` | Run authentication and authorization policies that require request identity. |
| 7 | `decode_validate` | Decode and validate request payload after cheap protection stages pass. |
| 8 | `business` | Run ordered business interceptors, then the handler. |

Important rules:

- Admission runs before authentication when the rule can use cheap dimensions such as route, IP, service, method, or global load.
- Identity-based limits, tenant quotas, and authorization belong in `security` because they require authenticated identity.
- `decode_validate` must not run before admission unless a route explicitly needs body-based policy keys.
- Observability starts early, but it must stay lightweight: no body capture and no expensive tag computation.

## Client Outbound Order

The client outbound chain is used by HTTP client and gRPC client.

| Order | Stage | Purpose |
| --- | --- | --- |
| 1 | `recovery` | Catch panics in client interceptors. |
| 2 | `observe` | Attach request id and start lightweight client trace/log/metric envelope. |
| 3 | `deadline` | Bound the logical outbound call. |
| 4 | `admission` | Fast reject circuit-open, local rate limit, and concurrency limit before signing or I/O. |
| 5 | `retry` | Orchestrate retry attempts according to governance rules. |
| 6 | `business` | Run ordered business interceptors before signing. |
| 7 | `security` | Apply auth/signature as late as possible, close to the real transport call. |

Important rules:

- Circuit-open and local rate-limit checks run before signing because signing can be expensive and should not happen for locally rejected calls.
- Retry must not count local validation, local rate-limit, or local auth failures as downstream failures.
- Signing runs close to the transport call because signatures often depend on timestamp, nonce, headers, or body hash.

## Governance Rules

`governance.Store` stores an immutable local snapshot:

```go
store := stellar.NewGovernanceStore()
store.Replace(stellar.GovernanceSnapshot{
	Version: "v1",
	Rules: []stellar.GovernanceRule{
		{
			ID:       "user-service-rate-limit",
			Kind:     stellar.GovernanceRuleRateLimit,
			Enabled:  true,
			Priority: 100,
			Scope: stellar.GovernanceScope{
				Transport: "http.client",
				Service:   "user-service",
			},
			Spec: map[string]any{
				"qps": 100,
			},
		},
	},
})
```

Framework interceptors should read rules from the store:

```go
rules := app.Governance().Rules(stellar.GovernanceRuleRateLimit, func(rule stellar.GovernanceRule) bool {
	return rule.Enabled && rule.Scope.Service == "user-service"
})
```

The interceptor chain remains stable while rule snapshots change dynamically.

## Business Interceptors

Business interceptors are registered with `stellar.WithInterceptor`.

```go
func main() {
	if err := stellar.Run(
		stellar.WithInterceptor(
			stellar.HTTPServerInterceptor("tenant", 100, tenantInterceptor),
			stellar.GRPCClientInterceptor("signature-context", 200, signatureContextInterceptor),
		),
	); err != nil {
		log.Fatal(err)
	}
}
```

The `order` value only controls the order among business interceptors of the same transport kind. It does not move business interceptors ahead of framework protection stages.

## Framework Interceptors

Framework interceptors are registered into stable stages such as `admission`, `security`, and `retry`.

Examples:

- route policy evaluator: `route_resolve`
- rate limit evaluator: `admission`
- circuit breaker evaluator: `admission`
- load shedding evaluator: `admission`
- authentication evaluator: `security`
- authorization evaluator: `security`
- retry evaluator: `retry`
- signing evaluator: `security`

Application code should prefer business interceptors unless it is implementing a framework or governance module.
