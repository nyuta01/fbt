# FBT-DOCS-UX-006 Simplify README mental model

## Observation

The README had become more concrete but still tried to explain too many
features at once. It asked first-time readers to infer the core product model
from examples, command lists, and detailed capabilities instead of stating the
simple user-facing shape of fbt first.

## Decision

Make the README lead with one mental model:
`sources + instructions + runner -> artifact + build receipt`. Explain only the
core elements needed to understand fbt, then show the support manual example as
an instance of that model. Move feature breadth into linked docs instead of the
README opening.

## Permanent Fix

README now focuses on the user's starting point and outcome: existing files go
through an external runner and become a generated artifact plus an inspectable
local receipt. The support example now starts from the support lead's actual
problem: prior cases contain the answer, but not in a reusable or reviewable
form. It then shows the generated manual, the receipt, the controlled process,
and only then the YAML and CLI commands.

## Next Check

```sh
make verify
```
