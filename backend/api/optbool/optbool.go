package optbool

type OptBool struct {
	Value bool
}

func (ob *OptBool) Bool() bool {
	return ob.Value
}
