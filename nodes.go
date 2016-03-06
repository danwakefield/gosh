package main

type Node interface {
	NodeType() NodeType
}

type NodeType int

const (
	NCommand NodeType = iota
	NPipe
	NRedirection
	NBackground
	NSubshell
	NAnd
	NOr
	NSemicolon
	NIf
	NWhile
	NUntil
	NFor
	NCase
	NFunction
	NArg
	NRedirTo
	NRedirAppend
	NRedirClobber
	NNot
)
