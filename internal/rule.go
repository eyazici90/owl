package internal

type Rule struct {
	Group, Type, Name, Query, Labels string
	EvalDuration                     float64
}
