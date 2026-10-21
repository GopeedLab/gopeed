package download

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/util"
)

func (d *Downloader) protocolManager(name string) (fetcher.FetcherManager, error) {
	for _, fm := range d.cfg.FetchManagers {
		if fm.Name() == name {
			return fm, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrUnSupportedProtocol, name)
}
func jsonObject(v any) (map[string]json.RawMessage, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var object map[string]json.RawMessage
	if err = json.Unmarshal(raw, &object); err != nil {
		return nil, err
	}
	if object == nil {
		object = map[string]json.RawMessage{}
	}
	return object, nil
}
func normalizeProtocol(fm fetcher.FetcherManager, v any) (any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	target := fm.DefaultConfig()
	if target == nil {
		return nil, errors.New("protocol does not expose configuration")
	}
	// Managers may return either typed pointers or value/map defaults.
	valueDefault := reflect.TypeOf(target).Kind() != reflect.Pointer
	if valueDefault {
		pointer := reflect.New(reflect.TypeOf(target))
		defaults, err := json.Marshal(target)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(defaults, pointer.Interface()); err != nil {
			return nil, err
		}
		target = pointer.Interface()
	}
	if err = json.Unmarshal(raw, target); err != nil {
		return nil, err
	}
	if normalizer, ok := fm.(fetcher.ConfigNormalizer); ok {
		raw, err = json.Marshal(target)
		if err != nil {
			return nil, err
		}
		return normalizer.NormalizeConfig(raw)
	}
	if valueDefault {
		return reflect.ValueOf(target).Elem().Interface(), nil
	}
	return target, nil
}

// preserveProtocolState ignores all client-supplied state, including null or
// malformed state values, before decoding/validating the writable config.
func preserveProtocolState(next, previous any) (any, error) {
	object, err := jsonObject(next)
	if err != nil {
		return nil, err
	}
	before, err := jsonObject(previous)
	if err != nil {
		return nil, err
	}
	// Match encoding/json field folding as well as the canonical spelling.
	for key := range object {
		if strings.EqualFold(key, "state") {
			delete(object, key)
		}
	}
	if state, ok := before["state"]; ok {
		object["state"] = state
	}
	return object, nil
}
func (d *Downloader) readProtocolConfig(name string) (any, error) {
	if _, err := d.protocolManager(name); err != nil {
		return nil, err
	}
	d.configLock.RLock()
	defer d.configLock.RUnlock()
	value := d.cfg.ProtocolConfig[name]
	return *util.DeepClone(&value), nil
}

// updateProtocolState is the only internal write path for backend-owned state.
// It never publishes callback edits outside the reserved state section.
func (d *Downloader) updateProtocolState(name string, change func(config, state json.RawMessage) (any, error)) error {
	fm, err := d.protocolManager(name)
	if err != nil {
		return err
	}
	d.configLock.Lock()
	defer d.configLock.Unlock()
	if d.closed.Load() {
		return errors.New("downloader is closed")
	}
	next := util.DeepClone(d.cfg.DownloaderStoreConfig)
	object, err := jsonObject(next.ProtocolConfig[name])
	if err != nil {
		return err
	}
	state, ok := object["state"]
	if !ok {
		return fmt.Errorf("protocol %s does not expose persistent state", name)
	}
	raw, err := json.Marshal(object)
	if err != nil {
		return err
	}
	value, err := change(raw, state)
	if err != nil {
		return err
	}
	// Keep the transaction's original config isolated from callback mutations.
	object["state"], err = json.Marshal(value)
	if err != nil {
		return err
	}
	normalized, err := normalizeProtocol(fm, object)
	if err != nil {
		return err
	}
	next.ProtocolConfig[name] = normalized
	if err = d.storage.Put(bucketConfig, "config", next); err != nil {
		return err
	}
	d.cfg.DownloaderStoreConfig.ProtocolConfig = next.ProtocolConfig
	return nil
}

// initProtocolConfigs loads the current schema and initializes missing defaults.
// This release intentionally does not import settings from older config layouts.
func (d *Downloader) initProtocolConfigs() error {
	cfg := d.cfg.DownloaderStoreConfig
	cfg.Init()
	if err := cfg.GithubMirror.Validate(); err != nil {
		return err
	}
	for _, fm := range d.cfg.FetchManagers {
		value, ok := cfg.ProtocolConfig[fm.Name()]
		if !ok {
			value = map[string]any{}
		}
		normalized, err := normalizeProtocol(fm, value)
		if err != nil {
			return fmt.Errorf("%s config: %w", fm.Name(), err)
		}
		cfg.ProtocolConfig[fm.Name()] = normalized
	}
	return d.storage.Put(bucketConfig, "config", cfg)
}
