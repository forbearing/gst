package constants

// Import paths
const (
	// Framework import paths
	//nolint:godoclint
	ImportPathModel     = "github.com/forbearing/gst/model"
	ImportPathService   = "github.com/forbearing/gst/service"
	ImportPathRouter    = "github.com/forbearing/gst/router"
	ImportPathTypes     = "github.com/forbearing/gst/types"
	ImportPathConsts    = "github.com/forbearing/gst/types/consts"
	ImportPathBootstrap = "github.com/forbearing/gst/bootstrap"
	ImportPathUtil      = "github.com/forbearing/gst/util"

	// ModelPackagePath is the package path for comparison
	ModelPackagePath = `"github.com/forbearing/gst/model"`
)

// File patterns and extensions
const (
	ExtensionGo     = ".go"
	PatternTestFile = "_test.go"
	PrefixIgnore    = "_"
)

// Directory names
const (
	DirVendor   = "vendor"
	DirTestData = "testdata"
	DirModel    = "model"
	DirService  = "service"
	DirRouter   = "router"
)

// Package names
const (
	PkgMain      = "main"
	PkgModel     = "model"
	PkgService   = "service"
	PkgRouter    = "router"
	PkgModule    = "module"
	PkgBootstrap = "bootstrap"
)

// Model field names
const (
	FieldBase  = "Base"
	FieldEmpty = "Empty"
)

// Function names
const (
	FuncInit     = "init"
	FuncMain     = "main"
	FuncInit2    = "Init"
	FuncRegister = "Register"
	FuncRunOrDie = "RunOrDie"
)

// Prefix for model package conversion
const (
	PrefixModel         = "model"
	PrefixService       = "service"
	SeparatorUnderscore = "_"
)

// Cache file
const (
	CacheFileName = ".gg_cache.json"
)

// Project subdirectories for main.go imports
const (
	SubDirConfigx    = "configx"
	SubDirCronjob    = "cronjob"
	SubDirMiddleware = "middleware"
	SubDirModel      = "model"
	SubDirModule     = "module"
	SubDirService    = "service"
	SubDirRouter     = "router"
)

// Bootstrap method names
const (
	BootstrapBootstrap = "Bootstrap"
	BootstrapRun       = "Run"
	RouterInit         = "Init"
	ModuleInit         = "Init"
)
