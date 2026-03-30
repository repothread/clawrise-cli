package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/pflag"

	accountstore "github.com/clawrise/clawrise-cli/internal/account"
	"github.com/clawrise/clawrise-cli/internal/authflow"
	"github.com/clawrise/clawrise-cli/internal/config"
	"github.com/clawrise/clawrise-cli/internal/output"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

func runAuthMethods(args []string, stdout io.Writer, manager *pluginruntime.Manager) error {
	if manager == nil {
		return fmt.Errorf("plugin manager is required")
	}

	flags := pflag.NewFlagSet("clawrise auth methods", pflag.ContinueOnError)
	flags.SetOutput(stdout)

	var platform string
	flags.StringVar(&platform, "platform", "", "list auth methods only for the selected platform")
	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) != 0 {
		return fmt.Errorf("usage: clawrise auth methods [--platform <name>]")
	}

	methods, err := manager.ListAuthMethods(context.Background(), strings.TrimSpace(platform))
	if err != nil {
		return err
	}
	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"methods": methods,
		},
	})
}

func runAuthPresets(args []string, stdout io.Writer, manager *pluginruntime.Manager) error {
	if manager == nil {
		return fmt.Errorf("plugin manager is required")
	}

	flags := pflag.NewFlagSet("clawrise auth presets", pflag.ContinueOnError)
	flags.SetOutput(stdout)

	var platform string
	flags.StringVar(&platform, "platform", "", "list account presets only for the selected platform")
	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) != 0 {
		return fmt.Errorf("usage: clawrise auth presets [--platform <name>]")
	}

	presets, err := manager.ListAuthPresets(context.Background(), strings.TrimSpace(platform))
	if err != nil {
		return err
	}
	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"presets": presets,
		},
	})
}

func runAuthInspectV2(args []string, cfg *config.Config, store *config.Store, stdout io.Writer, manager *pluginruntime.Manager) error {
	if manager == nil {
		return fmt.Errorf("plugin manager is required")
	}
	if len(args) > 1 {
		return fmt.Errorf("usage: clawrise auth inspect [account]")
	}

	accountName, account, ok, err := resolveAccountSelection(cfg, args)
	if err != nil {
		return writeCLIError(stdout, "ACCOUNT_REQUIRED", err.Error())
	}
	if !ok {
		return writeCLIError(stdout, "ACCOUNT_NOT_FOUND", "the selected account does not exist")
	}

	authAccount, err := buildPluginAuthAccount(cfg, store, accountName, account)
	if err != nil {
		return err
	}
	result, err := manager.InspectAuth(context.Background(), account.Platform, pluginruntime.AuthInspectParams{
		Account: authAccount,
	})
	if err != nil {
		return err
	}
	return output.WriteJSON(stdout, map[string]any{
		"ok": result.Ready,
		"data": map[string]any{
			"account": map[string]any{
				"name":        accountName,
				"platform":    account.Platform,
				"subject":     account.Subject,
				"auth_method": account.Auth.Method,
			},
			"inspection": result,
		},
	})
}

func runAuthCheckV2(args []string, cfg *config.Config, store *config.Store, stdout io.Writer, manager *pluginruntime.Manager) error {
	if err := runAuthInspectV2(args, cfg, store, stdout, manager); err != nil {
		return err
	}
	accountName, account, ok, err := resolveAccountSelection(cfg, args)
	if err != nil {
		return err
	}
	if !ok {
		return ExitError{Code: 1}
	}
	authAccount, err := buildPluginAuthAccount(cfg, store, accountName, account)
	if err != nil {
		return err
	}
	result, err := manager.InspectAuth(context.Background(), account.Platform, pluginruntime.AuthInspectParams{
		Account: authAccount,
	})
	if err != nil {
		return err
	}
	if !result.Ready {
		return ExitError{Code: 1}
	}
	return nil
}

