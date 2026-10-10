#!/bin/sh
# Pane entrypoint: runs lazyworktree in the directory herdr was focused on.
# Plugin commands start in the plugin root, so cd to the invoking context's
# directory first (the context JSON has no documented schema; sed avoids a
# jq dependency).
ctx=${HERDR_PLUGIN_CONTEXT_JSON:-}
field() { printf '%s' "$ctx" | sed -n "s/.*\"$1\":\"\([^\"]*\)\".*/\1/p"; }
dir=$(field focused_pane_cwd)
[ -n "$dir" ] || dir=$(field workspace_cwd)
[ -n "$dir" ] && cd "$dir" 2>/dev/null

if command -v lazyworktree >/dev/null 2>&1; then
	exec lazyworktree
fi
echo "lazyworktree not found on PATH. Install it: brew install dev-shimada/lazyworktree/lazyworktree" >&2
sleep 5
exit 1
