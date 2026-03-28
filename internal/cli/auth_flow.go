package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/pflag"

	feishuadapter "github.com/clawrise/clawrise-cli/internal/adapter/feishu"
	notionadapter "github.com/clawrise/clawrise-cli/internal/adapter/notion"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"github.com/clawrise/clawrise-cli/internal/authflow"
	"github.com/clawrise/clawrise-cli/internal/config"
	"github.com/clawrise/clawrise-cli/internal/output"
	"github.com/clawrise/clawrise-cli/internal/secretstore"
)

var (
	newFeishuAuthFlowClient = func(sessionStore authcache.Store) (*feishuadapter.Client, error) {
		return feishuadapter.NewClient(feishuadapter.Options{
			SessionStore: sessionStore,
		})
	}
	newNotionAuthFlowClient = func(sessionStore authcache.Store) (*notionadapter.Client, error) {
		return notionadapter.NewClient(notionadapter.Options{
			SessionStore: sessionStore,
		})
	}
)

func runAuthFlow(args []string, cfg *config.Config, store *config.Store, stdout io.Writer) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		printAuthFlowHelp(stdout)
		return nil
	}

	switch args[0] {
	case "begin":
		return runAuthFlowBegin(args[1:], cfg, store, stdout)
	case "status":
		return runAuthFlowStatus(args[1:], cfg, store, stdout)
	case "continue":
		return runAuthFlowContinue(args[1:], cfg, store, stdout)
	default:
		return fmt.Errorf("unknown auth flow command: %s", args[0])
	}
}

func runAuthFlowBegin(args []string, cfg *config.Config, store *config.Store, stdout io.Writer) error {
	flags := pflag.NewFlagSet("clawrise auth begin", pflag.ContinueOnError)
	flags.SetOutput(stdout)

	var mode string
	var redirectURI string
	var callbackHost string
	var callbackPath string

	flags.StringVar(&mode, "mode", "", "授权交互模式：local_browser/manual_url/manual_code")
	flags.StringVar(&redirectURI, "redirect-uri", "", "显式指定 OAuth redirect_uri")
	flags.StringVar(&callbackHost, "callback-host", "127.0.0.1", "loopback 模式使用的本地回调主机")
	flags.StringVar(&callbackPath, "callback-path", "/callback", "loopback 模式使用的本地回调路径")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) > 1 {
		return fmt.Errorf("usage: clawrise auth begin [connection] [--mode <name>] [--redirect-uri <uri>]")
	}

	connectionName, connection, ok, err := resolveAuthConnection(cfg, flags.Args())
	if err != nil {
		return writeCLIError(stdout, "CONNECTION_REQUIRED", err.Error())
	}
	if !ok {
		return writeCLIError(stdout, "CONNECTION_NOT_FOUND", "the selected connection does not exist")
	}

	methodSpec, ok := authflow.LookupMethodSpec(connection.Method)
	if !ok {
		return writeCLIError(stdout, "AUTH_METHOD_NOT_SUPPORTED", fmt.Sprintf("auth method %s is not supported", connection.Method))
	}
	if !methodSpec.Interactive || !methodSpec.SupportsCodeFlow {
		return writeCLIError(stdout, "AUTH_METHOD_NOT_INTERACTIVE", fmt.Sprintf("auth method %s does not support interactive authorization flows", connection.Method))
	}

	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = methodSpec.DefaultMode
	}
	if !stringSliceContains(methodSpec.Modes, mode) {
		return writeCLIError(stdout, "AUTH_MODE_NOT_SUPPORTED", fmt.Sprintf("mode %s is not supported for auth method %s", mode, connection.Method))
	}

	flow, appErr := buildAuthorizationFlow(connectionName, connection, mode, redirectURI, callbackHost, callbackPath)
	if appErr != nil {
		return writeCLIError(stdout, appErr.Code, appErr.Message)
	}

	flowStore := authflow.NewFileStore(store.Path())
	if err := flowStore.Save(flow); err != nil {
		return err
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"flow":    buildAuthFlowView(flow),
			"actions": authflow.BuildActions(flow),
			"method":  methodSpec,
		},
	})
}

func runAuthFlowStatus(args []string, cfg *config.Config, store *config.Store, stdout io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: clawrise auth status <flow_id>")
	}

	flowStore := authflow.NewFileStore(store.Path())
	flow, err := flowStore.Load(strings.TrimSpace(args[0]))
	if err != nil {
		return writeCLIError(stdout, "AUTH_FLOW_NOT_FOUND", "the selected auth flow does not exist")
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"flow":    buildAuthFlowView(*flow),
			"actions": authflow.BuildActions(*flow),
		},
	})
}

