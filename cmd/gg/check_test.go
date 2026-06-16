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
	writeCheckTestFile(t, filepath.Join(projectDir, "model", "user.go"), `package model

import (
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

type User struct {
	model.Base
}

func (User) Design() {
	Create(func() {
		Enabled(true)
		Service(true)
		Payload[*User]()
		Result[*User]()
	})
}
`)
	writeCheckTestFile(t, filepath.Join(projectDir, "model", "search.go"), `package model

import (
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

type Search struct {
	model.Empty
}

type SearchReq struct {
	Query string `+"`json:\"query\"`"+`
}

type SearchRsp struct {
	Items []string `+"`json:\"items\"`"+`
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
`)
	writeCheckTestFile(t, filepath.Join(projectDir, "model", "ping.go"), `package model

import (
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

type Ping struct {
	model.Empty
}

type PingRsp struct {
	Msg string `+"`json:\"msg\"`"+`
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

func TestCheckRunPrintsCompactSuccessReport(t *testing.T) {
	projectDir := t.TempDir()
	writeCheckTestFile(t, filepath.Join(projectDir, "model", "user.go"), `package model

import "github.com/forbearing/gst/model"

type User struct {
	model.Base
}
`)

	out, err := runCheckCommandForTest(t, projectDir)
	if err != nil {
		t.Fatalf("expected checkRun to pass, err=%v output:\n%s", err, out)
	}

	for _, want := range []string{
		"▶ Project Checks",
		"✔ Architecture dependencies",
		"✔ Model singular naming",
		"✔ Model JSON tag naming",
		"✔ Model action type naming",
		"✔ Model file boundaries",
		"✔ Service file boundaries",
		"✔ Model package naming",
		"✔ Directory restrictions",
		"▶ Summary",
		"✔ All checks passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected compact check output to contain %q, output:\n%s", want, out)
		}
	}

	for _, oldSection := range []string{
		"Architecture Dependency Check",
		"Model Singular Naming Check",
		"Model JSON Tag Naming Check",
		"Model Action Type Naming Check",
		"Model File Boundary Check",
		"Service File Boundary Check",
		"Model Package Naming Check",
		"Directory Restriction Check",
	} {
		if strings.Contains(out, oldSection) {
			t.Fatalf("expected compact check output to omit old section %q, output:\n%s", oldSection, out)
		}
	}
}

func TestCheckRunRejectsMultipleModelStructsInOneFile(t *testing.T) {
	projectDir := t.TempDir()
	writeCheckTestFile(t, filepath.Join(projectDir, "model", "account.go"), `package model

import "github.com/forbearing/gst/model"

type Account struct {
	model.Base
}

type Session struct {
	model.Empty
}

type AccountReq struct {
	Name string `+"`json:\"name\"`"+`
}
`)

	out, err := runCheckCommandForTest(t, projectDir)
	if err == nil {
		t.Fatalf("expected checkRun to reject multiple model structs in one file, output:\n%s", out)
	}
	if !strings.Contains(out, "Model file 'model/account.go' should contain at most one model struct") {
		t.Fatalf("expected model file boundary violation, output:\n%s", out)
	}
}

func TestCheckRunAllowsOneModelStructWithMultipleDTOs(t *testing.T) {
	projectDir := t.TempDir()
	writeCheckTestFile(t, filepath.Join(projectDir, "model", "account.go"), `package model

import "github.com/forbearing/gst/model"

type Account struct {
	model.Base
}

type AccountCreateReq struct {
	Name string `+"`json:\"name\"`"+`
}

type AccountCreateRsp struct {
	ID string `+"`json:\"id\"`"+`
}

type AccountPatchReq struct {
	DisplayName string `+"`json:\"display_name\"`"+`
}

type accountHelper struct {
	normalizedName string
}
`)

	out, err := runCheckCommandForTest(t, projectDir)
	if err != nil {
		t.Fatalf("expected checkRun to allow one model struct with DTO/helper types, err=%v output:\n%s", err, out)
	}
}

func TestCheckRunRejectsMultipleServiceStructsInOneFile(t *testing.T) {
	projectDir := t.TempDir()
	writeCheckTestFile(t, filepath.Join(projectDir, "service", "account", "action.go"), `package account

import (
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
)

type Creator struct {
	service.Base[*model.Empty, *model.Empty, *model.Empty]
}

type Patcher struct {
	service.Base[*model.Empty, *model.Empty, *model.Empty]
}
`)

	out, err := runCheckCommandForTest(t, projectDir)
	if err == nil {
		t.Fatalf("expected checkRun to reject multiple service structs in one file, output:\n%s", out)
	}
	if !strings.Contains(out, "Service file 'service/account/action.go' should contain at most one service struct") {
		t.Fatalf("expected service file boundary violation, output:\n%s", out)
	}
}

func TestCheckRunAllowsOneServiceStructWithHelpers(t *testing.T) {
	projectDir := t.TempDir()
	writeCheckTestFile(t, filepath.Join(projectDir, "service", "account", "create.go"), `package account

import (
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
)

type Creator struct {
	service.Base[*model.Empty, *model.Empty, *model.Empty]
}

type createHelper struct {
	normalizedName string
}
`)

	out, err := runCheckCommandForTest(t, projectDir)
	if err != nil {
		t.Fatalf("expected checkRun to allow one service struct with helper types, err=%v output:\n%s", err, out)
	}
}

func TestGenRunRejectsChecksRegisteredForCheckRun(t *testing.T) {
	projectDir := t.TempDir()
	writeCheckTestFile(t, filepath.Join(projectDir, "go.mod"), "module example.com/app\n\nrequire github.com/forbearing/gst v0.0.0\n")
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

	out, err := runGenCommandForTest(t, projectDir)
	if err == nil {
		t.Fatalf("expected genRun to reject checkRun violations, output:\n%s", out)
	}
	if !strings.Contains(out, "Payload type 'SearchRequest' should end with Req") {
		t.Fatalf("expected genRun to run action type naming check, output:\n%s", out)
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
	t.Chdir(projectDir)

	modelDir = "model"
	serviceDir = "service"
	routerDir = "router"
	daoDir = "dao"

	checkRun()
}

func TestGenRunHelperProcess(t *testing.T) {
	if os.Getenv("GG_GEN_HELPER") != "1" {
		return
	}

	projectDir := os.Getenv("GG_GEN_PROJECT_DIR")
	if projectDir == "" {
		t.Fatal("GG_GEN_PROJECT_DIR is required")
	}
	t.Chdir(projectDir)

	modelDir = "model"
	serviceDir = "service"
	routerDir = "router"
	daoDir = "dao"
	module = ""
	prune = false

	genRun()
}

func runCheckCommandForTest(t *testing.T, projectDir string) (string, error) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestCheckRunHelperProcess")
	cmd.Env = append(os.Environ(), "GG_CHECK_HELPER=1", "GG_CHECK_PROJECT_DIR="+projectDir)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func runGenCommandForTest(t *testing.T, projectDir string) (string, error) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestGenRunHelperProcess")
	cmd.Env = append(os.Environ(), "GG_GEN_HELPER=1", "GG_GEN_PROJECT_DIR="+projectDir)
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
