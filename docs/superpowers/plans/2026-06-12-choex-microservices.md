# ChoexManager Microservices Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the monolithic choex-server into 8 independent gRPC microservices with API Gateway, independent databases per service, and Agent Tool Calling via gRPC.

**Architecture:** gRPC + Protobuf, BFF Gateway (HTTP REST → gRPC), per-service MySQL databases on shared instance, multi-repo with proto submodule, docker-compose orchestration.

**Tech Stack:** Go 1.25, gRPC/Protobuf (buf.build), GORM, MySQL 9.6, Gin (gateway only), JWT HS256, AES-256-GCM, bcrypt, DeepSeek API (openai-go SDK).

**Work directory:** `/Users/zhangsongzhi/ClaudeProjects/`
**GitHub org:** `choex2025-ops`

---

## File Structure Map

```
choex-proto/                   # Task 1
├── go.mod, buf.yaml, buf.gen.yaml
├── common/common.proto
├── auth/v1/auth.proto, calendar/v1/calendar.proto, bill/v1/bill.proto
├── password/v1/password.proto, memory/v1/memory.proto, agent/v1/agent.proto
└── gen/ (generated)

choex-gateway/                 # Task 2 — HTTP :8080
├── go.mod, proto/ (submodule), cmd/server/main.go
├── internal/config/config.go, internal/server/http.go, internal/handler/proxy.go
└── Dockerfile

choex-{auth,calendar,bill,password,memory,agent}/  # Tasks 3-8
  (each: go.mod, proto/ submodule, cmd/server/main.go,
   internal/{config,database,model,service,server}/*.go, Dockerfile)

docker-compose.yml             # Task 9 — root of ~/ClaudeProjects/
```

---

## Phase 1: Foundation

### Task 1: choex-proto — Proto Definitions + Code Generation

- [ ] **Step 1: Create repo**

```bash
mkdir ~/ClaudeProjects/choex-proto && cd ~/ClaudeProjects/choex-proto
git init && go mod init github.com/choex2025-ops/choex-proto
```

- [ ] **Step 2: Write buf.yaml**

```yaml
version: v2
modules:
  - path: .
    name: buf.build/choex2025-ops/choex-proto
```

- [ ] **Step 3: Write buf.gen.yaml**

```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen
    opt: paths=source_relative
  - remote: buf.build/grpc/go
    out: gen
    opt: paths=source_relative
```

- [ ] **Step 4: Write common/common.proto**

```protobuf
syntax = "proto3";
package common;
option go_package = "github.com/choex2025-ops/choex-proto/gen/common";

message UserInfo {
  uint64 user_id = 1;
  string username = 2;
  string email = 3;
}
message Empty {}
```

- [ ] **Step 5: Write auth/v1/auth.proto**

```protobuf
syntax = "proto3";
package auth.v1;
option go_package = "github.com/choex2025-ops/choex-proto/gen/auth/v1";

service AuthService {
  rpc Register(RegisterRequest) returns (AuthResponse);
  rpc Login(LoginRequest) returns (AuthResponse);
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
}

message RegisterRequest { string username = 1; string email = 2; string password = 3; }
message LoginRequest { string email = 1; string password = 2; }
message AuthResponse { string token = 1; uint64 user_id = 2; string username = 3; string email = 4; }
message ValidateTokenRequest { string token = 1; }
message ValidateTokenResponse { bool valid = 1; uint64 user_id = 2; string email = 3; }
```

- [ ] **Step 6: Write calendar/v1/calendar.proto**

```protobuf
syntax = "proto3";
package calendar.v1;
option go_package = "github.com/choex2025-ops/choex-proto/gen/calendar/v1";

service CalendarService {
  rpc ListEvents(ListEventsRequest) returns (ListEventsResponse);
  rpc CreateEvent(CreateEventRequest) returns (CreateEventResponse);
  rpc UpdateEvent(UpdateEventRequest) returns (UpdateEventResponse);
  rpc DeleteEvent(DeleteEventRequest) returns (DeleteEventResponse);
}

message Event {
  uint64 id = 1; uint64 user_id = 2; string title = 3; string description = 4;
  string location = 5; string start_time = 6; string end_time = 7; bool all_day = 8;
  string color = 9; string created_at = 10; string updated_at = 11;
}
message ListEventsRequest { uint64 user_id = 1; }
message ListEventsResponse { repeated Event events = 1; }
message CreateEventRequest {
  uint64 user_id = 1; string title = 2; string description = 3;
  string location = 4; string start_time = 5; string end_time = 6;
  bool all_day = 7; string color = 8;
}
message CreateEventResponse { uint64 id = 1; }
message UpdateEventRequest {
  uint64 id = 1; uint64 user_id = 2; string title = 3; string description = 4;
  string location = 5; string start_time = 6; string end_time = 7;
  bool all_day = 8; string color = 9;
}
message UpdateEventResponse {}
message DeleteEventRequest { uint64 id = 1; uint64 user_id = 2; }
message DeleteEventResponse {}
```

- [ ] **Step 7: Write bill/v1/bill.proto**

```protobuf
syntax = "proto3";
package bill.v1;
option go_package = "github.com/choex2025-ops/choex-proto/gen/bill/v1";

service BillService {
  rpc ListBills(ListBillsRequest) returns (ListBillsResponse);
  rpc CreateBill(CreateBillRequest) returns (CreateBillResponse);
  rpc DeleteBill(DeleteBillRequest) returns (DeleteBillResponse);
  rpc GetStats(GetStatsRequest) returns (GetStatsResponse);
}

message Bill {
  uint64 id = 1; uint64 user_id = 2; double amount = 3; string type = 4;
  string category = 5; string note = 6; string bill_date = 7;
  string created_at = 8; string updated_at = 9;
}
message ListBillsRequest { uint64 user_id = 1; string date = 2; }
message ListBillsResponse { repeated Bill bills = 1; }
message CreateBillRequest {
  uint64 user_id = 1; double amount = 2; string type = 3;
  string category = 4; string note = 5; string bill_date = 6;
}
message CreateBillResponse { uint64 id = 1; }
message DeleteBillRequest { uint64 id = 1; uint64 user_id = 2; }
message DeleteBillResponse {}
message GetStatsRequest { uint64 user_id = 1; string month = 2; }
message GetStatsResponse { double total_income = 1; double total_expense = 2; map<string, double> by_category = 3; }
```

