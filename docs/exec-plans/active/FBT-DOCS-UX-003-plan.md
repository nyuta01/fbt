# FBT-DOCS-UX-003 Define quickstart scope

## Observation

The quickstart page showed commands and outputs but did not clearly state what
the scenario represented. Readers could reasonably interpret it as a support
product demo, model-quality benchmark, realistic manual-generation workflow, or
general fbt architecture walkthrough.

## Decision

Define quickstart as a small control-plane acceptance demo. The support
template is only a fixture used to demonstrate doctor, plan, build, artifact
versioning, local inspection, and standard exports.

## Permanent Fix

The quickstart now opens with a scope statement, an explicit table of lifecycle
stages, and per-step "what this proves" text. README, usage guide, and
"What you can do today" now state that quickstart is not a model-quality
benchmark or realistic manual-generation workflow. The docs sidebar labels it
as a quickstart demo.

## Next Check

```sh
make verify
```
