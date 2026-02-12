package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/forbearing/gst/internal/codegen/gen"
	"github.com/forbearing/gst/types/consts"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	Long:  "Generate and execute database migration code based on current models",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Get module name
		moduleName, err := gen.GetModulePath()
		if err != nil {
			return fmt.Errorf("failed to get module path: %w", err)
		}

		// 2. Prepare content
		content := migrateTemplate
		content = strings.ReplaceAll(content, "{{MODULE}}", moduleName)

		fullContent := fmt.Sprintf("%s\n%s", consts.CodeGeneratedComment(), content)

		// 3. Create directory
		targetDir := "cmd/migrate"
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
		}

		// 4. Write file
		targetFile := filepath.Join(targetDir, "main.go")
		if err := os.WriteFile(targetFile, []byte(fullContent), 0o600); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetFile, err)
		}
		fmt.Printf("Generated %s\n", targetFile)

		// 4.5 Run go mod tidy to ensure dependencies are met
		fmt.Println("Running go mod tidy...")
		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Stdout = os.Stdout
		tidyCmd.Stderr = os.Stderr
		if err := tidyCmd.Run(); err != nil {
			fmt.Printf("Warning: go mod tidy failed: %v\n", err)
			// Continue anyway, as go run might still work or give better error
		}

		// 5. Run the generated code
		fmt.Println("Running migration...")
		runCmd := exec.Command("go", "run", targetFile)
		runCmd.Stdout = os.Stdout
		runCmd.Stderr = os.Stderr
		runCmd.Stdin = os.Stdin
		if err := runCmd.Run(); err != nil {
			return fmt.Errorf("failed to run migration: %w", err)
		}

		return nil
	},
}

const migrateTemplate = `package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	_ "{{MODULE}}/model"
	_ "{{MODULE}}/module"

	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/middleware"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/module"
	"github.com/forbearing/gst/pkg/dbmigrate"
	"github.com/forbearing/gst/router"
)

func main() {
	// Initialize system components and suppress stdout during initialization
	// to avoid cluttering the migration output.
	initComponents()
	// Ensure config resources are cleaned up when the program exits.
	defer config.Clean()

	// Wait for all models to be registered.
	// This sleep is necessary because some models might be registered asynchronously
	// or during the initialization phase of modules.
	time.Sleep(1 * time.Second)

	// Collect all registered models.
	models := collectModels()

	// Dump the schema for the collected models.
	schema, err := dumpSchema(models)
	if err != nil {
		panic(err)
	}

	// Write the schema to a file for reference or debugging.
	if err := os.WriteFile("schema.sql", []byte(schema), 0o644); err != nil {
		fmt.Printf("Failed to write schema.sql: %v\n", err)
	}

	// Get database configuration based on the configured database type.
	dbConfig := getDatabaseConfig()

	// Perform migration.
	if err := performMigration(schema, dbConfig); err != nil {
		panic(err)
	}
}

// initComponents initializes the application components (config, middleware, router, module).
// It temporarily suppresses stdout to prevent initialization logs from appearing in the console.
func initComponents() {
	oldStdout := os.Stdout
	null, err := os.Open(os.DevNull)
	if err != nil {
		panic(err)
	}
	os.Stdout = null
	defer func() {
		os.Stdout = oldStdout
		null.Close()
	}()

	if err = config.Init(); err != nil {
		panic(err)
	}
	if err = middleware.Init(); err != nil {
		panic(err)
	}
	if err = router.Init(); err != nil {
		panic(err)
	}
	if err = module.Init(); err != nil {
		panic(err)
	}
}

// collectModels collects all models registered in model.TableChan.
func collectModels() []any {
	models := make([]any, 0)
	maxCount := len(model.TableChan)
	if maxCount == 0 {
		return models
	}
	count := 0
	for m := range model.TableChan {
		models = append(models, m)
		count++
		if count >= maxCount {
			break
		}
	}
	return models
}

// dumpSchema creates a schema dump for the provided models using the configured database type.
func dumpSchema(models []any) (string, error) {
	dumper, err := dbmigrate.NewSchemaDumper()
	if err != nil {
		return "", err
	}
	dbtyp := config.App.Database.Type
	return dumper.Dump(dbtyp, models...)
}

// getDatabaseConfig constructs the database configuration based on the application config.
func getDatabaseConfig() *dbmigrate.DatabaseConfig {
	var cfg *dbmigrate.DatabaseConfig
	switch config.App.Database.Type {
	case config.DBMySQL:
		cfg = &dbmigrate.DatabaseConfig{
			Host:     config.App.MySQL.Host,
			Port:     int(config.App.MySQL.Port),
			Database: config.App.MySQL.Database,
			Username: config.App.MySQL.Username,
			Password: config.App.MySQL.Password,
		}
	case config.DBPostgres:
		cfg = &dbmigrate.DatabaseConfig{
			Host:     config.App.Postgres.Host,
			Port:     int(config.App.Postgres.Port),
			Database: config.App.Postgres.Database,
			Username: config.App.Postgres.Username,
			Password: config.App.Postgres.Password,
			SSLMode:  config.App.Postgres.SSLMode,
		}
	case config.DBSqlite:
		cfg = &dbmigrate.DatabaseConfig{
			Database: config.App.Sqlite.Database,
		}
	default:
		panic("unsupported database type: " + config.App.Database.Type)
	}
	return cfg
}

// performMigration executes the migration process: dry run, confirmation, and actual execution.
func performMigration(schema string, cfg *dbmigrate.DatabaseConfig) error {
	dbtyp := config.App.Database.Type

	// Dry Run: Check for changes without executing.
	hasChange, err := dbmigrate.Migrate([]string{schema}, dbtyp, cfg, &dbmigrate.MigrateOption{
		DryRun:     true,
		EnableDrop: true,
	})
	if err != nil {
		return err
	}

	if !hasChange {
		fmt.Println("No changes detected.")
		return nil
	}

	// Confirm execution with the user.
	if !confirmExecution() {
		fmt.Println("Migration canceled.")
		return nil
	}

	// Execute Migration.
	_, err = dbmigrate.Migrate([]string{schema}, dbtyp, cfg, &dbmigrate.MigrateOption{
		DryRun:     false,
		EnableDrop: true,
	})
	if err != nil {
		return err
	}
	fmt.Println("Migration executed successfully.")
	return nil
}

// confirmExecution prompts the user for confirmation to proceed.
func confirmExecution() bool {
	fmt.Print("\nDo you want to execute the migration? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	return strings.ToLower(input) == "y" || strings.ToLower(input) == "yes"
}
`
