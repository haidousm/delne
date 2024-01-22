package proxy

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Proxy struct {
	Target   map[string]string
	RevProxy map[string]*httputil.ReverseProxy
}

func (p *Proxy) ProxyRequest(w http.ResponseWriter, r *http.Request) {
	host := r.Host

	// if we already have a rev proxy for this host setup
	if rev, ok := p.RevProxy[host]; ok {
		rev.ServeHTTP(w, r)
		return
	}

	// otherwise, create one
	if target, ok := p.Target[host]; ok {
		remote, err := url.Parse(target)
		if err != nil {
			println(err.Error())
			return
		}

		rev := httputil.NewSingleHostReverseProxy(remote)
		p.RevProxy[host] = rev
		rev.ServeHTTP(w, r)
		return
	}
	err := errors.New("forbidden host")
	println(err.Error())
}