- [ ] **Step 8: Write password/v1/password.proto**

```protobuf
syntax = "proto3";
package password.v1;
option go_package = "github.com/choex2025-ops/choex-proto/gen/password/v1";

service PasswordService {
  rpc ListPasswords(ListPasswordsRequest) returns (ListPasswordsResponse);
  rpc CreatePassword(CreatePasswordRequest) returns (CreatePasswordResponse);
  rpc GetPassword(GetPasswordRequest) returns (GetPasswordResponse);
  rpc UpdatePassword(UpdatePasswordRequest) returns (UpdatePasswordResponse);
  rpc DeletePassword(DeletePasswordRequest) returns (DeletePasswordResponse);
}

message PasswordItem {
  uint64 id = 1; uint64 user_id = 2; string title = 3; string url = 4;
  string username = 5; string password = 6; string note = 7; string category = 8;
  string created_at = 9;
}
message ListPasswordsRequest { uint64 user_id = 1; }
message ListPasswordsResponse { repeated PasswordItem items = 1; }
message CreatePasswordRequest {
  uint64 user_id = 1; string title = 2; string url = 3; string username = 4;
  string password = 5; string note = 6; string category = 7;
}
message CreatePasswordResponse { uint64 id = 1; }
message GetPasswordRequest { uint64 id = 1; uint64 user_id = 2; }
message GetPasswordResponse { PasswordItem item = 1; }
message UpdatePasswordRequest {
  uint64 id = 1; uint64 user_id = 2; string title = 3; string url = 4;
  string username = 5; string password = 6; string note = 7; string category = 8;
}
message UpdatePasswordResponse {}
message DeletePasswordRequest { uint64 id = 1; uint64 user_id = 2; }
message DeletePasswordResponse {}
```

- [ ] **Step 9: Write memory/v1/memory.proto**

```protobuf
syntax = "proto3";
package memory.v1;
option go_package = "github.com/choex2025-ops/choex-proto/gen/memory/v1";

service MemoryService {
  rpc ListMemories(ListMemoriesRequest) returns (ListMemoriesResponse);
  rpc CreateMemory(CreateMemoryRequest) returns (CreateMemoryResponse);
  rpc ActivateMemory(ActivateMemoryRequest) returns (ActivateMemoryResponse);
  rpc DeleteMemory(DeleteMemoryRequest) returns (DeleteMemoryResponse);
  rpc GetVersions(GetVersionsRequest) returns (GetVersionsResponse);
  rpc SaveVersion(SaveVersionRequest) returns (SaveVersionResponse);
  rpc RestoreMemory(RestoreMemoryRequest) returns (RestoreMemoryResponse);
}

message Memory { uint64 id = 1; uint64 user_id = 2; string name = 3; string icon = 4; bool is_active = 5; string created_at = 6; }
message ListMemoriesRequest { uint64 user_id = 1; }
message ListMemoriesResponse { repeated Memory memories = 1; }
message CreateMemoryRequest { uint64 user_id = 1; string name = 2; string icon = 3; }
message CreateMemoryResponse { uint64 id = 1; }
message ActivateMemoryRequest { uint64 id = 1; uint64 user_id = 2; }
message ActivateMemoryResponse {}
message DeleteMemoryRequest { uint64 id = 1; uint64 user_id = 2; }
message DeleteMemoryResponse {}
message GetVersionsRequest { uint64 memory_id = 1; }
message GetVersionsResponse { string current = 1; string backup = 2; string custom = 3; }
message SaveVersionRequest { uint64 memory_id = 1; string type = 2; string content = 3; }
message SaveVersionResponse {}
message RestoreMemoryRequest { uint64 memory_id = 1; }
message RestoreMemoryResponse {}
```

- [ ] **Step 10: Write agent/v1/agent.proto**

```protobuf
syntax = "proto3";
package agent.v1;
option go_package = "github.com/choex2025-ops/choex-proto/gen/agent/v1";

service AgentService { rpc Chat(ChatRequest) returns (stream ChatResponse); }

message ChatRequest { uint64 user_id = 1; string message = 2; repeated ChatMessage history = 3; }
message ChatMessage { string role = 1; string content = 2; }
message ChatResponse { string content = 1; bool done = 2; string error = 3; }
```

- [ ] **Step 11: Generate code and push**

```bash
cd ~/ClaudeProjects/choex-proto
brew install bufbuild/buf/buf 2>/dev/null || true
buf generate
ls gen/calendar/v1/calendar.pb.go gen/auth/v1/auth_grpc.pb.go  # verify
git add -A && git commit -m "feat: add all proto definitions and generated code"
gh repo create choex2025-ops/choex-proto --public --source=. --push
```

---

### Task 2: choex-gateway — API Gateway (HTTP :8080, BFF pattern)

- [ ] **Step 1: Create repo and setup**

```bash
mkdir ~/ClaudeProjects/choex-gateway && cd ~/ClaudeProjects/choex-gateway
git init && go mod init github.com/choex2025-ops/choex-gateway
git submodule add https://github.com/choex2025-ops/choex-proto.git proto
echo 'replace github.com/choex2025-ops/choex-proto => ./proto' >> go.mod
go get google.golang.org/grpc github.com/gin-gonic/gin github.com/golang-jwt/jwt/v5
go mod tidy
```

- [ ] **Step 2: Write internal/config/config.go**

```go
package config

import "os"

type Config struct {
	HTTPPort, JWTSecret                                    string
	AuthAddr, CalendarAddr, BillAddr, PasswordAddr, MemoryAddr, AgentAddr string
}

func Load() *Config {
	return &Config{
		HTTPPort:     getEnv("HTTP_PORT", "8080"),
		JWTSecret:    getEnv("JWT_SECRET", "dev-secret"),
		AuthAddr:     getEnv("AUTH_ADDR", "localhost:9001"),
		CalendarAddr: getEnv("CALENDAR_ADDR", "localhost:9002"),
		BillAddr:     getEnv("BILL_ADDR", "localhost:9003"),
		PasswordAddr: getEnv("PASSWORD_ADDR", "localhost:9004"),
		MemoryAddr:   getEnv("MEMORY_ADDR", "localhost:9005"),
		AgentAddr:    getEnv("AGENT_ADDR", "localhost:9006"),
	}
}

func getEnv(key, def string) string { if v := os.Getenv(key); v != "" { return v }; return def }
```

