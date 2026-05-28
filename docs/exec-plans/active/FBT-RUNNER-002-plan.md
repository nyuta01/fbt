# FBT-RUNNER-002 Harden runner process invocation with args, env, and cwd

## Observation

Runner config carried a command and environment variable names, but process
startup did not support first-class args or working directories. Declared env
names were not diagnosed as missing, and runner processes inherited the ambient
environment rather than an explicit fbt-controlled environment.

## Decision

Add `args` and `cwd` to project runner config and plugin manifests. Keep `cwd`
optional so existing wrappers keep their current working-directory behavior;
when set, project config cwd is resolved from the project directory and plugin
cwd is resolved from the plugin manifest directory. Pass a small base
environment plus declared env names to runner processes, and report missing
declared env vars through runner diagnostics.

## Permanent Fix

Updated config, manifest, plugin, discovery, protocol startup, CLI diagnostics,
docs, and tests. Runner list/doctor now shows args/cwd/env names without
printing env values.

## Next Check

Run:

```sh
make verify
```
