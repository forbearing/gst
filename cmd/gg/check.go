package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/forbearing/gst/dsl"
	"github.com/gertd/go-pluralize"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "check architecture dependencies in generated code",
	Long: `Check architecture dependencies in generated code:
1. Service code should not call other service code
2. DAO code should not call service code
3. Model code should not call service code
4. Model directories and files must be singular
5. Model file names should not contain hyphens (use underscores instead)
6. Model struct json tags should use snake_case naming
7. Model package names must match their directory names
8. Only allowed directories are enforced for gst framework projects`,
	Run: func(cmd *cobra.Command, args []string) {
		checkRun()
	},
}

func checkRun() {
	var totalViolations int

	// Architecture Dependency Check
	totalViolations += CheckArchitectureDependency()

	// Model Singular Naming Check
	totalViolations += CheckModelSingularNaming()

	// Model JSON Tag Naming Check
	totalViolations += CheckModelJSONTagNaming()

	// Model Package Naming Check
	totalViolations += CheckModelPackageNaming()

	// Directory Restriction Check
	totalViolations += CheckAllowedDirectories()

	// Summary
	logSection("Summary")
	if totalViolations > 0 {
		fmt.Printf("  %s %d violations found\n", red("✘"), totalViolations)
		os.Exit(1)
	} else {
		fmt.Printf("  %s All checks passed\n", green("✔"))
	}
}

// CheckArchitectureDependency performs architecture dependency checks.
func CheckArchitectureDependency() int {
	//nolint:prealloc
	var violations []string

	// Check service files
	serviceViolations := checkServiceDependencies()
	violations = append(violations, serviceViolations...)

	// Check dao files
	daoViolations := checkDAODependencies()
	violations = append(violations, daoViolations...)

	// Check model files
	modelViolations := checkModelDependencies()
	violations = append(violations, modelViolations...)

	logSection("Architecture Dependency Check")
	if len(violations) > 0 {
		for _, violation := range violations {
			fmt.Printf("  %s %s\n", red("→"), violation)
		}
	} else {
		fmt.Printf("  %s No architecture violations found\n", green("✔"))
	}

	return len(violations)
}

// checkServiceDependencies checks if service code calls other service code
func checkServiceDependencies() []string {
	var violations []string

	if _, err := os.Stat(serviceDir); os.IsNotExist(err) {
		return violations
	}

	err := filepath.Walk(serviceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.Contains(path, "_test.go") {
			return nil
		}

		// Skip service.go registration file
		if strings.HasSuffix(path, "service.go") {
			return nil
		}

		fileViolations := checkFileForServiceImports(path, "service")
		violations = append(violations, fileViolations...)

		return nil
	})
	if err != nil {
		logError(fmt.Sprintf("walking service directory: %v", err))
	}

	return violations
}

// checkDAODependencies checks if DAO code calls service code
func checkDAODependencies() []string {
	var violations []string

	if _, err := os.Stat(daoDir); os.IsNotExist(err) {
		return violations
	}

	err := filepath.Walk(daoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.Contains(path, "_test.go") {
			return nil
		}

		fileViolations := checkFileForServiceImports(path, "dao")
		violations = append(violations, fileViolations...)

		return nil
	})
	if err != nil {
		logError(fmt.Sprintf("walking dao directory: %v", err))
	}

	return violations
}

// checkModelDependencies checks if model code calls service code
func checkModelDependencies() []string {
	var violations []string

	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
		return violations
	}

	err := filepath.Walk(modelDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.Contains(path, "_test.go") {
			return nil
		}

		// Skip model.go registration file
		if strings.HasSuffix(path, "model.go") {
			return nil
		}

		fileViolations := checkFileForServiceImports(path, "model")
		violations = append(violations, fileViolations...)

		return nil
	})
	if err != nil {
		logError(fmt.Sprintf("walking model directory: %v", err))
	}

	return violations
}