func runAuthFlowContinue(args []string, cfg *config.Config, store *config.Store, stdout io.Writer) error {
	flags := pflag.NewFlagSet("clawrise auth continue", pflag.ContinueOnError)
	flags.SetOutput(stdout)

	var code string
	var callbackURL string

	flags.StringVar(&code, "code", "", "直接传入 OAuth 授权码")
	flags.StringVar(&callbackURL, "callback-url", "", "传入浏览器最终跳转的完整回调 URL")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) != 1 {
		return fmt.Errorf("usage: clawrise auth continue <flow_id> [--callback-url <url> | --code <text>]")
	}

	flowStore := authflow.NewFileStore(store.Path())
	flow, err := flowStore.Load(strings.TrimSpace(flags.Args()[0]))
	if err != nil {
		return writeCLIError(stdout, "AUTH_FLOW_NOT_FOUND", "the selected auth flow does not exist")
	}
	if flow.State == "completed" {
		return output.WriteJSON(stdout, map[string]any{
			"ok": true,
			"data": map[string]any{
				"flow": buildAuthFlowView(*flow),
			},
		})
	}
	if time.Now().UTC().After(flow.ExpiresAt) {
		flow.State = "failed"
		flow.ErrorCode = "AUTH_FLOW_EXPIRED"
		flow.ErrorMessage = "authorization flow has expired"
		_ = flowStore.Save(*flow)
		return writeCLIError(stdout, flow.ErrorCode, flow.ErrorMessage)
	}

	authorizationCode, appErr := extractAuthorizationCode(*flow, code, callbackURL)
	if appErr != nil {
		return writeCLIError(stdout, appErr.Code, appErr.Message)
	}

	connection, ok := cfg.Connections[flow.ConnectionName]
	if !ok {
		return writeCLIError(stdout, "CONNECTION_NOT_FOUND", "the flow connection does not exist in current config")
	}

	sessionStore := authcache.NewFileStore(store.Path())
	secretStore, err := openCLISecretStore(cfg, store)
	if err != nil {
		return err
	}

	session, appErr := continueAuthorizationFlow(context.Background(), sessionStore, secretStore, flow.ConnectionName, connection, *flow, authorizationCode)
	if appErr != nil {
		flow.State = "failed"
		flow.ErrorCode = appErr.Code
		flow.ErrorMessage = appErr.Message
		_ = flowStore.Save(*flow)
		return writeCLIError(stdout, appErr.Code, appErr.Message)
	}

	completedAt := time.Now().UTC()
	flow.State = "completed"
	flow.CompletedAt = &completedAt
	flow.Result = map[string]any{
		"session": buildSessionView(sessionStore.Path(flow.ConnectionName), session, connection),
	}
	flow.ErrorCode = ""
	flow.ErrorMessage = ""
	if err := flowStore.Save(*flow); err != nil {
		return err
	}

	return output.WriteJSON(stdout, map[string]any{
		"ok": true,
		"data": map[string]any{
			"flow": buildAuthFlowView(*flow),
		},
	})
}

func resolveAuthConnection(cfg *config.Config, args []string) (string, config.Profile, bool, error) {
	name := ""
	if len(args) == 1 {
		name = strings.TrimSpace(args[0])
	} else if platform := strings.TrimSpace(cfg.Defaults.Platform); platform != "" {
		name = strings.TrimSpace(cfg.Defaults.Connections[platform])
		if name == "" {
			name = strings.TrimSpace(cfg.Defaults.Profile)
		}
	}
	if name == "" {
		return "", config.Profile{}, false, fmt.Errorf("no connection was provided and no default connection is configured")
	}

	_, connection, ok := lookupConnection(cfg, name)
	if !ok {
		return name, config.Profile{}, false, nil
	}
	return name, connection, true, nil
}

