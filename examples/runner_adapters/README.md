# Runner Adapter Examples

These are source-checkout examples of external commands that speak the fbt
runner protocol. They are not fbt core packages.

Use them to understand packaging boundaries, protocol behavior, and local
smoke workflows before publishing a real adapter package or plugin.

| Directory | Purpose |
|---|---|
| `demo_llm/` | Deterministic LLM-shaped runner for offline templates and examples. |
| `demo_agent/` | Deterministic agent-shaped runner with redacted tool-call events. |
| `command/` | Runner adapter that executes a configured local command. |
| `openai/` | Optional OpenAI Responses adapter used by practical examples. |

Real provider or CLI-agent integrations should normally live in their own
package, plugin, or project-local adapter directory. fbt only requires that the
command advertises compatible capabilities, writes candidates under
`work.outputs`, and returns JSON-RPC protocol messages.
