package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseModelRoutesFromProject(t *testing.T) {
	projectDir := setupModelRoutesProject(t)

	routes, err := parseModelRoutesFromProject(filepath.Join(projectDir, "router", "router.go"), filepath.Join(projectDir, "model"))
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 4 {
		t.Fatalf("expected 4 routes, got %d: %#v", len(routes), routes)
	}

	assertModelRoute(t, routes[0], modelRoute{
		Model:  "*auth.Login",
		Req:    "*auth.Login",
		Rsp:    "*auth.LoginRsp",
		Source: filepath.Join("model", "auth", "login.go"),
		Scope:  "public",
		Path:   "auth/login",
		Method: "GET",
		Phase:  "List",
	})
	assertModelRoute(t, routes[2], modelRoute{
		Model:  "*conversation.Message",
		Req:    "*conversation.Message",
		Rsp:    "*conversation.Message",
		Source: filepath.Join("model", "conversation", "message.go"),
		Scope:  "auth",
		Path:   "conversations/:conv/messages/:id",
		Method: "GET",
		Phase:  "Get",
		Param:  "id",
	})
}

func TestFilterModelRoutesMatchesModelSourcePathAndPhase(t *testing.T) {
	projectDir := setupModelRoutesProject(t)
	routes, err := parseModelRoutesFromProject(filepath.Join(projectDir, "router", "router.go"), filepath.Join(projectDir, "model"))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		filter string
		want   []string
	}{
		{
			name:   "model",
			filter: "message",
			want: []string{
				"conversations/:conv/messages",
				"conversations/:conv/messages/:id",
			},
		},
		{
			name:   "source",
			filter: "auth/login.go",
			want:   []string{"auth/login"},
		},
		{
			name:   "path",
			filter: "config/files",
			want:   []string{"config/files/encrypt"},
		},
		{
			name:   "phase",
			filter: "get",
			want:   []string{"conversations/:conv/messages/:id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterModelRoutes(routes, tt.filter)
			got := make([]string, 0, len(filtered))
			for _, route := range filtered {
				got = append(got, route.Path)
			}
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Fatalf("filter %q got %v, want %v", tt.filter, got, tt.want)
			}
		})
	}
}

func TestPrintModelRoutesGroupsByModelFile(t *testing.T) {
	routes := modelRoutesForTest(t)
	var out bytes.Buffer
	printModelRoutes(&out, routes, modelRoutesPrintOptions{})

	got := out.String()
	requireContainsAll(
		t, got,
		"▶ Model Routes",
		"→ models: 3, routes: 4, public: 1, auth: 3",
		"auth/",
		"└─ login.go  *auth.Login [public]",
		"GET    /auth/login",
		"conversation/",
		"└─ message.go  *conversation.Message [auth]",
		"GET    /conversations/:conv/messages/:id",
		"file/",
		"└─ encrypt.go  *file.Encrypt [auth]",
	)
	requireContainsNone(
		t, got,
		"model/",
		"└─ *auth.Login [public]",
		"└─ *conversation.Message [auth]",
		"└─ *file.Encrypt [auth]",
		" /auth/login List",
		" /conversations/:conv/messages List",
		" /conversations/:conv/messages/:id Get",
		" /config/files/encrypt Create",
		"req:",
		"rsp:",
		"param:",
	)
}

func TestPrintModelRoutesExpandsFileWithMultipleModels(t *testing.T) {
	routes := []modelRoute{
		{
			Model:  "*namespace.KV",
			Source: filepath.Join("model", "config", "namespace", "mixed.go"),
			Scope:  "auth",
			Path:   "config/namespaces/:ns/kv",
			Method: "GET",
			Phase:  "List",
		},
		{
			Model:  "*namespace.KV",
			Source: filepath.Join("model", "config", "namespace", "mixed.go"),
			Scope:  "auth",
			Path:   "config/namespaces/:ns/kv/:kv",
			Method: "GET",
			Phase:  "Get",
			Param:  "kv",
		},
		{
			Model:  "*namespace.Tag",
			Source: filepath.Join("model", "config", "namespace", "mixed.go"),
			Scope:  "public",
			Path:   "config/namespaces/:ns/tags",
			Method: "POST",
			Phase:  "Create",
		},
	}

	var out bytes.Buffer
	printModelRoutes(&out, routes, modelRoutesPrintOptions{})

	requireContainsAll(
		t, out.String(),
		"└─ mixed.go  (2 models)",
		"├─ *namespace.KV [auth]",
		"└─ *namespace.Tag [public]",
		"GET    /config/namespaces/:ns/kv",
		"GET    /config/namespaces/:ns/kv/:kv",
		"POST   /config/namespaces/:ns/tags",
	)
}

