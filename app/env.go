package app

import (
	oas "github.com/charmixer/oas/exporter"
	cache "github.com/patrickmn/go-cache"
)

type Environment struct {
	Ip     string
	Port   int
	Addr   string
	Domain string

	OpenAPI oas.Openapi

	BaseUrl               string
	AuthorizationEndpoint string
	TokenEndpoint         string

	PollIntervalInSeconds int

	CacheDefaultExpiration int
	CachePurgeExpired      int
	Cache                  *cache.Cache
}

var Env Environment
