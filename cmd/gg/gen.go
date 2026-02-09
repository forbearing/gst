package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/forbearing/gst/ds/tree/trie"
	"github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/internal/codegen"
	"github.com/forbearing/gst/internal/codegen/gen"
	pkgnew "github.com/forbearing/gst/internal/codegen/new"
	"github.com/forbearing/gst/types/consts"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "generate service code",
	Run: func(cmd *cobra.Command, args []string) {
		genRun()
	},
}

var tsCmd = &cobra.Command{
	Use:   "ts",
	Short: "generate typescript interface code",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("not implemented")
	},
}

func init() {
	genCmd.AddCommand(tsCmd)
}

func genRun() {
	if len(module) == 0 {
		var err error
		module, err = gen.GetModulePath()
		checkErr(err)
	}

	// Architecture dependency check
	if CheckArchitectureDependency() > 0 {
		os.Exit(1)
	}
	// Model Singular Naming Check
	if CheckModelSingularNaming() > 0 {
		os.Exit(1)
	}
	// Model JSON Tag Naming Check
	if CheckModelJSONTagNaming() > 0 {
		os.Exit(1)
	}
	// Model Package Naming Check
	if CheckModelPackageNaming() > 0 {
		os.Exit(1)
	}
	// Directory Restriction Check
	if CheckAllowedDirectories() > 0 {
		os.Exit(1)
	}

	// Ensure required files exist
	logSection("Ensure Required Files")
	_ = pkgnew.EnsureFileExists()

	if !fileExists(modelDir) {
		logError(fmt.Sprintf("model dir not found: %s", modelDir))
		os.Exit(1)
	}

	// Scan all models
	logSection("Scan Models")
	allModels, err := codegen.FindModels(module, modelDir, serviceDir, excludes)
	buildHierarchicalEndpoints(allModels)
	propagateParentParams(allModels)

	checkErr(err)
	if len(allModels) == 0 {
		fmt.Println(gray("  No models found, nothing to do"))
		return
	}
	fmt.Printf("  %s %d models found\n", green("✔"), len(allModels))

	// Record old service files list (if prune option is enabled)
	var oldServiceFiles []string
	if prune {
		oldServiceFiles = scanExistingServiceFiles(serviceDir)
	}

	modelStmts := make([]ast.Stmt, 0)
	serviceStmts := make([]ast.Stmt, 0)
	routerStmts := make([]ast.Stmt, 0)
	modelImportMap := make(map[string]struct{})
	routerImportMap := make(map[string]struct{})
	serviceImportMap := make(map[string]struct{})

	for _, m := range allModels {
		if m.Design.Enabled && m.Design.Migrate {
			// If the ModelFileDir is "model" or "model/", the model package name is the same as the model name,
			// and the statement in model/model.go will be "Register[*Project]()".
			// otherwise, the model package name is the last segment of the model file dir.
			//
			// For example:
			// If the ModelFileDir is "model/setting", the model package name is "setting",
			// then the statement in model/model.go should be "Register[*setting.Project]()"
			if m.ModelPkgName == strings.TrimRight(m.ModelFileDir, "/") {
				modelStmts = append(modelStmts, gen.StmtModelRegister(m.ModelName))
			} else {
				modelStmts = append(modelStmts, gen.StmtModelRegister(fmt.Sprintf("%s.%s", m.ModelPkgName, m.ModelName)))
			}

			if path, shouldImport := m.ModelImportPath(); shouldImport {
				modelImportMap[path] = struct{}{}
			}
		}

		m.Design.Range(func(s string, a *dsl.Action) {
			if a.Service {
				serviceImportMap[m.ServiceImportPath(modelDir, serviceDir)] = struct{}{}
			}
			routerImportMap[m.RouterImportPath()] = struct{}{}
		})
	}

	// Resolve import conflicts
	serviceImports := lo.Keys(serviceImportMap)
	sort.Strings(serviceImports)
	serviceAliasMap := gen.ResolveImportConflicts(serviceImports)
	for _, m := range allModels {
		m.Design.Range(func(route string, act *dsl.Action) {
			if act.Service {
				if alias := serviceAliasMap[m.ServiceImportPath(modelDir, serviceDir)]; len(alias) > 0 {
					// alias import package, eg:
					// pkg1_user "service/pkg1/user"
					// pkg2_user "service/pkg2/user"
					serviceStmts = append(serviceStmts, gen.StmtServiceRegister(fmt.Sprintf("%s.%s", alias, act.Phase.RoleName()), act.Phase))
				} else {
					// Use lowercase ModelName as package name to maintain original naming style
					// For example: ModelName "ConfigSetting" -> package name "configsetting"
					serviceStmts = append(serviceStmts, gen.StmtServiceRegister(fmt.Sprintf("%s.%s", strings.ToLower(m.ModelName), act.Phase.RoleName()), act.Phase))
				}
			}
			base := "Auth"
			if act.Public {
				base = "Pub"
			}
			// If the phase is matched, the model endpoint will append the param, eg:
			// Endpoint: tenant, param is ":tenant", new endpoint is "tenant/:tenant"
			// Endpoint: tenant, param is ":id", new endpoint is "tenant/:id"
			switch act.Phase {
			case consts.PHASE_DELETE, consts.PHASE_UPDATE, consts.PHASE_PATCH, consts.PHASE_GET:
				if len(m.Design.Param) == 0 {
					route = filepath.Join(route, ":id") // empty param will append default ":id" to endpoint.
				} else {
					route = filepath.Join(route, m.Design.Param)
				}
			case consts.PHASE_CREATE_MANY, consts.PHASE_DELETE_MANY, consts.PHASE_UPDATE_MANY, consts.PHASE_PATCH_MANY:
				route = filepath.Join(route, "batch")
			case consts.PHASE_IMPORT:
				route = filepath.Join(route, "import")
			case consts.PHASE_EXPORT:
				route = filepath.Join(route, "export")

			}

			switch act.Phase {
			case consts.PHASE_DELETE, consts.PHASE_UPDATE, consts.PHASE_PATCH, consts.PHASE_GET:
				items := strings.Split(route, "/")
				lastSegment := strings.TrimLeft(items[len(items)-1], ":")
				routerStmts = append(routerStmts, gen.StmtRouterRegister(m.ModelPkgName, m.ModelName, act.Payload, act.Result, base, route, lastSegment, act.Phase.MethodName()))
			default:
				routerStmts = append(routerStmts, gen.StmtRouterRegister(m.ModelPkgName, m.ModelName, act.Payload, act.Result, base, route, "", act.Phase.MethodName()))
			}
		})

		// for key, actions := range m.Design.Routes {
		// 	for _, act := range actions {
		// 		route := key
		// 		base := "Auth"
		// 		if act.Public {
		// 			base = "Pub"
		// 		}
		// 		// If the phase is matched, the model endpoint will append the param, eg:
		// 		// Endpoint: tenant, param is ":tenant", new endpoint is "tenant/:tenant"
		// 		// Endpoint: tenant, param is ":id", new endpoint is "tenant/:id"
		// 		switch act.Phase {
		// 		case consts.PHASE_DELETE, consts.PHASE_UPDATE, consts.PHASE_PATCH, consts.PHASE_GET:
		// 			if len(m.Design.Param) == 0 {
		// 				route = filepath.Join(route, ":id") // empty param will append default ":id" to endpoint.
		// 			} else {
		// 				route = filepath.Join(route, m.Design.Param)
		// 			}
		// 		case consts.PHASE_CREATE_MANY, consts.PHASE_DELETE_MANY, consts.PHASE_UPDATE_MANY, consts.PHASE_PATCH_MANY:
		// 			route = filepath.Join(route, "batch")
		// 		case consts.PHASE_IMPORT:
		// 			route = filepath.Join(route, "import")
		// 		case consts.PHASE_EXPORT:
		// 			route = filepath.Join(route, "export")
		//
		// 		}
		//
		// 		switch act.Phase {
		// 		case consts.PHASE_DELETE, consts.PHASE_UPDATE, consts.PHASE_PATCH, consts.PHASE_GET:
		// 			items := strings.Split(route, "/")
		// 			lastSegment := strings.TrimLeft(items[len(items)-1], ":")
		// 			routerStmts = append(routerStmts, gen.StmtRouterRegister(m.ModelPkgName, m.ModelName, act.Payload, act.Result, base, route, lastSegment, act.Phase.MethodName()))
		// 		default:
		// 			routerStmts = append(routerStmts, gen.StmtRouterRegister(m.ModelPkgName, m.ModelName, act.Payload, act.Result, base, route, "", act.Phase.MethodName()))
		// 		}
		// 	}
		// }
	}

	// ============================================================
	// Generate model/service/router/main files
	// ============================================================
	logSection("Generate Files")

	modelImports := lo.Keys(modelImportMap)
	sort.Strings(modelImports)
	modelCode, err := gen.BuildModelFile("model", modelImports, modelStmts...)
	checkErr(err)
	writeFileWithLog(filepath.Join(modelDir, "model.go"), modelCode)

	// generate service/service.go
	serviceImports = lo.Keys(serviceImportMap)
	sort.Strings(serviceImports)
	serviceCode, err := gen.BuildServiceFile("service", serviceImports, serviceStmts...)
	checkErr(err)
	writeFileWithLog(filepath.Join(serviceDir, "service.go"), serviceCode)

	// generate router/router.go
	// router always imports "github.com/forbearing/gst/types"
	routerImportMap["github.com/forbearing/gst/types"] = struct{}{}
	routerImports := lo.Keys(routerImportMap)
	sort.Strings(routerImports)
	routerCode, err := gen.BuildRouterFile("router", routerImports, routerStmts...)
	checkErr(err)
	writeFileWithLog(filepath.Join(routerDir, "router.go"), routerCode)

	// generate main.go
	mainCode, err := gen.BuildMainFile(module)
	checkErr(err)
	writeFileWithLog("main.go", mainCode)

	// ============================================================
	// Apply actions to services
	// ============================================================
	logSection("Apply Actions To Services")

	fset := token.NewFileSet()
	applyFile := func(filename string, code string, action *dsl.Action, servicePkgName string, modelInfo *gen.ModelInfo) {
		if fileExists(filename) {
			// Read original file content to preserve comments and formatting
			src, err := os.ReadFile(filename)
			checkErr(err)
			f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
			checkErr(err)

			// Calculate the correct model import path and package name
			correctModelImportPath := filepath.Join(modelInfo.ModulePath, modelInfo.ModelFileDir)
			correctModelPkgName := modelInfo.ModelPkgName

			// Apply changes and sync model imports to handle import path and package name updates
			changed := gen.ApplyServiceFileWithModelSync(f, action, servicePkgName, correctModelImportPath, correctModelPkgName)

			if changed {
				// Only reformat and write file when there are changes
				// Use original FileSet to preserve comment positions
				code, err = gen.FormatNodeExtraWithFileSet(f, fset)
				checkErr(err)
				logUpdate(filename)
				checkErr(ensureParentDir(filename))
				checkErr(os.WriteFile(filename, []byte(code), 0o600))
			} else {
				logSkip(filename)
			}
		} else {
			logCreate(filename)
			checkErr(ensureParentDir(filename))
			checkErr(os.WriteFile(filename, []byte(code), 0o600))
		}
	}

	for _, m := range allModels {
		m.Design.Range(func(route string, act *dsl.Action) {
			if file := gen.GenerateService(m, act, act.Phase); file != nil {
				fset := token.NewFileSet()
				code, err := gen.FormatNodeExtraWithFileSet(file, fset)
				// pretty.Println(file)
				checkErr(err)
				// code = gen.MethodAddComments(code, m.ModelName)
				dir := strings.Replace(m.ModelFilePath, modelDir, serviceDir, 1)
				dir = strings.TrimSuffix(dir, ".go")
				filename := filepath.Join(dir, act.ServiceFilename())
				// Use lowercase ModelName as service package name to ensure consistency
				// with service registration logic and maintain original naming style
				// For example: ModelName "ConfigSetting" -> package name "configsetting"
				servicePkgName := strings.ToLower(m.ModelName)
				applyFile(filename, code, act, servicePkgName, m)
			}
		})
	}

	// ============================================================
	// Prune disabled service files
	// ============================================================
	if prune && len(oldServiceFiles) > 0 {
		pruneServiceFiles(oldServiceFiles, allModels)
	}

	// ============================================================
	// Completion message
	// ============================================================
	logSection("Done")
	fmt.Printf("\n%s Code generation completed successfully!\n", green("🎉"))
}

