package api

import (
	"context"
	"fmt"
	"net/http"
	"github.com/gorilla/mux"
	"openaloha.io/openaloha-devpod/coding/handler"
)

type DevPodServer struct {
	srv    *http.Server
	coding handler.CodingHandler
}

func NewDevPodServer(addr string, coding handler.CodingHandler) *DevPodServer {
	srv := &DevPodServer{
		srv: &http.Server{
			Addr: addr,
		},
		coding: coding,
	}
	router := mux.NewRouter()
	router.HandleFunc("/ask", srv.CreateAskHandler)
	srv.srv.Handler = router
	return srv
}

func (s *DevPodServer) CreateAskHandler(w http.ResponseWriter, r *http.Request) {
	// 对于GET请求，使用URL查询参数
	question := r.URL.Query().Get("question")
	if question == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error: question parameter is required"))
		return
	}

	// 使用echo管道的方式直接向gemini传递指令
	err := s.coding.Coding(question, w, w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Error: %v", err)))
		return
	}

	// 清理输出中的多余空白字符
	w.Write([]byte("success"))
}

func (s *DevPodServer) ListenAndServe() <-chan error {
	errChan := make(chan error)
	go func() {
		err := s.srv.ListenAndServe()
		if err != nil {
			errChan <- err
		}
	}()
	return errChan
}

// Shutdown 优雅停止服务器
func (s *DevPodServer) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
