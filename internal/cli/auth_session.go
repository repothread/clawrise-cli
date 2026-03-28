package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	feishuadapter "github.com/clawrise/clawrise-cli/internal/adapter/feishu"
	notionadapter "github.com/clawrise/clawrise-cli/internal/adapter/notion"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"github.com/clawrise/clawrise-cli/internal/config"
	"github.com/clawrise/clawrise-cli/internal/output"
)

var (
	newFeishuAuthSessionClient = func(sessionStore authcache.Store) (*feishuadapter.Client, error) {
		return feishuadapter.NewClient(feishuadapter.Options{
			SessionStore: sessionStore,
		})
	}
	newNotionAuthSessionClient = func(sessionStore authcache.Store) (*notionadapter.Client, error) {
		return notionadapter.NewClient(notionadapter.Options{
			SessionStore: sessionStore,
		})
	}
)

func runAuthSession(args []string, cfg *config.Config, store *config.Store, stdout io.Writer) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printAuthSessionHelp(stdout)
		return nil
	}

	sessionStore := authcache.NewFileStore(store.Path())
	switch args[0] {
	case "inspect":
		if len(args) > 2 {
			return fmt.Errorf("usage: clawrise auth session inspect [profile]")
		}
		name, profile, ok, err := resolveAuthSessionProfile(cfg, args[1:])
		if err != nil {
			return writeCLIError(stdout, "PROFILE_REQUIRED", err.Error())
		}
		if !ok {
			return writeCLIError(stdout, "PROFILE_NOT_FOUND", "the selected profile does not exist")
		}

		sessionView, err := inspectAuthSession(sessionStore, name, profile)
		if err != nil {
			return writeCLIError(stdout, "SESSION_LOAD_FAILED", err.Error())
		}
		return output.WriteJSON(stdout, map[string]any{
			"ok": true,
			"data": map[string]any{
				"profile": map[string]any{
					"name":       name,
					"platform":   profile.Platform,
					"subject":    profile.Subject,
					"grant_type": profile.Grant.Type,
				},
				"session": sessionView,
			},
		})
	case "clear":
		if len(args) > 2 {
			return fmt.Errorf("usage: clawrise auth session clear [profile]")
		}
		name, profile, ok, err := resolveAuthSessionProfile(cfg, args[1:])
		if err != nil {
			return writeCLIError(stdout, "PROFILE_REQUIRED", err.Error())
		}
		if !ok {
			return writeCLIError(stdout, "PROFILE_NOT_FOUND", "the selected profile does not exist")
		}

		path := sessionStore.Path(name)
		_, statErr := os.Stat(path)
		existed := statErr == nil
		if statErr != nil && !os.IsNotExist(statErr) {
			return writeCLIError(stdout, "SESSION_STAT_FAILED", statErr.Error())
		}
		if err := sessionStore.Delete(name); err != nil {
			return writeCLIError(stdout, "SESSION_CLEAR_FAILED", err.Error())
		}

		return output.WriteJSON(stdout, map[string]any{
			"ok": true,
			"data": map[string]any{
				"profile": map[string]any{
					"name":       name,
					"platform":   profile.Platform,
					"subject":    profile.Subject,
					"grant_type": profile.Grant.Type,
				},
				"session": map[string]any{
					"path":    path,
					"deleted": existed,
					"exists":  false,
				},
			},
		})
	case "refresh":
		if len(args) > 2 {
			return fmt.Errorf("usage: clawrise auth session refresh [profile]")
		}
		name, profile, ok, err := resolveAuthSessionProfile(cfg, args[1:])
		if err != nil {
			return writeCLIError(stdout, "PROFILE_REQUIRED", err.Error())
		}
		if !ok {
			return writeCLIError(stdout, "PROFILE_NOT_FOUND", "the selected profile does not exist")
		}

		session, appErr := refreshAuthSession(context.Background(), sessionStore, name, profile)
		if appErr != nil {
			return writeCLIError(stdout, appErr.Code, appErr.Message)
		}

		return output.WriteJSON(stdout, map[string]any{
			"ok": true,
			"data": map[string]any{
				"profile": map[string]any{
					"name":       name,
					"platform":   profile.Platform,
					"subject":    profile.Subject,
					"grant_type": profile.Grant.Type,
				},
				"session": buildSessionView(sessionStore.Path(name), session, profile),
			},
		})
	default:
		return fmt.Errorf("unknown auth session command: %s", args[0])
	}
}

