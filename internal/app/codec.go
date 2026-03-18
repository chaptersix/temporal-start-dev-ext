package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

type CodecServer struct {
	Endpoint string

	server   *http.Server
	listener net.Listener
	stopOnce sync.Once
}

type passthroughCodec struct{}

func (passthroughCodec) Encode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	return payloads, nil
}

func (passthroughCodec) Decode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	return payloads, nil
}

func StartCodecServer() (*CodecServer, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen codec server: %w", err)
	}

	h := converter.NewPayloadCodecHTTPHandler(passthroughCodec{})
	h = withCORS(h)

	srv := &http.Server{Handler: h}
	codec := &CodecServer{
		Endpoint: fmt.Sprintf("http://%s", ln.Addr().String()),
		server:   srv,
		listener: ln,
	}

	go func() {
		err := srv.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			_ = ln.Close()
		}
	}()

	return codec, nil
}

func (s *CodecServer) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.server.Shutdown(ctx)
		_ = s.listener.Close()
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Namespace")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