- [ ] **Step 3: Write cmd/server/main.go**

```go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/choex2025-ops/choex-gateway/internal/config"
	"github.com/choex2025-ops/choex-gateway/internal/server"
)

func main() {
	cfg := config.Load()
	srv := server.New(cfg)
	go func() {
		log.Printf("Gateway listening on :%s", cfg.HTTPPort)
		if err := srv.Run(":" + cfg.HTTPPort); err != nil {
			log.Fatalf("server: %v", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
}
```

- [ ] **Step 4: Write internal/handler/proxy.go** (copy from existing monolith)
- [ ] **Step 5: Write internal/server/http.go** (core gateway file)
- [ ] **Step 6: Write Dockerfile and push**

For the full http.go and remaining code, see the detailed sub-tasks below.

- [ ] **Step 5a: http.go — imports and setup**

```go
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/choex2025-ops/choex-gateway/internal/config"
	"github.com/choex2025-ops/choex-gateway/internal/handler"
	agentpb "github.com/choex2025-ops/choex-proto/gen/agent/v1"
	authpb "github.com/choex2025-ops/choex-proto/gen/auth/v1"
	billpb "github.com/choex2025-ops/choex-proto/gen/bill/v1"
	calpb "github.com/choex2025-ops/choex-proto/gen/calendar/v1"
	mempb "github.com/choex2025-ops/choex-proto/gen/memory/v1"
	pwdpb "github.com/choex2025-ops/choex-proto/gen/password/v1"
)

type Gateway struct {
	g                                           *gin.Engine
	cfg                                         *config.Config
	authCli   authpb.AuthServiceClient
	calCli    calpb.CalendarServiceClient
	billCli   billpb.BillServiceClient
	pwdCli    pwdpb.PasswordServiceClient
	memCli    mempb.MemoryServiceClient
	agentCli  agentpb.AgentServiceClient
}

func New(cfg *config.Config) *gin.Engine {
	g := &Gateway{cfg: cfg, g: gin.Default()}
	g.initClients()
	g.setupRoutes()
	return g.g
}

func (g *Gateway) initClients() {
	conn, _ := grpc.NewClient(g.cfg.AuthAddr, grpc.WithTransportCredentials(insecure.NewCredentials())); g.authCli = authpb.NewAuthServiceClient(conn)
	conn, _ = grpc.NewClient(g.cfg.CalendarAddr, grpc.WithTransportCredentials(insecure.NewCredentials())); g.calCli = calpb.NewCalendarServiceClient(conn)
	conn, _ = grpc.NewClient(g.cfg.BillAddr, grpc.WithTransportCredentials(insecure.NewCredentials())); g.billCli = billpb.NewBillServiceClient(conn)
	conn, _ = grpc.NewClient(g.cfg.PasswordAddr, grpc.WithTransportCredentials(insecure.NewCredentials())); g.pwdCli = pwdpb.NewPasswordServiceClient(conn)
	conn, _ = grpc.NewClient(g.cfg.MemoryAddr, grpc.WithTransportCredentials(insecure.NewCredentials())); g.memCli = mempb.NewMemoryServiceClient(conn)
	conn, _ = grpc.NewClient(g.cfg.AgentAddr, grpc.WithTransportCredentials(insecure.NewCredentials())); g.agentCli = agentpb.NewAgentServiceClient(conn)
}

func (g *Gateway) ctxWithUserID(c *gin.Context) context.Context {
	uid := strconv.FormatUint(c.GetUint64("user_id"), 10)
	return metadata.NewOutgoingContext(c.Request.Context(), metadata.Pairs("user-id", uid))
}

func (g *Gateway) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			c.JSON(401, gin.H{"error": "missing authorization header"}); c.Abort(); return
		}
		t, err := jwt.Parse(strings.TrimPrefix(h, "Bearer "), func(t *jwt.Token) (interface{}, error) { return []byte(g.cfg.JWTSecret), nil })
		if err != nil || !t.Valid { c.JSON(401, gin.H{"error": "invalid token"}); c.Abort(); return }
		claims := t.Claims.(jwt.MapClaims)
		c.Set("user_id", uint64(claims["user_id"].(float64)))
		c.Set("email", claims["email"])
		c.Next()
	}
}
```

- [ ] **Step 5b: http.go — route setup**

```go
func (g *Gateway) setupRoutes() {
	g.g.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if c.Request.Method == "OPTIONS" { c.AbortWithStatus(204); return }
		c.Next()
	})
	api := g.g.Group("/api")
	{
		api.POST("/auth/register", g.handleRegister)
		api.POST("/auth/login", g.handleLogin)
		api.GET("/proxy", handler.ProxyHandler)
		protected := api.Group(""); protected.Use(g.authMiddleware())
		{
			protected.POST("/agent/chat", g.handleAgentChatSSE)
			protected.GET("/events", g.handleListEvents)
			protected.POST("/events", g.handleCreateEvent)
			protected.PUT("/events/:id", g.handleUpdateEvent)
			protected.DELETE("/events/:id", g.handleDeleteEvent)
			protected.GET("/bills", g.handleListBills)
			protected.POST("/bills", g.handleCreateBill)
			protected.DELETE("/bills/:id", g.handleDeleteBill)
			protected.GET("/bills/stats", g.handleGetStats)
			protected.GET("/passwords", g.handleListPasswords)
			protected.POST("/passwords", g.handleCreatePassword)
			protected.GET("/passwords/:id", g.handleGetPassword)
			protected.PUT("/passwords/:id", g.handleUpdatePassword)
			protected.DELETE("/passwords/:id", g.handleDeletePassword)
			protected.GET("/memories", g.handleListMemories)
			protected.POST("/memories", g.handleCreateMemory)
			protected.PUT("/memories/:id/activate", g.handleActivateMemory)
			protected.DELETE("/memories/:id", g.handleDeleteMemory)
			protected.GET("/memories/:id/versions", g.handleGetVersions)
			protected.PUT("/memories/:id/versions/:type", g.handleSaveVersion)
			protected.PUT("/memories/:id/restore", g.handleRestoreMemory)
		}
	}
}
```

- [ ] **Step 5c: http.go — auth handlers**

