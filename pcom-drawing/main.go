package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"time"
)

//go:embed static/* data/presets.json
var assets embed.FS
var requestShutdown = make(chan struct{}, 1)
var lastRequest atomic.Int64

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, err error) {
	var e ValidationError
	if errors.As(err, &e) {
		writeJSON(w, http.StatusUnprocessableEntity, e)
		return
	}
	log.Printf("요청 처리 오류: %v", err)
	writeJSON(w, 500, map[string]string{"detail": "도면을 만들지 못했습니다. 입력을 확인하고 다시 시도해 주세요."})
}
func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, "POST 요청이 필요합니다.", 405)
		return false
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeJSON(w, 415, map[string]string{"detail": "JSON 형식으로 요청해 주세요."})
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 65536)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		writeJSON(w, 422, map[string][]FieldError{"detail": {{"config", "올바른 JSON 사양 파일인지 확인해 주세요. " + err.Error()}}})
		return false
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		writeJSON(w, 422, map[string]string{"detail": "JSON 객체는 하나만 입력할 수 있습니다."})
		return false
	}
	return true
}
func catalog() map[string]any {
	return map[string]any{"title": "pCOM 도면 작성", "version": "1.0.0", "defaults": defaultConfig(), "options": options, "presets": presets, "source_notes": []string{"8비트 전용 · 확인된 I/O 유형 0 / 1 / 2 / F / L을 지원합니다.", "커넥터 코드 0은 개방선입니다. 검사·제거용 커넥터는 완성품 도면에 포함하지 않습니다.", "도면번호에 담기지 않는 BOARD 핀 순서, 보조 치수와 색상은 별도로 보존합니다.", "PPTX는 편집 가능한 개략 연결도입니다. 원본 커넥터 기구 형상과 전체 제조 BOM을 복제하지 않습니다."}, "capabilities": []string{"Windows 실행 파일", "8비트 코드 생성·해석", "PPTX / SVG / JSON 내보내기", "원본 18장 프리셋", "전원 핀 순서·검정 커넥터 보존"}}
}
func newHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok", "version": "1.0.0", "app": "pcom-drawing"})
	})
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("/api/catalog", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}
		writeJSON(w, 200, catalog())
	})
	mux.HandleFunc("/api/decode", func(w http.ResponseWriter, r *http.Request) {
		var v struct {
			Code string `json:"code"`
		}
		if !readJSON(w, r, &v) {
			return
		}
		c, warnings, err := decodeCode(v.Code)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"config": c, "warnings": warnings})
	})
	mux.HandleFunc("/api/preview", func(w http.ResponseWriter, r *http.Request) {
		var c Config
		if !readJSON(w, r, &c) {
			return
		}
		m, err := makeModel(c)
		if err != nil {
			writeError(w, err)
			return
		}
		m.SVG = renderSVG(m)
		writeJSON(w, 200, m)
	})
	for _, kind := range []string{"pptx", "svg", "json"} {
		kind := kind
		mux.HandleFunc("/api/export/"+kind, func(w http.ResponseWriter, r *http.Request) {
			var c Config
			if !readJSON(w, r, &c) {
				return
			}
			m, err := makeModel(c)
			if err != nil {
				writeError(w, err)
				return
			}
			var data []byte
			var contentType string
			switch kind {
			case "pptx":
				data, err = renderPPTX(m)
				contentType = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
			case "svg":
				data = []byte(renderSVG(m))
				contentType = "image/svg+xml; charset=utf-8"
			case "json":
				data, err = json.MarshalIndent(map[string]any{"app": "pCOM Drawing", "version": "1.0.0", "code": m.Code, "config": m.Config, "source": m.Source, "warnings": m.Warnings}, "", "  ")
				contentType = "application/json; charset=utf-8"
			}
			if err != nil {
				writeError(w, err)
				return
			}
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, m.Code, kind))
			w.Header().Set("Content-Length", fmt.Sprint(len(data)))
			_, _ = w.Write(data)
		})
	}
	mux.HandleFunc("/api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		var v map[string]any
		if !readJSON(w, r, &v) {
			return
		}
		writeJSON(w, 200, map[string]string{"status": "stopping"})
		select {
		case requestShutdown <- struct{}{}:
		default:
		}
	})
	sub, _ := fs.Sub(assets, "static")
	static := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeJSON(w, 404, map[string]string{"detail": "존재하지 않는 API입니다."})
			return
		}
		static.ServeHTTP(w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastRequest.Store(time.Now().Unix())
		host := r.Host
		if h, _, e := net.SplitHostPort(host); e == nil {
			host = h
		}
		if host != "127.0.0.1" && host != "localhost" && host != "[::1]" && host != "::1" {
			http.Error(w, "Local requests only", 403)
			return
		}
		origin := r.Header.Get("Origin")
		if origin != "" && origin != "http://"+r.Host {
			http.Error(w, "Cross-origin request blocked", 403)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; object-src 'none'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		mux.ServeHTTP(w, r)
	})
}
func main() {
	addr := flag.String("addr", "127.0.0.1:8752", "로컬 수신 주소")
	headless := flag.Bool("headless", false, "전용 창을 열지 않음 (검증/개발용)")
	flag.Parse()
	data, err := assets.ReadFile("data/presets.json")
	if err != nil {
		log.Fatal(err)
	}
	if err = loadPresets(data); err != nil {
		log.Fatal(err)
	}
	host, _, err := net.SplitHostPort(*addr)
	if err != nil || host != "127.0.0.1" {
		log.Fatal("주소는 127.0.0.1:<port> 형식이어야 합니다.")
	}
	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		// Reopening the EXE should reuse an existing local app, not start a second server.
		if !*headless {
			client := &http.Client{Timeout: time.Second}
			if res, e := client.Get("http://" + *addr + "/health"); e == nil {
				var health map[string]string
				_ = json.NewDecoder(io.LimitReader(res.Body, 1024)).Decode(&health)
				res.Body.Close()
				if health["app"] == "pcom-drawing" {
					_ = openAppWindow("http://" + *addr)
					return
				}
			}
		}
		notifyError("프로그램을 시작하지 못했습니다: " + err.Error())
		return
	}
	url := "http://" + listener.Addr().String()
	lastRequest.Store(time.Now().Unix())
	server := &http.Server{Handler: newHandler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server: %v", err)
			select {
			case requestShutdown <- struct{}{}:
			default:
			}
		}
	}()
	log.Printf("pCOM Drawing v1.0.0 · %s", url)
	if !*headless {
		if err := openAppWindow(url); err != nil {
			notifyError("화면을 열지 못했습니다. 브라우저 주소: " + url + "\n" + err.Error())
		}
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	running := true
	for running {
		select {
		case <-requestShutdown:
			running = false
		case <-stop:
			running = false
		case <-ticker.C:
			if !*headless && time.Now().Unix()-lastRequest.Load() > 150 {
				running = false
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