// scanExistingServiceFiles scans existing service files in the service directory
// Only includes files that match phase names (e.g., create.go, update.go, etc.)
func scanExistingServiceFiles(serviceDir string) []string {
	var files []string

	// Check if service directory exists
	if _, err := os.Stat(serviceDir); os.IsNotExist(err) {
		return files
	}

	// Get all valid phase names
	validPhases := map[string]bool{
		consts.PHASE_CREATE.Filename():      true,
		consts.PHASE_DELETE.Filename():      true,
		consts.PHASE_UPDATE.Filename():      true,
		consts.PHASE_PATCH.Filename():       true,
		consts.PHASE_LIST.Filename():        true,
		consts.PHASE_GET.Filename():         true,
		consts.PHASE_CREATE_MANY.Filename(): true,
		consts.PHASE_DELETE_MANY.Filename(): true,
		consts.PHASE_UPDATE_MANY.Filename(): true,
		consts.PHASE_PATCH_MANY.Filename():  true,
		consts.PHASE_IMPORT.Filename():      true,
		consts.PHASE_EXPORT.Filename():      true,
	}

	// Walk through the service directory
	err := filepath.Walk(serviceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			fileName := filepath.Base(path)
			// Only include files that match phase names
			if validPhases[fileName] {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Warning: failed to scan existing service files: %v\n", err)
	}
	return files
}

// filterIgnoredFiles filters out files that match any ignore pattern
// Supports both string matching (contains) and regex matching
// Returns filtered files and ignored files
func filterIgnoredFiles(files []string, ignorePatterns []string) (filtered []string, ignored []string) {
	if len(ignorePatterns) == 0 {
		return files, []string{}
	}

	for _, file := range files {
		shouldIgnore := false

		for _, pattern := range ignorePatterns {
			// Try regex match first
			if matched, err := regexp.MatchString(pattern, file); err == nil && matched {
				shouldIgnore = true
				break
			}
			// Fallback to string contains match
			if strings.Contains(file, pattern) {
				shouldIgnore = true
				break
			}
		}

		if shouldIgnore {
			ignored = append(ignored, file)
		} else {
			filtered = append(filtered, file)
		}
	}

	return filtered, ignored
}

// pruneServiceFiles prunes disabled service files
func pruneServiceFiles(oldServiceFiles []string, allModels []*gen.ModelInfo) {
	// Get list of service files that should currently exist
	currentServiceFiles := make(map[string]bool)
	for _, m := range allModels {
		m.Design.Range(func(route string, act *dsl.Action) {
			if act.Enabled && act.Service {
				dir := strings.Replace(m.ModelFilePath, modelDir, serviceDir, 1)
				dir = strings.TrimSuffix(dir, ".go")
				filename := filepath.Join(dir, act.ServiceFilename())
				currentServiceFiles[filename] = true
			}
		})
	}

	// Find files to delete (exist in old list but not in current list)
	filesToDelete := make([]string, 0)
	for _, oldFile := range oldServiceFiles {
		if !currentServiceFiles[oldFile] {
			filesToDelete = append(filesToDelete, oldFile)
		}
	}

	// Apply ignore patterns from config
	ignorePatterns := getPruneIgnorePatterns()
	var ignoredFiles []string
	if len(ignorePatterns) > 0 {
		filesToDelete, ignoredFiles = filterIgnoredFiles(filesToDelete, ignorePatterns)
	}

	// Display ignored files if any
	if len(ignoredFiles) > 0 {
		fmt.Printf("\n%s Files ignored by config:\n", gray("→"))
		for _, file := range ignoredFiles {
			fmt.Printf("  %s %s\n", gray("→ IGNORE"), file)
		}
	}

	if len(filesToDelete) == 0 {
		if len(ignoredFiles) > 0 {
			fmt.Printf("\n  %s No disabled service files to prune (all files are ignored)\n", green("✔"))
		} else {
			fmt.Printf("  %s No disabled service files to prune\n", green("✔"))
		}
		// Still check for empty directories even if no files to delete
		removeEmptyDirectories(serviceDir)
		return
	}

	// Display list of files to be deleted
	fmt.Printf("\n%s Files to be deleted:\n", yellow("⚠"))
	for _, file := range filesToDelete {
		fmt.Printf("  %s %s\n", red("✘"), file)
	}

	// Ask user for confirmation
	fmt.Printf("\n%s Do you want to delete these files? (y/N): ", cyan("?"))
	var response string
	_, _ = fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" && response != "yes" {
		fmt.Printf("  %s Deletion canceled\n", gray("→"))
		return
	}

	// Execute deletion operation
	for _, file := range filesToDelete {
		if err := os.Remove(file); err != nil {
			fmt.Printf("  %s Failed to delete %s: %v\n", red("✘"), file, err)
		} else {
			fmt.Printf("  %s Deleted %s\n", green("✔"), file)
		}
	}

	// Remove empty directories after deleting files
	removeEmptyDirectories(serviceDir)
}

