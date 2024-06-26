// Copyright 2023 james dotter.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://github.com/jcdotter/go/LICENSE
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser

import (
	"testing"

	"github.com/jcdotter/go/test"
)

func TestNum(t *testing.T) {
	var n []byte
	n, _ = Num([]byte("123 "), 0)
	test.Assert(t, Int(n), 123, "parse int")
	n, _ = Num([]byte("123.456 "), 0)
	test.Assert(t, Float(n), 123.456, "parse float")
	n, _ = Num([]byte("123.456e-2 "), 0)
	test.Assert(t, Float(n), 1.23456, "parse exponent")
}

func TestString(t *testing.T) {
	var b []byte
	var s string
	b, _ = StringLit([]byte("\"hello\nworld\" "), 0)
	test.Assert(t, string(b), "\"hello\nworld\"", "parse string literal")
	s = String(b)
	test.Assert(t, s, "hello\nworld", "parse string")
}

func TestBool(t *testing.T) {
	var b []byte
	b, _ = Bool([]byte("true "), 0)
	test.Assert(t, string(b), "true", "parse bool")
	b, _ = Bool([]byte("false "), 0)
	test.Assert(t, string(b), "false", "parse bool")
}

func TestNull(t *testing.T) {
	var b []byte
	b, _ = Null([]byte("null "), 0)
	test.Assert(t, string(b), "null", "parse null")
}

func TestExists(t *testing.T) {
	ok, _ := Exists([]byte("ts"), []byte("exists "), 0)
	test.Assert(t, false, ok, "parse exists fail")
	ok, _ = Exists([]byte("ex"), []byte("exists "), 0)
	test.Assert(t, true, ok, "parse exists pass")
}

func TestSearch(t *testing.T) {
	_, i := Search([]byte{'}'}, []byte("{{}}"), 0)
	test.Assert(t, 2, i, "search")
	_, i = Find('}', []byte("{{}}"), 0)
	test.Assert(t, 2, i, "find")
}
