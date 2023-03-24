module github.com/alecthomas/participle/v2/_examples

go 1.18

require (
	github.com/alecthomas/assert/v2 v2.2.2
	github.com/alecthomas/go-thrift v0.0.0-20220915213326-b383ff0e9ca1
	github.com/alecthomas/kong v0.7.1
	github.com/alecthomas/participle/v2 v2.0.0
	github.com/alecthomas/repr v0.2.0
)

require github.com/hexops/gotextdiff v1.0.3 // indirect

replace github.com/alecthomas/participle/v2 => ../
