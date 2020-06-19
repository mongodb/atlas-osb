package broker

type Mode int

const (
	BasicAuth Mode = iota
	MultiGroup
	MultiGroupAutoPlans
	DynamicPlans
)