func runAuthLogin(args []string, cfg *config.Config, store *config.Store, stdout io.Writer, manager *pluginruntime.Manager) error {
	if manager == nil {
		return fmt.Errorf("plugin manager is required")
	}

	flags := pflag.NewFlagSet("clawrise auth login", pflag.ContinueOnError)
	flags.SetOutput(stdout)

	var mode string
	var redirectURI string
	var callbackHost string
	var callbackPath string
	var openBrowser bool

	flags.StringVar(&mode, "mode", "", "set the auth mode, for example local_browser or manual_code")
	flags.StringVar(&redirectURI, "redirect-uri", "", "set the OAuth redirect_uri explicitly")
	flags.StringVar(&callbackHost, "callback-host", "", "set the loopback callback host")
	flags.StringVar(&callbackPath, "callback-path", "/callback", "set the loopback callback path")
	flags.BoolVar(&openBrowser, "open-browser", true, "launch the primary auth action automatically when possible")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) > 1 {
		return fmt.Errorf("usage: clawrise auth login [account] [--mode <name>] [--redirect-uri <uri>]")
	}

	accountName, account, ok, err := resolveAccountSelection(cfg, flags.Args())
	if err != nil {
		return writeCLIError(stdout, "ACCOUNT_REQUIRED", err.Error())
	}
	if !ok {
		return writeCLIError(stdout, "ACCOUNT_NOT_FOUND", "the selected account does not exist")
	}

	authAccount, err := buildPluginAuthAccount(cfg, store, accountName, account)
	if err != nil {
		return err
	}
	result, err := manager.BeginAuth(context.Background(), account.Platform, pluginruntime.AuthBeginParams{
		Account:      authAccount,
		Mode:         strings.TrimSpace(mode),
		RedirectURI:  strings.TrimSpace(redirectURI),
		CallbackHost: strings.TrimSpace(callbackHost),
		CallbackPath: strings.TrimSpace(callbackPath),
	})
	if err != nil {
		return err
	}

	flow := authFlowFromPluginResult(accountName, account, result.Flow)
	flowStore, err := openCLIAuthFlowStore(cfg, store)
	if err != nil {
		return err
	}
	if err := flowStore.Save(flow); err != nil {
		return err
	}

	launcherInfo := map[string]any{
		"attempted": openBrowser,
		"handled":   false,
	}
	if openBrowser {
		launchResult, launchErr := launchPrimaryAuthAction(context.Background(), manager, accountName, account, result.Flow, result.NextActions)
		launcherInfo["handled"] = launchResult.Handled
		if strings.TrimSpace(launchResult.Status) != "" {
			launcherInfo["status"] = launchResult.Status
		}
		if strings.TrimSpace(launchResult.LauncherID) != "" {
			launcherInfo["launcher_id"] = launchResult.LauncherID
		}
		if strings.TrimSpace(launchResult.Message) != "" {
			launcherInfo["message"] = launchResult.Message
		}
		if len(launchResult.Metadata) > 0 {
			launcherInfo["metadata"] = launchResult.Metadata
		}
		if launchErr != nil {
			launcherInfo["error"] = launchErr.Error()
		}
	}

	browserInfo := map[string]any{
		"attempted": openBrowser,
		"opened":    false,
	}
	if handled, _ := launcherInfo["handled"].(bool); handled {
		browserInfo["opened"] = true
	}
	if status, _ := launcherInfo["status"].(string); strings.TrimSpace(status) != "" {
		browserInfo["status"] = status
	}
	if message, _ := launcherInfo["message"].(string); strings.TrimSpace(message) != "" {
		browserInfo["message"] = message
	}
	if launcherID, _ := launcherInfo["launcher_id"].(string); strings.TrimSpace(launcherID) != "" {
		browserInfo["launcher_id"] = launcherID
	}
	if launchErr, _ := launcherInfo["error"].(string); strings.TrimSpace(launchErr) != "" {
		browserInfo["error"] = launchErr
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"account": map[string]any{
				"name":        accountName,
				"platform":    account.Platform,
				"subject":     account.Subject,
				"auth_method": account.Auth.Method,
			},
			"flow": map[string]any{
				"id":                result.Flow.ID,
				"method":            result.Flow.Method,
				"mode":              result.Flow.Mode,
				"state":             result.Flow.State,
				"redirect_uri":      result.Flow.RedirectURI,
				"authorization_url": result.Flow.AuthorizationURL,
				"device_code":       result.Flow.DeviceCode,
				"user_code":         result.Flow.UserCode,
				"verification_url":  result.Flow.VerificationURL,
				"interval_sec":      result.Flow.IntervalSec,
				"expires_at":        result.Flow.ExpiresAt,
			},
			"launcher":     launcherInfo,
			"browser":      browserInfo,
			"next_actions": result.NextActions,
		},
	})
}

