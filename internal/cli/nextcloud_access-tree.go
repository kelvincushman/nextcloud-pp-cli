// Copyright 2026 kelvincushman. Licensed under Apache-2.0.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

type accessEntry struct {
	ShareType   int      `json:"share_type"`
	ShareWith   string   `json:"share_with"`
	Permissions int      `json:"permissions"`
	Members     []string `json:"members,omitempty"`
}

func newAccessTreeCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "access-tree <path>",
		Short:   "Show who has access to a path (users + groups)",
		Example: "  nextcloud-pp-cli access-tree /Documents/Report.pdf",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			filePath := args[0]
			ocsHeaders := map[string]string{"OCS-APIRequest": "true"}

			raw, err := c.GetWithHeaders("/ocs/v2.php/apps/files_sharing/api/v1/shares",
				map[string]string{"format": "json", "path": filePath, "subfiles": "true"}, ocsHeaders)
			if err != nil {
				return classifyAPIError(err, flags)
			}

			var sharesEnv struct {
				OCS struct {
					Data []struct {
						ID          string `json:"id"`
						ShareType   int    `json:"share_type"`
						ShareWith   string `json:"share_with"`
						Permissions int    `json:"permissions"`
					} `json:"data"`
				} `json:"ocs"`
			}
			if err := json.Unmarshal(raw, &sharesEnv); err != nil {
				return fmt.Errorf("parsing shares: %w", err)
			}

			var entries []accessEntry

			for _, s := range sharesEnv.OCS.Data {
				entry := accessEntry{
					ShareType:   s.ShareType,
					ShareWith:   s.ShareWith,
					Permissions: s.Permissions,
				}

				// Share type 1 = group share — expand members
				if s.ShareType == 1 && s.ShareWith != "" {
					groupPath := fmt.Sprintf("/ocs/v2.php/cloud/groups/%s", s.ShareWith)
					groupRaw, gErr := c.GetWithHeaders(groupPath, map[string]string{"format": "json"}, ocsHeaders)
					if gErr == nil {
						var groupEnv struct {
							OCS struct {
								Data struct {
									Users []string `json:"users"`
								} `json:"data"`
							} `json:"ocs"`
						}
						if json.Unmarshal(groupRaw, &groupEnv) == nil {
							entry.Members = groupEnv.OCS.Data.Users
						}
					}
				}

				entries = append(entries, entry)
			}

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(entries)
			}

			// Tree-style output
			shareTypeLabel := func(t int) string {
				switch t {
				case 0:
					return "user"
				case 1:
					return "group"
				case 3:
					return "public-link"
				case 6:
					return "federated"
				default:
					return fmt.Sprintf("type-%d", t)
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Access tree for: %s\n", filePath)
			for _, e := range entries {
				label := shareTypeLabel(e.ShareType)
				fmt.Fprintf(cmd.OutOrStdout(), "  [%s] %s (perms: %d)\n", label, e.ShareWith, e.Permissions)
				for _, member := range e.Members {
					fmt.Fprintf(cmd.OutOrStdout(), "    └─ %s\n", member)
				}
			}
			if len(entries) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "  (no shares found)\n")
			}
			return nil
		},
	}

	return cmd
}
