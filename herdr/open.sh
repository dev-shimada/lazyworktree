#!/bin/sh
# Action entrypoint: opens the lazyworktree popup pane.
exec "$HERDR_BIN_PATH" plugin pane open --plugin "$HERDR_PLUGIN_ID" --entrypoint main
