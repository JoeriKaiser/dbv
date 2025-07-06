# MVP, not production ready

# Basic usage
go run main.go -d "postgres://user:pass@localhost/mydb" -f mermaid -o schema.md

# MySQL with PlantUML
go run main.go -d "mysql://user:pass@localhost/mydb" -f plantuml -o schema.puml

# SQLite with Graphviz
go run main.go -d "sqlite:///path/to/database.db" -f graphviz -o schema.dot

# Include views and exclude certain tables
go run main.go -d "postgres://user:pass@localhost/mydb" -v -e migrations,logs -f mermaid

# Only include specific tables
go run main.go -d "postgres://user:pass@localhost/mydb" -i users,posts,comments -f mermaid

## Building and Installing

```bash
go build -o dbv

go install

GOOS=linux GOARCH=amd64 go build -o dbv-linux
GOOS=windows GOARCH=amd64 go build -o dbv-windows.exe
GOOS=darwin GOARCH=amd64 go build -o dbv-darwin
```

## Features

- ✅ Support for PostgreSQL, MySQL, and SQLite
- ✅ Multiple output formats (Mermaid, PlantUML, Graphviz)
- ✅ Include/exclude tables and views
- ✅ Foreign key relationship detection
- ✅ Primary key and column type information
- ✅ Configuration file support
- ✅ Cross-platform compatibility