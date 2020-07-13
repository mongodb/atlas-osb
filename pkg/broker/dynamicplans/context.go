package dynamicplans

type Context map[string]interface{}

func (c Context) With(key string, value interface{}) Context {
	newCtx := Context{}
	for k, v := range c {
		newCtx[k] = v
	}
	newCtx[key] = value
	return newCtx
}
