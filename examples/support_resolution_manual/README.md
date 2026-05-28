# Support Resolution Manual Example

This example is a production-shaped fbt project for turning user inquiries,
support responses, product references, and existing macros into an approved
support resolution manual.

It does not use the bundled demo runners. From a source checkout it uses the
project-local `bin/fbt-runner-openai` wrapper, which calls the optional
`runners/openai` external runner implementation and requires `OPENAI_API_KEY`.

## Workflow

1. Export new support tickets into `data/support/tickets/`.
2. Store agent response notes in `data/support/response_logs/`.
3. Keep current product docs in `data/reference/product_docs/`.
4. Keep approved macros in `data/reference/macros/`.
5. Run:

```sh
fbt parse --project-dir examples/support_resolution_manual
fbt doctor --project-dir examples/support_resolution_manual
fbt plan --project-dir examples/support_resolution_manual --select support_resolution_manual
fbt build --project-dir examples/support_resolution_manual --select support_resolution_manual
fbt review show support_resolution_manual --project-dir examples/support_resolution_manual
fbt review approve support_resolution_manual --project-dir examples/support_resolution_manual --comment "Support lead approved"
fbt docs generate --project-dir examples/support_resolution_manual
```

`fbt doctor` and `fbt build` require credentials. The example is intentionally
configured for a real runner boundary:

```yaml
runners:
  - name: openai.responses
    command: bin/fbt-runner-openai
    args: ["responses"]
    env: ["OPENAI_API_KEY"]
```

If you install a separately packaged runner, replace `command` with that
installed command.

## Output

The generated official artifact is:

```text
target/artifacts/support/support_resolution_manual.md
```

The required format is defined in `assets/support_resolution_manual_format.md`.
The deterministic eval in `evals/support.yml` checks that required sections are
present before approval.
