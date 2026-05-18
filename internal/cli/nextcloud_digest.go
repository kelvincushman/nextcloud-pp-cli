// Copyright 2026 kelvincushman. Licensed under Apache-2.0.

package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

type digestEntry struct {
	User         string `json:"user"`
	File         string `json:"file"`
	Count        int    `json:"count"`
	LastActivity string `json:"last_activity"`
}

// parseDuration supports "24h", "7d", "30m" etc.
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "d") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func newDigestCmd(flags *rootFlags) *cobra.Command {
	var since string

	cmd := &cobra.Command{
		Use:     "digest",
		Short:   "Recent activity digest grouped by user and file",
		Example: "  nextcloud-pp-cli digest\n  nextcloud-pp-cli digest --since 7d",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}

			dur, err := parseDuration(since)
			if err != nil {
				return fmt.Errorf("invalid --since value %q: %w", since, err)
			}
			sinceTime := time.Now().Add(-dur)
			sinceUnix := strconv.FormatInt(sinceTime.Unix(), 10)

			c, err2 := flags.newClient()
			if err2 != nil {
				return err2
			}

			ocsHeaders := map[string]string{"OCS-APIRequest": "true"}
			raw, fetchErr := c.GetWithHeaders("/ocs/v2.php/apps/activity/api/v2/activity/filter",
				map[string]string{
					"format": "json",
					"sort":   "desc",
					"limit":  "200",
					"since":  sinceUnix,
				}, ocsHeaders)
			if fetchErr != nil {
				return classifyAPIError(fetchErr, flags)
			}

			var activityEnv struct {
				OCS struct {
					Data []struct {
						User        string `json:"user"`
						ObjectName  string `json:"object_name"`
						Datetime    string `json:"datetime"`
					} `json:"data"`
				} `json:"ocs"`
			}
			if err := json.Unmarshal(raw, &activityEnv); err != nil {
				return fmt.Errorf("parsing activity: %w", err)
			}

			type key struct{ user, file string }
			countMap := map[key]int{}
			lastMap := map[key]string{}

			for _, a := range activityEnv.OCS.Data {
				k := key{a.User, a.ObjectName}
				countMap[k]++
				if lastMap[k] == "" || a.Datetime > lastMap[k] {
					lastMap[k] = a.Datetime
				}
			}

			var entries []digestEntry
			for k, cnt := range countMap {
				entries = append(entries, digestEntry{
					User:         k.user,
					File:         k.file,
					Count:        cnt,
					LastActivity: lastMap[k],
				})
			}
			sort.Slice(entries, func(i, j int) bool {
				if entries[i].LastActivity != entries[j].LastActivity {
					return entries[i].LastActivity > entries[j].LastActivity
				}
				return entries[i].Count > entries[j].Count
			})

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(entries)
			}

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 2, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "USER\tFILE\tCOUNT\tLAST ACTIVITY")
			for _, e := range entries {
				fmt.Fprintf(tw, "%s\t%s\t%d\t%s\n", e.User, truncate(e.File, 50), e.Count, e.LastActivity)
			}
			return tw.Flush()
		},
	}

	cmd.Flags().StringVar(&since, "since", "24h", "How far back to look (e.g. 24h, 7d)")
	return cmd
}
