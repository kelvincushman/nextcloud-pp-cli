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

func newRemotePhpMkdirCmd(flags *rootFlags) *cobra.Command {
	var parents bool

	cmd := &cobra.Command{
		Use:     "mkdir <user> <path>",
		Short:   "Create a directory (WebDAV MKCOL)",
		Example: "  nextcloud-pp-cli remote-php mkdir admin /NewFolder\n  nextcloud-pp-cli remote-php mkdir admin /a/b/c --parents",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
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
			dirPath := args[1]

			if parents {
				return davMkdirAll(c.HTTPClient, c.BaseURL, c.Config.AuthHeader(), c.DryRun, flags, user, dirPath)
			}
			return davMkcol(cmd, c.HTTPClient, c.BaseURL, c.Config.AuthHeader(), c.DryRun, flags, user, dirPath)
		},
	}

	cmd.Flags().BoolVar(&parents, "parents", false, "Create intermediate directories as needed")
	return cmd
}

func davMkcol(cmd *cobra.Command, httpClient *http.Client, baseURL, authHeader string, dryRun bool, flags *rootFlags, user, dirPath string) error {
	davPath := fmt.Sprintf("/remote.php/dav/files/%s%s", user, dirPath)
	targetURL := strings.TrimRight(baseURL, "/") + davPath

	if dryRun {
		fmt.Fprintf(os.Stderr, "MKCOL %s\n\n(dry run - no request sent)\n", targetURL)
		return nil
	}

	req, err := http.NewRequest("MKCOL", targetURL, nil)
	if err != nil {
		return fmt.Errorf("creating MKCOL request: %w", err)
	}
	req.Header.Set("OCS-APIRequest", "true")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("MKCOL %s: %w", davPath, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case 201:
		fmt.Fprintf(cmd.OutOrStdout(), "created: %s\n", dirPath)
		return nil
	case 405:
		if flags != nil && flags.idempotent {
			fmt.Fprintf(os.Stderr, "already exists (no-op): %s\n", dirPath)
			return nil
		}
		return fmt.Errorf("MKCOL %s: directory already exists (HTTP 405)", davPath)
	default:
		return fmt.Errorf("MKCOL %s returned HTTP %d: %s", davPath, resp.StatusCode, truncate(string(body), 512))
	}
}

func davMkdirAll(httpClient *http.Client, baseURL, authHeader string, dryRun bool, flags *rootFlags, user, dirPath string) error {
	// Build list of paths to create
	parts := strings.Split(strings.Trim(dirPath, "/"), "/")
	var paths []string
	var cur string
	for _, p := range parts {
		if p == "" {
			continue
		}
		cur += "/" + p
		paths = append(paths, cur)
	}

	for _, p := range paths {
		davPath := fmt.Sprintf("/remote.php/dav/files/%s%s", user, p)
		targetURL := strings.TrimRight(baseURL, "/") + davPath

		if dryRun {
			fmt.Fprintf(os.Stderr, "MKCOL %s\n", targetURL)
			continue
		}

		req, err := http.NewRequest("MKCOL", targetURL, nil)
		if err != nil {
			return fmt.Errorf("creating MKCOL request: %w", err)
		}
		req.Header.Set("OCS-APIRequest", "true")
		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("MKCOL %s: %w", davPath, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		switch resp.StatusCode {
		case 201:
			fmt.Fprintf(os.Stderr, "created: %s\n", p)
		case 405:
			// Already exists — continue
		default:
			return fmt.Errorf("MKCOL %s returned HTTP %d: %s", davPath, resp.StatusCode, truncate(string(body), 512))
		}
	}

	if dryRun {
		fmt.Fprintf(os.Stderr, "\n(dry run - no request sent)\n")
	} else {
		fmt.Fprintf(os.Stdout, "created: %s\n", dirPath)
	}
	return nil
}
