package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/openaloha/openaloha-devpod/run/handler"
)

type DevPodServer struct {
	srv *http.Server
	run handler.RunHandler
}

func NewDevPodServer(addr string, run handler.RunHandler) *DevPodServer {
	srv := &DevPodServer{
		srv: &http.Server{
			Addr: addr,
		},
		run: run,
	}
	router := mux.NewRouter()
    router.HandleFunc("/ask", srv.CreateAskHandler)
    srv.srv.Handler = router
	return srv
}

func(s *DevPodServer) CreateAskHandler(w http.ResponseWriter, r *http.Request) {
	// 对于GET请求，使用URL查询参数
	question := r.URL.Query().Get("question")
	if question == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error: question parameter is required"))
		return
	}
	
	// 使用echo管道的方式直接向gemini传递指令
	err := s.run.Run([]string{"gemini -y -a -p " + question})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Error: %v", err)))
		return
	}
	
	// 清理输出中的多余空白字符
	w.Write([]byte("success"))
}


 func (s *DevPodServer) ListenAndServe() (<-chan error, error) {
     var err error
     errChan := make(chan error)
     // 通过 Goroutine 运行，避免阻塞代码继续运行
     go func() {
         err = s.srv.ListenAndServe()
         errChan <- err
     }()

	 // 监听 http server Goroutine 运行状态
     select {
     case err = <-errChan:
         return nil, err
     case <-time.After(time.Second):
         return errChan, nil
     }
 }

// Shutdown 优雅停止服务器
func (s *DevPodServer) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}