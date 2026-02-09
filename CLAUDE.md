This project provides a TUI (using the bubbletea library) for interactively searching for and connecting to servers in Teleport inventory.  It assumes that the `tsh` command is installed and configured on the system, and acts as a wrapper around it.  The main goal is to provide a more user-friendly interface for navigating and connecting to servers managed by Teleport.

## Dependency Management

This project uses Flox to manage dependencies.  There is a flox-mcp-server that Claude can use to learn how to use flox.  Even though the project uses mise as a task runner, do not install tools with mise.  All tools will be managed with Flox.

## Build & Test

This project uses `mise` as a task runner. Available tasks:

- `mise run build` -- builds the `teash` binary
- `mise run test` -- runs all tests
- `mise run test:verbose` -- runs all tests with verbose output
- `mise run lint` -- runs `go vet`

## End to End Tests

Claude must never attempt to connect to a server during testing.  Any end-to-end tests that invoke `tsh ssh` should only be run by the user.  Claude will also not have access to credentials necessary to `tsh login`.  All tests that need to get the list of servers will need to mock the json output of tsh commands using pre-recorded json responses provided by the user.