```go
func (g *Gateway) handleRegister(c *gin.Context) {
	var b struct{ Username, Email, Password string }
	if err := c.ShouldBindJSON(&b); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	resp, err := g.authCli.Register(context.Background(), &authpb.RegisterRequest{Username: b.Username, Email: b.Email, Password: b.Password})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(201, resp)
}

func (g *Gateway) handleLogin(c *gin.Context) {
	var b struct{ Email, Password string }
	if err := c.ShouldBindJSON(&b); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	resp, err := g.authCli.Login(context.Background(), &authpb.LoginRequest{Email: b.Email, Password: b.Password})
	if err != nil { c.JSON(401, gin.H{"error": "invalid email or password"}); return }
	c.JSON(200, resp)
}
```

- [ ] **Step 5d: http.go — calendar handlers**

```go
func (g *Gateway) handleListEvents(c *gin.Context) {
	resp, err := g.calCli.ListEvents(g.ctxWithUserID(c), &calpb.ListEventsRequest{UserId: c.GetUint64("user_id")})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, resp.Events)
}

func (g *Gateway) handleCreateEvent(c *gin.Context) {
	var b struct{ Title, Description, Location, StartTime, EndTime, Color string; AllDay bool }
	if err := c.ShouldBindJSON(&b); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	resp, err := g.calCli.CreateEvent(g.ctxWithUserID(c), &calpb.CreateEventRequest{UserId: c.GetUint64("user_id"), Title: b.Title, Description: b.Description, Location: b.Location, StartTime: b.StartTime, EndTime: b.EndTime, AllDay: b.AllDay, Color: b.Color})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(201, gin.H{"id": resp.Id})
}

func (g *Gateway) handleUpdateEvent(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var b struct{ Title, Description, Location, StartTime, EndTime, Color string; AllDay bool }
	if err := c.ShouldBindJSON(&b); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	_, err := g.calCli.UpdateEvent(g.ctxWithUserID(c), &calpb.UpdateEventRequest{Id: id, UserId: c.GetUint64("user_id"), Title: b.Title, Description: b.Description, Location: b.Location, StartTime: b.StartTime, EndTime: b.EndTime, AllDay: b.AllDay, Color: b.Color})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, gin.H{"message": "updated"})
}

func (g *Gateway) handleDeleteEvent(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	_, err := g.calCli.DeleteEvent(g.ctxWithUserID(c), &calpb.DeleteEventRequest{Id: id, UserId: c.GetUint64("user_id")})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, gin.H{"message": "deleted"})
}
```

- [ ] **Step 5e: http.go — bill handlers**

```go
func (g *Gateway) handleListBills(c *gin.Context) {
	resp, err := g.billCli.ListBills(g.ctxWithUserID(c), &billpb.ListBillsRequest{UserId: c.GetUint64("user_id"), Date: c.Query("date")})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, resp.Bills)
}

func (g *Gateway) handleCreateBill(c *gin.Context) {
	var b struct{ Amount float64; Type, Category, Note, BillDate string }
	if err := c.ShouldBindJSON(&b); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	resp, err := g.billCli.CreateBill(g.ctxWithUserID(c), &billpb.CreateBillRequest{UserId: c.GetUint64("user_id"), Amount: b.Amount, Type: b.Type, Category: b.Category, Note: b.Note, BillDate: b.BillDate})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(201, gin.H{"id": resp.Id})
}

func (g *Gateway) handleDeleteBill(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	_, err := g.billCli.DeleteBill(g.ctxWithUserID(c), &billpb.DeleteBillRequest{Id: id, UserId: c.GetUint64("user_id")})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, gin.H{"message": "deleted"})
}

func (g *Gateway) handleGetStats(c *gin.Context) {
	resp, err := g.billCli.GetStats(g.ctxWithUserID(c), &billpb.GetStatsRequest{UserId: c.GetUint64("user_id"), Month: c.Query("month")})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, resp)
}
```

- [ ] **Step 5f: http.go — password handlers**

```go
func (g *Gateway) handleListPasswords(c *gin.Context) {
	resp, err := g.pwdCli.ListPasswords(g.ctxWithUserID(c), &pwdpb.ListPasswordsRequest{UserId: c.GetUint64("user_id")})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, resp.Items)
}

func (g *Gateway) handleCreatePassword(c *gin.Context) {
	var b struct{ Title, Url, Username, Password, Note, Category string }
	if err := c.ShouldBindJSON(&b); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	resp, err := g.pwdCli.CreatePassword(g.ctxWithUserID(c), &pwdpb.CreatePasswordRequest{UserId: c.GetUint64("user_id"), Title: b.Title, Url: b.Url, Username: b.Username, Password: b.Password, Note: b.Note, Category: b.Category})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(201, gin.H{"id": resp.Id})
}

func (g *Gateway) handleGetPassword(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	resp, err := g.pwdCli.GetPassword(g.ctxWithUserID(c), &pwdpb.GetPasswordRequest{Id: id, UserId: c.GetUint64("user_id")})
	if err != nil { c.JSON(404, gin.H{"error": "not found"}); return }
	c.JSON(200, resp.Item)
}

func (g *Gateway) handleUpdatePassword(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var b struct{ Title, Url, Username, Password, Note, Category string }
	if err := c.ShouldBindJSON(&b); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	_, err := g.pwdCli.UpdatePassword(g.ctxWithUserID(c), &pwdpb.UpdatePasswordRequest{Id: id, UserId: c.GetUint64("user_id"), Title: b.Title, Url: b.Url, Username: b.Username, Password: b.Password, Note: b.Note, Category: b.Category})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, gin.H{"message": "updated"})
}

func (g *Gateway) handleDeletePassword(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	_, err := g.pwdCli.DeletePassword(g.ctxWithUserID(c), &pwdpb.DeletePasswordRequest{Id: id, UserId: c.GetUint64("user_id")})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, gin.H{"message": "deleted"})
}
```

- [ ] **Step 5g: http.go — memory handlers**

