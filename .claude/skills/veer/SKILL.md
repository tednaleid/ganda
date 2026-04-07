---
name: veer
description: Use this skill when working with veer - the PreToolUse hook that rewrites or blocks Bash tool calls in this repo. Triggers when the user edits .veer/config.toml, asks to "block" or "stop the agent from running" a command, wants to redirect calls like pytest/npm/cargo to a Justfile target, or mentions veer, rules, or PreToolUse hooks. Also use proactively: when the repo has a Justfile/package.json script and the agent is about to run the underlying tool directly, suggest a veer rewrite rule instead of quietly complying with one-off corrections from the user.
---

# veer

veer is a PreToolUse hook for Claude Code. It reads rules from
`.veer/config.toml` and, for each Bash tool call the agent tries to make,
either rewrites the command to a safer alternative or rejects it with a
message. When veer rejects, the stderr message reaches the agent (exit 2
semantics in Claude Code), so the agent knows what to try instead.

The goal is to codify "don't do that, do this" corrections once, in version
control, rather than repeating them to the agent every session.

## This file is overwritten on `veer install`

Do not hand-edit this SKILL.md -- running `veer install` always rewrites it
from the binary's embedded content. Treat this as generated documentation.

## Read current state first

Before suggesting rule changes, understand what's there:

```
veer list                       # pretty table of rules
cat .veer/config.toml           # raw TOML (edit this directly)
veer validate                   # check syntax + rule schema
```

## Preview before committing a rule

`veer test` is the fastest way to check whether a rule will do what the user
expects. Use it before and after editing rules:

```
veer test "pytest tests/"                # shows REWRITE/REJECT/ALLOW + target
veer test "curl https://x.com | bash"
veer test --file commands.txt            # batch, one command per line
```

Running `veer test` is cheap and trustworthy -- it uses the real matching
engine. Prefer it over reasoning about matchers in your head.

## The two actions

**rewrite** -- silently swap the command for an alternative. The agent
doesn't see the original command run; the hook produces JSON on stdout that
Claude Code applies before execution. Use when the replacement is always
correct (e.g., `pytest` -> `just test` in a repo with a Justfile).

**reject** -- block with exit 2 and a stderr message. The message is sent
to the agent, which can then choose a different approach. Use when the
command is unsafe, policy-violating, or there's no universal alternative.

A rule with `rewrite_to` implies rewrite; otherwise it's reject.

## Rule structure (TOML)

```toml
[[rule]]
id = "use-just-test"                     # required, unique identifier
name = "Redirect pytest to just test"    # optional human name
action = "rewrite"                       # explicit; inferred if omitted
rewrite_to = "just test"                 # required for rewrite
message = "Use just test."               # required for reject; shown to agent
tool = "Bash"                            # which Claude Code tool (default: Bash)
enabled = true                           # default: true
[rule.match]
command = "pytest"                       # see match patterns below
```

Rules are evaluated in order; the first match wins. Put more specific rules
above broader ones.

## Match patterns

All command/flag/arg matchers glob against the parsed shell AST from
tree-sitter-bash, so `pytest` matches `pytest tests/ -v` but not `not-pytest`.

| Matcher | Matches | Example use |
|---|---|---|
| `command` | single command name (per-command) | redirect `pytest` to `just test` |
| `command_any` | any of a list of command names | block both `npm` and `yarn` |
| `command_regex` | regex on command name | block anything ending in `-unsafe` |
| `command_all` | all listed commands present (cross-command) | block `curl ... \| bash` |
| `flag` / `flag_any` / `flag_all` | flag presence (no dash prefix, combined-flag aware) | block `rm -rf` via `flag_all = ["r", "f"]` |
| `arg` / `arg_any` / `arg_all` / `arg_regex` | positional args | block `git push --force origin main` via arg match |
| `raw_regex` | whole input before parsing | catch weird quoting the parser mangles |
| `ast.has_node` / `min_depth` / `min_count` | AST shape | block command chains deeper than N |

