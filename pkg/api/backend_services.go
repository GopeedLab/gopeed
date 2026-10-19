package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/GopeedLab/gopeed/pkg/download"
	restmodel "github.com/GopeedLab/gopeed/pkg/rest/model"
)

func (c *Context) requestContext() context.Context {
	if c.RequestContext != nil {
		return c.RequestContext
	}
	return context.Background()
}
func backendError(err error) *Response {
	status := http.StatusBadRequest
	if errors.Is(err, download.ErrUnSupportedProtocol) || errors.Is(err, download.ErrProtocolJobNotFound) {
		status = http.StatusNotFound
	}
	return newStatusResponse(status, restmodel.NewErrorResult(err.Error()))
}
func (s *Service) triggerProtocolJob(c *Context) *Response {
	if err := s.Downloader.TriggerProtocolJob(c.requestContext(), c.PathParam("protocol"), c.PathParam("name")); err != nil {
		return backendError(err)
	}
	return newNilResponse()
}
func (s *Service) checkAppUpdate(c *Context) *Response {
	var request struct {
		CurrentVersion string `json:"currentVersion"`
	}
	if response := decodeRequest(c.Body, &request); response != nil {
		return response
	}
	result, err := s.Downloader.CheckAppUpdate(c.requestContext(), request.CurrentVersion)
	if err != nil {
		return backendError(err)
	}
	return newOKResponse(result)
}
func (s *Service) appUpdateCandidates(c *Context) *Response {
	var request struct {
		Version string `json:"version"`
		Asset   string `json:"asset"`
	}
	if response := decodeRequest(c.Body, &request); response != nil {
		return response
	}
	result, err := s.Downloader.GetAppUpdateCandidates(request.Version, request.Asset)
	if err != nil {
		return backendError(err)
	}
	return newOKResponse(result)
}
func (s *Service) storeExtensions(c *Context) *Response {
	result, err := s.Downloader.StoreExtensions(c.requestContext(), c.Query)
	if err != nil {
		return backendError(err)
	}
	return newOKResponse(result)
}
func (s *Service) storeExtension(c *Context) *Response {
	result, err := s.Downloader.StoreExtension(c.requestContext(), c.PathParam("id"), c.Query.Get("version"))
	if err != nil {
		return backendError(err)
	}
	return newOKResponse(result)
}
func (s *Service) reportStoreInstall(c *Context) *Response {
	if err := s.Downloader.ReportStoreInstall(c.requestContext(), c.PathParam("id")); err != nil {
		return backendError(err)
	}
	return newNilResponse()
}
