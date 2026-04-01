package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/stockyard-dev/stockyard-rangeland/internal/store"
)

type Server struct {
	db     *store.DB
	mux    *http.ServeMux
	port   int
	limits Limits
}

func New(db *store.DB, port int, limits Limits) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), port: port, limits: limits}
	s.routes()
	return s
}

func (s *Server) routes() {
	// Sites
	s.mux.HandleFunc("POST /api/sites", s.handleCreateSite)
	s.mux.HandleFunc("GET /api/sites", s.handleListSites)
	s.mux.HandleFunc("GET /api/sites/{id}", s.handleGetSite)
	s.mux.HandleFunc("DELETE /api/sites/{id}", s.handleDeleteSite)

	// Analytics
	s.mux.HandleFunc("GET /api/sites/{id}/overview", s.handleOverview)
	s.mux.HandleFunc("GET /api/sites/{id}/timeseries", s.handleTimeseries)
	s.mux.HandleFunc("GET /api/sites/{id}/pages", s.handleTopPages)
	s.mux.HandleFunc("GET /api/sites/{id}/referrers", s.handleTopReferrers)
	s.mux.HandleFunc("GET /api/sites/{id}/countries", s.handleTopCountries)
	s.mux.HandleFunc("GET /api/sites/{id}/browsers", s.handleTopBrowsers)
	s.mux.HandleFunc("GET /api/sites/{id}/os", s.handleTopOS)
	s.mux.HandleFunc("GET /api/sites/{id}/devices", s.handleTopDevices)
	s.mux.HandleFunc("GET /api/sites/{id}/realtime", s.handleRealtime)

	// Tracking — the hot path
	s.mux.HandleFunc("POST /api/event", s.handleEvent)
	s.mux.HandleFunc("GET /api/event", s.handleEventGET) // img pixel fallback

	// Tracking script
	s.mux.HandleFunc("GET /js/script.js", s.handleScript)

	// Status
	s.mux.HandleFunc("GET /api/status", s.handleStatus)
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /ui", s.handleUI)

	s.mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"product": "stockyard-rangeland", "version": "0.1.0"})
	})
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("[rangeland] listening on %s", addr)
	return http.ListenAndServe(addr, s.mux)
}

// --- Tracking (hot path) ---

func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(204)
		return
	}

	var evt struct {
		Domain   string `json:"d"`
		Path     string `json:"p"`
		Referrer string `json:"r"`
		Screen   string `json:"s"`
	}
	if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
		w.WriteHeader(400)
		return
	}

	s.recordEvent(w, r, evt.Domain, evt.Path, evt.Referrer, evt.Screen)
}

func (s *Server) handleEventGET(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	q := r.URL.Query()
	s.recordEvent(w, r, q.Get("d"), q.Get("p"), q.Get("r"), q.Get("s"))
	// Return 1x1 transparent GIF
	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01, 0x00, 0x3b})
}

func (s *Server) recordEvent(w http.ResponseWriter, r *http.Request, domain, path, referrer, screen string) {
	if domain == "" {
		w.WriteHeader(400)
		return
	}

	site, err := s.db.GetSiteByDomain(domain)
	if err != nil {
		w.WriteHeader(404)
		return
	}

	if path == "" {
		path = "/"
	}

	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.Split(fwd, ",")[0]
	}
	ua := r.UserAgent()

	// Parse referrer domain
	refDomain := ""
	if referrer != "" {
		if u, err := url.Parse(referrer); err == nil {
			refDomain = u.Hostname()
			// Strip own domain from referrer
			if refDomain == domain {
				refDomain = ""
				referrer = ""
			}
		}
	}

	device, browser, osName := parseUA(ua)

	visitorHash := store.HashVisitor(site.ID, ip, ua)
	sessionHash := store.HashSession(visitorHash, path)

	s.db.RecordPageview(site.ID, path, referrer, refDomain, "", device, browser, osName, screen, visitorHash, sessionHash)

	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(202)
}

// --- Tracking script ---

func (s *Server) handleScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	origin := fmt.Sprintf("http://localhost:%d", s.port)
	if r.Host != "" {
		proto := "https"
		if strings.HasPrefix(r.Host, "localhost") || strings.HasPrefix(r.Host, "127.") {
			proto = "http"
		}
		origin = proto + "://" + r.Host
	}
	fmt.Fprintf(w, trackingScript, origin)
}

// --- Site handlers ---

func (s *Server) handleCreateSite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain string `json:"domain"`
		Name   string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Domain == "" {
		writeJSON(w, 400, map[string]string{"error": "domain is required"})
		return
	}

	if s.limits.MaxSites > 0 {
		sites, _ := s.db.ListSites()
		if LimitReached(s.limits.MaxSites, len(sites)) {
			writeJSON(w, 402, map[string]string{
				"error":   fmt.Sprintf("free tier limit: %d site(s) max — upgrade to Pro", s.limits.MaxSites),
				"upgrade": "https://stockyard.dev/rangeland/",
			})
			return
		}
	}

	site, err := s.db.CreateSite(req.Domain, req.Name)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	snippet := fmt.Sprintf(`<script defer data-domain="%s" src="http://localhost:%d/js/script.js"></script>`, site.Domain, s.port)
	writeJSON(w, 201, map[string]any{"site": site, "snippet": snippet})
}

