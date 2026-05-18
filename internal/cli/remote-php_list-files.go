// Copyright 2026 kelvincushman. Licensed under Apache-2.0.

package cli

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// davMultistatus is the top-level element of a WebDAV PROPFIND response.
type davMultistatus struct {
	XMLName   xml.Name      `xml:"multistatus"`
	Responses []davResponse `xml:"response"`
}

type davResponse struct {
	Href     string       `xml:"href"`
	Propstat []davPropstat `xml:"propstat"`
}

type davPropstat struct {
	Status string  `xml:"status"`
	Prop   davProp `xml:"prop"`
}

type davProp struct {
	DisplayName     string          `xml:"displayname"`
	ResourceType    davResourceType `xml:"resourcetype"`
	ContentLength   int64           `xml:"getcontentlength"`
	ContentType     string          `xml:"getcontenttype"`
	ETag            string          `xml:"getetag"`
	LastModified    string          `xml:"getlastmodified"`
	FileID          string          `xml:"fileid"`
	Size            int64           `xml:"size"`
}

type davResourceType struct {
	Collection *struct{} `xml:"collection"`
}

type fileEntry struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
	Path     string `json:"path"`
}

func formatBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func newRemotePhpListFilesCmd(flags *rootFlags) *cobra.Command {
	var recursive bool

	cmd := &cobra.Command{
		Use:   "list-files <user> [path]",
		Short: "List files in a directory (WebDAV PROPFIND)",
		Example: "  nextcloud-pp-cli remote-php list-files admin /\n" +
			"  nextcloud-pp-cli remote-php list-files admin /Documents --recursive",
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

			user := args[0]
			filePath := "/"
			if len(args) >= 2 {
				filePath = args[1]
			}
			filePath = strings.TrimRight(filePath, "/")
			if filePath == "" {
				filePath = "/"
			}

			davPath := fmt.Sprintf("/remote.php/dav/files/%s%s", user, filePath)
			targetURL := strings.TrimRight(c.BaseURL, "/") + davPath

			depth := "1"
			if recursive {
				depth = "infinity"
			}

			propfindBody := `<?xml version="1.0"?>
<d:propfind xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns" xmlns:nc="http://nextcloud.org/ns">
  <d:prop>
    <d:displayname/><d:resourcetype/><d:getcontentlength/><d:getcontenttype/>
    <d:getetag/><d:getlastmodified/><oc:fileid/><oc:size/>
  </d:prop>
</d:propfind>`

			req, err := http.NewRequest("PROPFIND", targetURL, strings.NewReader(propfindBody))
			if err != nil {
				return fmt.Errorf("creating PROPFIND request: %w", err)
			}
			req.Header.Set("Content-Type", "application/xml")
			req.Header.Set("Depth", depth)
			req.Header.Set("OCS-APIRequest", "true")
			if authHeader := c.Config.AuthHeader(); authHeader != "" {
				req.Header.Set("Authorization", authHeader)
			}

			if c.DryRun {
				fmt.Fprintf(os.Stderr, "PROPFIND %s\n  Depth: %s\n\n(dry run - no request sent)\n", targetURL, depth)
				return nil
			}

			resp, err := c.HTTPClient.Do(req)
			if err != nil {
				return fmt.Errorf("PROPFIND %s: %w", davPath, err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("reading PROPFIND response: %w", err)
			}

			if resp.StatusCode >= 400 {
				return fmt.Errorf("PROPFIND %s returned HTTP %d: %s", davPath, resp.StatusCode, truncate(string(body), 512))
			}

			var ms davMultistatus
			if err := xml.Unmarshal(body, &ms); err != nil {
				return fmt.Errorf("parsing PROPFIND XML: %w", err)
			}

			var entries []fileEntry
			for i, r := range ms.Responses {
				// Skip the first entry (the directory itself) when depth > 0
				if i == 0 && depth != "0" {
					continue
				}
				for _, ps := range r.Propstat {
					if !strings.Contains(ps.Status, "200") {
						continue
					}
					p := ps.Prop
					name := p.DisplayName
					if name == "" {
						// Fall back to extracting from href
						parts := strings.Split(strings.TrimRight(r.Href, "/"), "/")
						if len(parts) > 0 {
							name = parts[len(parts)-1]
						}
					}
					entryType := "file"
					if p.ResourceType.Collection != nil {
						entryType = "dir"
					}
					size := p.Size
					if size == 0 {
						size = p.ContentLength
					}
					entries = append(entries, fileEntry{
						Name:     name,
						Type:     entryType,
						Size:     size,
						Modified: p.LastModified,
						Path:     r.Href,
					})
					break
				}
			}

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(entries)
			}

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 2, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "NAME\tTYPE\tSIZE\tMODIFIED")
			for _, e := range entries {
				sizeStr := "-"
				if e.Type == "file" {
					sizeStr = formatBytes(e.Size)
				}
				mod := e.Modified
				if len(mod) > 25 {
					mod = mod[:25]
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", e.Name, e.Type, sizeStr, mod)
			}
			return tw.Flush()
		},
	}

	cmd.Flags().BoolVar(&recursive, "recursive", false, "Recurse into subdirectories (Depth: infinity)")
	return cmd
}