```go
func (g *Gateway) handleListMemories(c *gin.Context) {
	resp, err := g.memCli.ListMemories(g.ctxWithUserID(c), &mempb.ListMemoriesRequest{UserId: c.GetUint64("user_id")})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, resp.Memories)
}

func (g *Gateway) handleCreateMemory(c *gin.Context) {
	var b struct{ Name, Icon string }
	if err := c.ShouldBindJSON(&b); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	resp, err := g.memCli.CreateMemory(g.ctxWithUserID(c), &mempb.CreateMemoryRequest{UserId: c.GetUint64("user_id"), Name: b.Name, Icon: b.Icon})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(201, gin.H{"id": resp.Id})
}

func (g *Gateway) handleActivateMemory(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	_, err := g.memCli.ActivateMemory(g.ctxWithUserID(c), &mempb.ActivateMemoryRequest{Id: id, UserId: c.GetUint64("user_id")})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, gin.H{"message": "activated"})
}

func (g *Gateway) handleDeleteMemory(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	_, err := g.memCli.DeleteMemory(g.ctxWithUserID(c), &mempb.DeleteMemoryRequest{Id: id, UserId: c.GetUint64("user_id")})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, gin.H{"message": "deleted"})
}

func (g *Gateway) handleGetVersions(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	resp, err := g.memCli.GetVersions(g.ctxWithUserID(c), &mempb.GetVersionsRequest{MemoryId: id})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, resp)
}

func (g *Gateway) handleSaveVersion(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	t := c.Param("type")
	var b struct{ Content string }
	if err := c.ShouldBindJSON(&b); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	_, err := g.memCli.SaveVersion(g.ctxWithUserID(c), &mempb.SaveVersionRequest{MemoryId: id, Type: t, Content: b.Content})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, gin.H{"message": "saved"})
}

func (g *Gateway) handleRestoreMemory(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	_, err := g.memCli.RestoreMemory(g.ctxWithUserID(c), &mempb.RestoreMemoryRequest{MemoryId: id})
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, gin.H{"message": "restored"})
}
```

- [ ] **Step 5h: http.go — agent SSE handler**

```go
func (g *Gateway) handleAgentChatSSE(c *gin.Context) {
	var b struct {
		Message string                   `json:"message"`
		History []agentpb.ChatMessage    `json:"history"`
	}
	if err := c.ShouldBindJSON(&b); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(200)

	stream, err := g.agentCli.Chat(g.ctxWithUserID(c), &agentpb.ChatRequest{
		UserId: c.GetUint64("user_id"), Message: b.Message, History: b.History,
	})
	if err != nil { fmt.Fprintf(c.Writer, "data: {\"error\":\"%s\"}\n\n", err.Error()); c.Writer.Flush(); return }

	for {
		resp, err := stream.Recv()
		if err == io.EOF { break }
		if err != nil {
			fmt.Fprintf(c.Writer, "data: {\"error\":\"%s\"}\n\n", err.Error())
			c.Writer.Flush(); break
		}
		if resp.Error != "" {
			fmt.Fprintf(c.Writer, "data: {\"error\":\"%s\"}\n\n", resp.Error)
		} else if resp.Done {
			fmt.Fprint(c.Writer, "data: {\"done\":true}\n\n")
		} else {
			data, _ := json.Marshal(map[string]string{"content": resp.Content})
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		}
		c.Writer.Flush()
	}
}
```

- [ ] **Step 6: Write Dockerfile and push**

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /gateway ./cmd/server/

FROM scratch
COPY --from=builder /gateway /gateway
EXPOSE 8080
ENTRYPOINT ["/gateway"]
```

```bash
cd ~/ClaudeProjects/choex-gateway
git add -A && git commit -m "feat: gateway with gRPC clients, JWT, SSE streaming"
gh repo create choex2025-ops/choex-gateway --public --source=. --push
```

---

### Task 3: choex-auth — Auth Service (gRPC :9001)

- [ ] **Step 1: Create repo and setup**

```bash
mkdir ~/ClaudeProjects/choex-auth && cd ~/ClaudeProjects/choex-auth
git init && go mod init github.com/choex2025-ops/choex-auth
git submodule add https://github.com/choex2025-ops/choex-proto.git proto
echo 'replace github.com/choex2025-ops/choex-proto => ./proto' >> go.mod
go get google.golang.org/grpc gorm.io/gorm gorm.io/driver/mysql golang.org/x/crypto github.com/golang-jwt/jwt/v5
go mod tidy
```

- [ ] **Step 2: Write config.go**

```go
package config
import "os"

type Config struct { GRPCPort, DBHost, DBPort, DBUser, DBPassword, DBName, JWTSecret string }

