package api

import (
	"fmt"
	pathpkg "path"
	"strings"
)

type HandlerFunc func(*Context) *Response

type RouteSpec struct {
	Method  string
	Pattern string
	Handler HandlerFunc
	parts   []routePart
}

type routePart struct {
	literal string
	param   string
}

func newRoute(method, pattern string, handler HandlerFunc) RouteSpec {
	pattern = cleanPath(pattern)
	return RouteSpec{
		Method:  strings.ToUpper(method),
		Pattern: pattern,
		Handler: handler,
		parts:   parseRouteParts(pattern),
	}
}

func validateRoutes(routes []RouteSpec) error {
	patterns := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		if route.Method == "" || route.Pattern == "" || route.Handler == nil {
			return fmt.Errorf("invalid API route: %+v", route)
		}
		key := route.Method + " " + route.Pattern
		if _, exists := patterns[key]; exists {
			return fmt.Errorf("duplicate API route: %s", key)
		}
		patterns[key] = struct{}{}
	}
	return nil
}

func parseRouteParts(pattern string) []routePart {
	pattern = strings.Trim(pattern, "/")
	if pattern == "" {
		return nil
	}
	segments := strings.Split(pattern, "/")
	parts := make([]routePart, 0, len(segments))
	for _, segment := range segments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			name := strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
			parts = append(parts, routePart{param: name})
			continue
		}
		parts = append(parts, routePart{literal: segment})
	}
	return parts
}

func (r RouteSpec) Match(rawPath string) (map[string]string, bool) {
	path := strings.Trim(cleanPath(rawPath), "/")
	if path == "" && len(r.parts) == 0 {
		return map[string]string{}, true
	}
	segments := strings.Split(path, "/")
	if len(segments) != len(r.parts) {
		return nil, false
	}
	params := make(map[string]string)
	for index, part := range r.parts {
		if part.param != "" {
			params[part.param] = segments[index]
			continue
		}
		if part.literal != segments[index] {
			return nil, false
		}
	}
	return params, true
}

func cleanPath(rawPath string) string {
	clean := pathpkg.Clean("/" + strings.TrimSpace(rawPath))
	if clean == "." {
		return "/"
	}
	return clean
}
