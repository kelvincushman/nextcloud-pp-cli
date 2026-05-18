// Copyright 2026 kelvincushman. Licensed under Apache-2.0.

package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newRemotePhpMoveCmd(flags *rootFlags) *cobra.Command {
	var overwrite bool

	cmd := &cobra.Command{
		Use:     "move <user> <src> <dest>",
		Short:   "Move or rename a file/directory (WebDAV MOVE)",
		Example: "  nextcloud-pp-cli remote-php move admin /old.txt /new.txt\n  nextcloud-pp-cli remote-php move admin /old.txt /new.txt --overwrite",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			user := args[0]
			src := args[1]
			dest := args[2]

			srcDAVPath := fmt.Sprintf("/remote.php/dav/files/%s%s", user, src)
			destDAVPath := fmt.Sprintf("/remote.php/dav/files/%s%s", user, dest)
			srcURL := strings.TrimRight(c.BaseURL, "/") + srcDAVPath
			destURL := strings.TrimRight(c.BaseURL, "/") + destDAVPath

			overwriteHeader := "F"
			if overwrite {
				overwriteHeader = "T"
			}

			if c.DryRun {
				fmt.Fprintf(os.Stderr, "MOVE %s\n  Destination: %s\n  Overwrite: %s\n\n(dry run - no request sent)\n", srcURL, destURL, overwriteHeader)
				return nil
			}

			req, err := http.NewRequest("MOVE", srcURL, nil)
			if err != nil {
				return fmt.Errorf("creating MOVE request: %w", err)
			}
			req.Header.Set("Destination", destURL)
			req.Header.Set("Overwrite", overwriteHeader)
			req.Header.Set("OCS-APIRequest", "true")
			if authHeader := c.Config.AuthHeader(); authHeader != "" {
				req.Header.Set("Authorization", authHeader)
			}

			resp, err := c.HTTPClient.Do(req)
			if err != nil {
				return fmt.Errorf("MOVE %s: %w", srcDAVPath, err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode == 201 || resp.StatusCode == 204 {
				fmt.Fprintf(cmd.OutOrStdout(), "moved: %s -> %s\n", src, dest)
				return nil
			}
			return fmt.Errorf("MOVE %s returned HTTP %d: %s", srcDAVPath, resp.StatusCode, truncate(string(body), 512))
		},
	}

	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Allow overwriting the destination if it exists")
	return cmd
}
