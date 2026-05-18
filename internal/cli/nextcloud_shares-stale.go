// Copyright 2026 kelvincushman. Licensed under Apache-2.0.

package cli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

type staleShare struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	ShareType  int    `json:"share_type"`
	Owner      string `json:"owner"`
	Created    string `json:"created"`
	Expiry     string `json:"expiry"`
	Reason     string `json:"reason"`
}

func newSharesStaleCmd(flags *rootFlags) *cobra.Command {
	var days int

	cmd := &cobra.Command{
		Use:     "shares-stale",
		Short:   "Report shares that are expired or stale",
		Example: "  nextcloud-pp-cli shares-stale\n  nextcloud-pp-cli shares-stale --days 7",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			ocsHeaders := map[string]string{"OCS-APIRequest": "true"}
			raw, err := c.GetWithHeaders("/ocs/v2.php/apps/files_sharing/api/v1/shares",
				map[string]string{"format": "json"}, ocsHeaders)
			if err != nil {
				return classifyAPIError(err, flags)
			}

			var sharesEnv struct {
				OCS struct {
					Data []struct {
						ID         string `json:"id"`
						Path       string `json:"path"`
						ShareType  int    `json:"share_type"`
						UIDOwner   string `json:"uid_owner"`
						STime      int64  `json:"stime"`
						Expiration string `json:"expiration"`
					} `json:"data"`
				} `json:"ocs"`
			}
			if err := json.Unmarshal(raw, &sharesEnv); err != nil {
				return fmt.Errorf("parsing shares: %w", err)
			}

			now := time.Now()
			staleAfter := now.AddDate(0, 0, -days)

			var stale []staleShare
			for _, s := range sharesEnv.OCS.Data {
				created := time.Unix(s.STime, 0)
				reason := ""

				if s.Expiration != "" {
					// Parse expiration date (typically "YYYY-MM-DD 00:00:00")
					expTime, parseErr := time.Parse("2006-01-02 15:04:05", s.Expiration)
					if parseErr != nil {
						expTime, parseErr = time.Parse("2006-01-02", s.Expiration)
					}
					if parseErr == nil && expTime.Before(now) {
						reason = "expired"
					}
				} else if created.Before(staleAfter) {
					reason = "stale"
				}

				if reason == "" {
					continue
				}

				expStr := s.Expiration
				if expStr == "" {
					expStr = "-"
				}

				stale = append(stale, staleShare{
					ID:        s.ID,
					Path:      s.Path,
					ShareType: s.ShareType,
					Owner:     s.UIDOwner,
					Created:   created.Format("2006-01-02"),
					Expiry:    expStr,
					Reason:    reason,
				})
			}

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(stale)
			}

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 2, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tPATH\tTYPE\tOWNER\tCREATED\tEXPIRY\tREASON")
			for _, s := range stale {
				fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\t%s\t%s\n",
					s.ID, truncate(s.Path, 40), s.ShareType, s.Owner, s.Created, s.Expiry, s.Reason)
			}
			return tw.Flush()
		},
	}

	cmd.Flags().IntVar(&days, "days", 30, "Consider shares stale if older than N days with no expiry")
	return cmd
}
