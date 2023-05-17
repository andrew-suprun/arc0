package ui

type Position struct {
	X int
	Y int
}

type Size struct {
	Width  int
	Height int
}

type Flex struct {
	X int
	Y int
}

type Constraint struct {
	Size
	Flex
}