// removeEmptyDirectories recursively removes empty directories starting from the given root directory
func removeEmptyDirectories(rootDir string) {
	_ = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			//nolint:nilerr
			return nil // Continue walking even if there's an error
		}

		// Skip the root directory itself
		if path == rootDir {
			return nil
		}

		// Only process directories
		if !info.IsDir() {
			return nil
		}

		// Check if directory is empty
		entries, err := os.ReadDir(path)
		if err != nil {
			//nolint:nilerr
			return nil // Continue if we can't read the directory
		}

		// If directory is empty, remove it
		if len(entries) == 0 {
			if err := os.Remove(path); err == nil {
				fmt.Printf("  %s Removed empty directory %s\n", green("✔"), path)
			}
		}

		return nil
	})

	// Run multiple passes to handle nested empty directories
	// After removing a directory, its parent might become empty
	for range 3 {
		emptyDirsFound := false
		_ = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || path == rootDir || !info.IsDir() {
				//nolint:nilerr
				return nil
			}

			entries, err := os.ReadDir(path)
			if err != nil {
				//nolint:nilerr
				return nil
			}

			if len(entries) == 0 {
				if err := os.Remove(path); err == nil {
					fmt.Printf("  %s Removed empty directory %s\n", green("✔"), path)
					emptyDirsFound = true
				}
			}

			return nil
		})

		// If no empty directories were found in this pass, we're done
		if !emptyDirsFound {
			break
		}
	}
}

