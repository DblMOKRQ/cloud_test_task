package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/DblMOKRQ/cloud_test_task/internal/router/errs"
	logger "github.com/DblMOKRQ/cloud_test_task/pkg"
)

// Proxy создает reverse proxy для указанного целевого URL.
// Логирует ошибки проксирования запросов.
func Proxy(target *url.URL, log *logger.Logger) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error("Proxying a request to the server " + target.String() + " failed: " + err.Error())
		errs.JSONError(w, errs.ErrorResponse{Error: "Service is unavailable"}, http.StatusBadGateway)
	}
	log.Info("Proxying a request to the server " + target.String())
	return proxy
}
