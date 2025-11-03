//nolint:predeclared
package new

import (
	"github.com/forbearing/gst/types/consts"
)

var modelContent = consts.CodeGeneratedComment() + `
package model

func init() {
}
`

var serviceContent = consts.CodeGeneratedComment() + `
package service

func init() {
}
`

var routerContent = consts.CodeGeneratedComment() + `
package router

func Init() error {
	return nil
}
`

var moduleContent = consts.CodeGeneratedComment() + `
package module

func Init() error {
	return nil
}
`

var mainContent = consts.CodeGeneratedComment() + `
package main

import (
	_ "%s/configx"
	_ "%s/cronjob"
	_ "%s/middleware"
	_ "%s/model"
	"%s/module"
	"%s/router"
	_ "%s/service"

	"github.com/forbearing/gst/bootstrap"
	. "github.com/forbearing/gst/util"
)

func main() {
	RunOrDie(bootstrap.Bootstrap)
	RunOrDie(module.Init)
	RunOrDie(router.Init)
	RunOrDie(bootstrap.Run)
}
`

const configxContent = `package configx

func init() {
}
`

const cronjobContent = `package cronjob

func init() {
}
`

const middlewareContent = `package middleware

func init() {
}
`

const gitignoreContent = `# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with 'go test -c'
*.test

# Output of the go coverage tool, specifically when used with LiteIDE
*.out

# Dependency directories (remove the comment below to include it)
# vendor/

# Go workspace file
go.work

# IDE files
.vscode/
.idea/
*.swp
*.swo
*~

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# Log files
*.log

# Temporary files
tmp/
temp/

# Build output
dist/
build/`