func Load() *Config {
	return &Config{
		GRPCPort:   getEnv("GRPC_PORT", "9001"),
		DBHost:     getEnv("DB_HOST", "localhost"), DBPort: getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"), DBPassword: getEnv("DB_PASSWORD", "choex2025"),
		DBName:     getEnv("DB_NAME", "choex_auth"), JWTSecret: getEnv("JWT_SECRET", "dev-secret"),
	}
}
func (c *Config) DSN() string { return c.DBUser + ":" + c.DBPassword + "@tcp(" + c.DBHost + ":" + c.DBPort + ")/" + c.DBName + "?charset=utf8mb4&parseTime=True" }
func getEnv(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
```

- [ ] **Step 3: Write database.go**

```go
package database
import ("log"; "gorm.io/driver/mysql"; "gorm.io/gorm"; "github.com/choex2025-ops/choex-auth/internal/config"; "github.com/choex2025-ops/choex-auth/internal/model")

var DB *gorm.DB

func Init(cfg *config.Config) {
	var err error
	DB, err = gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
	if err != nil { log.Fatalf("failed to connect database: %v", err) }
	DB.AutoMigrate(&model.User{})
}
```

- [ ] **Step 4: Write model/user.go**

```go
package model
import "time"

type User struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string    `gorm:"size:100;not null" json:"username"`
	Email        string    `gorm:"size:255;uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
func (User) TableName() string { return "users" }
```

- [ ] **Step 5: Write service/auth.go**

```go
package service
import (
	"errors"; "time"
	"github.com/golang-jwt/jwt/v5"; "golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"; "github.com/choex2025-ops/choex-auth/internal/database"; "github.com/choex2025-ops/choex-auth/internal/config"; "github.com/choex2025-ops/choex-auth/internal/model"
)

type AuthService struct { cfg *config.Config }

func New(cfg *config.Config) *AuthService { return &AuthService{cfg: cfg} }

func (s *AuthService) HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), 10)
	return string(b), err
}
func (s *AuthService) CheckPassword(pw, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}
func (s *AuthService) GenerateToken(userID uint64, email string) (string, error) {
	c := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id": userID, "email": email, "exp": time.Now().Add(24 * time.Hour).Unix()})
	return c.SignedString([]byte(s.cfg.JWTSecret))
}
func (s *AuthService) Register(username, email, password string) (string, uint64, string, string, error) {
	var existing model.User
	if err := database.DB.Where("email = ?", email).First(&existing).Error; err == nil { return "", 0, "", "", errors.New("email already registered") }
	hash, err := s.HashPassword(password); if err != nil { return "", 0, "", "", err }
	u := model.User{Username: username, Email: email, PasswordHash: hash}
	if err := database.DB.Create(&u).Error; err != nil { return "", 0, "", "", err }
	token, err := s.GenerateToken(u.ID, u.Email); if err != nil { return "", 0, "", "", err }
	return token, u.ID, u.Username, u.Email, nil
}
func (s *AuthService) Login(email, password string) (string, uint64, string, string, error) {
	var u model.User
	if err := database.DB.Where("email = ?", email).First(&u).Error; err != nil { return "", 0, "", "", errors.New("invalid credentials") }
	if !s.CheckPassword(password, u.PasswordHash) { return "", 0, "", "", errors.New("invalid credentials") }
	token, err := s.GenerateToken(u.ID, u.Email); if err != nil { return "", 0, "", "", err }
	return token, u.ID, u.Username, u.Email, nil
}
func (s *AuthService) ValidateToken(tokenStr string) (bool, uint64, string) {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) { return []byte(s.cfg.JWTSecret), nil })
	if err != nil || !t.Valid { return false, 0, "" }
	c := t.Claims.(jwt.MapClaims)
	return true, uint64(c["user_id"].(float64)), c["email"].(string)
}
```

- [ ] **Step 6: Write server/grpc.go**

```go
package server
import (
	"context"; "log"; "net"; "strings"
	"google.golang.org/grpc"; "google.golang.org/grpc/metadata"
	"github.com/choex2025-ops/choex-auth/internal/config"; "github.com/choex2025-ops/choex-auth/internal/database"; "github.com/choex2025-ops/choex-auth/internal/service"
	authpb "github.com/choex2025-ops/choex-proto/gen/auth/v1"
)

type AuthServer struct {
	authpb.UnimplementedAuthServiceServer
	svc *service.AuthService
}

func Run(cfg *config.Config) {
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil { log.Fatalf("failed to listen: %v", err) }
	s := grpc.NewServer()
	authpb.RegisterAuthServiceServer(s, &AuthServer{svc: service.New(cfg)})
	log.Printf("Auth service listening on :%s", cfg.GRPCPort)
	if err := s.Serve(lis); err != nil { log.Fatalf("failed to serve: %v", err) }
}

func (s *AuthServer) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.AuthResponse, error) {
	token, uid, uname, email, err := s.svc.Register(req.Username, req.Email, req.Password)
	if err != nil {
		if strings.Contains(err.Error(), "already") { return nil, err }
		return nil, err
	}
	return &authpb.AuthResponse{Token: token, UserId: uid, Username: uname, Email: email}, nil
}
func (s *AuthServer) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.AuthResponse, error) {
	token, uid, uname, email, err := s.svc.Login(req.Email, req.Password)
	if err != nil { return nil, err }
	return &authpb.AuthResponse{Token: token, UserId: uid, Username: uname, Email: email}, nil
}
func (s *AuthServer) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	valid, uid, email := s.svc.ValidateToken(req.Token)
	return &authpb.ValidateTokenResponse{Valid: valid, UserId: uid, Email: email}, nil
}
```

- [ ] **Step 7: Write main.go**

```go
package main
import ("github.com/choex2025-ops/choex-auth/internal/config"; "github.com/choex2025-ops/choex-auth/internal/database"; "github.com/choex2025-ops/choex-auth/internal/server")

func main() {
	cfg := config.Load()
	database.Init(cfg)
	server.Run(cfg)
}
```

- [ ] **Step 8: Write Dockerfile and push**

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /auth ./cmd/server/
FROM scratch
COPY --from=builder /auth /auth
EXPOSE 9001
ENTRYPOINT ["/auth"]
```

```bash
cd ~/ClaudeProjects/choex-auth
git add -A && git commit -m "feat: auth service with register/login/token validation"
gh repo create choex2025-ops/choex-auth --public --source=. --push
```

---

## Phase 2: Core Business Services (Tasks 4-6)

### Task 4: choex-calendar — Calendar Service (gRPC :9002)

**Setup:** Same pattern as Task 3 (mkdir, git init, go mod init `choex-calendar`, git submodule add proto, `go get grpc gorm mysql`, replace directive in go.mod).

**Config:** `GRPC_PORT=9002`, `DB_HOST/PORT/USER/PASSWORD`, `DB_NAME=choex_calendar`, DSN() method.

**Database:** Open gorm, AutoMigrate `model.Event`.

**Model (model/event.go):**
```go
package model
import "time"
type Event struct {
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID uint64 `gorm:"index;not null" json:"user_id"`
	Title string `gorm:"size:255" json:"title"`
	Description string `gorm:"type:text" json:"description"`
	Location string `gorm:"size:255" json:"location"`
	StartTime string `gorm:"size:50" json:"start_time"`
	EndTime string `gorm:"size:50" json:"end_time"`
	AllDay bool `json:"all_day"`
	Color string `gorm:"size:20" json:"color"`
	CreatedAt time.Time `json:"created_at"`; UpdatedAt time.Time `json:"updated_at"`
}
func (Event) TableName() string { return "events" }
```

**Service (service/calendar.go):**
```go
package service
import ("github.com/choex2025-ops/choex-calendar/internal/database"; "github.com/choex2025-ops/choex-calendar/internal/model")

func ListEvents(userID uint64) ([]model.Event, error) {
	var events []model.Event
	err := database.DB.Where("user_id = ?", userID).Order("start_time DESC").Find(&events).Error
	return events, err
}
func CreateEvent(e *model.Event) error { return database.DB.Create(e).Error }
func UpdateEvent(id, userID uint64, updates map[string]any) error {
	return database.DB.Model(&model.Event{}).Where("id = ? AND user_id = ?", id, userID).Updates(updates).Error
}
func DeleteEvent(id, userID uint64) error {
	return database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Event{}).Error
}
```

**Server (server/grpc.go):** implement calpb.CalendarServiceServer. Extract user_id from gRPC metadata (actually for business services, user_id comes from request). Run() starts gRPC listener on port.

**Main, Dockerfile:** Same pattern as auth, port 9002.

**Push:** `gh repo create choex2025-ops/choex-calendar --public --source=. --push`

---

### Task 5: choex-bill — Bill Service (gRPC :9003)

**Setup:** Same pattern. Port 9003, DB_NAME=choex_bill.

**Model (model/bill.go):**
```go
package model
import "time"
type Bill struct {
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID uint64 `gorm:"index;not null" json:"user_id"`
	Amount float64 `gorm:"type:decimal(10,2)" json:"amount"`
	Type string `gorm:"size:20" json:"type"` // "income" or "expense"
	Category string `gorm:"size:50" json:"category"`
	Note string `gorm:"size:255" json:"note"`
	BillDate string `gorm:"size:10;index" json:"bill_date"` // YYYY-MM-DD
	CreatedAt time.Time `json:"created_at"`; UpdatedAt time.Time `json:"updated_at"`
}
func (Bill) TableName() string { return "bills" }
```

**Service (service/bill.go):** ListBills with optional date filter, CreateBill, DeleteBill, GetStats with `WHERE bill_date LIKE 'month%'` and per-category aggregation.

**Server:** GRPC implementation with all four RPCs.

**Dockerfile, main, push.**

---

### Task 6: choex-password — Password Service (gRPC :9004)

**Setup:** Port 9004, DB_NAME=choex_password, extra env var `ENCRYPTION_KEY`.

**Model (model/password.go):**
```go
package model
import "time"
type Password struct {
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID uint64 `gorm:"index;not null" json:"user_id"`
	Title string `gorm:"size:255" json:"title"`
	URL string `gorm:"size:500" json:"url"`
	Username string `gorm:"size:255" json:"username"`
	EncryptedPassword string `gorm:"type:text" json:"-"`
	Note string `gorm:"type:text" json:"note"`
	Category string `gorm:"size:50" json:"category"`
	CreatedAt time.Time `json:"created_at"`; UpdatedAt time.Time `json:"updated_at"`
}
func (Password) TableName() string { return "passwords" }
```

**Service (service/password.go):** AES-256-GCM encrypt/decrypt with Base64 encoding. Key padded to 32 bytes. Encrypt/decrypt methods (same as existing monolith). List, CreateRaw (using map + `database.DB.Table("passwords").Create(data)`), Get (with decrypt), Update, Delete.

**Server/Dockerfile/main/push.**

---

## Phase 3: Agent Services (Tasks 7-8)

### Task 7: choex-memory — Memory Service (gRPC :9005)

**Setup:** Port 9005, DB_NAME=choex_memory. Same structure.

**Model (model/memory.go):**
```go
package model
import "time"
type AgentMemory struct {
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID uint64 `gorm:"index;not null" json:"user_id"`
	Name string `gorm:"size:100" json:"name"`; Icon string `gorm:"size:10" json:"icon"`
	IsActive bool `gorm:"default:false" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`; UpdatedAt time.Time `json:"updated_at"`
}
func (AgentMemory) TableName() string { return "agent_memories" }
type MemoryVersion struct {
	ID uint64 `gorm:"primaryKey;autoIncrement"`
	MemoryID uint64 `gorm:"index;not null"`
	VersionType string `gorm:"size:20;not null"` // "current", "backup", "custom"
	Content string `gorm:"type:text"`
	CreatedAt time.Time
}
func (MemoryVersion) TableName() string { return "memory_versions" }
```

**Service (service/memory.go):** List, Create (transaction: create memory + empty "current" version), Activate (transaction: deactivate all → activate target), Delete (transaction: delete versions → delete memory), GetVersions (map[string]string), SaveVersion (saving "current" auto-backups old content to "backup"), Restore (copy "backup" → "current").

**Server/Dockerfile/main/push.**

---

### Task 8: choex-agent — Agent Service (gRPC :9006 + Tool Calling)

**Setup:** Port 9006, DB_NAME=choex_agent. Extra env var `DEEPSEEK_API_KEY` (required), `DEEPSEEK_BASE_URL` (default `https://api.deepseek.com/`).