// checkFileForServiceImports checks a single file for service imports
func checkFileForServiceImports(filePath, layerType string) []string {
	var violations []string

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		// Treat parse errors as violations to prevent code generation
		violation := fmt.Sprintf("%s file '%s' has parse error: %v",
			cases.Title(language.English).String(layerType), filePath, err)
		violations = append(violations, violation)
		return violations
	}

	// Check imports
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		// Check for service imports
		if containsServiceImport(importPath, layerType) {
			violation := fmt.Sprintf("%s file '%s' imports service code: %s",
				cases.Title(language.English).String(layerType), filePath, importPath)
			violations = append(violations, violation)
		}
	}

	return violations
}

// CheckModelSingularNaming checks if model directories and files use singular names
func CheckModelSingularNaming() int {
	var violations []string

	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
		return 0
	}

	// Common plural file names that are allowed in Go projects
	allowedPluralFiles := map[string]bool{
		"types":       true,
		"errors":      true,
		"constants":   true,
		"consts":      true,
		"vars":        true,
		"handlers":    true,
		"models":      true,
		"examples":    true,
		"configs":     true,
		"options":     true,
		"helpers":     true,
		"utils":       true,
		"interfaces":  true,
		"services":    true,
		"clients":     true,
		"controllers": true,
		"apis":        true,
		"schemas":     true,
		"entities":    true,
		"records":     true,
	}

	client := pluralize.NewClient()

	err := filepath.Walk(modelDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from model directory
		relPath, err := filepath.Rel(modelDir, path)
		if err != nil {
			return err
		}

		// Skip the root model directory itself
		if relPath == "." {
			return nil
		}

		if info.IsDir() {
			// Check directory name.
			// Directory name length must greater than 3 before check.
			// Check singular must before plural.
			dirName := info.Name()
			if len(dirName) > 3 && !client.IsSingular(dirName) && client.IsPlural(dirName) {
				violation := fmt.Sprintf("Model directory '%s' should be singular (suggested: %s)",
					path, client.Singular(dirName))
				violations = append(violations, violation)
			}
		} else if strings.HasSuffix(path, ".go") && !strings.Contains(path, "_test.go") {
			// Skip model.go registration file
			if strings.HasSuffix(path, "model.go") {
				return nil
			}

			// Check Go file name (without .go extension)
			fileName := strings.TrimSuffix(info.Name(), ".go")

			// Check if file name contains hyphen
			if strings.Contains(fileName, "-") {
				suggestedName := strings.ReplaceAll(fileName, "-", "_")
				violation := fmt.Sprintf("Model file '%s' should not contain hyphens (suggested: %s.go)",
					path, suggestedName)
				violations = append(violations, violation)
			}

			// File name length must greater than 3 before check.
			// Check singular must before plural.
			// Skip check for allowed plural file names
			if len(fileName) > 3 && !allowedPluralFiles[fileName] && !client.IsSingular(fileName) && client.IsPlural(fileName) {
				violation := fmt.Sprintf("Model file '%s' should be singular (suggested: %s.go)",
					path, client.Singular(fileName))
				violations = append(violations, violation)
			}
		}

		return nil
	})
	if err != nil {
		logError(fmt.Sprintf("walking model directory: %v", err))
	}

	logSection("Model Singular Naming Check")
	if len(violations) > 0 {
		for _, violation := range violations {
			fmt.Printf("  %s %s\n", red("→"), violation)
		}
	} else {
		fmt.Printf("  %s No singular naming violations found\n", green("✔"))
	}

	return len(violations)
}

// containsServiceImport checks if an import path contains service code
func containsServiceImport(importPath, layerType string) bool {
	// Split import path by '/'
	parts := strings.SplitSeq(importPath, "/")

	for part := range parts {
		if part == "service" {
			// For service layer, check if it's importing other service packages
			if layerType == "service" {
				// Allow importing the base gst service package only
				if strings.Contains(importPath, "github.com/forbearing/gst/service") {
					return false
				}
				// Forbid importing any other service implementations
				return true
			}
			// For dao and model layers, any service import is forbidden except gst service
			if layerType == "dao" || layerType == "model" {
				// Allow importing the base gst service package for interfaces
				if strings.Contains(importPath, "github.com/forbearing/gst/service") {
					return false
				}
				// Forbid importing service implementations
				return true
			}
		}
	}
	return false
}

