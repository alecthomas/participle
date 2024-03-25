module github.com/alecthomas/participle/v2/_examples

go 1.18

require (
	github.com/alecthomas/assert/v2 v2.6.0
	github.com/alecthomas/go-thrift v0.0.0-20220915213326-b383ff0e9ca1
	github.com/alecthomas/kong v0.8.1
	github.com/alecthomas/participle/v2 v2.1.0
	github.com/alecthomas/repr v0.4.0
)

require github.com/hexops/gotextdiff v1.0.3 // indirect

replace github.com/alecthomas/participle/v2 => ../
