# go-mupdf

A Go wrapper for the MuPDF library, providing PDF manipulation capabilities.

## Features

- PDF document handling
- Document information extraction
- Page rendering (planned)
- Text extraction (planned)

## Installation

```bash
go get bitbucket.org/lexmata/go-mupdf
```

## Usage


## Development

### Prerequisites

- Go 1.16 or higher
- MuPDF library (included as a git submodule tracking version 1.26.3)

### Setup

```bash
# Clone the repository with submodules
git clone --recurse-submodules bitbucket.org/lexmata/go-mupdf
# Or if you've already cloned the repository:
git submodule update --init --recursive
```

### Testing

```bash
go test ./...
```

## License

This project is dual-licensed under both the [MIT](LICENSE) and [Apache-2.0](LICENSE) licenses. You may choose either license at your option.