package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/go-redis/redis/v7"
	rollout "github.com/salesloft/gorollout"
	"github.com/urfave/cli/v2"
	"github.com/vmihailenco/msgpack/v4"
)

var (
	app = &cli.App{
		Name:  "Deals Feature Flags",
		Usage: "manage feature flags in SalesLoft Deals.",

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "host",
				Usage: "Redis host connection string (comma separated)",
				Value: "localhost:6379",
			},
			&cli.StringFlag{
				Name:  "prefix",
				Usage: "Key prefix for Deals feature flags",
				Value: "dealsff",
			},
		},

		Commands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "List all active feature flags",
				Action: listFeatureFlags,
			},
			{
				Name:      "rollout",
				Usage:     "Rollout a feature flag the given percentage",
				Action:    rolloutFeatureFlag,
				ArgsUsage: "[feature name] [percentage]",
			},
			{
				Name:      "activate",
				Usage:     "Activate a feature flag for all teams",
				Action:    activateFeatureFlag,
				ArgsUsage: "[feature name]",
			},
			{
				Name:      "deactivate",
				Usage:     "Deactivate a feature flag for all teams",
				Action:    deactivateFeatureFlag,
				ArgsUsage: "[feature name]",
			},
			{
				Name:      "activate-team",
				Usage:     "Activate a feature flag for a specific team",
				Action:    activateTeamFeatureFlag,
				ArgsUsage: "[feature name] [team_id]",
			},
			{
				Name:      "deactivate-team",
				Usage:     "Deactivate a feature flag for a specific team",
				Action:    deactivateTeamFeatureFlag,
				ArgsUsage: "[feature name] [team_id]",
			},
			{
				Name:      "delete",
				Usage:     "Delete a feature flag from the database",
				Action:    deleteFeatureFlag,
				ArgsUsage: "[feature name]",
			},
		},
	}
)

func main() {
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func listFeatureFlags(c *cli.Context) error {
	client := redis.NewUniversalClient(
		&redis.UniversalOptions{
			Addrs: strings.Split(c.String("host"), ","),
		},
	)

	var cursor uint64
	var allKeys []string

	for {
		var keys []string
		var err error
		keys, cursor, err = client.Scan(cursor, c.String("prefix")+":*", 100).Result()
		if err != nil {
			return err
		}

		allKeys = append(allKeys, keys...)

		if cursor == 0 {
			break
		}
	}

	sort.Strings(allKeys)

	var results []interface{}
	var err error
	if len(allKeys) > 0 {
		results, err = client.MGet(allKeys...).Result()
		if err != nil {
			return err
		}
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 8, 0, '\t', 0)
	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t%s\t", "flag", "percentage", "active_teams")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t", "----", "----------", "------------")

	for i, result := range results {
		dec := msgpack.NewDecoder(bytes.NewBufferString(result.(string)))

		percentage, err := dec.DecodeUint8()
		if err != nil {
			return err
		}

		var teamIDs []string
		n, err := dec.DecodeArrayLen()
		if err != nil {
			return err
		}

		if n > 0 {
			teamIDs = make([]string, n)
			for i := 0; i < n; i++ {
				teamID, err := dec.DecodeInt64()
				if err != nil {
					return err
				}

				teamIDs[i] = strconv.FormatInt(teamID, 10)
			}
		}

		fmt.Fprintf(w, "\n %s\t%d\t%s\t", allKeys[i][8:], percentage, strings.Join(teamIDs, ","))
	}

	fmt.Fprint(w, "\n")

	return nil
}

func rolloutFeatureFlag(c *cli.Context) error {
	ff := rollout.NewFeature(c.Args().Get(0))
	if ff.Name() == "" {
		return cli.NewExitError("Missing required feature flag name", 1)
	}

	percentageStr := c.Args().Get(1)
	if percentageStr == "" {
		return cli.NewExitError("Missing required percentage", 1)
	}

	percentage, err := strconv.ParseUint(percentageStr, 10, 8)
	if err != nil {
		return err
	}

	manager := rollout.NewManager(
		redis.NewUniversalClient(
			&redis.UniversalOptions{
				Addrs: strings.Split(c.String("host"), ","),
			},
		),
		c.String("prefix"),
	)

	return manager.ActivatePercentage(ff, uint8(percentage))
}

func activateFeatureFlag(c *cli.Context) error {
	ff := rollout.NewFeature(c.Args().Get(0))
	if ff.Name() == "" {
		return cli.NewExitError("Missing required feature flag name", 1)
	}

	manager := rollout.NewManager(
		redis.NewUniversalClient(
			&redis.UniversalOptions{
				Addrs: strings.Split(c.String("host"), ","),
			},
		),
		c.String("prefix"),
	)

	return manager.Activate(ff)
}

func deactivateFeatureFlag(c *cli.Context) error {
	ff := rollout.NewFeature(c.Args().Get(0))
	if ff.Name() == "" {
		return cli.NewExitError("Missing required feature flag name", 1)
	}

	manager := rollout.NewManager(
		redis.NewUniversalClient(
			&redis.UniversalOptions{
				Addrs: strings.Split(c.String("host"), ","),
			},
		),
		c.String("prefix"),
	)

	return manager.Deactivate(ff)
}

func activateTeamFeatureFlag(c *cli.Context) error {
	ff := rollout.NewFeature(c.Args().Get(0))
	if ff.Name() == "" {
		return cli.NewExitError("Missing required feature flag name", 1)
	}

	teamIDStr := c.Args().Get(1)
	if teamIDStr == "" {
		return cli.NewExitError("Missing required team id", 1)
	}

	teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
	if err != nil {
		return err
	}

	manager := rollout.NewManager(
		redis.NewUniversalClient(
			&redis.UniversalOptions{
				Addrs: strings.Split(c.String("host"), ","),
			},
		),
		c.String("prefix"),
	)

	return manager.ActivateTeam(teamID, ff)
}

func deactivateTeamFeatureFlag(c *cli.Context) error {
	ff := rollout.NewFeature(c.Args().Get(0))
	if ff.Name() == "" {
		return cli.NewExitError("Missing required feature flag name", 1)
	}

	teamIDStr := c.Args().Get(1)
	if teamIDStr == "" {
		return cli.NewExitError("Missing required team id", 1)
	}

	teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
	if err != nil {
		return err
	}

	manager := rollout.NewManager(
		redis.NewUniversalClient(
			&redis.UniversalOptions{
				Addrs: strings.Split(c.String("host"), ","),
			},
		),
		c.String("prefix"),
	)

	return manager.DeactivateTeam(teamID, ff)
}

func deleteFeatureFlag(c *cli.Context) error {
	ff := rollout.NewFeature(c.Args().Get(0))
	if ff.Name() == "" {
		return cli.NewExitError("Missing required feature flag name", 1)
	}

	client := redis.NewUniversalClient(
		&redis.UniversalOptions{
			Addrs: strings.Split(c.String("host"), ","),
		},
	)

	count, err := client.Del(c.String("prefix") + ":" + ff.Name()).Result()
	if err != nil {
		return err
	}
	if count == 0 {
		return cli.NewExitError("Feature flag was not found", 0)
	}

	return nil
}
