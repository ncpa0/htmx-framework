package configuration

import (
	"time"

	"github.com/labstack/echo"
	servestatic "github.com/ncpa0/hardwire/serve-static"
)

type CachingPolicy struct {
	// The maximum age of the cached resource, in seconds.
	MaxAge int
	// When `MaxAge` is 0 and this is enabled, the browser
	// will revalidate the cached resource on every request.
	NoCache bool
	// When true, CDN's will be instructed to not cache the resource.
	Private bool
	// When true, the browser will not cache the resource at all.
	NoStore bool
}

type CachingConfig struct {
	StaticRoutes  *CachingPolicy
	DynamicRoutes *CachingPolicy
	Fragments     *CachingPolicy
}

type Configuration struct {
	// When enabled, the `.html` extension will be stripped from the URL pathnames.
	StripExtension bool
	// When enabled, the server will print debug information to the console.
	DebugMode bool
	// The entrypoint file containing the JSX pages used to generate the views html files.
	//
	// Defaults to `index.tsx`.
	Entrypoint string
	// The directory to which output the generated html files, and from which those will be hosted.
	//
	// Defaults to `views`.
	HtmlDir string
	// The directory to which output the static files, and from which those will be hosted.
	//
	// Defaults to `static`.
	StaticDir string
	// The URL path from under which the static files will be hosted.
	//
	// Defaults to `/static`.
	StaticURL string
	// Skip the html generation step.
	//
	// Defaults to `false`.
	NoBuild bool
	// Clean the html directory before generating the html files.
	//
	// Defaults to `false`.
	CleanBuild       bool
	Caching          *CachingConfig
	BeforeStaticSend func(resp *servestatic.StaticResponse, c echo.Context) error
}

var Current *Configuration = &Configuration{
	StripExtension: false,
	DebugMode:      false,
	Entrypoint:     "index.tsx",
	HtmlDir:        "views",
	StaticDir:      "static",
	StaticURL:      "/static",
	NoBuild:        false,
	CleanBuild:     false,
	Caching: &CachingConfig{
		StaticRoutes: &CachingPolicy{
			MaxAge: int(time.Hour.Seconds()),
		},
		DynamicRoutes: &CachingPolicy{
			NoStore: true,
		},
		Fragments: &CachingPolicy{
			NoStore: true,
		},
	},
}

func Configure(newConfig *Configuration) {
	Current.StripExtension = newConfig.StripExtension
	Current.DebugMode = newConfig.DebugMode

	if newConfig.Entrypoint != "" {
		Current.Entrypoint = newConfig.Entrypoint
	}
	if newConfig.HtmlDir != "" {
		Current.HtmlDir = newConfig.HtmlDir
	}
	if newConfig.StaticDir != "" {
		Current.StaticDir = newConfig.StaticDir
	}
	if newConfig.StaticURL != "" {
		Current.StaticURL = newConfig.StaticURL
	}
	if newConfig.BeforeStaticSend != nil {
		Current.BeforeStaticSend = newConfig.BeforeStaticSend
	}
	if newConfig.NoBuild {
		Current.NoBuild = true
	}
	if newConfig.CleanBuild {
		Current.CleanBuild = true
	}
	if newConfig.Caching != nil {
		if newConfig.Caching.StaticRoutes != nil {
			Current.Caching.StaticRoutes = newConfig.Caching.StaticRoutes
		}
		if newConfig.Caching.DynamicRoutes != nil {
			Current.Caching.DynamicRoutes = newConfig.Caching.DynamicRoutes
		}
		if newConfig.Caching.Fragments != nil {
			Current.Caching.Fragments = newConfig.Caching.Fragments
		}
	}
}
