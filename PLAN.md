# Teash Improvement Plan

## Phase 0: Build Tooling

Set up mise tasks so building and testing have consistent, repeatable entry points.

### 0.1 Initialize mise configuration
- Create a `mise.toml` at the project root
- Configure the Go tool version (matching the `go.mod` version target)

### 0.2 Add build task
- `mise run build` -- runs `go build -o teash .`
- Should produce the binary in the project root (or a `bin/` directory)

### 0.3 Add test task
- `mise run test` -- runs `go test ./...`
- Consider a `test:verbose` variant that runs with `-v`

### 0.4 Add lint task
- `mise run lint` -- runs `go vet ./...` (and optionally `staticcheck` or `golangci-lint` if desired)

### 0.5 Document in CLAUDE.md
- Add build/test instructions referencing mise tasks so they're discoverable

## Phase 1: Add Tests

Add tests against the current codebase before making any changes. This gives us a safety net so that cleanup and refactoring in later phases can be verified against known-good behavior -- and if a test breaks, we can decide whether the change was intentional.

All tests must work without `tsh` installed or any Teleport credentials. Tests that need server data will use mock JSON responses. Tests must never invoke `tsh ssh` or attempt to connect to any server.

### 1.1 Unit test `stripInvalidJSONPrefix`
- Valid JSON passes through unchanged
- JSON with leading garbage text is cleaned
- Empty input returns empty
- Input with no valid JSON returns empty

### 1.2 Unit test `filterNodesBySearch`
- All-column fuzzy search returns correct matches
- Single-column search filters on the right column
- Empty search returns all nodes
- Search with no matches returns empty
- Search is case-insensitive
- Results are ordered by match quality

### 1.3 Unit test `fillTable`
- Correct number of rows/columns generated from nodes
- Dynamic label columns are discovered and sorted
- Nodes with different label sets handled (sparse labels)
- Empty node list produces empty table

### 1.4 Test the Teleport interface with mocks
- Create a `mockTeleport` implementation for tests (similar to existing `demo` but more configurable)
- Test `GetNodes` JSON parsing with recorded tsh output fixtures (user will provide sample JSON)
- Test `GetCluster` JSON parsing with recorded tsh status output
- Test error cases: invalid JSON, missing fields, not-logged-in response

### 1.5 Bubbletea model tests
- Test `Init` triggers node fetch
- Test key handling: `/` enters search mode, `esc` exits it, `q` quits, `enter` sets tsh command
- Test `View` output in different states (loading, searching, normal)
- Use bubbletea's `teatest` package for model testing where appropriate

## Phase 2: Clean Up

Remove noise and fix obvious issues. Tests from Phase 1 will catch any unintended behavioral changes.

### 2.1 Remove dead code and debug artifacts
- Delete unused type definitions in `teleport.go:137-155` (`teleportItem`, `metadata`, `spec`, `cmdLabels`, `Result`)
- Remove `litter.Sdump("WTF", ...)` call in `main.go:108`
- Remove `tea.LogToFile("debug.log", ...)` and associated log separator in `main.go:324-328`
- Remove commented-out log lines throughout `main.go`
- Remove `sanity-io/litter` from imports and `go.mod`
- Remove `davecgh/go-spew` from imports and `go.mod` (only used in empty test stubs)

### 2.2 Remove global state
- Delete the global `var tsh string` in `main.go:36`
- In `NewTeleport()`, use a local variable instead of assigning to the global

### 2.3 Replace panics with graceful error handling
- `main.go:131` (`case error: panic(msg)`) -- display the error in the TUI and quit gracefully
- `main.go:321-322` (`NewTeleport` panic) -- already handled with `fmt.Println` + `os.Exit(1)` pattern nearby; make consistent
- `main.go:364` (program run panic) -- same treatment
- `teleport.go:41` (`Connect` panic) -- show error and exit
- `teleport.go:181` (demo `Connect` panic) -- same

### 2.4 Delete empty test stubs
- Remove `teleport_test.go` entirely. The current tests are auto-generated stubs with no actual test cases, and they call real `tsh` commands which can't run without credentials. Proper tests were added in Phase 1.

## Phase 3: Update Dependencies

