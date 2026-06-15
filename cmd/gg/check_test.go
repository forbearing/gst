package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckRunRejectsExplicitActionTypesWithoutReqRspSuffix(t *testing.T) {
	projectDir := t.TempDir()
	writeCheckTestFile(t, filepath.Join(projectDir, "model", "search.go"), `package model

import (
	"github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

type Search struct {
	model.Empty
}

type SearchRequest struct {
	Query string `+"`json:\"query\"`"+`
}

type SearchResponse struct {
	Items []string `+"`json:\"items\"`"+`
}

func (Search) Design() {
	dsl.Route("search", func() {
		dsl.List(func() {
			dsl.Enabled(true)
			dsl.Service(true)
			dsl.Payload[*SearchRequest]()
			dsl.Result[*SearchResponse]()
		})
	})
}
`)

	out, err := runCheckCommandForTest(t, projectDir)
	if err == nil {
		t.Fatalf("expected checkRun to reject action types without Req/Rsp suffixes, output:\n%s", out)
	}
	if !strings.Contains(out, "Payload type 'SearchRequest' should end with Req") {
		t.Fatalf("expected Payload suffix violation, output:\n%s", out)
	}
	if !strings.Contains(out, "Result type 'SearchResponse' should end with Rsp") {
		t.Fatalf("expected Result suffix violation, output:\n%s", out)
	}
}

func TestCheckRunAllowsModelTypesReqRspTypesAndResponseOnlyActions(t *testing.T) {
	projectDir := t.TempDir()
	writeCheckTestFile(t, filepath.Join(projectDir, "model", "action.go"), `package model

import (
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

type User struct {
	model.Base
}

type Search struct {
	model.Empty
}

type SearchReq struct {
	Query string `+"`json:\"query\"`"+`
}

type SearchRsp struct {
	Items []string `+"`json:\"items\"`"+`
}

type Ping struct {
	model.Empty
}

type PingRsp struct {
	Msg string `+"`json:\"msg\"`"+`
}

func (User) Design() {
	Create(func() {
		Enabled(true)
		Service(true)
		Payload[*User]()
		Result[*User]()
	})
}

func (Search) Design() {
	Route("search", func() {
		List(func() {
			Enabled(true)
			Service(true)
			Payload[*SearchReq]()
			Result[*SearchRsp]()
		})
	})
}

func (Ping) Design() {
	List(func() {
		Enabled(true)
		Service(true)
		Result[*PingRsp]()
	})
}
`)

	out, err := runCheckCommandForTest(t, projectDir)
	if err != nil {
		t.Fatalf("expected checkRun to allow model, Req/Rsp, and response-only action types, err=%v output:\n%s", err, out)
	}
}

func TestCheckRunHelperProcess(t *testing.T) {
	if os.Getenv("GG_CHECK_HELPER") != "1" {
		return
	}

	projectDir := os.Getenv("GG_CHECK_PROJECT_DIR")
	if projectDir == "" {
		t.Fatal("GG_CHECK_PROJECT_DIR is required")
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}

	modelDir = "model"
	serviceDir = "service"
	routerDir = "router"
	daoDir = "dao"

	checkRun()
}

func runCheckCommandForTest(t *testing.T, projectDir string) (string, error) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestCheckRunHelperProcess")
	cmd.Env = append(os.Environ(), "GG_CHECK_HELPER=1", "GG_CHECK_PROJECT_DIR="+projectDir)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func writeCheckTestFile(t *testing.T, filename, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
