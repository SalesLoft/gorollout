# gorollout
Fast and concurrent-safe feature flags for golang based on Redis. Inspired by the [ruby rollout gem](https://github.com/fetlife/rollout).

[![](https://godoc.org/github.com/salesloft/gorollout?status.svg)](http://godoc.org/github.com/salesloft/gorollout)
[![Build Status](https://travis-ci.org/salesloft/gorollout.svg?branch=master)](https://travis-ci.org/salesloft/gorollout)
[![Code Coverage](https://codecov.io/gh/salesloft/gorollout/branch/master/graph/badge.svg)](https://codecov.io/gh/salesloft/gorollout)
[![Go Report Card](https://goreportcard.com/badge/github.com/salesloft/gorollout)](https://goreportcard.com/report/github.com/salesloft/gorollout)

## Installation

```bash
go get -u github.com/salesloft/gorollout
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
    // instantiate a feature manager
    manager := rollout.NewManager(redis.NewClient(&redis.Options{}))

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

    // check if a feature is active for a specific team
    manager.IsTeamActive(99, apples)

    // check multiple feature flags at once
    manager.IsActiveMulti(apples, bananas)
}
```
