// Package http defines a storage backend over a HTTP API.
// This is meant to make basic interoperability possible for users
// who do not want to change their data model (or use views on compatible RDBMS systems)
//
// Users will only need to create compatible endpoints in their codebases that we can communicate with
// It is the users job at that point to do any conversions of data type, names, etc. required to be
// compatible with their system
package http
