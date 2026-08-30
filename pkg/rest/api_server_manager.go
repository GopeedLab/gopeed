package rest

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/rest/model"
	"github.com/GopeedLab/gopeed/pkg/util"
	"github.com/pkg/errors"
)

type apiServerManager struct {
	server       *http.Server
	runningPort  int
	activeConfig *base.APIServerConfig
	lastError    string
}

var apiServer = &apiServerManager{}

func GetAPIServerState() (*model.APIServerState, error) {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	desired, err := desiredAPIServerConfigLocked()
	return apiServer.stateLocked(desired), err
}

func StartAPIServer() (*model.APIServerState, error) {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	desired, err := desiredAPIServerConfigLocked()
	if err != nil {
		return apiServer.stateLocked(nil), err
	}
	if !desired.Enable {
		err = errors.New("API server is disabled in persisted config")
		apiServer.lastError = err.Error()
		return apiServer.stateLocked(desired), err
	}
	_, err = apiServer.startLocked(desired)
	return apiServer.stateLocked(desired), err
}

func StopAPIServer() (*model.APIServerState, error) {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	desired, configErr := desiredAPIServerConfigLocked()
	err := apiServer.stopLocked()
	if err == nil {
		err = configErr
	}
	return apiServer.stateLocked(desired), err
}

func RestartAPIServer() (*model.APIServerState, error) {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	desired, err := desiredAPIServerConfigLocked()
	if err != nil {
		return apiServer.stateLocked(nil), err
	}
	if !desired.Enable {
		err = errors.New("API server is disabled in persisted config")
		apiServer.lastError = err.Error()
		return apiServer.stateLocked(desired), err
	}
	err = apiServer.restartLocked(desired)
	return apiServer.stateLocked(desired), err
}

func desiredAPIServerConfigLocked() (*base.APIServerConfig, error) {
	if Downloader == nil {
		return nil, errors.New("service not started")
	}
	config, err := Downloader.GetConfig()
	if err != nil {
		return nil, err
	}
	if config.API == nil {
		return (&base.APIServerConfig{}).Init(), nil
	}
	return cloneAPIServerConfig(config.API).Init(), nil
}

func (m *apiServerManager) startLocked(config *base.APIServerConfig) (int, error) {
	if config == nil {
		return 0, errors.New("API server config is nil")
	}
	if m.server != nil {
		if sameAPIServerConfig(m.activeConfig, config) {
			m.lastError = ""
			return m.runningPort, nil
		}
		err := errors.New("API server is already running with different config")
		m.lastError = err.Error()
		return m.runningPort, err
	}

	return m.startWithConfigLocked(startConfigForAPIServer(config), config)
}

func (m *apiServerManager) startWithConfigLocked(
	startConfig *model.StartConfig,
	activeConfig *base.APIServerConfig,
) (int, error) {
	server, listener, err := buildServer(startConfig, false)
	if err != nil {
		m.lastError = err.Error()
		return 0, err
	}
	m.server = server
	m.runningPort = 0
	m.activeConfig = cloneAPIServerConfig(activeConfig)
	m.activeConfig.Enable = true
	m.lastError = ""
	if address, ok := listener.Addr().(*net.TCPAddr); ok {
		m.runningPort = address.Port
	}
	go m.serve(server, listener)
	return m.runningPort, nil
}

func (m *apiServerManager) serve(server *http.Server, listener net.Listener) {
	err := server.Serve(listener)
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return
	}
	runtimeMu.Lock()
	if m.server == server {
		m.server = nil
		m.runningPort = 0
		m.activeConfig = nil
		m.lastError = err.Error()
	}
	downloader := Downloader
	runtimeMu.Unlock()
	if downloader != nil {
		downloader.Logger.Error().Err(err).Msg("API server stopped unexpectedly")
	}
}

func (m *apiServerManager) stopLocked() error {
	server := m.server
	config := cloneAPIServerConfig(m.activeConfig)
	if server == nil {
		m.runningPort = 0
		m.activeConfig = nil
		m.lastError = ""
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	err := server.Shutdown(shutdownCtx)
	cancel()
	if err != nil {
		if closeErr := server.Close(); closeErr != nil {
			err = errors.Wrapf(err, "force close API server failed: %v", closeErr)
		}
	}
	m.server = nil
	m.runningPort = 0
	m.activeConfig = nil
	if config != nil && config.Network == "unix" {
		util.SafeRemove(config.Address)
	}
	if err != nil {
		m.lastError = err.Error()
		return errors.Wrap(err, "shutdown API server failed")
	}
	m.lastError = ""
	return nil
}

func (m *apiServerManager) restartLocked(desired *base.APIServerConfig) error {
	if m.server == nil {
		_, err := m.startLocked(desired)
		return err
	}
	if sameAPIServerConfig(m.activeConfig, desired) {
		m.lastError = ""
		return nil
	}
	if err := m.stopLocked(); err != nil {
		return err
	}
	_, err := m.startLocked(desired)
	return err
}

func (m *apiServerManager) stateLocked(desired *base.APIServerConfig) *model.APIServerState {
	state := &model.APIServerState{Running: m.server != nil, RunningPort: m.runningPort, LastError: m.lastError}
	if desired != nil {
		state.Enabled = desired.Enable
	}
	if m.activeConfig != nil {
		state.Network = m.activeConfig.Network
		state.Address = m.activeConfig.Address
	}
	state.PendingApply = !apiServerConfigApplied(desired, m.activeConfig)
	return state
}

func apiServerConfigApplied(desired, active *base.APIServerConfig) bool {
	if desired == nil {
		return false
	}
	if !desired.Enable {
		return active == nil
	}
	return active != nil && sameAPIServerConfig(desired, active)
}