// buildHierarchicalEndpoints constructs complete hierarchical endpoint paths for all models.
// It maps directory structures to their corresponding endpoint names and builds full endpoint paths
// by replacing directory names with their custom endpoint names (if defined).
//
// For example:
//   - model/config/namespace.go with Endpoint("namespaces") -> config/namespaces
//   - model/config/namespace/app.go with Endpoint("apps") -> config/namespaces/apps
//   - model/config/namespace/app/env.go with Endpoint("envs") -> config/namespaces/apps/envs
func buildHierarchicalEndpoints(allModels []*gen.ModelInfo) {
	// Create a map to store directory-to-endpoint mappings
	// This will store what endpoint name should be used for each directory
	dirEndpointMap := make(map[string]string)

	// First pass: build directory-to-endpoint mapping
	for _, m := range allModels {
		if m.Design == nil {
			continue
		}

		// Extract directory from model file path
		modelFilePath := strings.TrimPrefix(m.ModelFilePath, "model/")
		modelDir_ := filepath.Dir(modelFilePath)
		if modelDir_ == "." {
			modelDir_ = ""
		}

		// Get the filename without extension
		fileName := strings.TrimSuffix(filepath.Base(modelFilePath), ".go")

		// Determine the directory path that this model defines endpoint for
		// The rule is: model file defines endpoint for the directory path formed by modelDir + fileName
		var targetDir string
		if modelDir_ == "" {
			targetDir = fileName
		} else {
			targetDir = filepath.Join(modelDir_, fileName)
		}

		// Store the endpoint mapping for the target directory
		if m.Design.Endpoint != "" {
			dirEndpointMap[targetDir] = m.Design.Endpoint
		}
	}

	// Second pass: build complete endpoints by replacing directory names with mapped endpoints
	for _, m := range allModels {
		if m.Design == nil {
			continue
		}

		// Extract directory from model file path
		modelFilePath := strings.TrimPrefix(m.ModelFilePath, "model/")
		modelDir_ := filepath.Dir(modelFilePath)
		if modelDir_ == "." {
			modelDir_ = ""
		}

		// Store the original endpoint from DSL
		originalEndpoint := m.Design.Endpoint

		if modelDir_ == "" {
			// Model is in root model directory, keep original endpoint
			continue
		}

		// Build the complete endpoint path by replacing directory names with mapped endpoints
		var endpointParts []string
		pathParts := strings.Split(modelDir_, "/")

		// For each directory level, use mapped endpoint or directory name
		for i := range pathParts {
			currentPath := strings.Join(pathParts[:i+1], "/")
			if mappedEndpoint, exists := dirEndpointMap[currentPath]; exists {
				// Use the mapped endpoint for this directory
				endpointParts = append(endpointParts, mappedEndpoint)
			} else {
				// No mapping found, use directory name
				endpointParts = append(endpointParts, pathParts[i])
			}
		}

		// Add the current model's original endpoint
		endpointParts = append(endpointParts, originalEndpoint)

		// Join all parts to form the complete endpoint
		m.Design.Endpoint = strings.Join(endpointParts, "/")
	}

	// for _, m := range allModels {
	// 	fmt.Println("-----", m.ModelFilePath, "=>", m.Design.Endpoint)
	// }
}