### 3.1 Bump Go version
- Update `go.mod` from `go 1.20` to current stable (1.23+)
- Replace `golang.org/x/exp/maps` with stdlib `maps` (available since Go 1.21)
- Replace `golang.org/x/exp/slices` with stdlib `slices` (available since Go 1.21)
- Remove `golang.org/x/exp` from `go.mod`

### 3.2 Update charmbracelet stack
- Update `charmbracelet/bubbletea`, `charmbracelet/bubbles`, `charmbracelet/lipgloss` to latest versions
- Adapt code to any API changes (lipgloss v1.x changed the styling API)

## Phase 4: Feature Improvements

Build on the tested, cleaned-up, and updated foundation. These are ordered by impact and independence -- each can be done as a separate PR.

### 4.1 Dynamic table sizing
- Calculate column widths based on the longest value in each column
- Set table height based on terminal size (use bubbletea's `WindowSizeMsg`)
- Handle terminal resize events

### 4.2 Fix search ranking
- When Levenshtein distances tie, prefer prefix matches
- Consider weighting exact matches highest, then prefix, then substring, then fuzzy

### 4.3 Improve column selection
- Replace the number-key-based column selection with arrow-key navigation
- Remove the hardcoded 1-9 limit
- Show visual indicator of selected column

### 4.4 Refresh node list
- Add a keybinding (e.g., `r`) to re-fetch nodes from tsh
- Show loading indicator during refresh while keeping current list visible

### 4.5 Error display in UI
- Instead of panicking or exiting on errors, show error messages inline in the TUI
- Allow recovery where possible (e.g., retry node fetch)

### 4.6 Support other tsh resource types
- Currently teash only supports `tsh ssh` to nodes. Extend to support other Teleport resource types:
  - **Databases:** list via `tsh db ls --format json`, connect with `tsh db connect <name>`
  - **Kubernetes:** list via `tsh kube ls --format json`, connect with `tsh kube login <name>`
- Add a resource type selector (e.g., tab bar or keybinding to switch between Nodes / Databases / Kube)
- Each resource type will need its own JSON parsing since the schema differs
- The `Teleport` interface will need to be generalized or extended to support different resource types and their respective columns/labels
- The `Connect` method will need to dispatch to the appropriate `tsh` subcommand based on resource type

### 4.7 Inventory cache for fast startup
- On first run, fetch inventory from `tsh` and write it to a local cache file (e.g., `~/.config/teash/cache/<cluster>.json`)
- On subsequent runs, load from cache immediately so the table is populated instantly
- Kick off a background refresh that updates the cache and swaps in fresh data when ready
- Cache should be keyed by cluster name to support multiple `tsh` profiles
- Consider a configurable TTL or staleness indicator so users know when data is stale

### 4.8 User-defined labels
- Allow users to define their own labels that get merged with Teleport labels for display and search
- Store in a local config file (e.g., `~/.config/teash/labels.yaml`) mapping hostname to key-value pairs:
  ```yaml
  host1.example.com:
    role: redis
    notes: "primary cache node"
  host5.example.com:
    role: postgres
  ```
- User labels appear as additional columns alongside Teleport labels
- User labels are searchable with the same fuzzy search
- Provide a keybinding to add/edit a label for the currently selected server without leaving the TUI

### 4.9 Shared label definitions
- Allow importing user-defined labels from a URL (e.g., a raw GitHub file)
- Support YAML or TOML format
- Configure shared label sources in a config file (e.g., `~/.config/teash/config.yaml`):
  ```yaml
  shared_labels:
    - url: https://raw.githubusercontent.com/team/teash-labels/main/labels.yaml
      name: team-labels
  ```
- Fetch and cache shared labels on startup (or on demand with a keybinding)
- Merge priority: shared labels < user labels (user labels win on conflict)
- Shared labels should be read-only in the TUI; editing creates a user-label override

### 4.10 Additional features (to discuss)
- Configurable SSH user per connection
- Bookmark/favorite servers
- Persist last search or filter across sessions
- Sort by clicking column headers

## Sequencing

Phases 1-3 should be done in order before Phase 4. Phase 1 (tests) establishes a safety net, Phase 2 (cleanup) removes noise with tests to catch regressions, and Phase 3 (dependency updates) builds on a clean codebase. Within Phase 4, features are independent and can be done in any order based on priority. I'd suggest starting with 4.1 (dynamic sizing) and 4.5 (error display) since they improve the experience most broadly.
