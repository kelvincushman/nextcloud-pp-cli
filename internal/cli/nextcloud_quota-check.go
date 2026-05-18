// Copyright 2026 kelvincushman. Licensed under Apache-2.0.

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newQuotaCheckCmd(flags *rootFlags) *cobra.Command {
	var threshold float64

	cmd := &cobra.Command{
		Use:   "quota-check",
		Short: "Alert when users exceed quota threshold",
		Long: `Lists users whose quota usage exceeds the threshold percentage.
Exits with code 1 if any user exceeds the threshold (useful in monitoring scripts).`,
		Example: "  nextcloud-pp-cli quota-check\n  nextcloud-pp-cli quota-check --threshold 90",
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

			raw, err := c.GetWithHeaders("/ocs/v2.php/cloud/users", map[string]string{"format": "json"}, ocsHeaders)
			if err != nil {
				return classifyAPIError(err, flags)
			}

			var usersEnv struct {
				OCS struct {
					Data struct {
						Users []string `json:"users"`
					} `json:"data"`
				} `json:"ocs"`
			}
			if err := json.Unmarshal(raw, &usersEnv); err != nil {
				return fmt.Errorf("parsing users list: %w", err)
			}

			sem := make(chan struct{}, 10)
			var mu sync.Mutex
			var overLimit []userQuota
			var wg sync.WaitGroup

			for _, uid := range usersEnv.OCS.Data.Users {
				uid := uid
				wg.Add(1)
				go func() {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()

					userPath := fmt.Sprintf("/ocs/v2.php/cloud/users/%s", uid)
					userRaw, err := c.GetWithHeaders(userPath, map[string]string{"format": "json"}, ocsHeaders)
					if err != nil {
						return
					}

					var userEnv struct {
						OCS struct {
							Data struct {
								Quota struct {
									Used     int64   `json:"used"`
									Total    int64   `json:"total"`
									Relative float64 `json:"relative"`
								} `json:"quota"`
							} `json:"data"`
						} `json:"ocs"`
					}
					if err := json.Unmarshal(userRaw, &userEnv); err != nil {
						return
					}

					q := userEnv.OCS.Data.Quota
					pct := q.Relative
					if pct == 0 && q.Total > 0 {
						pct = float64(q.Used) / float64(q.Total) * 100
					}

					if pct >= threshold {
						mu.Lock()
						overLimit = append(overLimit, userQuota{
							UserID:    uid,
							UsedBytes: q.Used,
							TotalBytes: q.Total,
							QuotaPct:  pct,
						})
						mu.Unlock()
					}
				}()
			}
			wg.Wait()

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(overLimit)
			}

			if len(overLimit) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "All users are below %.1f%% quota threshold.\n", threshold)
				return nil
			}

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 2, 4, 2, ' ', 0)
			fmt.Fprintf(tw, "WARNING: %d user(s) exceed %.1f%% quota threshold\n\n", len(overLimit), threshold)
			fmt.Fprintln(tw, "USER\tUSED\tTOTAL\tQUOTA%")
			for _, q := range overLimit {
				totalStr := "-"
				if q.TotalBytes > 0 {
					totalStr = formatBytes(q.TotalBytes)
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%.1f%%\n", q.UserID, formatBytes(q.UsedBytes), totalStr, q.QuotaPct)
			}
			tw.Flush()

			// Exit code 1 signals monitoring system
			fmt.Fprintf(os.Stderr, "\nquota-check: %d user(s) above threshold\n", len(overLimit))
			return &cliError{code: 1, err: fmt.Errorf("%d user(s) exceed %.1f%% quota threshold", len(overLimit), threshold)}
		},
	}

	cmd.Flags().Float64Var(&threshold, "threshold", 80.0, "Quota usage percentage threshold (0-100)")
	return cmd
}
