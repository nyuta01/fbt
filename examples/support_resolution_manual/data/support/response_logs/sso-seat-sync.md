# SSO Seat Sync Response Log

## Pattern

When group sync is enabled, members added to the billing group can increase the
billable seat count at the next sync.

## Agent Steps That Worked

1. Confirm the workspace has SSO group sync enabled.
2. Ask the admin to compare the identity provider billing group with the
   workspace member list.
3. Explain that removing users from the billing group takes effect after the
   next sync.
4. Escalate billing disputes when the customer reports a mismatch after sync.

## Customer Language Used

"The seat estimate reflects users currently synced from your identity provider
billing group. After you remove users from that group, the estimate should
update after the next sync."