func TestPrintModelRoutesDetailIncludesReqRspAndParam(t *testing.T) {
	routes := modelRoutesForTest(t)
	var out bytes.Buffer
	printModelRoutes(&out, filterModelRoutes(routes, "messages/:id"), modelRoutesPrintOptions{
		Detail: true,
		Filter: "messages/:id",
	})

	requireContainsAll(
		t, out.String(),
		"→ filter: messages/:id",
		"GET    /conversations/:conv/messages/:id",
		"phase: Get",
		"param: id",
		"req:   *conversation.Message",
		"rsp:   *conversation.Message",
		"source: model/conversation/message.go",
	)
}

func TestPrintRouterRoutesGroupsByModel(t *testing.T) {
	routes := modelRoutesForTest(t)
	var out bytes.Buffer
	printRouterRoutes(&out, routes, modelRoutesPrintOptions{})

	got := out.String()
	requireContainsAll(
		t, got,
		"▶ Router Routes",
		"→ models: 3, routes: 4, public: 1, auth: 3",
		"*conversation.Message [auth]",
		"GET    /conversations/:conv/messages/:id",
		"*auth.Login [public]",
		"GET    /auth/login",
	)
	requireContainsNone(
		t, got,
		"router.Auth()",
		"router.Pub()",
		" /auth/login List",
		" /conversations/:conv/messages List",
		" /conversations/:conv/messages/:id Get",
		" /config/files/encrypt Create",
		"req:",
		"rsp:",
		"source:",
	)
}

func TestRunModelRoutesDefaultsToRouterView(t *testing.T) {
	projectDir := setupModelRoutesProject(t)
	t.Chdir(projectDir)

	var out bytes.Buffer
	if err := runModelRoutes(&out, "", &modelRoutesCommandOptions{}); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if !strings.Contains(got, "▶ Router Routes") {
		t.Fatalf("default routes output should use router view:\n%s", got)
	}
	if strings.Contains(got, "▶ Model Routes") {
		t.Fatalf("default routes output should not use model view:\n%s", got)
	}
}

func TestRunModelRoutesCanSelectModelView(t *testing.T) {
	projectDir := setupModelRoutesProject(t)
	t.Chdir(projectDir)

	var out bytes.Buffer
	if err := runModelRoutes(&out, "", &modelRoutesCommandOptions{model: true}); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if !strings.Contains(got, "▶ Model Routes") {
		t.Fatalf("model routes output should use model view:\n%s", got)
	}
	if strings.Contains(got, "▶ Router Routes") {
		t.Fatalf("model routes output should not use router view:\n%s", got)
	}
}

func TestPrintRouterRoutesDetailIncludesReqRspParamAndSource(t *testing.T) {
	routes := modelRoutesForTest(t)
	var out bytes.Buffer
	printRouterRoutes(&out, filterModelRoutes(routes, "messages/:id"), modelRoutesPrintOptions{
		Detail: true,
		Filter: "messages/:id",
	})

	requireContainsAll(
		t, out.String(),
		"▶ Router Routes",
		"→ filter: messages/:id",
		"*conversation.Message [auth]",
		"GET    /conversations/:conv/messages/:id",
		"phase: Get",
		"scope: auth",
		"param: id",
		"req:   *conversation.Message",
		"rsp:   *conversation.Message",
		"source: model/conversation/message.go",
	)
}

