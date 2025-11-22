package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v3"
)

var (
	listConfigs  bool
	outputFormat string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long:  "Configuration management commands for gst framework",
}

var dumpCmd = &cobra.Command{
	Use:   "dump [config-name]",
	Short: "Dump default configuration",
	Long: `Dump default configuration to stdout.

Examples:
  gg config dump                    # Dump all default configurations (JSON format)
  gg config dump --list             # List available configuration names
  gg config dump --format yaml     # Dump all configurations in YAML format
  gg config dump --format ini      # Dump all configurations in INI format
  gg config dump redis             # Dump redis configuration only (JSON format)
  gg config dump redis --format yaml  # Dump redis configuration in YAML format`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDump,
}

func init() {
	dumpCmd.Flags().BoolVar(&listConfigs, "list", false, "List available configuration names")
	dumpCmd.Flags().StringVarP(&outputFormat, "format", "f", "json", "Output format (json, yaml, ini)")
	configCmd.AddCommand(dumpCmd)
}

// getAvailableConfigs returns a map of available configuration names and their types
func getAvailableConfigs() map[string]reflect.Type {
	configs := make(map[string]reflect.Type)

	// Get the Config struct type
	configType := reflect.TypeFor[config.Config]()

	// Iterate through all fields in the Config struct
	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)

		// Get the json tag name as the config name
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			// Remove omitempty and other options from tag
			configName := strings.Split(jsonTag, ",")[0]
			configs[configName] = field.Type
		}
	}

	return configs
}

func runDump(cmd *cobra.Command, args []string) error {
	availableConfigs := getAvailableConfigs()

	// If --list flag is provided, list all available configurations
	if listConfigs {
		fmt.Println("Available configurations:")

		// Sort config names for consistent output
		var names []string
		for name := range availableConfigs {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			fmt.Printf("  %s\n", name)
		}
		return nil
	}

	// Validate output format
	if outputFormat != "json" && outputFormat != "yaml" && outputFormat != "ini" {
		return fmt.Errorf("unsupported format: %s. Supported formats: json, yaml, ini", outputFormat)
	}

	// If a specific config name is provided
	if len(args) == 1 {
		configName := args[0]

		// Check if the config exists
		if _, exists := availableConfigs[configName]; !exists {
			return fmt.Errorf("configuration '%s' not found. Use 'gg config dump --list' to see available configurations", configName)
		}

		// Initialize config to get default values
		if err := config.Init(); err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}
		defer config.Clean()

		// Get the specific configuration value using reflection
		configValue := reflect.ValueOf(config.App).Elem()
		configType := reflect.TypeFor[config.Config]()

		var specificConfig any
		for i := 0; i < configType.NumField(); i++ {
			field := configType.Field(i)
			jsonTag := field.Tag.Get("json")
			if jsonTag != "" && jsonTag != "-" {
				name := strings.Split(jsonTag, ",")[0]
				if name == configName {
					specificConfig = configValue.Field(i).Interface()
					break
				}
			}
		}

		if specificConfig == nil {
			return fmt.Errorf("configuration '%s' not found", configName)
		}

		// Create a map with the specific config for consistent output format
		configMap := map[string]any{
			configName: specificConfig,
		}

		// Create temporary file with appropriate extension
		tempFile := fmt.Sprintf("temp_config_%s.%s", configName, outputFormat)
		defer os.Remove(tempFile)

		// Format and output the specific configuration
		switch outputFormat {
		case "json":
			content, err := json.MarshalIndent(configMap, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Print(string(content))
		case "yaml":
			content, err := yaml.Marshal(configMap)
			if err != nil {
				return fmt.Errorf("failed to marshal YAML: %w", err)
			}
			fmt.Print(string(content))
		case "ini":
			// For INI format, we need to handle it specially
			cfg := ini.Empty()
			section, err := cfg.NewSection(configName)
			if err != nil {
				return fmt.Errorf("failed to create INI section: %w", err)
			}

			// Convert the config struct to key-value pairs
			if err := convertStructToINI(section, specificConfig); err != nil {
				return fmt.Errorf("failed to convert config to INI: %w", err)
			}

			var buf bytes.Buffer
			if _, err := cfg.WriteTo(&buf); err != nil {
				return fmt.Errorf("failed to write INI: %w", err)
			}
			fmt.Print(buf.String())
		}

		return nil
	}

	// Save stdout and redirect to capture output
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Create a pipe to capture output
	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}
	defer r.Close()
	defer w.Close()

	// Redirect stdout to pipe
	os.Stdout = w

	// Initialize config
	if err = config.Init(); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}
	defer config.Clean()

	// Create temporary file with appropriate extension
	tempFile, err := os.CreateTemp("", fmt.Sprintf("temp_config.*.%s", outputFormat))
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	// Save config to temporary file
	if err = config.Save(tempFile); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read the generated file and output to stdout
	content, err := os.ReadFile(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to read generated config file: %w", err)
	}

	fmt.Print(string(content))
	return nil
}

