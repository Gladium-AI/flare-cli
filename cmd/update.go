package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	accesspkg "github.com/paoloanzn/flare-cli/internal/access"
	"github.com/paoloanzn/flare-cli/internal/session"
	"github.com/paoloanzn/flare-cli/internal/ui"
)

var updateCmd = &cobra.Command{
	Use:   "update <session-id>",
	Short: "Update an active session's Access policy, TTL, or hostname",
	Args:  cobra.ExactArgs(1),
	RunE:  runUpdate,
}

func init() {
	f := updateCmd.Flags()
	f.String("ttl", "", "New session lifetime")
	f.String("session-duration", "", "New Access session duration")
	f.StringSlice("allow-email", nil, "Modify allowed emails (prefix with add: or remove:)")
	f.StringSlice("allow-domain", nil, "Modify allowed domains (prefix with add: or remove:)")
	f.String("health-path", "", "New health check path")

	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	sessionID := args[0]

	store, err := getStore()
	if err != nil {
		return err
	}

	sess, err := store.Resolve(ctx, sessionID)
	if err != nil {
		return err
	}

	if sess.State != session.StateActive {
		return fmt.Errorf("session %s is not active (state: %s)", sess.ShortID(), sess.State)
	}

	updated := false

	// Update TTL.
	if ttlStr, _ := cmd.Flags().GetString("ttl"); ttlStr != "" {
		ttl, err := time.ParseDuration(ttlStr)
		if err != nil {
			return fmt.Errorf("invalid --ttl: %w", err)
		}
		exp := time.Now().UTC().Add(ttl)
		sess.ExpiresAt = &exp
		updated = true
		ui.PrintSuccess("TTL updated: expires at %s", exp.Format(time.RFC3339))
	}

	// Update Access policy.
	policyUpdated := false

	if emails, _ := cmd.Flags().GetStringSlice("allow-email"); len(emails) > 0 {
		for _, e := range emails {
			if strings.HasPrefix(e, "add:") {
				email := strings.TrimPrefix(e, "add:")
				sess.AllowedEmails = appendUnique(sess.AllowedEmails, email)
			} else if strings.HasPrefix(e, "remove:") {
				email := strings.TrimPrefix(e, "remove:")
				sess.AllowedEmails = removeStr(sess.AllowedEmails, email)
			} else {
				sess.AllowedEmails = appendUnique(sess.AllowedEmails, e)
			}
		}
		policyUpdated = true
	}

	if domains, _ := cmd.Flags().GetStringSlice("allow-domain"); len(domains) > 0 {
		for _, d := range domains {
			if strings.HasPrefix(d, "add:") {
				domain := strings.TrimPrefix(d, "add:")
				sess.AllowedDomains = appendUnique(sess.AllowedDomains, domain)
			} else if strings.HasPrefix(d, "remove:") {
				domain := strings.TrimPrefix(d, "remove:")
				sess.AllowedDomains = removeStr(sess.AllowedDomains, domain)
			} else {
				sess.AllowedDomains = appendUnique(sess.AllowedDomains, d)
			}
		}
		policyUpdated = true
	}

	if sessionDur, _ := cmd.Flags().GetString("session-duration"); sessionDur != "" {
		sess.SessionDuration = sessionDur
		policyUpdated = true
	}

	if policyUpdated && sess.AccessAppID != "" && sess.AccessPolicyID != "" {
		// Get access manager from injected services or create real one.
		var accessMgr accesspkg.Manager
		if svc := getServices(); svc != nil && svc.AccessMgr != nil {
			accessMgr = svc.AccessMgr
		} else {
			svc, buildErr := buildProductionServices()
			if buildErr != nil {
				return buildErr
			}
			accessMgr = svc.AccessMgr
		}

		err = accessMgr.UpdatePolicy(ctx, sess.AccountID, sess.AccessAppID, sess.AccessPolicyID, accesspkg.Policy{
			AllowedEmails:   sess.AllowedEmails,
			AllowedDomains:  sess.AllowedDomains,
			SessionDuration: sess.SessionDuration,
		})
		if err != nil {
			return fmt.Errorf("updating Access policy: %w", err)
		}
		ui.PrintSuccess("Access policy updated")
		updated = true
	}

	if !updated {
		return fmt.Errorf("no changes specified")
	}

	sess.UpdatedAt = time.Now().UTC()
	if err := store.Save(ctx, sess); err != nil {
		return fmt.Errorf("saving session: %w", err)
	}

	return nil
}

func appendUnique(slice []string, val string) []string {
	for _, v := range slice {
		if v == val {
			return slice
		}
	}
	return append(slice, val)
}

func removeStr(slice []string, val string) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if v != val {
			result = append(result, v)
		}
	}
	return result
}
