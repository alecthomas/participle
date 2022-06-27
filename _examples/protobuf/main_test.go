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
