package main

import (
  "strings"
  "testing"

  "github.com/gotestyourself/gotestyourself/assert"

  "github.com/alecthomas/go-thrift/parser"

  "github.com/alecthomas/participle"
)

var (
  source = strings.TrimSpace(`
namespace cpp thrift.example
namespace java thrift.example

enum TweetType {
    TWEET
    RETWEET = 2
    DM = 3
    REPLY
}

struct Location {
    1: required double latitude
    2: required double longitude
}

struct Tweet {
    1: required i32 userId
    2: required string userName
    3: required string text
    4: optional Location loc
    5: optional TweetType tweetType = TweetType.TWEET
    16: optional string language = "english"
}

typedef list<Tweet> TweetList

struct TweetSearchResult {
    1: TweetList tweets
}

exception TwitterUnavailable {
    1: string message
}

const i32 MAX_RESULTS = 100

service Twitter {
    void ping()
    bool postTweet(1:Tweet tweet) throws (1:TwitterUnavailable unavailable)
    TweetSearchResult searchTweets(1:string query)
    void zip()
}
`)
)

func BenchmarkParticipleThrift(b *testing.B) {
  b.ReportAllocs()
  parser, err := participle.Build(&Thrift{}, nil)
  assert.NilError(b, err)

  thrift := &Thrift{}
  err = parser.ParseString(source, thrift)
  assert.NilError(b, err)

  b.ResetTimer()

  for i := 0; i < b.N; i++ {
    thrift := &Thrift{}
    _ = parser.ParseString(source, thrift)
  }
}

func BenchmarkGoThriftParser(b *testing.B) {
  b.ReportAllocs()
  _, err := parser.ParseReader("user.thrift", strings.NewReader(source))
  assert.NilError(b, err)

  b.ResetTimer()

  for i := 0; i < b.N; i++ {
    _, _ = parser.ParseReader("user.thrift", strings.NewReader(source))
  }
}
