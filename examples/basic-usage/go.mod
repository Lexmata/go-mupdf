module example-basic-usage

go 1.24.0

require bitbucket.org/lexmata/go-mupdf v1.0.0

require (
	github.com/clipperhouse/uax29/v2 v2.2.0 // indirect
	github.com/hhrutter/lzw v1.0.0 // indirect
	github.com/hhrutter/pkcs7 v0.2.0 // indirect
	github.com/hhrutter/tiff v1.0.2 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/pdfcpu/pdfcpu v0.11.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/image v0.32.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

// For local development, use replace directive to point to parent directory
replace bitbucket.org/lexmata/go-mupdf => ../..
