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

func newRemotePhpCopyCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "copy <user> <src> <dest>",
		Short:   "Copy a file/directory (WebDAV COPY)",
		Example: "  nextcloud-pp-cli remote-php copy admin /original.txt /backup.txt",
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

			if c.DryRun {
				fmt.Fprintf(os.Stderr, "COPY %s\n  Destination: %s\n\n(dry run - no request sent)\n", srcURL, destURL)
				return nil
			}

			req, err := http.NewRequest("COPY", srcURL, nil)
			if err != nil {
				return fmt.Errorf("creating COPY request: %w", err)
			}
			req.Header.Set("Destination", destURL)
			req.Header.Set("OCS-APIRequest", "true")
			if authHeader := c.Config.AuthHeader(); authHeader != "" {
				req.Header.Set("Authorization", authHeader)
			}

			resp, err := c.HTTPClient.Do(req)
			if err != nil {
				return fmt.Errorf("COPY %s: %w", srcDAVPath, err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode == 201 || resp.StatusCode == 204 {
				fmt.Fprintf(cmd.OutOrStdout(), "copied: %s -> %s\n", src, dest)
				return nil
			}
			return fmt.Errorf("COPY %s returned HTTP %d: %s", srcDAVPath, resp.StatusCode, truncate(string(body), 512))
		},
	}

	return cmd
}
