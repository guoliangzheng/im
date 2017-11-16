package gate

import (
	"fmt"

	"github.com/golang/snappy"
)

func compress(ci *connInfo, msg []byte) ([]byte, error) {
	needC := ci.compress % 10
	typeC := ci.compress / 100
	levelC := ((ci.compress % 100) - needC) / 10

	// 判断是否需要压缩
	if needC == 1 {
		switch typeC {
		// case 1: //gzip
		// 	out, err := gzipC(msg.Msg, levelC)
		// 	return out, err

		case 2: //snappy
			out := snappyC(msg, levelC)
			return out, nil

		default:
			return nil, fmt.Errorf("unsupported compress type: %v", ci.compress)
		}
	}

	return msg, nil
}

// func gzipC(data []byte, lv int) ([]byte, error) {
// 	return data, nil
// }

func snappyC(data []byte, lv int) []byte {
	out := snappy.Encode(nil, data)
	return out
}

func snappyUnC(data []byte) ([]byte, error) {
	out, err := snappy.Decode(nil, data)
	return out, err
}
