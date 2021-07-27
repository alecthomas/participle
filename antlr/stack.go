package antlr

type strStack struct {
	stack []string
}

func (ss *strStack) push(s string) {
	ss.stack = append(ss.stack, s)
}

func (ss *strStack) pop() string { //nolint:unparam
	s := ss.peek()
	ss.stack = ss.stack[:len(ss.stack)-1]
	return s
}

func (ss *strStack) peek() string {
	return ss.stack[len(ss.stack)-1]
}

func (ss *strStack) safePeek() string {
	if len(ss.stack) == 0 {
		return ""
	}
	return ss.peek()
}

func (ss *strStack) contains(s string) bool {
	for _, v := range ss.stack {
		if v == s {
			return true
		}
	}
	return false
}

type boolStack struct {
	stack []bool
}

func (bs *boolStack) push(b bool) {
	bs.stack = append(bs.stack, b)
}

func (bs *boolStack) pop() bool { //nolint:unparam
	b := bs.peek()
	bs.stack = bs.stack[:len(bs.stack)-1]
	return b
}

func (bs *boolStack) peek() bool {
	return bs.stack[len(bs.stack)-1]
}

func (bs *boolStack) safePeek() bool {
	if len(bs.stack) == 0 {
		return false
	}
	return bs.peek()
}
