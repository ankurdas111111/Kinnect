package api

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var gzPool = sync.Pool{
	New: func() any {
		gz, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		return gz
	},
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gz          *gzip.Writer
	wroteHeader bool
	// passthrough is set when the handler below supplied its own
	// Content-Encoding (serveStaticCompressed shipping a precompressed .br/.gz
	// sibling). Gzipping those bytes would double-compress them and leave the
	// stream mislabelled as the inner encoding.
	passthrough bool
}

// decide latches whether this response is ours to gzip. It must run before the
// first byte reaches the client, and only once - the handler may set
// Content-Encoding any time before that.
func (g *gzipResponseWriter) decide() {
	enc := g.Header().Get("Content-Encoding")
	g.passthrough = enc != "" && !strings.EqualFold(enc, "gzip")
	g.wroteHeader = true
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	if !g.wroteHeader {
		if g.Header().Get("Content-Type") == "" {
			g.Header().Set("Content-Type", http.DetectContentType(b))
		}
		g.decide()
	}
	if g.passthrough {
		return g.ResponseWriter.Write(b)
	}
	return g.gz.Write(b)
}

func (g *gzipResponseWriter) WriteHeader(code int) {
	if !g.wroteHeader {
		g.decide()
	}
	// Length is only unknown when we recompress; a passed-through body keeps
	// the Content-Length its handler already computed.
	if !g.passthrough {
		g.Header().Del("Content-Length")
	}
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipResponseWriter) Flush() {
	if !g.passthrough {
		g.gz.Flush()
	}
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (g *gzipResponseWriter) Unwrap() http.ResponseWriter {
	return g.ResponseWriter
}

var skipGzipExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
	".woff2": true, ".woff": true, ".br": true, ".gz": true, ".zst": true,
	// Static assets served by http.ServeFile — gzip here corrupts the stream
	// because gz.Close() runs after ServeFile finalizes the response.
	// These files are cached immutable so compression isn't needed.
	".js": true, ".css": true, ".map": true, ".svg": true, ".ico": true,
}

// GzipMiddleware transparently gzip-compresses responses for clients that accept it.
// WebSocket upgrades and pre-compressed formats (images, fonts) are skipped.
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		path := r.URL.Path
		for ext := range skipGzipExt {
			if strings.HasSuffix(path, ext) {
				next.ServeHTTP(w, r)
				return
			}
		}

		gz := gzPool.Get().(*gzip.Writer)
		defer gzPool.Put(gz)
		gz.Reset(w)

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")

		grw := &gzipResponseWriter{ResponseWriter: w, gz: gz}
		next.ServeHTTP(grw, r)
		// Closing would emit a gzip trailer over a body we never compressed.
		if !grw.passthrough {
			gz.Close()
		}
	})
}

// NopWriteCloser wraps an io.Writer into a WriteCloser (Close is a no-op).
type NopWriteCloser struct{ io.Writer }

func (NopWriteCloser) Close() error { return nil }