// CheckModelJSONTagNaming checks if model struct json tags use camelCase naming
func CheckModelJSONTagNaming() int {
	var violations []string

	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
		return 0
	}

	err := filepath.Walk(modelDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip generated files
		if strings.HasSuffix(path, "model.go") {
			return nil
		}

		fileViolations := checkFileJSONTagNaming(path)
		violations = append(violations, fileViolations...)

		return nil
	})
	if err != nil {
		logError(fmt.Sprintf("walking model directory: %v", err))
	}

	logSection("Model JSON Tag Naming Check")
	if len(violations) > 0 {
		for _, violation := range violations {
			fmt.Printf("  %s %s\n", red("→"), violation)
		}
	} else {
		fmt.Printf("  %s No JSON tag naming violations found\n", green("✔"))
	}

	return len(violations)
}

// checkFileJSONTagNaming checks json tag naming in a single file
func checkFileJSONTagNaming(filePath string) []string {
	var violations []string

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return violations
	}

	// Find all model structs in this file
	modelBaseNames := dsl.FindAllModelBase(node)
	modelEmptyNames := dsl.FindAllModelEmpty(node)
	allModelNames := append(modelBaseNames, modelEmptyNames...)

	// If no model structs found, skip this file
	if len(allModelNames) == 0 {
		return violations
	}

	// Get relative path for cleaner output
	cwd, _ := os.Getwd()
	relPath, _ := filepath.Rel(cwd, filePath)

	// Check only model structs
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			// Check if this struct is a model
			isModel := slices.Contains(allModelNames, typeSpec.Name.Name)
			if !isModel {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok || structType.Fields == nil {
				continue
			}

			// Check JSON tags in this model struct
			for _, field := range structType.Fields.List {
				if field.Tag != nil {
					tagValue := strings.Trim(field.Tag.Value, "`")
					if jsonTag := extractJSONTag(tagValue); jsonTag != "" {
						if !isSnakeCase(jsonTag) {
							fieldName := ""
							if len(field.Names) > 0 {
								fieldName = field.Names[0].Name
							}
							violations = append(violations, fmt.Sprintf(
								"%s: field '%s' json tag '%s' should be '%s'",
								relPath, fieldName, jsonTag, toSnakeCase(jsonTag)))
						}
					}
				}
			}
		}
	}

	return violations
}

// extractJSONTag extracts the json tag value from struct tag
func extractJSONTag(tag string) string {
	re := regexp.MustCompile(`json:"([^"]+)"`)
	matches := re.FindStringSubmatch(tag)
	if len(matches) > 1 {
		// Remove options like omitempty
		parts := strings.Split(matches[1], ",")
		return parts[0]
	}
	return ""
}

// isSnakeCase checks if a string is in snake_case format
func isSnakeCase(s string) bool {
	if s == "" {
		return true
	}

	// Skip special cases like "-" or single characters
	if s == "-" || len(s) == 1 {
		return true
	}

	// Check if it contains hyphens (kebab-case) or uppercase letters
	if strings.Contains(s, "-") {
		return false
	}

	// Check for uppercase letters (not snake_case)
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return false
		}
	}

	return true
}

