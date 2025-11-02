package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"tiny11-builder/internal/app"
	"tiny11-builder/internal/config"
	"tiny11-builder/internal/logger"
	"tiny11-builder/internal/types"
)

type Server struct {
	port   int
	log    *logger.Logger
	mu     sync.RWMutex
	status *types.BuildStatus
}

func NewServer(port int, log *logger.Logger) *Server {
	return &Server{
		port: port, log: log, status: &types.BuildStatus{
			Phase: "idle", Progress: 0}}
}
func (s *Server) Start() error {
	http.HandleFunc("/api/build", s.handleBuild)
	http.HandleFunc("/api/status", s.handleStatus)
	http.HandleFunc("/api/themes", s.handleThemes)
	http.HandleFunc("/api/preinstall", s.handlePreinstall)
	addr := fmt.Sprintf(":%d", s.port)
	s.log.Info("API服务器启动在 http://localhost%s", addr)
	return http.ListenAndServe(addr, nil)
}
func (s *Server) handleBuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持 POST", http.StatusMethodNotAllowed)
		return
	}
	var req types.BuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "无效的请求", err)
		return
	}
	go s.executeBuild(&req)
	s.sendJSON(w, types.BuildResponse{
		Success: true, Message: "构建已启动"})
}
func (s *Server) executeBuild(req *types.BuildRequest) {
	s.updateStatus("preparing", 0, "准备构建环境")
	cfg := config.NewConfig()
	cfg.ISODrive = req.ISODrive
	cfg.ThemeName = req.Theme
	cfg.PreinstallApps = req.PreinstallApps
	cfg.ImageIndex = req.ImageIndex
	if req.ScratchDrive != "" {
		cfg.ScratchDrive = req.ScratchDrive
	}
	log := logger.NewLogger("api-build")
	defer log.Close()
	var builder app.Builder
	if req.Mode == types.ModeCore {
		cfg.CoreMode = true
		builder = app.NewTiny11CoreBuilder(cfg, log)
	} else {
		builder = app.NewTiny11Builder(cfg, log)
	}
	s.updateStatus("building", 10, "开始构建")
	if err := builder.Build(); err != nil {
		s.updateStatus("error", 0, err.Error())
		return
	}
	s.updateStatus("complete", 100, "构建完成")
	s.mu.Lock()
	s.status.OutputISO = builder.GetOutputISO()
	s.mu.Unlock()
}
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.sendJSON(w, s.status)
}
func (s *Server) handleThemes(w http.ResponseWriter, r *http.Request) {
	themes := []string{"default", "miku"}
	s.sendJSON(w, themes)
}
func (s *Server) handlePreinstall(w http.ResponseWriter, r *http.Request) {
	apps := []map[string]string{{"id": "chrome", "name": "Google Chrome"}, {"id": "7zip", "name": "7-Zip"}}
	s.sendJSON(w, apps)
}
func (s *Server) updateStatus(phase string, progress float64, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.Phase = phase
	s.status.Progress = progress
	s.status.Message = message
	s.status.IsComplete = phase == "complete"
	if phase == "error" {
		s.status.Error = message
	}
}
func (s *Server) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
func (s *Server) sendError(w http.ResponseWriter, message string, err error) {
	w.WriteHeader(http.StatusBadRequest)
	s.sendJSON(w, types.BuildResponse{
		Success: false, Message: message, Error: err.Error()})
}