func TestFilterRoutesByScopeKeepsOnlySelectedScope(t *testing.T) {
	routes := modelRoutesForTest(t)
	authRoutes := filterRoutesByScope(routes, "auth")
	if len(authRoutes) != 3 {
		t.Fatalf("auth filter got %d routes, want 3: %#v", len(authRoutes), authRoutes)
	}
	for _, route := range authRoutes {
		if route.Scope != "auth" {
			t.Fatalf("auth filter kept non-auth route: %#v", route)
		}
	}

	publicRoutes := filterRoutesByScope(routes, "public")
	if len(publicRoutes) != 1 {
		t.Fatalf("public filter got %d routes, want 1: %#v", len(publicRoutes), publicRoutes)
	}
	if publicRoutes[0].Path != "auth/login" {
		t.Fatalf("public filter kept unexpected route: %#v", publicRoutes[0])
	}
}

func TestNormalizeRouteScopeAcceptsAliasesAndRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: ""},
		{input: "auth", want: "auth"},
		{input: "pub", want: "public"},
		{input: "public", want: "public"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := normalizeRouteScope(tt.input)
			if err != nil {
				t.Fatalf("normalizeRouteScope(%q) failed: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("normalizeRouteScope(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}

	if _, err := normalizeRouteScope("admin"); err == nil {
		t.Fatal("normalizeRouteScope(admin) succeeded unexpectedly")
	}
}

func modelRoutesForTest(t *testing.T) []modelRoute {
	t.Helper()

	projectDir := setupModelRoutesProject(t)
	routes, err := parseModelRoutesFromProject(filepath.Join(projectDir, "router", "router.go"), filepath.Join(projectDir, "model"))
	if err != nil {
		t.Fatal(err)
	}
	return routes
}

func requireContainsAll(t *testing.T, got string, wants ...string) {
	t.Helper()

	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func requireContainsNone(t *testing.T, got string, values ...string) {
	t.Helper()

	for _, value := range values {
		if strings.Contains(got, value) {
			t.Fatalf("output should omit %q:\n%s", value, got)
		}
	}
}

func setupModelRoutesProject(t *testing.T) string {
	t.Helper()

	projectDir := t.TempDir()
	writeModelRoutesTestFile(t, filepath.Join(projectDir, "router", "router.go"), `package router

import (
	"demo/model/auth"
	"demo/model/config/file"
	"demo/model/conversation"

	"github.com/forbearing/gst/router"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
)

func Init() error {
	router.Register[*auth.Login, *auth.Login, *auth.LoginRsp](router.Pub(), "auth/login", &types.ControllerConfig[*auth.Login]{}, consts.List)
	router.Register[*conversation.Message, *conversation.Message, *conversation.Message](router.Auth(), "conversations/:conv/messages", &types.ControllerConfig[*conversation.Message]{}, consts.List)
	router.Register[*conversation.Message, *conversation.Message, *conversation.Message](router.Auth(), "conversations/:conv/messages/:id", &types.ControllerConfig[*conversation.Message]{ParamName: "id"}, consts.Get)
	router.Register[*file.Encrypt, *file.EncryptReq, *file.EncryptRsp](router.Auth(), "config/files/encrypt", &types.ControllerConfig[*file.Encrypt]{}, consts.Create)
	return nil
}
`)
	writeModelRoutesTestFile(t, filepath.Join(projectDir, "model", "auth", "login.go"), `package auth

type Login struct{}
type LoginRsp struct{}
`)
	writeModelRoutesTestFile(t, filepath.Join(projectDir, "model", "conversation", "message.go"), `package conversation

type Message struct{}
`)
	writeModelRoutesTestFile(t, filepath.Join(projectDir, "model", "config", "file", "encrypt.go"), `package file

type Encrypt struct{}
type EncryptReq struct{}
type EncryptRsp struct{}
`)
	return projectDir
}

func writeModelRoutesTestFile(t *testing.T, filename, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertModelRoute(t *testing.T, got modelRoute, want modelRoute) {
	t.Helper()

	if got != want {
		t.Fatalf("model route mismatch:\ngot:  %#v\nwant: %#v", got, want)
	}
}