**Dependencies:** `go get github.com/openai/openai-go` (OpenAI-compatible SDK).

**Model (model/chat.go):**
```go
package model
import "time"
type ChatHistory struct {
	ID uint64 `gorm:"primaryKey;autoIncrement"`
	UserID uint64 `gorm:"index;not null"`
	Role string `gorm:"size:20"`; Content string `gorm:"type:text"`
	CreatedAt time.Time
}
func (ChatHistory) TableName() string { return "chat_history" }
```

**Service (service/agent.go):**
- Creates gRPC clients to calendar, bill, password, memory services
- `Chat(userID, message, history) (<-chan StreamChunk, error)`
- System prompt: "你是 ChoexManager 个人生活管家..."
- Tools defined as OpenAI function definitions (query_calendar, manage_bills, search_password, get_memories)
- When LLM returns function_call → execute via gRPC → feed result back → stream final response
- Uses openai-go SDK with `client := openai.NewClient(deepseekURL, deepseekKey)`

**Server (server/grpc.go):** Implement `AgentServiceServer.Chat` as server-side streaming. Each token from service channel sent via `stream.Send()`.

**Dockerfile/main/push.**

---

## Phase 4: Integration (Tasks 9-10)

### Task 9: docker-compose orchestration

- [ ] **Step 1: Write docker-compose.yml to `~/ClaudeProjects/docker-compose.yml`**

