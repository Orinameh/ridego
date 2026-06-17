package proxy

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/ridego/pkg/resilience"
)

// NewReverseProxy builds a reverse proxy that resolves the upstream
// address via KubeResolver and wraps the call in a circuit breaker
// keyed by service name.
func NewReverseProxy(r *KubeResolver, svcName string) http.Handler {
	breaker := resilience.NewBreaker(svcName)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		addr, err := r.Resolve(svcName)
		if err != nil {
			slog.Error("resolve", "svc", svcName, "err", err)
			http.Error(w, `{"error":"service unavailable"}`, http.StatusServiceUnavailable)
			return
		}

		target, _ := url.Parse(fmt.Sprintf("http://%s", addr))
		rp := &httputil.ReverseProxy{
			Rewrite: func(pr *httputil.ProxyRequest) {
				pr.SetURL(target)
				pr.Out.Host = target.Host
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				slog.Error("proxy error", "svc", svcName, "err", err)
				http.Error(w, `{"error":"upstream error"}`, http.StatusBadGateway)
			},
		}

		_, cbErr := breaker.Execute(func() ([]byte, error) {
			rp.ServeHTTP(w, req)
			return nil, nil
		})
		if cbErr != nil {
			http.Error(w, `{"error":"circuit open, try later"}`, http.StatusServiceUnavailable)
		}
	})
}
