package generators

import (
	"dbv/internal/schema"
	"fmt"
	"strings"
)

func GeneratePlantUML(s *schema.Schema) string {
    var builder strings.Builder

    builder.WriteString("@startuml\n")
    builder.WriteString("!theme plain\n")
    builder.WriteString("skinparam linetype ortho\n\n")

    for _, table := range s.Tables {
        builder.WriteString(fmt.Sprintf("entity \"%s\" as %s {\n", table.Name, cleanEntityName(table.Name)))
        
        for _, col := range table.Columns {
            if col.IsPrimaryKey {
                builder.WriteString(fmt.Sprintf("  * %s : %s <<PK>>\n", col.Name, formatPlantUMLType(col)))
            }
        }
        
        builder.WriteString("  --\n")
        
        for _, col := range table.Columns {
            if !col.IsPrimaryKey {
                nullStr := ""
                if !col.IsNullable {
                    nullStr = " <<NOT NULL>>"
                }
                builder.WriteString(fmt.Sprintf("  %s : %s%s\n", col.Name, formatPlantUMLType(col), nullStr))
            }
        }
        
        builder.WriteString("}\n\n")
    }

    for _, view := range s.Views {
        builder.WriteString(fmt.Sprintf("entity \"%s\" as %s <<view>> {\n", view.Name, cleanEntityName(view.Name)))
        
        for _, col := range view.Columns {
            builder.WriteString(fmt.Sprintf("  %s : %s\n", col.Name, formatPlantUMLType(col)))
        }
        
        builder.WriteString("}\n\n")
    }

    for _, fk := range s.ForeignKeys {
        builder.WriteString(fmt.Sprintf("%s ||--o{ %s : %s\n", 
            cleanEntityName(fk.ReferencedTable), 
            cleanEntityName(fk.Table), 
            fk.Column))
    }

    builder.WriteString("\n@enduml\n")

    return builder.String()
}

func formatPlantUMLType(col schema.Column) string {
    switch strings.ToLower(col.Type) {
    case "varchar", "text", "char", "string":
        if col.Length != nil {
            return fmt.Sprintf("VARCHAR(%d)", *col.Length)
        }
        return "VARCHAR"
    case "int", "integer":
        return "INTEGER"
    case "bigint":
        return "BIGINT"
    case "decimal", "numeric":
        if col.Precision != nil && col.Scale != nil {
            return fmt.Sprintf("DECIMAL(%d,%d)", *col.Precision, *col.Scale)
        }
        return "DECIMAL"
    case "boolean", "bool":
        return "BOOLEAN"
    case "date":
        return "DATE"
    case "timestamp", "datetime":
        return "TIMESTAMP"
    default:
        return strings.ToUpper(col.Type)
    }
}

func cleanEntityName(name string) string {
    name = strings.ReplaceAll(name, "-", "_")
    name = strings.ReplaceAll(name, ".", "_")
    name = strings.ReplaceAll(name, " ", "_")
    return name
}