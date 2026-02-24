This project provides a TUI (using the bubbletea library) for interactively searching for and connecting to servers in Teleport inventory.  It assumes that the `tsh` command is installed and configured on the system, and acts as a wrapper around it.  The main goal is to provide a more user-friendly interface for navigating and connecting to servers managed by Teleport.

## Dependency Management

This project uses Flox to manage dependencies.  There is a flox-mcp-server that Claude can use to learn how to use flox.  Even though the project uses mise as a task runner, do not install tools with mise.  All tools will be managed with Flox.

## Build & Test

This project uses `mise` as a task runner. Available tasks:

- `mise run build` -- builds the `teash` binary
- `mise run test` -- runs all tests
- `mise run test:verbose` -- runs all tests with verbose output
- `mise run lint` -- runs `go vet`

## Testability

The code should be structured so that things like the Teleport data layer, user attribute management, caching, etc are not coupled to the TUI code.  This will allow us to write unit tests for the core logic without needing to worry about the TUI rendering.  The TUI code should be as thin as possible, ideally just taking data from the core logic and rendering it, and sending user input back to the core logic without doing any processing itself.  This separation of concerns will make it much easier to write tests for the core functionality of the program.  The core functionality of the program should have extensive unit tests.  

## Testing the TUI

Use https://github.com/charmbracelet/x/tree/main/exp/teatest to write tests that need to verify that the TUI is rendering correctly. 


## End to End Tests

Claude must never attempt to connect to a server during testing.  Any end-to-end tests that invoke `tsh ssh` should only be run by the user.  Claude will also not have access to credentials necessary to `tsh login`.  All tests that need to get the list of servers will need to mock the json output of tsh commands using pre-recorded json responses provided by the user.

## Requirements

When run, the program will use `tsh ls` to get the list of servers in the currently logged in cluster.  If there is no teleport session then attempt to run `tsh login` and allow the user to authenticate.  The data from `tsh ls` should be cached locally to avoid having to run `tsh ls` every time the user opens the program.  The user should be able to refresh the cache manually to get the latest list of servers.  The user should also be able to add custom attributes to servers (e.g. "production", "database", "web server") and these attributes should be stored locally and associated with the server in the cache.

The program needs to support multiple teleport clusters.  The user will only be logged in to one cluster at a time, but the cache and user-defined server attributes should be stored separately for each cluster.  The user should be able to switch between clusters without losing their server cache or user-defined attributes.

## TUI

The UI should show a table view of servers with static columns for Hostname, IP, OS followed by a column for each label associated with the server.  
The labels will come from the tsh json metadata and can also be user defined.  User defined labels will be discussed in more detail later. 
Each column in the table should be sortable and filterable.

### Keys

- `j`/`k` to move up and down the list of servers
- `enter` to connect to the selected server using `tsh ssh`
- `r` to refresh the server list cache by re-running `tsh ls`
- `a` to add a user defined label to the selected server
- `/` to start a search query to filter the list of servers.  The search should match against all columns (hostname, ip, os, and labels).
- `c` to choose a column to filter by, then enter a value to filter the list of servers by that column.  For example, the user could press `c`, then select the "OS" column, then enter "linux" to only show servers with "linux" in the OS column.

### User Defined Labels

User defined labels can be added interactively by selecting a server and pressing `a`.  The user will then be prompted to enter a label key and value (e.g. "environment:production").  This label will then be associated with the server and displayed as an additional column in the table.  

User defined labels should also be able to be imported in bulk from a yaml/toml file, either locally or from a git repository.  The file should have a simple format that maps server hostnames to label key/value pairs.  For example:

```yaml
cluster:
  server1.example.com:
    environment: production
    role: web
```

User defined labels should be easily exportable to a yaml/toml file as well, so that users can manage their labels in their preferred format and share them across machines.

### Cache

The cache should be stored in a local file in the user's home directory following the XDG Base Directory Specification (e.g. `~/.cache/teash/cache.json`).  The cache should simply store the raw JSON output from `tsh ls` for each cluster, keyed by the cluster name.  When the program starts, it should load the cache for the currently logged in cluster and use that to populate the server list.  When the user refreshes the cache, it should re-run `tsh ls`, update the cache file, and refresh the server list in the UI.

The UI should show an indicator of when the cache was last updated, and whether it is currently being refreshed.  If there is an error refreshing the cache (e.g. `tsh ls` fails), the UI should show an error message and keep displaying the old cache until a successful refresh.
