package uuid

import (
	"strconv"

	"github.com/uber-go/zap"
)

var sid int
var gen Generator
var Logger zap.Logger

func Init(i int, t string, l zap.Logger) {
	sid = i
	Logger = l
	switch t {
	case "ts16":
		g := &Ts16{}
		g.Start()

	case "standard":
		g := &Standard{}
		g.Start()

	default:
		Logger.Fatal("init uuid error: invalid type", zap.String("type", t))
	}
}

func Gen() int64 {
	s := gen.Gen()

	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		Logger.Fatal("gen id fatal error", zap.Error(err), zap.String("gens", s))
	}

	return id
}

func GenStr() string {
	s := gen.Gen()
	return s
}
