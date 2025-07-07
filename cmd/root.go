package cmd

import (
	"dbv/internal/database"
	"dbv/internal/generators"
	"dbv/internal/schema"
	"dbv/pkg/config"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"slices"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	styles = struct {
		success lipgloss.Style
		info    lipgloss.Style
		error   lipgloss.Style
	}{
		success: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true),
		info: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")),
		error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true),
	}
)

type AppConfig struct {
	DatabaseURL   string
	Format        string
	OutputFile    string
	IncludeViews  bool
	ExcludeTables []string
	IncludeTables []string
}

type ConnectionHistory struct {
	Recent []string `json:"recent"`
}

type formModel struct {
	form               *huh.Form
	config             AppConfig
	excludeTablesInput string
	includeTablesInput string
	connectionChoice   string
	customConnection   string
	format             string
	outputFile         string
	includeViews       bool
	quitting           bool
	cancelled          bool
}

var rootCmd = &cobra.Command{
	Use:   "dbv",
	Short: "Generate database schema visualizations",
	Long: `A CLI tool that connects to databases and generates schema visualizations
using Mermaid, PlantUML, or Graphviz formats.`,
	RunE: run,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	flags := rootCmd.Flags()
	flags.StringP("database-url", "d", "", "Database connection URL")
	flags.StringP("format", "f", "mermaid", "Output format: mermaid, plantuml, graphviz")
	flags.StringP("output", "o", "", "Output file path")
	flags.BoolP("include-views", "v", false, "Include database views")
	flags.StringSliceP("exclude-tables", "e", []string{}, "Tables to exclude")
	flags.StringSliceP("include-tables", "i", []string{}, "Only include these tables")
}

func run(cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed("database-url") {
		return runWithFlags(cmd)
	}

	return runInteractive()
}

func runWithFlags(cmd *cobra.Command) error {
	config := AppConfig{}

	config.DatabaseURL, _ = cmd.Flags().GetString("database-url")
	config.Format, _ = cmd.Flags().GetString("format")
	config.OutputFile, _ = cmd.Flags().GetString("output")
	config.IncludeViews, _ = cmd.Flags().GetBool("include-views")
	config.ExcludeTables, _ = cmd.Flags().GetStringSlice("exclude-tables")
	config.IncludeTables, _ = cmd.Flags().GetStringSlice("include-tables")

	return generateSchema(config)
}

func runInteractive() error {
	model := newFormModel()

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("problem running form: %w", err)
	}

	if m, ok := finalModel.(*formModel); ok {
		if m.cancelled {
			fmt.Println("Operation cancelled.")
			return nil
		}

		if m.config.DatabaseURL != "" {
			saveToHistory(m.config.DatabaseURL)
		}

		return generateSchema(m.config)
	}

	return fmt.Errorf("unexpected model type")
}

func newFormModel() *formModel {
	model := &formModel{
		format: "mermaid",
	}

	history := loadHistory()
	var connectionOptions []huh.Option[string]

	for _, conn := range history.Recent {
		connectionOptions = append(connectionOptions, huh.NewOption(conn, conn))
	}

	examples := []string{
		"postgres://user:pass@localhost/mydb",
		"mysql://user:pass@localhost/mydb",
		"sqlite://./database.db",
	}

	for _, example := range examples {
		if !contains(history.Recent, example) {
			connectionOptions = append(connectionOptions, huh.NewOption(example, example))
		}
	}
	connectionOptions = append(connectionOptions, huh.NewOption("Custom...", "custom"))

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose a Database Connection").
				Description("Select from recent connections or examples").
				Options(connectionOptions...).
				Value(&model.connectionChoice),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Database Connection URL").
				Placeholder("postgres://user:pass@localhost/db").
				Value(&model.customConnection).
				Validate(validateDatabaseURL),
		).WithHideFunc(func() bool {
			return model.connectionChoice != "custom"
		}),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Output Format").
				Options(
					huh.NewOption("Mermaid", "mermaid"),
					huh.NewOption("PlantUML", "plantuml"),
					huh.NewOption("Graphviz", "graphviz"),
				).
				Value(&model.format),
			huh.NewInput().
				Title("Output File (optional)").
				Placeholder("schema.md").
				Value(&model.outputFile),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Include Views?").
				Value(&model.includeViews),
			huh.NewInput().
				Title("Exclude Tables (comma-separated)").
				Placeholder("migrations,logs").
				Value(&model.excludeTablesInput),
			huh.NewInput().
				Title("Include Only Tables (comma-separated)").
				Placeholder("users,posts").
				Value(&model.includeTablesInput),
		),
	).WithTheme(huh.ThemeCharm()).
		WithShowHelp(true).
		WithShowErrors(true)

	model.form = form

	return model
}

func validateDatabaseURL(s string) error {
	if s == "" {
		return fmt.Errorf("database URL cannot be empty")
	}
	if _, _, err := database.ParseDatabaseURL(s); err != nil {
		return fmt.Errorf("invalid database URL: %w", err)
	}
	return nil
}

