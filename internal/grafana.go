package internal

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/go-openapi/runtime"
	rtclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/go-viper/mapstructure/v2"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/hashicorp/go-cleanhttp"
)

type GrafanaConfig struct {
	URL    string
	Scheme string
	APIKey string
}

func newGrafanaOAPI(cfg *GrafanaConfig) *goapi.GrafanaHTTPAPI {
	tc := &goapi.TransportConfig{
		Client: cleanhttp.DefaultPooledClient(),
		// Host is the domain name or IP address of the host that serves the API.
		Host: cfg.URL,
		// BasePath is the URL prefix for all API paths, relative to the host root.
		BasePath: "/api",
		// Schemes are the transfer protocols used by the API (http or https).
		Schemes: []string{cfg.Scheme},
		// TLSConfig provides an optional configuration for a TLS client
		TLSConfig: &tls.Config{},
		APIKey:    cfg.APIKey,
	}
	return goapi.New(newOAPITransportWithConfig(tc), tc, strfmt.Default)
}

// newOAPITransportWithConfig is inline from https://github.com/grafana/grafana-openapi-client-go/blob/main/client/grafana_http_api_client.go#L420-L462.
// As it is not allowed to configure through interface.
func newOAPITransportWithConfig(cfg *goapi.TransportConfig) *rtclient.Runtime {
	tr := rtclient.NewWithClient(cfg.Host, cfg.BasePath, cfg.Schemes, cfg.Client)
	tr.Transport = cfg.Client.Transport

	var auth []runtime.ClientAuthInfoWriter
	if cfg.BasicAuth != nil {
		pwd, _ := cfg.BasicAuth.Password()
		basicAuth := rtclient.BasicAuth(cfg.BasicAuth.Username(), pwd)
		auth = append(auth, basicAuth)
	}
	if cfg.OrgID != 0 {
		orgIDHeader := runtime.ClientAuthInfoWriterFunc(func(r runtime.ClientRequest, _ strfmt.Registry) error {
			return r.SetHeaderParam(goapi.OrgIDHeader, strconv.FormatInt(cfg.OrgID, 10))
		})
		auth = append(auth, orgIDHeader)
	}
	if cfg.APIKey != "" {
		APIKey := rtclient.BearerToken(cfg.APIKey)
		auth = append(auth, APIKey)
	}

	tr.DefaultAuthentication = rtclient.Compose(auth...)

	// The default runtime.JSONConsumer uses `json.Number` for numbers which is unwieldy to use.
	tr.Consumers[runtime.JSONMime] = runtime.ConsumerFunc(func(reader io.Reader, data interface{}) error {
		return json.NewDecoder(reader).Decode(data)
	})

	tr.Debug = cfg.Debug

	return tr
}

func getAllDashboards(ctx context.Context, graf *goapi.GrafanaHTTPAPI) ([]string, error) {
	var (
		typ               = "dash-db"
		page, limit int64 = 1, 100
		results     []string
	)
	for {
		resp, err := graf.Search.Search(&search.SearchParams{
			Limit:   &limit,
			Page:    &page,
			Type:    &typ,
			Context: ctx,
		})
		if err != nil {
			return nil, fmt.Errorf("dashboard search: %w", err)
		}
		if len(resp.Payload) == 0 {
			break
		}
		for _, db := range resp.Payload {
			results = append(results, db.UID)
		}
		page++
	}
	return results, nil
}

func getDashboardByUID(ctx context.Context, graf *goapi.GrafanaHTTPAPI, uid string) (*Board, error) {
	resp, err := graf.Dashboards.GetDashboardByUID(uid, func(op *runtime.ClientOperation) {
		op.Context = ctx
	})
	if err != nil {
		return nil, fmt.Errorf("get dashboard by uid: %w", err)
	}
	raw, ok := resp.Payload.Dashboard.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("payload can't be casted, uid: %s", uid)
	}
	var board Board
	if err = mapstructure.Decode(raw, &board); err != nil {
		return nil, fmt.Errorf("decode dashboard: %w", err)
	}
	return &board, nil
}
