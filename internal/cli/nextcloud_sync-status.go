// Copyright 2026 kelvincushman. Licensed under Apache-2.0.

package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"nextcloud-pp-cli/internal/store"
)

type syncStatusReport struct {
	LastSynced    string         `json:"last_synced"`
	ItemsCached   map[string]int `json:"items_cached"`
	ServerVersion string         `json:"server_version"`
	DBPath        string         `json:"db_path"`
}

func newSyncStatusCmd(flags *rootFlags) *cobra.Command {
	var dbPath string

	cmd := &cobra.Command{
		Use:     "sync-status",
		Short:   "Compare local cache vs live API to detect drift",
		Example: "  nextcloud-pp-cli sync-status",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}

			if dbPath == "" {
				dbPath = defaultDBPath("nextcloud-pp-cli")
			}

			// Open store read-only; it may not exist if never synced
			db, dbErr := store.Open(dbPath)
			var itemsCached map[string]int
			var lastSynced time.Time
			if dbErr == nil {
				defer db.Close()
				itemsCached, _ = db.Status()
				// Find the most recent sync time across all resources
				for rt := range itemsCached {
					_, syncedAt, _, err := db.GetSyncState(rt)
					if err == nil && syncedAt.After(lastSynced) {
						lastSynced = syncedAt
					}
				}
			} else {
				itemsCached = map[string]int{}
			}

			// Fetch server version from capabilities
			serverVersion := "unknown"
			c, err := flags.newClient()
			if err == nil {
				ocsHeaders := map[string]string{"OCS-APIRequest": "true"}
				raw, capErr := c.GetWithHeaders("/ocs/v2.php/cloud/capabilities",
					map[string]string{"format": "json"}, ocsHeaders)
				if capErr == nil {
					var capEnv struct {
						OCS struct {
							Data struct {
								Version struct {
									String string `json:"string"`
								} `json:"version"`
							} `json:"data"`
						} `json:"ocs"`
					}
					if json.Unmarshal(raw, &capEnv) == nil {
						serverVersion = capEnv.OCS.Data.Version.String
					}
				}
			}

			lastSyncedStr := "never"
			if !lastSynced.IsZero() {
				lastSyncedStr = lastSynced.UTC().Format(time.RFC3339)
			}

			report := syncStatusReport{
				LastSynced:    lastSyncedStr,
				ItemsCached:   itemsCached,
				ServerVersion: serverVersion,
				DBPath:        dbPath,
			}

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(report)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Last synced:    %s\n", report.LastSynced)
			fmt.Fprintf(cmd.OutOrStdout(), "Server version: %s\n", report.ServerVersion)
			fmt.Fprintf(cmd.OutOrStdout(), "DB path:        %s\n", report.DBPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Cached items:\n")
			if len(itemsCached) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "  (none — run 'nextcloud-pp-cli sync' first)\n")
			}
			for rt, count := range itemsCached {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-30s %d\n", rt, count)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/nextcloud-pp-cli/data.db)")
	return cmd
}