```yaml
services:
  mysql:
    image: mysql:9.6
    ports: ["3306:3306"]
    environment:
      MYSQL_ROOT_PASSWORD: choex2025
    volumes:
      - mysql_data:/var/lib/mysql
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s; timeout: 3s; retries: 10

  redis:
    image: redis:8.8
    ports: ["6379:6379"]
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]; interval: 5s; timeout: 3s; retries: 5

  auth:
    build: ./choex-auth
    ports: ["9001:9001"]
    environment:
      GRPC_PORT: "9001"; DB_HOST: mysql; DB_PORT: "3306"; DB_USER: root; DB_PASSWORD: choex2025; DB_NAME: choex_auth; JWT_SECRET: dev-secret
    depends_on: { mysql: { condition: service_healthy } }

  calendar:
    build: ./choex-calendar; ports: ["9002:9002"]
    environment: { GRPC_PORT: "9002", DB_HOST: mysql, DB_PORT: "3306", DB_USER: root, DB_PASSWORD: choex2025, DB_NAME: choex_calendar }
    depends_on: { mysql: { condition: service_healthy } }

  bill:
    build: ./choex-bill; ports: ["9003:9003"]
    environment: { GRPC_PORT: "9003", DB_HOST: mysql, DB_PORT: "3306", DB_USER: root, DB_PASSWORD: choex2025, DB_NAME: choex_bill }
    depends_on: { mysql: { condition: service_healthy } }

  password:
    build: ./choex-password; ports: ["9004:9004"]
    environment: { GRPC_PORT: "9004", DB_HOST: mysql, DB_PORT: "3306", DB_USER: root, DB_PASSWORD: choex2025, DB_NAME: choex_password, ENCRYPTION_KEY: "choex2025-32byte-secret-key!!!" }
    depends_on: { mysql: { condition: service_healthy } }

  memory:
    build: ./choex-memory; ports: ["9005:9005"]
    environment: { GRPC_PORT: "9005", DB_HOST: mysql, DB_PORT: "3306", DB_USER: root, DB_PASSWORD: choex2025, DB_NAME: choex_memory }
    depends_on: { mysql: { condition: service_healthy } }

  agent:
    build: ./choex-agent; ports: ["9006:9006"]
    environment:
      GRPC_PORT: "9006"; DB_HOST: mysql; DB_PORT: "3306"; DB_USER: root; DB_PASSWORD: choex2025; DB_NAME: choex_agent
      DEEPSEEK_API_KEY: "${DEEPSEEK_API_KEY}"; CALENDAR_ADDR: calendar:9002; BILL_ADDR: bill:9003
      PASSWORD_ADDR: password:9004; MEMORY_ADDR: memory:9005
    depends_on: { mysql: { condition: service_healthy }, calendar: { condition: service_started }, bill: { condition: service_started }, password: { condition: service_started }, memory: { condition: service_started } }

  gateway:
    build: ./choex-gateway; ports: ["8080:8080"]
    environment:
      HTTP_PORT: "8080"; JWT_SECRET: dev-secret
      AUTH_ADDR: auth:9001; CALENDAR_ADDR: calendar:9002; BILL_ADDR: bill:9003
      PASSWORD_ADDR: password:9004; MEMORY_ADDR: memory:9005; AGENT_ADDR: agent:9006
    depends_on: [auth, calendar, bill, password, memory, agent]

volumes:
  mysql_data:
```

- [ ] **Step 2: Write `~/ClaudeProjects/init.sql`**

```sql
CREATE DATABASE IF NOT EXISTS choex_auth;
CREATE DATABASE IF NOT EXISTS choex_calendar;
CREATE DATABASE IF NOT EXISTS choex_bill;
CREATE DATABASE IF NOT EXISTS choex_password;
CREATE DATABASE IF NOT EXISTS choex_memory;
CREATE DATABASE IF NOT EXISTS choex_agent;
```

---

### Task 10: End-to-end Verification

- [ ] **Step 1: Start services**

```bash
cd ~/ClaudeProjects && docker-compose up -d && sleep 15
```

- [ ] **Step 2: Register and login**

```bash
curl -s -X POST localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test","email":"test@test.com","password":"pass123"}' | jq .
# Expected: {"token":"eyJ...","user_id":1,"username":"test","email":"test@test.com"}

TOKEN=$(curl -s -X POST localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"pass123"}' | jq -r '.token')
echo "Token: ${TOKEN:0:20}..."
```

- [ ] **Step 3: Test calendar**

```bash
curl -s -H "Authorization: Bearer $TOKEN" -X POST localhost:8080/api/events \
  -H "Content-Type: application/json" \
  -d '{"title":"Meeting","start_time":"2026-06-12T10:00:00Z","end_time":"2026-06-12T11:00:00Z","color":"#58a6ff"}'
# Expected: {"id":1}

curl -s -H "Authorization: Bearer $TOKEN" localhost:8080/api/events | jq '.[].title'
# Expected: "Meeting"
```

- [ ] **Step 4: Test bills**

```bash
curl -s -H "Authorization: Bearer $TOKEN" -X POST localhost:8080/api/bills \
  -H "Content-Type: application/json" \
  -d '{"amount":30,"type":"expense","category":"餐饮","bill_date":"2026-06-12"}'
# Expected: {"id":1}

curl -s -H "Authorization: Bearer $TOKEN" "localhost:8080/api/bills/stats?month=2026-06" | jq .
# Expected: {"total_income":0,"total_expense":30,"by_category":{"餐饮":30}}
```

- [ ] **Step 5: Test passwords**

```bash
curl -s -H "Authorization: Bearer $TOKEN" -X POST localhost:8080/api/passwords \
  -H "Content-Type: application/json" \
  -d '{"title":"GitHub","url":"https://github.com","username":"myuser","password":"secret123"}'
# Expected: {"id":1}

curl -s -H "Authorization: Bearer $TOKEN" localhost:8080/api/passwords/1 | jq '.password'
# Expected: "secret123"
```

- [ ] **Step 6: Test memories**

```bash
curl -s -H "Authorization: Bearer $TOKEN" -X POST localhost:8080/api/memories \
  -H "Content-Type: application/json" \
  -d '{"name":"default","icon":"🧠"}'
# Expected: {"id":1}

curl -s -H "Authorization: Bearer $TOKEN" -X PUT localhost:8080/api/memories/1/activate
# Expected: {"message":"activated"}
```

- [ ] **Step 7: Test agent chat (SSE)**

```bash
curl -s -N -H "Authorization: Bearer $TOKEN" -X POST localhost:8080/api/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"你好","history":[]}'
# Expected: SSE stream with {"content":"..."} chunks, ending with {"done":true}
```

Expected: ALL steps return success responses. If any fail, investigate logs with `docker-compose logs <service>`.

- [ ] **Step 8: Cleanup**

```bash
cd ~/ClaudeProjects && docker-compose down -v
```