func resolveAuthSessionProfile(cfg *config.Config, args []string) (string, config.Profile, bool, error) {
	name := strings.TrimSpace(cfg.Defaults.Profile)
	if len(args) == 1 {
		name = strings.TrimSpace(args[0])
	}
	if name == "" {
		return "", config.Profile{}, false, fmt.Errorf("no profile was provided and no default profile is configured")
	}

	_, profile, ok := lookupProfile(cfg, name)
	if !ok {
		return name, config.Profile{}, false, nil
	}
	return name, profile, true, nil
}

func inspectAuthSession(sessionStore authcache.Store, profileName string, profile config.Profile) (map[string]any, error) {
	path := sessionStore.Path(profileName)
	session, err := sessionStore.Load(profileName)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{
				"path":    path,
				"exists":  false,
				"usable":  false,
				"matches": false,
			}, nil
		}
		return nil, err
	}
	return buildSessionView(path, session, profile), nil
}

func refreshAuthSession(ctx context.Context, sessionStore authcache.Store, profileName string, profile config.Profile) (*authcache.Session, *apperr.AppError) {
	switch profile.Platform {
	case "feishu":
		client, err := newFeishuAuthSessionClient(sessionStore)
		if err != nil {
			return nil, apperr.New("AUTH_CLIENT_INIT_FAILED", err.Error())
		}
		return client.RefreshSession(ctx, profileName, profile)
	case "notion":
		client, err := newNotionAuthSessionClient(sessionStore)
		if err != nil {
			return nil, apperr.New("AUTH_CLIENT_INIT_FAILED", err.Error())
		}
		return client.RefreshSession(ctx, profileName, profile)
	default:
		return nil, apperr.New("UNSUPPORTED_PLATFORM", fmt.Sprintf("session refresh is not supported for platform %s", profile.Platform))
	}
}

func buildSessionView(path string, session *authcache.Session, profile config.Profile) map[string]any {
	if session == nil {
		return map[string]any{
			"path":    path,
			"exists":  false,
			"usable":  false,
			"matches": false,
		}
	}

	now := time.Now().UTC()
	matches := strings.TrimSpace(session.Platform) == strings.TrimSpace(profile.Platform) &&
		strings.TrimSpace(session.Subject) == strings.TrimSpace(profile.Subject) &&
		strings.TrimSpace(session.GrantType) == strings.TrimSpace(profile.Grant.Type)

	view := map[string]any{
		"path":              path,
		"exists":            true,
		"matches":           matches,
		"version":           session.Version,
		"platform":          session.Platform,
		"subject":           session.Subject,
		"grant_type":        session.GrantType,
		"token_type":        session.TokenType,
		"has_access_token":  session.HasAccessToken(),
		"has_refresh_token": session.CanRefresh(),
		"usable":            session.UsableAt(now, authcache.DefaultRefreshSkew),
		"needs_refresh":     session.NeedsRefreshAt(now, authcache.DefaultRefreshSkew),
		"access_token":      redactSessionSecret(session.AccessToken),
		"refresh_token":     redactSessionSecret(session.RefreshToken),
	}

	if session.ExpiresAt != nil {
		view["expires_at"] = session.ExpiresAt.UTC().Format(time.RFC3339)
	}
	if session.CreatedAt != nil {
		view["created_at"] = session.CreatedAt.UTC().Format(time.RFC3339)
	}
	if session.UpdatedAt != nil {
		view["updated_at"] = session.UpdatedAt.UTC().Format(time.RFC3339)
	}
	if len(session.Metadata) > 0 {
		view["metadata"] = session.Metadata
	}
	return view
}

func redactSessionSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "***"
	}
	return value[:2] + "***" + value[len(value)-2:]
}

func printAuthSessionHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise auth session [inspect|clear|refresh] [profile]")
}
