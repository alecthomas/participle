package main

import (
	"testing"

	require "github.com/alecthomas/assert/v2"
)

func TestExe(t *testing.T) {
	_, err := parser.ParseString("", `
syntax = "proto3";

package test.test;

message SearchRequest {
  string query = 1;
  int32 page_number = 2;
  int32 result_per_page = 3;
  map<string, double> scores = 4;

  message Foo {}

  enum Bar {
    FOO = 0;
  }
}

message SearchResponse {
  string results = 1;
}

enum Type {
  INT = 0;
  DOUBLE = 1;
}

service SearchService {
  rpc Search(SearchRequest) returns (SearchResponse);
}
`)
	require.NoError(t, err)
}

func TestComments(t *testing.T) {
	proto, err := parser.ParseString("", `
// Package documentation
syntax = "proto3";

/* Block comment
   describing package */
package com.book;

// Comment before message
message Book {
    // Field comment
    int64 isbn = 1;
    string title = 2; // inline comment
    string author = 3;
}
`)
	require.NoError(t, err)
	require.Equal(t, "// Package documentation", proto.Entries[0].Comment)
	require.Equal(t, `"proto3"`, proto.Entries[1].Syntax)
	require.Equal(t, "/* Block comment\n   describing package */", proto.Entries[2].Comment)
	require.Equal(t, "com.book", proto.Entries[3].Package)
	require.Equal(t, "// Comment before message", proto.Entries[4].Comment)
	require.Equal(t, "Book", proto.Entries[5].Message.Name)
}
