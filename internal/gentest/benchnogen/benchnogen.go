// nolint: golint, stylecheck
package benchnogen

type Value struct {
	String string `  @String`
	Number int    `| @Int`
}

type Entry struct {
	Key   string `@Ident "="`
	Value *Value `@@`
}

type AST struct {
	Entries []*Entry `@@*`
}
