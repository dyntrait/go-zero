package rest

import (
	"net/http"
	"time"
)

type (
	// Middleware defines the middleware method.
	// 入参是函数，返回这也是一个函数
	Middleware func(next http.HandlerFunc) http.HandlerFunc

	// A Route is a http route.
	Route struct {
		Method  string
		Path    string
		Handler http.HandlerFunc
	}

	// RouteOption defines the method to customize a featured route.
	RouteOption func(r *featuredRoutes)

	jwtSetting struct {
		enabled    bool
		secret     string
		prevSecret string
	}

	signatureSetting struct {
		SignatureConf
		enabled bool
	}

	// Route上施加约束:接口超时，jwt认证，最大字节等
	featuredRoutes struct {
		timeout   time.Duration
		priority  bool
		jwt       jwtSetting
		signature signatureSetting
		routes    []Route
		maxBytes  int64
	}
)
