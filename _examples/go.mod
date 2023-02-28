module github.com/alecthomas/participle/v2/_examples

go 1.18

require (
	github.com/alecthomas/assert/v2 v2.1.0
	github.com/alecthomas/go-thrift v0.0.0-20170109061633-7914173639b2
	github.com/alecthomas/kong v0.7.1
	github.com/alecthomas/participle/v2 v2.0.0-alpha11
	github.com/alecthomas/repr v0.1.0
)

require github.com/hexops/gotextdiff v1.0.3 // indirect

replace github.com/alecthomas/participle/v2 => ../
