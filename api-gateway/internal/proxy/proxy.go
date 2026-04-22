package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Proxy is a reverse proxy to a single upstream service.
type Proxy struct {
	target  *url.URL
	handler *httputil.ReverseProxy
}

func New(targetURL string, logger *zap.Logger) (*Proxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("proxy: parse url %q: %w", targetURL, err)
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}

	rp := httputil.NewSingleHostReverseProxy(target)
	rp.Transport = transport

	original := rp.Director
	rp.Director = func(req *http.Request) {
		original(req)
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Host = target.Host
	}

	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error("upstream unreachable",
			zap.String("target", targetURL),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, `{"success":false,"message":"upstream service unavailable","code":"BAD_GATEWAY"}`)
	}

	return &Proxy{target: target, handler: rp}, nil
}

// Handler returns a Gin handler that forwards the request to the upstream service.
// It also injects X-Request-ID and X-User-* headers set by upstream middlewares.
func (p *Proxy) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Header.Del("X-User-ID")
		c.Request.Header.Del("X-User-Email")
		c.Request.Header.Del("X-User-Role")

		if reqID, ok := c.Get("request_id"); ok {
			c.Request.Header.Set("X-Request-ID", fmt.Sprintf("%v", reqID))
		}
		if userID, ok := c.Get("user_id"); ok {
			c.Request.Header.Set("X-User-ID", fmt.Sprintf("%v", userID))
		}
		if email, ok := c.Get("email"); ok {
			c.Request.Header.Set("X-User-Email", fmt.Sprintf("%v", email))
		}
		if role, ok := c.Get("role"); ok {
			c.Request.Header.Set("X-User-Role", fmt.Sprintf("%v", role))
		}
		p.handler.ServeHTTP(c.Writer, c.Request)
	}
}
