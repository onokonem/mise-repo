package main

import "fmt"

func main() { fmt.Println(Greet("sandbox")) }

func Greet(name string) string { return "hello, " + name }
