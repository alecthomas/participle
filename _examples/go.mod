module github.com/alecthomas/participle/v2/_examples

go 1.14

require (
	github.com/alecthomas/go-thrift v0.0.0-20170109061633-7914173639b2
	github.com/alecthomas/kong v0.2.17
	github.com/alecthomas/participle/v2 v2.0.0-alpha6.0.20210722022952-38e05ef27064
	github.com/alecthomas/repr v0.0.0-20200325044227-4184120f674c
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/stretchr/testify v1.7.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
)

replace github.com/alecthomas/participle/v2 => ../
