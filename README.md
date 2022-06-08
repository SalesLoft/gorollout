# gorollout
Fast and concurrent-safe feature flags for golang based on Redis. Inspired by the [ruby rollout gem](https://github.com/fetlife/rollout).

[![](https://godoc.org/github.com/salesloft/gorollout?status.svg)](http://godoc.org/github.com/salesloft/gorollout)
[![Build Status](https://github.com/SalesLoft/gorollout/workflows/Go/badge.svg)](https://github.com/SalesLoft/gorollout/actions)
[![Code Coverage](https://codecov.io/gh/salesloft/gorollout/branch/master/graph/badge.svg)](https://codecov.io/gh/salesloft/gorollout)
[![Go Report Card](https://goreportcard.com/badge/github.com/salesloft/gorollout)](https://goreportcard.com/report/github.com/salesloft/gorollout)

## Installation

```bash
go get github.com/salesloft/gorollout
```

## Usage

```golang
package main

import (
    "github.com/go-redis/redis/v7"
    rollout "github.com/salesloft/gorollout"
)

var (
    apples = rollout.NewFeature("apples")
    bananas = rollout.NewFeature("bananas")
)

func main() {
    // instantiate a feature manager with a redis client and namespace prefix
    manager := rollout.NewManager(redis.NewClient(&redis.Options{}), "rollout")

    // activate a feature
    manager.Activate(apples)

    // deactivate a feature
    manager.Deactivate(apples)

    // rollout a feature to 25% of teams
    manager.ActivatePercentage(apples, 25)

    // explicitly activate a feature for team with id 99
    manager.ActivateTeam(99, apples)

    // check if a feature is active, globally
    manager.IsActive(apples)

    // check if a feature is active for a specific team (randomize percentage disabled)
    manager.IsTeamActive(99, apples, false)

    // check multiple feature flags at once
    manager.IsActiveMulti(apples, bananas)
}
```

## Command Line Interface (CLI)

gorollout also includes a [command line interface](cmd/rollout/README.md) for viewing and managing feature flags.

## Building and pushing the Docker image

    docker buildx create --use
    docker buildx build --platform linux/amd64,linux/arm64/v8 -t salesloft/gorollout:v1.1.2 . --push
