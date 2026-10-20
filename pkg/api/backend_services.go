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
func (s *Service) mirrorCandidates(c *Context) *Response {
	var request struct {
		URL string `json:"url"`
	}
	if response := decodeRequest(c.Body, &request); response != nil {
		return response
	}
	urls, err := s.Downloader.GetMirrorCandidates(c.PathParam("provider"), request.URL)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, download.ErrMirrorProviderNotFound):
			status = http.StatusNotFound
		case errors.Is(err, download.ErrInvalidMirrorURL):
			status = http.StatusBadRequest
		}
		return newStatusResponse(status, restmodel.NewErrorResult(err.Error()))
	}
	return newOKResponse(struct {
		URLs []string `json:"urls"`
	}{URLs: urls})
}
