# `koff`

koff is a command-line tool that allows you to process kubernetes resources in yaml or json format, from either file or piped input.
It reads input, performs the specific filter operations based on the flags and arguments (if provided), and writes the output in either tabular (as default), json or yaml format.
<img src="./docs/images/preview.png" width="100%">

## Installation
### Using `go`
```
go install github.com/gmeghnag/koff 
```

