package cmd

import (
	"dbv/internal/database"
	"dbv/internal/generators"
	"dbv/pkg/config"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
    cfgFile string
    cfg     config.Config
)

var rootCmd = &cobra.Command{
    Use:   "dbv",
    Short: "Generate database schema visualizations",
    Long: `A CLI tool that connects to databases and generates schema visualizations
using Mermaid, PlantUML, or Graphviz formats.

Examples:
  dbv -d "postgres://user:pass@localhost/mydb?sslmode=disable" -f mermaid -o schema.md
  dbv -d "mysql://user:pass@localhost/mydb" -f plantuml -o schema.puml
  dbv -d "sqlite:///path/to/db.sqlite" -f graphviz -o schema.dot`,
    RunE: runSchemaViz,
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    cobra.OnInitialize(initConfig)

    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.dbv.yaml)")
    rootCmd.Flags().StringP("database-url", "d", "", "Database connection URL (required)")
    rootCmd.Flags().StringP("format", "f", "mermaid", "Output format: mermaid, plantuml, graphviz")
    rootCmd.Flags().StringP("output", "o", "", "Output file path (default: schema.<format>)")
    rootCmd.Flags().BoolP("include-views", "v", false, "Include database views in output")
    rootCmd.Flags().StringSliceP("exclude-tables", "e", []string{}, "Tables to exclude from visualization")
    rootCmd.Flags().StringSliceP("include-tables", "i", []string{}, "Only include these tables (if specified)")

    rootCmd.MarkFlagRequired("database-url")

    viper.BindPFlag("database.url", rootCmd.Flags().Lookup("database-url"))
    viper.BindPFlag("output.format", rootCmd.Flags().Lookup("format"))
    viper.BindPFlag("output.file", rootCmd.Flags().Lookup("output"))
    viper.BindPFlag("schema.include_views", rootCmd.Flags().Lookup("include-views"))
    viper.BindPFlag("schema.exclude_tables", rootCmd.Flags().Lookup("exclude-tables"))
    viper.BindPFlag("schema.include_tables", rootCmd.Flags().Lookup("include-tables"))
}

func initConfig() {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        home, err := os.UserHomeDir()
        cobra.CheckErr(err)

        viper.AddConfigPath(home)
        viper.SetConfigType("yaml")
        viper.SetConfigName(".dbv")
    }

    viper.AutomaticEnv()
    viper.SetEnvPrefix("SCHEMA_VIZ")
    viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

    if err := viper.ReadInConfig(); err == nil {
        fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
    }
}

func runSchemaViz(cmd *cobra.Command, args []string) error {
    if err := viper.Unmarshal(&cfg); err != nil {
        return fmt.Errorf("failed to unmarshal config: %w", err)
    }

    validFormats := []string{"mermaid", "plantuml", "graphviz"}
    if !contains(validFormats, cfg.Output.Format) {
        return fmt.Errorf("invalid format '%s'. Valid formats: %s", cfg.Output.Format, strings.Join(validFormats, ", "))
    }

    if cfg.Output.File == "" {
        ext := map[string]string{
            "mermaid":  ".md",
            "plantuml": ".puml",
            "graphviz": ".dot",
        }
        cfg.Output.File = "schema" + ext[cfg.Output.Format]
    }

    connector, err := database.NewConnector(cfg.Database.URL)
    if err != nil {
        return fmt.Errorf("failed to create database connector: %w", err)
    }
    defer connector.Close()

    schema, err := connector.ExtractSchema(cfg.Schema)
    if err != nil {
        return fmt.Errorf("failed to extract schema: %w", err)
    }

    var content string
    switch cfg.Output.Format {
    case "mermaid":
        content = generators.GenerateMermaid(schema)
    case "plantuml":
        content = generators.GeneratePlantUML(schema)
    case "graphviz":
        content = generators.GenerateGraphviz(schema)
    }

    if err := os.MkdirAll(filepath.Dir(cfg.Output.File), 0755); err != nil {
        return fmt.Errorf("failed to create output directory: %w", err)
    }

    if err := os.WriteFile(cfg.Output.File, []byte(content), 0644); err != nil {
        return fmt.Errorf("failed to write output file: %w", err)
    }

    fmt.Printf("Schema visualization generated: %s\n", cfg.Output.File)
    fmt.Printf("Format: %s\n", cfg.Output.Format)
    fmt.Printf("Tables: %d\n", len(schema.Tables))
    fmt.Printf("Relationships: %d\n", len(schema.ForeignKeys))

    return nil
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}