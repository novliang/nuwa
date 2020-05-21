package nuwa

// 补天用的石头
type Stone interface {
	// 石头发挥作用
	Work(args ...interface{}) error
}
