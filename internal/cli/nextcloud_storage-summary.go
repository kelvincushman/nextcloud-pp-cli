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

type userQuota struct {
	UserID    string  `json:"user_id"`
	UsedBytes int64   `json:"used_bytes"`
	TotalBytes int64  `json:"total_bytes"`
	QuotaPct  float64 `json:"quota_pct"`
}

func newStorageSummaryCmd(flags *rootFlags) *cobra.Command {
	var threshold float64

	cmd := &cobra.Command{
		Use:     "storage-summary",
		Short:   "Summarize storage usage across all users",
		Example: "  nextcloud-pp-cli storage-summary\n  nextcloud-pp-cli storage-summary --threshold 80",
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

			// List all users
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
			userIDs := usersEnv.OCS.Data.Users

			// Fetch each user in parallel (max 10 goroutines)
			sem := make(chan struct{}, 10)
			var mu sync.Mutex
			var quotas []userQuota
			var wg sync.WaitGroup
			var fetchErr error

			for _, uid := range userIDs {
				uid := uid
				wg.Add(1)
				go func() {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()

					userPath := fmt.Sprintf("/ocs/v2.php/cloud/users/%s", uid)
					userRaw, err := c.GetWithHeaders(userPath, map[string]string{"format": "json"}, ocsHeaders)
					if err != nil {
						mu.Lock()
						if fetchErr == nil {
							fetchErr = err
						}
						mu.Unlock()
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

					mu.Lock()
					quotas = append(quotas, userQuota{
						UserID:    uid,
						UsedBytes: q.Used,
						TotalBytes: q.Total,
						QuotaPct:  pct,
					})
					mu.Unlock()
				}()
			}
			wg.Wait()

			if fetchErr != nil {
				fmt.Fprintf(os.Stderr, "warning: some user fetches failed: %v\n", fetchErr)
			}

			// Filter by threshold
			var filtered []userQuota
			for _, q := range quotas {
				if q.QuotaPct >= threshold {
					filtered = append(filtered, q)
				}
			}

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(filtered)
			}

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 2, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "USER\tUSED\tTOTAL\tQUOTA%")

			var totalUsed, totalTotal int64
			for _, q := range filtered {
				totalStr := "-"
				if q.TotalBytes > 0 {
					totalStr = formatBytes(q.TotalBytes)
				}
				pctStr := "-"
				if q.TotalBytes > 0 {
					pctStr = fmt.Sprintf("%.1f%%", q.QuotaPct)
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", q.UserID, formatBytes(q.UsedBytes), totalStr, pctStr)
				totalUsed += q.UsedBytes
				totalTotal += q.TotalBytes
			}

			totalPct := "-"
			if totalTotal > 0 {
				totalPct = fmt.Sprintf("%.1f%%", float64(totalUsed)/float64(totalTotal)*100)
			}
			fmt.Fprintln(tw, "---\t---\t---\t---")
			fmt.Fprintf(tw, "TOTAL\t%s\t%s\t%s\n", formatBytes(totalUsed), formatBytes(totalTotal), totalPct)
			return tw.Flush()
		},
	}

	cmd.Flags().Float64Var(&threshold, "threshold", 0, "Only show users above this quota % (0 = show all)")
	return cmd
}
