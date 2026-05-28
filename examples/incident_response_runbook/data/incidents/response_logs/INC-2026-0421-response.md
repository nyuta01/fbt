# INC-2026-0421 Response Notes

## Timeline

- 09:12 UTC: Pager fired for checkout-api latency.
- 09:16 UTC: Incident commander assigned.
- 09:20 UTC: SRE confirmed database connection pool saturation.
- 09:27 UTC: Application owner confirmed recent deploy did not change checkout
  database access pattern.
- 09:31 UTC: Traffic shifted away from the us-east-1 read replica.
- 09:44 UTC: Customer support was told to acknowledge elevated checkout
  latency and avoid promising order recovery until payment status was checked.
- 10:06 UTC: Incident resolved.

## Actions That Worked

1. Use checkout latency, timeout rate, and database pool saturation together to
   confirm the incident.
2. Shift checkout read traffic away from the saturated replica.
3. Ask support to verify payment status before asking customers to retry.

## Gaps

- The runbook did not include a clear threshold for replica traffic shifting.
- Support did not have an approved customer-facing message for partial checkout
  failures.