// buildHierarchicalEndpointsV2 constructs complete hierarchical endpoint paths for all models using a trie data structure.
// This is an optimized version of buildHierarchicalEndpoints that leverages trie for efficient path management.
// It maps directory structures to their corresponding endpoint names and builds full endpoint paths
// by replacing directory names with their custom endpoint names (if defined).
//
// The trie structure provides several advantages:
// - Efficient prefix-based lookups for directory-to-endpoint mappings
// - Natural hierarchical organization that mirrors the directory structure
// - Better performance for deep directory hierarchies
// - Simplified path traversal and reconstruction
//
// For example:
//   - model/config/namespace.go with Endpoint("namespaces") -> config/namespaces
//   - model/config/namespace/app.go with Endpoint("apps") -> config/namespaces/apps
//   - model/config/namespace/app/env.go with Endpoint("envs") -> config/namespaces/apps/envs
func buildHierarchicalEndpointsV2(allModels []*gen.ModelInfo) {
	// Create a trie to store directory-to-endpoint mappings
	// The trie key is the directory path (as runes), and the value is the endpoint name
	dirEndpointTrie, err := trie.New[rune, string]()
	if err != nil {
		panic(err)
	}

	// First pass: build directory-to-endpoint mapping using trie
	for _, m := range allModels {
		if m.Design == nil {
			continue
		}

		// Extract directory from model file path
		modelFilePath := strings.TrimPrefix(m.ModelFilePath, "model/")
		modelDir_ := filepath.Dir(modelFilePath)
		if modelDir_ == "." {
			modelDir_ = ""
		}

		// Get the filename without extension
		fileName := strings.TrimSuffix(filepath.Base(modelFilePath), ".go")

		// Determine the directory path that this model defines endpoint for
		// The rule is: model file defines endpoint for the directory path formed by modelDir + fileName
		var targetDir string
		if modelDir_ == "" {
			targetDir = fileName
		} else {
			targetDir = filepath.Join(modelDir_, fileName)
		}

		// Store the endpoint mapping in the trie
		if m.Design.Endpoint != "" {
			// Convert directory path to runes for trie key
			dirEndpointTrie.Put([]rune(targetDir), m.Design.Endpoint)
		}
	}

	// Second pass: build complete endpoints by replacing directory names with mapped endpoints
	for _, m := range allModels {
		if m.Design == nil {
			continue
		}

		// Extract directory from model file path
		modelFilePath := strings.TrimPrefix(m.ModelFilePath, "model/")
		modelDir_ := filepath.Dir(modelFilePath)
		if modelDir_ == "." {
			modelDir_ = ""
		}

		// Store the original endpoint from DSL
		originalEndpoint := m.Design.Endpoint

		if modelDir_ == "" {
			// Model is in root model directory, keep original endpoint
			continue
		}

		// Build the complete endpoint path by replacing directory names with mapped endpoints
		var endpointParts []string
		pathParts := strings.Split(modelDir_, "/")

		// For each directory level, use trie to lookup mapped endpoint or directory name
		for i := range pathParts {
			currentPath := strings.Join(pathParts[:i+1], "/")
			// Use trie to lookup the mapped endpoint for this directory
			if mappedEndpoint, exists := dirEndpointTrie.Get([]rune(currentPath)); exists {
				// Use the mapped endpoint for this directory
				endpointParts = append(endpointParts, mappedEndpoint)
			} else {
				// No mapping found, use directory name
				endpointParts = append(endpointParts, pathParts[i])
			}
		}

		// Add the current model's original endpoint
		endpointParts = append(endpointParts, originalEndpoint)

		// Join all parts to form the complete endpoint
		m.Design.Endpoint = strings.Join(endpointParts, "/")
	}

	// for _, m := range allModels {
	// 	fmt.Println("-----", m.ModelFilePath, "=>", m.Design.Endpoint)
	// }
}

