SHELL=/bin/bash -o pipefail

refresh-codegen: codegen-lexer cleanup-internal-participle-cmd

internal-participle-cmd:
	@echo Building private build of participle utility...
	cd cmd/participle && go build -o ../../participle github.com/alecthomas/participle/v2/cmd/participle

internal-generateinternal:
	@echo Updating internal test files...
	cd lexer/internal/participlegeninternal && go build -o ../../../participlegeninternal github.com/alecthomas/participle/v2/lexer/internal/participlegeninternal
	./participlegeninternal

codegen-lexer: internal-participle-cmd internal-generateinternal
	@echo Regenerating lexers...
	(./participle gen lexer --name GeneratedA internal < lexer/internal/alexer.json | gofmt > lexer/internal/alexer.go) || rm -f lexer/internal/alexer.go
	(./participle gen lexer --name GeneratedBasic internal < lexer/internal/basiclexer.json | gofmt > lexer/internal/basiclexer.go) || rm -f lexer/internal/basiclexer.go
	(./participle gen lexer --name GeneratedHeredoc internal < lexer/internal/heredoclexer.json | gofmt > lexer/internal/heredoclexer.go) || rm -f lexer/internal/heredoclexer.go
	(./participle gen lexer --name GeneratedHeredocWithWhitespace internal < lexer/internal/heredocwithwhitespacelexer.json | gofmt > lexer/internal/heredocwithwhitespacelexer.go) || rm -f lexer/internal/heredocwithwhitespacelexer.go
	(./participle gen lexer --name GeneratedInterpolated internal < lexer/internal/interpolatedlexer.json | gofmt > lexer/internal/interpolatedlexer.go) || rm -f lexer/internal/interpolatedlexer.go
	(./participle gen lexer --name GeneratedInterpolatedWithWhitespace internal < lexer/internal/interpolatedwithwhitespacelexer.json | gofmt > lexer/internal/interpolatedwithwhitespacelexer.go) || rm -f lexer/internal/interpolatedwithwhitespacelexer.go
	(./participle gen lexer --name GeneratedReference internal < lexer/internal/referencelexer.json | gofmt > lexer/internal/referencelexer.go) || rm -f lexer/internal/referencelexer.go

cleanup-internal-participle-cmd:
	@echo Cleaning up...
	rm -f ./participle
	rm -f ./participlegeninternal
