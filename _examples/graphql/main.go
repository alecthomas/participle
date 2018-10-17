package main

import (
	"github.com/alecthomas/participle"
	"github.com/alecthomas/repr"
)

type File struct {
	Entries []*Entry `{ @@ }`
}

type Entry struct {
	Type   *Type   `  @@`
	Schema *Schema `| @@`
	Enum   *Enum   `| @@`
	Scalar string  `| "scalar" @Ident`
}

type Enum struct {
	Name  string   `"enum" @Ident`
	Cases []string `"{" { @Ident } "}"`
}

type Schema struct {
	Fields []*Field `"schema" "{" { @@ } "}"`
}

type Type struct {
	Name       string   `"type" @Ident`
	Implements string   `[ "implements" @Ident ]`
	Fields     []*Field `"{" { @@ } "}"`
}

type Field struct {
	Name       string      `@Ident`
	Arguments  []*Argument `[ "(" [ @@ { "," @@ } ] ")" ]`
	Type       *TypeRef    `":" @@`
	Annotation string      `[ "@" @Ident ]`
}

type Argument struct {
	Name    string   `@Ident`
	Type    *TypeRef `":" @@`
	Default *Value   `[ "=" @@ ]`
}

type TypeRef struct {
	Array       *TypeRef `(   "[" @@ "]"`
	Type        string   `  | @Ident )`
	NonNullable bool     `[ @"!" ]`
}

type Value struct {
	Symbol string `@Ident`
}

var parser = participle.MustBuild(&File{})

func main() {
	ast := &File{}
	err := parser.ParseString(`
type Tweet {
    id: ID!
    // The tweet text. No more than 140 characters!
    body: String
    // When the tweet was published
    date: Date
    // Who published the tweet
    Author: User
    // Views, retweets, likes, etc
    Stats: Stat
}

type User {
    id: ID!
    username: String
    first_name: String
    last_name: String
    full_name: String
    name: String @deprecated
    avatar_url: Url
}

type Stat {
    views: Int
    likes: Int
    retweets: Int
    responses: Int
}

type Notification {
    id: ID
    date: Date
    type: String
}

type Meta {
    count: Int
}

scalar Url
scalar Date

type Query {
    Tweet(id: ID!): Tweet
    Tweets(limit: Int, skip: Int, sort_field: String, sort_order: String): [Tweet]
    TweetsMeta: Meta
    User(id: ID!): User
    Notifications(limit: Int): [Notification]
    NotificationsMeta: Meta
}

type Mutation {
    createTweet (
        body: String
    ): Tweet
    deleteTweet(id: ID!): Tweet
    markTweetRead(id: ID!): Boolean
}

`, ast)
	if err != nil {
		panic(err)
	}
	repr.Println(ast)
}