// propagateParentParams propagates parent resource parameters to all child resource endpoints.
// This function uses a trie data structure to efficiently organize and traverse the hierarchical
// endpoint structure, ensuring that parent parameters are correctly inherited by all descendant resources.
//
// When a parent resource defines a parameter (e.g., Param("ns")), all its child resources
// should inherit this parameter in their endpoint paths to maintain proper REST hierarchy.
// This is essential for creating RESTful APIs that follow nested resource patterns.
//
// Real-world usage scenarios:
//
// 1. Kubernetes-style namespace hierarchy:
//
//   - model/config/namespace.go defines Endpoint("namespaces") with Param("ns")
//
//   - model/config/namespace/app.go defines Endpoint("apps") with Param("app")
//
//   - model/config/namespace/app/env.go defines Endpoint("envs")
//
//     Before propagation:
//
//   - config/namespaces (with Param("ns"))
//
//   - config/namespaces/apps (with Param("app"))
//
//   - config/namespaces/apps/envs
//
//     After propagation:
//
//   - config/namespaces
//
//   - config/namespaces/:ns/apps
//
//   - config/namespaces/:ns/apps/:app/envs
//
//     Generated API endpoints:
//     GET    /api/config/namespaces
//     POST   /api/config/namespaces
//     GET    /api/config/namespaces/:ns/apps
//     POST   /api/config/namespaces/:ns/apps
//     GET    /api/config/namespaces/:ns/apps/:app/envs
//     POST   /api/config/namespaces/:ns/apps/:app/envs
//
// 2. Multi-tenant organization structure:
//
//   - model/tenant.go defines Endpoint("tenants") with Param("tenant")
//
//   - model/tenant/project.go defines Endpoint("projects") with Param("project")
//
//   - model/tenant/project/resource.go defines Endpoint("resources")
//
//     Results in endpoints like:
//     /api/tenants/:tenant/projects/:project/resources
//
// 3. E-commerce category hierarchy:
//
//   - model/category.go defines Endpoint("categories") with Param("category")
//
//   - model/category/product.go defines Endpoint("products") with Param("product")
//
//   - model/category/product/variant.go defines Endpoint("variants")
//
//     Results in endpoints like:
//     /api/categories/:category/products/:product/variants
//
// The trie data structure provides several advantages:
// - Efficient hierarchical organization of endpoints
// - O(log n) lookup time for ancestor relationships
// - Natural representation of tree-like endpoint structures
// - Easy parameter propagation through PathAncestors method
//
// This ensures that child resources are properly nested under their parent's parameter scope,
// maintaining RESTful conventions and enabling proper resource identification in nested APIs.
func propagateParentParams(allModels []*gen.ModelInfo) {
	nodeFormater := trie.WithNodeFormatter[string, *gen.ModelInfo](func(v *gen.ModelInfo, depth int, hasValue bool) string {
		if !hasValue || v == nil {
			return "<nil>"
		}
		return fmt.Sprintf("%s (param: %s)", v.Design.Endpoint, v.Design.Param)
	})
	keyFormater := trie.WithKeyFormatter[string, *gen.ModelInfo](func(k string, v *gen.ModelInfo, depth int, hasValue bool) string {
		return k
	})

	// Create a trie tree to organize endpoints hierarchically
	// Key type is string, value type is *gen.ModelInfo
	tree, err := trie.New[string, *gen.ModelInfo](nodeFormater, keyFormater)
	if err != nil {
		panic(err)
	}

	// Build the trie tree
	for _, m := range allModels {
		// Split endpoint into segments for trie insertion
		// e.g., "config/namespaces/apps" -> ["config", "namespaces", "apps"]
		tree.Put(strings.Split(m.Design.Endpoint, "/"), m)
	}

	// Use trie's PathAncestors to collect parameters from all ancestor levels
	for _, model := range allModels {
		// Get all ancestors (including self) for this endpoint
		ancestors := tree.PathAncestors(strings.Split(model.Design.Endpoint, "/"))

		// Build the new endpoint path by inserting parameters from all ancestors
		newPathSegments := make([]string, 0)

		// Process each ancestor to build the hierarchical path with parameters
		// Note: ancestors[len(ancestors)-1] is the model itself, so we exclude it from parameter propagation
		for i, ancestor := range ancestors {
			// Add path segments from this ancestor level
			if i == 0 {
				// First ancestor: add all its path segments
				newPathSegments = append(newPathSegments, ancestor.Keys...)
			} else {
				// Subsequent ancestors: add only the new segments (difference from previous)
				prevAncestor := ancestors[i-1]
				if len(ancestor.Keys) > len(prevAncestor.Keys) {
					// Add the new segments
					newSegments := ancestor.Keys[len(prevAncestor.Keys):]
					newPathSegments = append(newPathSegments, newSegments...)
				}
			}

			// Add the parameter for this ancestor (if it has one)
			// But skip the last ancestor (which is the model itself) to avoid duplicate parameters
			if i < len(ancestors)-1 && ancestor.Value != nil && len(ancestor.Value.Design.Param) > 0 {
				param := ancestor.Value.Design.Param
				newPathSegments = append(newPathSegments, param)
			}
		}

		// Update the model's endpoint with the new path that includes all ancestor parameters
		if len(newPathSegments) > 0 {
			newEndpoint := strings.Join(newPathSegments, "/")
			model.Design.Endpoint = newEndpoint
		}
	}
}
