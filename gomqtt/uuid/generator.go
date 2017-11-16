package uuid

// Generator ... UUID生成器
type Generator interface {
	Start()
	Close()
	Gen() string
}