func runAuthCompleteV2(args []string, cfg *config.Config, store *config.Store, stdout io.Writer, manager *pluginruntime.Manager) error {
	if manager == nil {
		return fmt.Errorf("plugin manager is required")
	}

	flags := pflag.NewFlagSet("clawrise auth complete", pflag.ContinueOnError)
	flags.SetOutput(stdout)

	var code string
	var callbackURL string
	flags.StringVar(&code, "code", "", "pass the authorization code directly")
	flags.StringVar(&callbackURL, "callback-url", "", "pass the full callback URL")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) != 1 {
		return fmt.Errorf("usage: clawrise auth complete <flow_id> [--callback-url <url> | --code <text>]")
	}

	flowStore, err := openCLIAuthFlowStore(cfg, store)
	if err != nil {
		return err
	}
	flow, err := flowStore.Load(strings.TrimSpace(flags.Args()[0]))
	if err != nil {
		return writeCLIError(stdout, "AUTH_FLOW_NOT_FOUND", "the selected auth flow does not exist")
	}

	cfg.Ensure()
	accountName := strings.TrimSpace(flow.ConnectionName)
	account, ok := cfg.Accounts[accountName]
	if !ok {
		return writeCLIError(stdout, "ACCOUNT_NOT_FOUND", "the flow account does not exist in current config")
	}

	authAccount, err := buildPluginAuthAccount(cfg, store, accountName, account)
	if err != nil {
		return err
	}
	result, err := manager.CompleteAuth(context.Background(), account.Platform, pluginruntime.AuthCompleteParams{
		Account:     authAccount,
		Flow:        pluginFlowFromStoredFlow(*flow),
		Code:        strings.TrimSpace(code),
		CallbackURL: strings.TrimSpace(callbackURL),
	})
	if err != nil {
		return err
	}
	if !result.Ready {
		return output.WriteJSON(stdout, map[string]any{
			"ok": false,
			"data": map[string]any{
				"account": accountName,
				"result":  result,
			},
		})
	}

	if err := persistAuthPatches(cfg, store, accountName, account, result.SessionPatch, result.SecretPatches); err != nil {
		return err
	}

	completedAt := time.Now().UTC()
	flow.State = "completed"
	flow.CompletedAt = &completedAt
	flow.Result = map[string]any{
		"account": accountName,
	}
	flow.ErrorCode = ""
	flow.ErrorMessage = ""
	if err := flowStore.Save(*flow); err != nil {
		return err
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"account": accountName,
			"result":  result,
		},
	})
}

func runAuthLogout(args []string, cfg *config.Config, store *config.Store, stdout io.Writer) error {
	if len(args) > 1 {
		return fmt.Errorf("usage: clawrise auth logout [account]")
	}

	accountName, account, ok, err := resolveAccountSelection(cfg, args)
	if err != nil {
		return writeCLIError(stdout, "ACCOUNT_REQUIRED", err.Error())
	}
	if !ok {
		return writeCLIError(stdout, "ACCOUNT_NOT_FOUND", "the selected account does not exist")
	}

	sessionStore, err := openCLISessionStore(cfg, store)
	if err != nil {
		return err
	}
	if err := sessionStore.Delete(accountName); err != nil {
		return err
	}

	secretStore, err := openCLISecretStore(cfg, store)
	if err != nil {
		return err
	}
	for _, field := range []string{"access_token", "refresh_token"} {
		if _, ok := account.Auth.SecretRefs[field]; ok {
			if err := secretStore.Delete(accountName, field); err != nil {
				return err
			}
		}
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"account":    accountName,
			"logged_out": true,
		},
	})
}

func resolveAccountSelection(cfg *config.Config, args []string) (string, config.Account, bool, error) {
	explicitName := ""
	if len(args) == 1 {
		explicitName = strings.TrimSpace(args[0])
	}
	selection, err := accountstore.ResolveSelection(cfg, explicitName)
	if err != nil {
		return "", config.Account{}, false, err
	}
	return selection.Name, selection.Account, selection.Found, nil
}

func buildPluginAuthAccount(cfg *config.Config, store *config.Store, accountName string, account config.Account) (pluginruntime.AuthAccount, error) {
	cfg.Ensure()
	secrets := map[string]string{}
	for field, ref := range account.Auth.SecretRefs {
		value, err := config.ResolveSecret(ref)
		if err != nil {
			continue
		}
		secrets[field] = value
	}

	sessionStore, err := openCLISessionStore(cfg, store)
	if err != nil {
		return pluginruntime.AuthAccount{}, err
	}
	var sessionPayload *pluginruntime.AuthSessionPayload
	if session, err := sessionStore.Load(accountName); err == nil {
		sessionPayload = pluginruntime.AuthSessionPayloadFromSession(session)
	}

	return pluginruntime.AuthAccount{
		Name:       accountName,
		Platform:   account.Platform,
		Subject:    account.Subject,
		AuthMethod: account.Auth.Method,
		Public:     cloneAnyMap(account.Auth.Public),
		Secrets:    secrets,
		Session:    sessionPayload,
	}, nil
}

