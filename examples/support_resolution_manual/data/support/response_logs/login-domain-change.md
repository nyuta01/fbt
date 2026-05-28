# Login Domain Change Response Log

## Pattern

Customers sometimes lose access after an email domain change when their
identity provider sends a new email value before the workspace admin updates
the user record.

## Agent Steps That Worked

1. Verify the requester is a workspace admin or has an approved admin contact.
2. Confirm the old and new email values.
3. Ask the admin to update the user email in workspace settings.
4. Ask the user to request a new password reset or restart SSO sign-in.
5. Escalate to identity support if the user record cannot be edited.

## Customer Language Used

"Your workspace admin needs to update the email on your user profile before the
reset link can be delivered to the new address."

## Risk Notes

Do not change identity details for a requester who is not an admin or approved
admin contact.