// convertStructToINI converts a struct to INI key-value pairs
func convertStructToINI(section *ini.Section, value any) error {
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		// For non-struct values, convert directly to string
		section.Key("value").SetValue(fmt.Sprintf("%v", value))
		return nil
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		if !fieldValue.CanInterface() {
			continue
		}

		// Use field name or json tag as key
		keyName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			keyName = strings.Split(jsonTag, ",")[0]
		}

		// Convert field value to string
		switch fieldValue.Kind() {
		case reflect.String:
			section.Key(keyName).SetValue(fieldValue.String())
		case reflect.Bool:
			section.Key(keyName).SetValue(fmt.Sprintf("%t", fieldValue.Bool()))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			section.Key(keyName).SetValue(fmt.Sprintf("%d", fieldValue.Int()))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			section.Key(keyName).SetValue(fmt.Sprintf("%d", fieldValue.Uint()))
		case reflect.Float32, reflect.Float64:
			section.Key(keyName).SetValue(fmt.Sprintf("%g", fieldValue.Float()))
		case reflect.Slice, reflect.Array:
			// Handle slices by joining with commas
			var strValues []string
			for j := 0; j < fieldValue.Len(); j++ {
				strValues = append(strValues, fmt.Sprintf("%v", fieldValue.Index(j).Interface()))
			}
			section.Key(keyName).SetValue(strings.Join(strValues, ","))
		default:
			section.Key(keyName).SetValue(fmt.Sprintf("%v", fieldValue.Interface()))
		}
	}

	return nil
}

// GGConfig represents the configuration for gg command
type GGConfig struct {
	Prune PruneConfig `mapstructure:"prune" yaml:"prune"`
}

// PruneConfig represents the prune configuration
type PruneConfig struct {
	Ignore []string `mapstructure:"ignore" yaml:"ignore"`
}

var ggConfig *GGConfig

// loadGGConfig loads the .gg.yaml configuration file
func loadGGConfig() (*GGConfig, error) {
	if ggConfig != nil {
		return ggConfig, nil
	}

	// Initialize viper
	v := viper.New()
	v.SetConfigName(".gg")
	v.SetConfigType("yaml")

	// Look for config file in current directory
	v.AddConfigPath(".")

	// Try to read the config file
	if err := v.ReadInConfig(); err != nil {
		// Config file not found is not an error, return default config
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			ggConfig = &GGConfig{
				Prune: PruneConfig{
					Ignore: []string{},
				},
			}
			return ggConfig, nil
		}
		// Other errors should be returned
		return nil, errors.Wrap(err, "failed to read config file")
	}

	// Unmarshal config
	cfg := &GGConfig{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal config")
	}

	ggConfig = cfg
	return ggConfig, nil
}

// getPruneIgnorePatterns returns the list of ignore patterns from config
func getPruneIgnorePatterns() []string {
	cfg, err := loadGGConfig()
	if err != nil {
		// If config loading fails, return empty list
		return []string{}
	}
	return cfg.Prune.Ignore
}
