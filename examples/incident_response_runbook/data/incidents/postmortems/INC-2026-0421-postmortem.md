# INC-2026-0421 Postmortem

## Summary

Checkout experienced elevated latency and timeouts due to database connection
pool saturation on the us-east-1 read replica.

## Root Cause

A maintenance task increased read load during peak traffic. The checkout-api
connection pool approached its configured limit, which increased request queue
time.

## Corrective Actions

- Add an alert when active database connections exceed 80% of the configured
  limit for 5 minutes.
- Document the traffic shift procedure for checkout read replicas.
- Add a customer communication template for partial checkout failures.

## Preventive Follow-up

The platform team owns the alert. The checkout team owns the runbook update.
