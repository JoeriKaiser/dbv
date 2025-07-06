package config

type Config struct {
    Database DatabaseConfig `mapstructure:"database"`
    Output   OutputConfig   `mapstructure:"output"`
    Schema   SchemaConfig   `mapstructure:"schema"`
}

type DatabaseConfig struct {
    URL string `mapstructure:"url"`
}

type OutputConfig struct {
    Format string `mapstructure:"format"`
    File   string `mapstructure:"file"`
}

type SchemaConfig struct {
    IncludeViews   bool     `mapstructure:"include_views"`
    ExcludeTables  []string `mapstructure:"exclude_tables"`
    IncludeTables  []string `mapstructure:"include_tables"`
}