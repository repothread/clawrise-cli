package plugin

import (
	"context"
	"os"
	"strings"

	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"github.com/clawrise/clawrise-cli/internal/authflow"
)

func init() {
	authcache.RegisterExternalStoreResolver(resolvePluginSessionStore)
	authflow.RegisterExternalStoreResolver(resolvePluginAuthFlowStore)
}

func resolvePluginSessionStore(options authcache.StoreOptions) (authcache.Store, bool, error) {
	manifest, found, err := FindStorageBackendManifest(StorageBackendLookup{
		Target:  "session_store",
		Backend: strings.TrimSpace(options.Backend),
		Plugin:  strings.TrimSpace(options.Plugin),
	})
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	return &pluginSessionStore{
		client: NewProcessSessionStore(manifest),
	}, true, nil
}

func resolvePluginAuthFlowStore(options authflow.StoreOptions) (authflow.Store, bool, error) {
	manifest, found, err := FindStorageBackendManifest(StorageBackendLookup{
		Target:  "authflow_store",
		Backend: strings.TrimSpace(options.Backend),
		Plugin:  strings.TrimSpace(options.Plugin),
	})
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	return &pluginAuthFlowStore{
		client: NewProcessAuthFlowStore(manifest),
	}, true, nil
}

type pluginSessionStore struct {
	client *ProcessSessionStore
}

func (s *pluginSessionStore) Load(accountName string) (*authcache.Session, error) {
	result, err := s.client.Load(context.Background(), SessionStoreLoadParams{
		AccountName: accountName,
	})
	if err != nil {
		return nil, err
	}
	if !result.Found || result.Session == nil {
		return nil, os.ErrNotExist
	}
	return result.Session, nil
}

func (s *pluginSessionStore) Save(session authcache.Session) error {
	return s.client.Save(context.Background(), SessionStoreSaveParams{
		Session: session,
	})
}

func (s *pluginSessionStore) Delete(accountName string) error {
	return s.client.Delete(context.Background(), SessionStoreDeleteParams{
		AccountName: accountName,
	})
}

func (s *pluginSessionStore) Path(accountName string) string {
	backend := ""
	if s != nil && s.client != nil {
		backend = s.client.Backend()
	}
	return "plugin://" + backend + "/sessions/" + strings.TrimSpace(accountName)
}

type pluginAuthFlowStore struct {
	client *ProcessAuthFlowStore
}

func (s *pluginAuthFlowStore) Load(flowID string) (*authflow.Flow, error) {
	result, err := s.client.Load(context.Background(), AuthFlowStoreLoadParams{
		FlowID: flowID,
	})
	if err != nil {
		return nil, err
	}
	if !result.Found || result.Flow == nil {
		return nil, os.ErrNotExist
	}
	return result.Flow, nil
}

func (s *pluginAuthFlowStore) Save(flow authflow.Flow) error {
	return s.client.Save(context.Background(), AuthFlowStoreSaveParams{
		Flow: flow,
	})
}

func (s *pluginAuthFlowStore) Delete(flowID string) error {
	return s.client.Delete(context.Background(), AuthFlowStoreDeleteParams{
		FlowID: flowID,
	})
}

func (s *pluginAuthFlowStore) Path(flowID string) string {
	backend := ""
	if s != nil && s.client != nil {
		backend = s.client.Backend()
	}
	return "plugin://" + backend + "/authflows/" + strings.TrimSpace(flowID)
}
