package method

import (
	"log"
	"net/http"
)

// PushMetricsGetHashV2 is a function.
func PushMetricsGetHashV2(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	node, err := PgwNodeRing.GetNode(path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("get_node_from_hashring_error")); err != nil {
			log.Printf("[PushMetrics][request_path:%s][write_error:%s]", path, err.Error())
		}
	}

	nextUrl := "http://" + node + path
	log.Printf("[PushMetrics][request_path:%s][redirect_url:%s]", path, nextUrl)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("nextUrl:" + nextUrl)); err != nil {
		log.Printf("[PushMetrics][request_path:%s][write_error:%s]", path, err.Error())
	}
}

// PushMetricsRedirectV2 is a function.
func PushMetricsRedirectV2(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	node, err := PgwNodeRing.GetNode(path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("get_node_from_hashring_error")); err != nil {
			log.Printf("[PushMetrics][request_path:%s][write_error:%s]", path, err.Error())
		}
		return
	}
	nextUrl := "http://" + node + path
	log.Printf("[PushMetrics][request_path:%s][redirect_url:%s]", path, nextUrl)
	//c.Redirect(http.StatusMovedPermanently, nextUrl)
	http.Redirect(w, r, nextUrl, http.StatusTemporaryRedirect)
	//c.Redirect(http.StatusPermanentRedirect, nextUrl)
	http.Redirect(w, r, nextUrl, http.StatusPermanentRedirect)
}