func buildAuthorizationFlow(connectionName string, connection config.Connection, mode string, redirectURI string, callbackHost string, callbackPath string) (authflow.Flow, *apperr.AppError) {
	redirectURI = strings.TrimSpace(redirectURI)
	callbackHost = strings.TrimSpace(callbackHost)
	callbackPath = strings.TrimSpace(callbackPath)
	if callbackPath == "" {
		callbackPath = "/callback"
	}

	flow := authflow.Flow{
		ID:             "flow_" + randomToken(8),
		ConnectionName: connectionName,
		Platform:       connection.Platform,
		Method:         connection.Method,
		Mode:           mode,
		State:          "awaiting_user_action",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		ExpiresAt:      time.Now().UTC().Add(authflow.DefaultFlowTTL),
		Metadata:       map[string]string{},
	}

	if mode == "local_browser" {
		if callbackHost == "" {
			callbackHost = "127.0.0.1"
		}
		port, err := reserveLoopbackPort(callbackHost)
		if err != nil {
			return authflow.Flow{}, apperr.New("AUTH_CALLBACK_PREPARE_FAILED", err.Error())
		}
		flow.CallbackHost = callbackHost
		flow.CallbackPort = port
		flow.CallbackPath = callbackPath
		if redirectURI == "" {
			redirectURI = fmt.Sprintf("http://%s:%d%s", callbackHost, port, callbackPath)
		}
	} else if redirectURI == "" {
		return authflow.Flow{}, apperr.New("REDIRECT_URI_REQUIRED", "manual_url and manual_code modes require an explicit --redirect-uri")
	}

	flow.RedirectURI = redirectURI
	stateToken := randomToken(16)
	flow.Metadata["oauth_state"] = stateToken

	authorizationURL, appErr := buildAuthorizationURL(connection, redirectURI, stateToken)
	if appErr != nil {
		return authflow.Flow{}, appErr
	}
	flow.AuthorizationURL = authorizationURL
	return flow, nil
}

func buildAuthorizationURL(connection config.Connection, redirectURI string, stateToken string) (string, *apperr.AppError) {
	switch connection.Method {
	case "feishu.oauth_user":
		clientID := strings.TrimSpace(connection.Params.ClientID)
		if clientID == "" {
			return "", apperr.New("INVALID_AUTH_CONFIG", "client_id is required for feishu.oauth_user")
		}
		endpoint, _ := url.Parse("https://accounts.feishu.cn/open-apis/authen/v1/authorize")
		query := endpoint.Query()
		query.Set("client_id", clientID)
		query.Set("response_type", "code")
		query.Set("redirect_uri", strings.TrimSpace(redirectURI))
		query.Set("state", stateToken)
		query.Set("prompt", "consent")
		if len(connection.Params.Scopes) > 0 {
			query.Set("scope", strings.Join(connection.Params.Scopes, " "))
		}
		endpoint.RawQuery = query.Encode()
		return endpoint.String(), nil
	case "notion.oauth_public":
		clientID := strings.TrimSpace(connection.Params.ClientID)
		if clientID == "" {
			return "", apperr.New("INVALID_AUTH_CONFIG", "client_id is required for notion.oauth_public")
		}
		endpoint, _ := url.Parse("https://api.notion.com/v1/oauth/authorize")
		query := endpoint.Query()
		query.Set("client_id", clientID)
		query.Set("response_type", "code")
		query.Set("owner", "user")
		query.Set("redirect_uri", strings.TrimSpace(redirectURI))
		query.Set("state", stateToken)
		endpoint.RawQuery = query.Encode()
		return endpoint.String(), nil
	default:
		return "", apperr.New("AUTH_METHOD_NOT_SUPPORTED", fmt.Sprintf("auth method %s is not supported", connection.Method))
	}
}

func extractAuthorizationCode(flow authflow.Flow, code string, callbackURL string) (string, *apperr.AppError) {
	code = strings.TrimSpace(code)
	callbackURL = strings.TrimSpace(callbackURL)
	if callbackURL != "" {
		parsed, err := url.Parse(callbackURL)
		if err != nil {
			return "", apperr.New("INVALID_INPUT", fmt.Sprintf("failed to parse callback_url: %v", err))
		}
		query := parsed.Query()
		if returnedError := strings.TrimSpace(query.Get("error")); returnedError != "" {
			return "", apperr.New("AUTH_CALLBACK_FAILED", returnedError)
		}
		stateToken := strings.TrimSpace(query.Get("state"))
		expectedState := strings.TrimSpace(flow.Metadata["oauth_state"])
		if expectedState != "" && stateToken != expectedState {
			return "", apperr.New("AUTH_STATE_MISMATCH", "callback state does not match the current auth flow")
		}
		code = strings.TrimSpace(query.Get("code"))
	}

	if code == "" {
		return "", apperr.New("AUTH_CODE_REQUIRED", "authorization code is required; pass --callback-url or --code")
	}
	return code, nil
}