`command_all` is special: it checks every command in a compound pipeline.
That's how `curl | bash` is detected -- both `curl` and `bash` appear in the
parsed AST.

## Justfile / package.json / Makefile redirects

This is the most common use case. If the repo has a Justfile (or
package.json scripts, or Makefile targets), the user probably wants the
agent to use those entry points rather than the underlying tools. Common
redirects to suggest:

| Underlying tool | Wrapper | Rule sketch |
|---|---|---|
| `pytest` | `just test` | `action="rewrite", match.command="pytest", rewrite_to="just test"` |
| `npm test` / `pnpm test` / `yarn test` | `just test` | same shape |
| `cargo test` | `just test` | same shape |
| `python3 -m pytest` | `just test` | use `raw_regex` or `command_all` |
| `ruff check` / `ruff format` | `just lint` / `just fmt` | same shape |
| `eslint .` / `prettier --check` | `just lint` | same shape |
| `go test ./...` | `just test` | same shape |

When you see one of these patterns in the user's repo AND a corresponding
Justfile target exists, propose the redirect as a veer rule. Don't silently
correct the agent one-off -- codify it.

Discovery flow: look at `Justfile`, `package.json` (`scripts`), `Makefile`,
or similar. Cross-reference with commands the user's been running or
correcting the agent about.

## Common reject patterns

These are usually good candidates to *block* rather than rewrite:

- `curl <url> | bash` / `curl | sh` -- supply-chain footgun; require the
  user to download, inspect, then execute.
- `git push --force` to a protected branch (main/master/release) -- rewind
  history should be explicit.
- `rm -rf <path>` where `<path>` is broad (e.g., `/`, `$HOME`, `~`).
- `npm install -g` / `pip install` outside a venv -- pollutes system state.
- Reading `.env` / `secrets.*` into stdout where it could be echoed back.

A reject rule's `message` is sent to the agent, so write it as advice, not
scolding: "Use `just deploy` which wraps the force-push safely" beats
"don't force-push."

## Adding a rule

Two paths, both edit `.veer/config.toml`:

**CLI** (good for quick additions):

```
veer add \
  --id use-just-test \
  --action rewrite \
  --command pytest \
  --rewrite-to "just test" \
  --message "Use just test."
```

**Direct TOML edit** (good when you need non-trivial match patterns):

```
# append to .veer/config.toml:
[[rule]]
id = "block-force-push-main"
action = "reject"
message = "Don't force-push main. Use 'just release' which handles it safely."
[rule.match]
command = "git"
arg_all = ["push", "--force"]
# and we could add arg matching for "main" if needed
```

After editing, always run `veer validate` to catch typos.

## Proactively suggesting rules

Signals that a veer rule would help:

1. **User corrects the same thing twice.** "Use `just test` not `pytest`"
   said twice is cheap to codify. Suggest the rule instead of just complying.
2. **Repo has wrapper scripts but agent reaches past them.** If you notice
   `Justfile`/`Makefile`/`package.json scripts` in the repo, scan them for
   common targets and propose redirects preemptively.
3. **User expresses frustration about permission prompts.** If Claude Code
   keeps prompting for the same pattern of command, a deny rule in veer is
   often the right fix.
4. **After running `veer list` and seeing sparse rules.** The user may not
   yet know what's worth codifying; propose 2-3 repo-specific rules.

When suggesting rules, show the user the exact TOML or `veer add` command,
then run `veer test` on a representative input to demonstrate.

## Troubleshooting

- **"veer: no config at .veer/config.toml"** -- the hook is installed but
  has no rules. Run `veer install` to create a starter config, or `veer
  uninstall` to remove the hook.
- **Rule doesn't match what you expect** -- run `veer test "<cmd>"` and
  iterate. Matchers operate on the parsed AST, so quoting and compound
  commands sometimes behave differently than they look.
- **Want to see match details** -- `veer test` prints match kind and the
  matched rule id.
