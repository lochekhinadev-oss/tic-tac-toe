package domain

const (
	BoardSize = 3

	CellEmpty    = 0
	CellUser     = 1
	CellComputer = 2
	CellX        = CellUser
	CellO        = CellComputer
)

type Field [][]int