func generateSchema(appConfig AppConfig) error {
	if err := validateConfig(&appConfig); err != nil {
		return err
	}

	cfg := config.Config{
		Database: config.DatabaseConfig{
			URL: appConfig.DatabaseURL,
		},
		Output: config.OutputConfig{
			Format: appConfig.Format,
			File:   appConfig.OutputFile,
		},
		Schema: config.SchemaConfig{
			IncludeViews:  appConfig.IncludeViews,
			ExcludeTables: appConfig.ExcludeTables,
			IncludeTables: appConfig.IncludeTables,
		},
	}

	connector, err := database.NewConnector(cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("failed to create database connector: %w", err)
	}
	defer connector.Close()

	dbSchema, err := connector.ExtractSchema(cfg.Schema)
	if err != nil {
		return fmt.Errorf("failed to extract schema: %w", err)
	}

	content := generateContent(cfg.Output.Format, dbSchema)

	if err := writeOutput(cfg.Output.File, content); err != nil {
		return err
	}

	displayResults(cfg.Output.File, cfg.Output.Format, dbSchema)
	return nil
}

func validateConfig(config *AppConfig) error {
	if config.DatabaseURL == "" {
		return fmt.Errorf("database URL cannot be empty")
	}

	validFormats := []string{"mermaid", "plantuml", "graphviz"}
	if !contains(validFormats, config.Format) {
		return fmt.Errorf("invalid format '%s'. Valid formats: %s",
			config.Format, strings.Join(validFormats, ", "))
	}

	if config.OutputFile == "" {
		ext := map[string]string{
			"mermaid":  ".md",
			"plantuml": ".puml",
			"graphviz": ".dot",
		}
		config.OutputFile = "schema" + ext[config.Format]
	}

	return nil
}

func generateContent(format string, dbSchema *schema.Schema) string {
	switch format {
	case "mermaid":
		return generators.GenerateMermaid(dbSchema)
	case "plantuml":
		return generators.GeneratePlantUML(dbSchema)
	case "graphviz":
		return generators.GenerateGraphviz(dbSchema)
	default:
		return ""
	}
}

func writeOutput(filename, content string) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

func displayResults(filename, format string, dbSchema *schema.Schema) {
	fmt.Printf("%s %s\n", styles.success.Render("✓"), styles.info.Render("Schema generated: "+filename))
	fmt.Printf("  %s\n", styles.info.Render(fmt.Sprintf("Format: %s", format)))
	fmt.Printf("  %s\n", styles.info.Render(fmt.Sprintf("Tables: %d", len(dbSchema.Tables))))
	fmt.Printf("  %s\n", styles.info.Render(fmt.Sprintf("Relationships: %d", len(dbSchema.ForeignKeys))))
}

func getHistoryPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".dbv_history.json"
	}
	return filepath.Join(home, ".dbv_history.json")
}

func loadHistory() ConnectionHistory {
	var history ConnectionHistory

	data, err := os.ReadFile(getHistoryPath())
	if err != nil {
		return history
	}

	json.Unmarshal(data, &history)
	return history
}

func saveToHistory(connectionURL string) {
	history := loadHistory()

	for i, conn := range history.Recent {
		if conn == connectionURL {
			history.Recent = append(history.Recent[:i], history.Recent[i+1:]...)
			break
		}
	}

	history.Recent = append([]string{connectionURL}, history.Recent...)

	if len(history.Recent) > 10 {
		history.Recent = history.Recent[:10]
	}

	data, _ := json.MarshalIndent(history, "", "  ")
	os.WriteFile(getHistoryPath(), data, 0644)
}

func (m *formModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *formModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			m.quitting = true
			return m, tea.Quit
		}
	}

	if m.quitting {
		return m, nil
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		m.quitting = true
		m.processFormData()
		return m, tea.Quit
	}

	return m, cmd
}

func (m *formModel) View() string {
	if m.quitting {
		return ""
	}

	view := m.form.View()

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("Press Ctrl+C or Esc to quit")

	return view + "\n\n" + help
}

func (m *formModel) processFormData() {
	if m.connectionChoice == "custom" {
		m.config.DatabaseURL = m.customConnection
	} else {
		m.config.DatabaseURL = m.connectionChoice
	}

	m.config.Format = m.format
	m.config.OutputFile = m.outputFile
	m.config.IncludeViews = m.includeViews

	if m.excludeTablesInput != "" {
		m.config.ExcludeTables = parseTableList(m.excludeTablesInput)
	}
	if m.includeTablesInput != "" {
		m.config.IncludeTables = parseTableList(m.includeTablesInput)
	}
}

func parseTableList(input string) []string {
	if input == "" {
		return nil
	}

	tables := strings.Split(input, ",")
	for i, table := range tables {
		tables[i] = strings.TrimSpace(table)
	}

	return tables
}

func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}
