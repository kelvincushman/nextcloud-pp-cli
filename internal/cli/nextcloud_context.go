// Copyright 2026 kelvincushman. Licensed under Apache-2.0.

package cli

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/spf13/cobra"
)

type nextcloudContext struct {
	ServerVersion    json.RawMessage `json:"server_version"`
	CurrentQuota     json.RawMessage `json:"current_quota"`
	RecentFiles      json.RawMessage `json:"recent_files"`
	RecentShares     json.RawMessage `json:"recent_shares"`
	RecentActivity   json.RawMessage `json:"recent_activity"`
	ActiveTalkRooms  json.RawMessage `json:"active_talk_rooms"`
	UnreadNotifications int          `json:"unread_notifications"`
	Errors           []string        `json:"errors,omitempty"`
}

func newNextcloudContextCmd(flags *rootFlags) *cobra.Command {
	var user string

	cmd := &cobra.Command{
		Use:     "context",
		Short:   "Dump all relevant Nextcloud state as JSON for AI agents",
		Example: "  nextcloud-pp-cli context\n  nextcloud-pp-cli context --user admin",
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
			ctx := nextcloudContext{}
			var mu sync.Mutex
			var wg sync.WaitGroup

			addErr := func(msg string) {
				mu.Lock()
				ctx.Errors = append(ctx.Errors, msg)
				mu.Unlock()
			}

			// 1. Server capabilities/version
			wg.Add(1)
			go func() {
				defer wg.Done()
				raw, err := c.GetWithHeaders("/ocs/v2.php/cloud/capabilities",
					map[string]string{"format": "json"}, ocsHeaders)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					ctx.ServerVersion = json.RawMessage(`null`)
					addErr(fmt.Sprintf("capabilities: %v", err))
					return
				}
				ctx.ServerVersion = raw
			}()

			// 2. Current user's quota
			wg.Add(1)
			go func() {
				defer wg.Done()
				targetUser := user
				if targetUser == "" && c.Config != nil {
					targetUser = c.Config.NextcloudUsername
				}
				if targetUser == "" {
					mu.Lock()
					ctx.CurrentQuota = json.RawMessage(`null`)
					mu.Unlock()
					return
				}
				userPath := fmt.Sprintf("/ocs/v2.php/cloud/users/%s", targetUser)
				raw, err := c.GetWithHeaders(userPath, map[string]string{"format": "json"}, ocsHeaders)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					ctx.CurrentQuota = json.RawMessage(`null`)
					addErr(fmt.Sprintf("quota: %v", err))
					return
				}
				ctx.CurrentQuota = raw
			}()

			// 3. Recent files (top 10 from shares/activity, simplified)
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Use activity feed as a proxy for recently touched files
				raw, err := c.GetWithHeaders("/ocs/v2.php/apps/activity/api/v2/activity/filter",
					map[string]string{"format": "json", "sort": "desc", "limit": "10"}, ocsHeaders)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					ctx.RecentFiles = json.RawMessage(`[]`)
					addErr(fmt.Sprintf("recent_files: %v", err))
					return
				}
				ctx.RecentFiles = raw
			}()

			// 4. Recent shares (last 5)
			wg.Add(1)
			go func() {
				defer wg.Done()
				raw, err := c.GetWithHeaders("/ocs/v2.php/apps/files_sharing/api/v1/shares",
					map[string]string{"format": "json"}, ocsHeaders)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					ctx.RecentShares = json.RawMessage(`[]`)
					addErr(fmt.Sprintf("recent_shares: %v", err))
					return
				}
				// Trim to last 5 items
				var sharesEnv struct {
					OCS struct {
						Data json.RawMessage `json:"data"`
					} `json:"ocs"`
				}
				if json.Unmarshal(raw, &sharesEnv) == nil {
					var items []json.RawMessage
					if json.Unmarshal(sharesEnv.OCS.Data, &items) == nil {
						if len(items) > 5 {
							items = items[:5]
						}
						trimmed, _ := json.Marshal(items)
						ctx.RecentShares = trimmed
						return
					}
				}
				ctx.RecentShares = raw
			}()

			// 5. Recent activity (last 10)
			wg.Add(1)
			go func() {
				defer wg.Done()
				raw, err := c.GetWithHeaders("/ocs/v2.php/apps/activity/api/v2/activity/filter",
					map[string]string{"format": "json", "sort": "desc", "limit": "10"}, ocsHeaders)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					ctx.RecentActivity = json.RawMessage(`[]`)
					addErr(fmt.Sprintf("recent_activity: %v", err))
					return
				}
				ctx.RecentActivity = raw
			}()

			// 6. Active Talk rooms
			wg.Add(1)
			go func() {
				defer wg.Done()
				raw, err := c.GetWithHeaders("/ocs/v2.php/apps/spreed/api/v4/room",
					map[string]string{"format": "json"}, ocsHeaders)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					ctx.ActiveTalkRooms = json.RawMessage(`[]`)
					addErr(fmt.Sprintf("talk_rooms: %v", err))
					return
				}
				ctx.ActiveTalkRooms = raw
			}()

			// 7. Unread notification count
			wg.Add(1)
			go func() {
				defer wg.Done()
				raw, err := c.GetWithHeaders("/ocs/v2.php/apps/admin_notifications/api/v1/notifications",
					map[string]string{"format": "json"}, ocsHeaders)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					// Notifications app may not be installed — not fatal
					addErr(fmt.Sprintf("notifications: %v", err))
					return
				}
				var notifEnv struct {
					OCS struct {
						Data []json.RawMessage `json:"data"`
					} `json:"ocs"`
				}
				if json.Unmarshal(raw, &notifEnv) == nil {
					ctx.UnreadNotifications = len(notifEnv.OCS.Data)
				}
			}()

			wg.Wait()

			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(ctx)
		},
	}

	cmd.Flags().StringVar(&user, "user", "", "User to scope to (default: authenticated user from config)")
	return cmd
}
