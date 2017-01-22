package lexer

var (
	DefaultLexer = `
		unicode_letter = "a" … "z" | "A" … "Z"
		unicode_digit  = "0" … "9"

		letter        = unicode_letter | "_" .
		decimal_digit = "0" … "9" .
		octal_digit   = "0" … "7" .
		hex_digit     = "0" … "9" | "A" … "F" | "a" … "f" .

		identifier = letter { letter | unicode_digit } .

		int_lit     = decimal_lit | octal_lit | hex_lit .
		decimal_lit = ( "1" … "9" ) { decimal_digit } .
		octal_lit   = "0" { octal_digit } .
		hex_lit     = "0" ( "x" | "X" ) hex_digit { hex_digit } .

		float_lit = decimals "." [ decimals ] [ exponent ] |
		            decimals exponent |
		            "." decimals [ exponent ] .
		decimals  = decimal_digit { decimal_digit } .
		exponent  = ( "e" | "E" ) [ "+" | "-" ] decimals .
	`
)
