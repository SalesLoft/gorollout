# gorollout - Command Line Interface (CLI)

A command line interface for viewing and managing features.

## Installation

```bash
GO111MODULE=on GOPROXY=direct GOSUMDB=off go get github.com/salesloft/gorollout/cmd/rollout
```

## Build Executable

Add the generated executables to the release.

```
env GOOS=darwin GOARCH=amd64 go build -o rollout-darwin-amd64
env GOOS=linux GOARCH=amd64 go build -o rollout-linux-amd64
```

### Help

```
~  rollout help
NAME:
   rollout - Fast and concurrent-safe feature flags for golang based on Redis.

USAGE:
   rollout [global options] command [command options] [arguments...]

COMMANDS:
   list                 List all active feature flags
   activate-percentage  Rollout a feature flag the given percentage
   activate             Activate a feature flag for all teams
   deactivate           Deactivate a feature flag for all teams
   activate-team        Activate a feature flag for a specific team
   deactivate-team      Deactivate a feature flag for a specific team
   delete               Delete a feature flag from the database
   help, h              Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --host value    Redis host connection string (comma separated) (default: "localhost:6379")
   --prefix value  Key prefix for feature flags (default: "rollout")
   --help, -h      show help (default: false)
```

### Example Usage

```
~  rollout activate apples
~  rollout activate-team bananas 99
~  rollout activate-percentage cherries 25
~  rollout list
 flag		percentage	active_teams
 ----		----------	------------
 apples		100
 bananas	0		99
 cherries	25
```