func continueAuthorizationFlow(ctx context.Context, sessionStore authcache.Store, secretStore secretstore.Store, connectionName string, connection config.Connection, flow authflow.Flow, code string) (*authcache.Session, *apperr.AppError) {
	switch connection.Method {
	case "feishu.oauth_user":
		client, err := newFeishuAuthFlowClient(sessionStore)
		if err != nil {
			return nil, apperr.New("AUTH_CLIENT_INIT_FAILED", err.Error())
		}
		session, appErr := client.ExchangeAuthorizationCode(ctx, connectionName, connection, code, flow.RedirectURI)
		if appErr != nil {
			return nil, appErr
		}
		if strings.TrimSpace(session.RefreshToken) != "" {
			if err := secretStore.Set(connectionName, "refresh_token", strings.TrimSpace(session.RefreshToken)); err != nil {
				return nil, apperr.New("SECRET_STORE_WRITE_FAILED", err.Error())
			}
		}
		return session, nil
	case "notion.oauth_public":
		client, err := newNotionAuthFlowClient(sessionStore)
		if err != nil {
			return nil, apperr.New("AUTH_CLIENT_INIT_FAILED", err.Error())
		}
		session, appErr := client.ExchangeAuthorizationCode(ctx, connectionName, connection, code, flow.RedirectURI)
		if appErr != nil {
			return nil, appErr
		}
		if strings.TrimSpace(session.RefreshToken) != "" {
			if err := secretStore.Set(connectionName, "refresh_token", strings.TrimSpace(session.RefreshToken)); err != nil {
				return nil, apperr.New("SECRET_STORE_WRITE_FAILED", err.Error())
			}
		}
		return session, nil
	default:
		return nil, apperr.New("AUTH_METHOD_NOT_SUPPORTED", fmt.Sprintf("auth method %s does not support continue", connection.Method))
	}
}

func buildAuthFlowView(flow authflow.Flow) map[string]any {
	view := map[string]any{
		"id":              flow.ID,
		"connection_name": flow.ConnectionName,
		"platform":        flow.Platform,
		"method":          flow.Method,
		"mode":            flow.Mode,
		"state":           flow.State,
		"redirect_uri":    flow.RedirectURI,
		"created_at":      flow.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":      flow.UpdatedAt.UTC().Format(time.RFC3339),
		"expires_at":      flow.ExpiresAt.UTC().Format(time.RFC3339),
	}
	if flow.AuthorizationURL != "" {
		view["authorization_url"] = flow.AuthorizationURL
	}
	if flow.CallbackHost != "" {
		view["callback"] = map[string]any{
			"host": flow.CallbackHost,
			"port": flow.CallbackPort,
			"path": flow.CallbackPath,
		}
	}
	if flow.CompletedAt != nil {
		view["completed_at"] = flow.CompletedAt.UTC().Format(time.RFC3339)
	}
	if flow.Result != nil {
		view["result"] = flow.Result
	}
	if flow.ErrorCode != "" {
		view["error"] = map[string]any{
			"code":    flow.ErrorCode,
			"message": flow.ErrorMessage,
		}
	}
	return view
}

func openCLISecretStore(cfg *config.Config, store *config.Store) (secretstore.Store, error) {
	backend := strings.TrimSpace(cfg.Auth.SecretStore.Backend)
	if backend == "" {
		backend = "auto"
	}
	return secretstore.Open(secretstore.Options{
		ConfigPath:      store.Path(),
		Backend:         backend,
		FallbackBackend: cfg.Auth.SecretStore.FallbackBackend,
	})
}

func reserveLoopbackPort(host string) (int, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected callback listener address type")
	}
	return tcpAddr.Port, nil
}

func randomToken(byteLength int) string {
	if byteLength <= 0 {
		byteLength = 16
	}
	data := make([]byte, byteLength)
	if _, err := rand.Read(data); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(data)
}

func printAuthFlowHelp(stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, "Usage: clawrise auth [begin|status|continue]")
	_, _ = fmt.Fprintln(stdout, "       clawrise auth begin [connection] [--mode <name>] [--redirect-uri <uri>]")
	_, _ = fmt.Fprintln(stdout, "       clawrise auth status <flow_id>")
	_, _ = fmt.Fprintln(stdout, "       clawrise auth continue <flow_id> [--callback-url <url> | --code <text>]")
}
