## Project setup

### Folder structure
- https://github.com/golang-standards/project-layout

### For Mac OS we need to send `-ldflags="-linkmode=external"` when building or running
```sh
go run -ldflags="-linkmode=external" cmd/gosync/main.go --source <source_path> --dest <destination_path>
```