// toSnakeCase converts camelCase or kebab-case to snake_case
func toSnakeCase(s string) string {
	if s == "" {
		return s
	}

	// Replace hyphens with underscores
	s = strings.ReplaceAll(s, "-", "_")

	// Convert camelCase to snake_case
	var result strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(r - 'A' + 'a')
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// CheckModelPackageNaming checks if model package names match their directory names
func CheckModelPackageNaming() int {
	var violations []string

	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
		return 0
	}

	err := filepath.Walk(modelDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip files in the root model directory
		relPath, err := filepath.Rel(modelDir, path)
		if err != nil {
			return err
		}
		if !strings.Contains(relPath, string(filepath.Separator)) {
			return nil
		}

		// Get the directory name (should match package name)
		dir := filepath.Dir(path)
		dirName := filepath.Base(dir)

		// Parse the Go file to get package name
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.PackageClauseOnly)
		if err != nil {
			return err
		}

		packageName := node.Name.Name

		// Check if package name matches directory name
		if packageName != dirName {
			relativePath, _ := filepath.Rel(modelDir, path)
			violations = append(violations, fmt.Sprintf("%s: package name '%s' should match directory name '%s'", relativePath, packageName, dirName))
		}

		return nil
	})
	if err != nil {
		fmt.Printf("Error walking model directory: %v\n", err)
	}

	// Model Package Naming Check
	logSection("Model Package Naming Check")
	if len(violations) > 0 {
		for _, violation := range violations {
			fmt.Printf("  %s %s\n", red("→"), violation)
		}
	} else {
		fmt.Printf("  %s No package naming violations found\n", green("✔"))
	}

	return len(violations)
}

// CheckAllowedDirectories checks if only allowed directories exist in the project
func CheckAllowedDirectories() int {
	projectDir := "."
	var violations []string

	// Check if this is a gst framework project by reading go.mod
	if isGstFrameworkProject(projectDir) {
		// Skip directory restriction check for gst framework itself
		return 0
	}

	// Check if this project uses gst framework
	if !usesGstFramework(projectDir) {
		// Skip directory restriction check for projects not using gst framework
		return 0
	}

	// Define allowed directories for gst framework projects
	allowedDirs := map[string]bool{
		"model":      true,
		"module":     true,
		"service":    true,
		"router":     true,
		"dao":        true,
		"provider":   true,
		"middleware": true,
		"cronjob":    true,
		"configx":    true,
		"config":     true,
		"typesx":     true,
		"consts":     true,
		"constx":     true,
		"type":       true,
		"typex":      true,
		"helper":     true,
		"internal":   true,
		"cmd":        true,
		"errorx":     true,
		"testcode":   true,
		"testdata":   true,
		"docs":       true,
		"doc":        true,
	}

	whitelistDirs := map[string]bool{
		"tmp":  true,
		"logs": true,
		"dist": true,
	}

	// Read directory contents
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return 0
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()

		// Skip hidden directories and common project files
		if strings.HasPrefix(dirName, ".") {
			continue
		}

		// Check if directory is allowed
		if !allowedDirs[dirName] && !whitelistDirs[dirName] {
			violations = append(violations, fmt.Sprintf("Directory '%s' is not allowed in project structure", dirName))
		}
	}

	// Directory Restriction Check
	logSection("Directory Restriction Check")
	if len(violations) > 0 {
		for _, violation := range violations {
			fmt.Printf("  %s %s\n", red("→"), violation)
		}
	} else {
		fmt.Printf("  %s No directory restriction violations found\n", green("✔"))
	}

	return len(violations)
}

// isGstFrameworkProject checks if this is the gst framework project itself
func isGstFrameworkProject(projectDir string) bool {
	goModPath := filepath.Join(projectDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return false
	}

	// Check if module name is github.com/forbearing/gst
	lines := strings.SplitSeq(string(content), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			moduleName := strings.TrimSpace(strings.TrimPrefix(line, "module"))
			return moduleName == "github.com/forbearing/gst"
		}
	}
	return false
}

// usesGstFramework checks if the project uses gst framework as a dependency
func usesGstFramework(projectDir string) bool {
	goModPath := filepath.Join(projectDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return false
	}

	// Check if github.com/forbearing/gst is in dependencies
	return strings.Contains(string(content), "github.com/forbearing/gst")
}