func (s *Server) handleListSites(w http.ResponseWriter, r *http.Request) {
	sites, err := s.db.ListSites()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if sites == nil {
		sites = []store.Site{}
	}
	writeJSON(w, 200, map[string]any{"sites": sites, "count": len(sites)})
}

func (s *Server) handleGetSite(w http.ResponseWriter, r *http.Request) {
	site, err := s.db.GetSite(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "site not found"})
		return
	}
	writeJSON(w, 200, map[string]any{"site": site})
}

func (s *Server) handleDeleteSite(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.db.GetSite(id); err != nil {
		writeJSON(w, 404, map[string]string{"error": "site not found"})
		return
	}
	s.db.DeleteSite(id)
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

// --- Analytics handlers ---

func (s *Server) getDays(r *http.Request) int {
	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	return days
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.db.GetSite(id); err != nil {
		writeJSON(w, 404, map[string]string{"error": "site not found"})
		return
	}
	writeJSON(w, 200, s.db.Overview(id, s.getDays(r)))
}

func (s *Server) handleTimeseries(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	days := s.getDays(r)
	data := s.db.DailyStats(id, days)
	if data == nil {
		data = []store.DailyStat{}
	}
	writeJSON(w, 200, map[string]any{"data": data, "period_days": days})
}

func (s *Server) handleTopPages(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"pages": s.db.TopPages(r.PathValue("id"), s.getDays(r), 20)})
}

func (s *Server) handleTopReferrers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"referrers": s.db.TopReferrers(r.PathValue("id"), s.getDays(r), 20)})
}

func (s *Server) handleTopCountries(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"countries": s.db.TopCountries(r.PathValue("id"), s.getDays(r), 20)})
}

func (s *Server) handleTopBrowsers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"browsers": s.db.TopBrowsers(r.PathValue("id"), s.getDays(r), 20)})
}

func (s *Server) handleTopOS(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"os": s.db.TopOS(r.PathValue("id"), s.getDays(r), 20)})
}

func (s *Server) handleTopDevices(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"devices": s.db.TopDevices(r.PathValue("id"), s.getDays(r), 20)})
}

func (s *Server) handleRealtime(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	count := s.db.RealtimeVisitors(id)
	writeJSON(w, 200, map[string]any{"visitors": count})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.db.Stats())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// --- UA parsing ---

func parseUA(ua string) (device, browser, osName string) {
	ua = strings.ToLower(ua)

	// Device
	device = "desktop"
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") && !strings.Contains(ua, "tablet") {
		device = "mobile"
	} else if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		device = "tablet"
	}
	if strings.Contains(ua, "bot") || strings.Contains(ua, "crawl") || strings.Contains(ua, "spider") {
		device = "bot"
	}

	// Browser
	switch {
	case strings.Contains(ua, "firefox"):
		browser = "Firefox"
	case strings.Contains(ua, "edg/"):
		browser = "Edge"
	case strings.Contains(ua, "chrome") || strings.Contains(ua, "crios"):
		browser = "Chrome"
	case strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome"):
		browser = "Safari"
	case strings.Contains(ua, "opera") || strings.Contains(ua, "opr/"):
		browser = "Opera"
	case strings.Contains(ua, "curl"):
		browser = "curl"
	default:
		browser = "Other"
	}

	// OS
	switch {
	case strings.Contains(ua, "windows"):
		osName = "Windows"
	case strings.Contains(ua, "mac os") || strings.Contains(ua, "macintosh"):
		osName = "macOS"
	case strings.Contains(ua, "linux") && !strings.Contains(ua, "android"):
		osName = "Linux"
	case strings.Contains(ua, "android"):
		osName = "Android"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad"):
		osName = "iOS"
	default:
		osName = "Other"
	}

	return
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// trackingScript is a lightweight analytics script (~800 bytes minified).
// No cookies, no local storage, GDPR compliant.
const trackingScript = `(function(){
  'use strict';
  var endpoint='%s/api/event';
  var domain=document.currentScript.getAttribute('data-domain');
  if(!domain)return;
  function send(){
    var data={d:domain,p:location.pathname,r:document.referrer,s:screen.width+'x'+screen.height};
    if(navigator.sendBeacon){
      navigator.sendBeacon(endpoint,JSON.stringify(data));
    }else{
      var x=new XMLHttpRequest();x.open('POST',endpoint);x.setRequestHeader('Content-Type','application/json');x.send(JSON.stringify(data));
    }
  }
  if(document.visibilityState==='prerender'){document.addEventListener('visibilitychange',function(){if(document.visibilityState!=='prerender')send()},{once:true});}
  else{send();}
  var pushState=history.pushState;
  history.pushState=function(){pushState.apply(this,arguments);send();};
  window.addEventListener('popstate',send);
})();`