func persistAuthPatches(cfg *config.Config, store *config.Store, accountName string, account config.Account, sessionPatch *pluginruntime.AuthSessionPayload, secretPatches map[string]string) error {
	if sessionPatch != nil {
		sessionStore, err := openCLISessionStore(cfg, store)
		if err != nil {
			return err
		}
		session := sessionPatch.ToSession()
		session.ProfileName = accountName
		session.Platform = account.Platform
		session.Subject = account.Subject
		session.GrantType = config.LegacyGrantTypeForMethod(account.Auth.Method)
		if err := sessionStore.Save(session); err != nil {
			return err
		}
	}

	if len(secretPatches) > 0 {
		secretStore, err := openCLISecretStore(cfg, store)
		if err != nil {
			return err
		}
		for field, value := range secretPatches {
			if strings.TrimSpace(value) == "" {
				continue
			}
			if err := secretStore.Set(accountName, field, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func authFlowFromPluginResult(accountName string, account config.Account, flow pluginruntime.AuthFlowPayload) authflow.Flow {
	result := authflow.Flow{
		ID:               flow.ID,
		ConnectionName:   accountName,
		Platform:         account.Platform,
		Method:           account.Auth.Method,
		Mode:             flow.Mode,
		State:            flow.State,
		RedirectURI:      flow.RedirectURI,
		AuthorizationURL: flow.AuthorizationURL,
		DeviceCode:       flow.DeviceCode,
		UserCode:         flow.UserCode,
		VerificationURL:  flow.VerificationURL,
		IntervalSec:      flow.IntervalSec,
		Metadata:         flow.Metadata,
	}
	if flow.ExpiresAt != "" {
		if expiresAt, err := time.Parse(time.RFC3339, flow.ExpiresAt); err == nil {
			result.ExpiresAt = expiresAt.UTC()
		}
	}
	return result
}

func pluginFlowFromStoredFlow(flow authflow.Flow) pluginruntime.AuthFlowPayload {
	payload := pluginruntime.AuthFlowPayload{
		ID:               flow.ID,
		Method:           flow.Method,
		Mode:             flow.Mode,
		State:            flow.State,
		RedirectURI:      flow.RedirectURI,
		AuthorizationURL: flow.AuthorizationURL,
		DeviceCode:       flow.DeviceCode,
		UserCode:         flow.UserCode,
		VerificationURL:  flow.VerificationURL,
		IntervalSec:      flow.IntervalSec,
		Metadata:         flow.Metadata,
	}
	if !flow.ExpiresAt.IsZero() {
		payload.ExpiresAt = flow.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return payload
}

func launchPrimaryAuthAction(ctx context.Context, manager *pluginruntime.Manager, accountName string, account config.Account, flow pluginruntime.AuthFlowPayload, actions []pluginruntime.AuthAction) (pluginruntime.AuthLaunchResult, error) {
	if manager == nil {
		return pluginruntime.AuthLaunchResult{
			Handled: false,
			Status:  "no_launcher_available",
			Message: "plugin manager is not available",
		}, nil
	}

	action, ok := selectLaunchableAuthAction(actions)
	if !ok {
		return pluginruntime.AuthLaunchResult{
			Handled: false,
			Status:  "no_launchable_action",
			Message: "the auth flow does not expose a launchable action",
		}, nil
	}

	return manager.LaunchAuth(ctx, pluginruntime.AuthLaunchParams{
		Context: pluginruntime.AuthLaunchContext{
			AccountName: accountName,
			Platform:    account.Platform,
			Subject:     account.Subject,
			AuthMethod:  account.Auth.Method,
		},
		Flow:   flow,
		Action: action,
	})
}

func selectLaunchableAuthAction(actions []pluginruntime.AuthAction) (pluginruntime.AuthAction, bool) {
	for _, preferredType := range []string{"device_code", "open_url"} {
		for _, action := range actions {
			if strings.TrimSpace(action.Type) == preferredType {
				return action, true
			}
		}
	}
	return pluginruntime.AuthAction{}, false
}
