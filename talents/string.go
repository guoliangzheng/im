package talents

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"unsafe"
)

//生成随机字符串
func RandString() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

// zero-copy, []byte转为string类型
// 注意，这种做法下，一旦[]byte变化，string也会变化
// 谨慎，黑科技！！除非性能瓶颈，否则请使用string(b)1

func Bytes2String(b []byte) (s string) {
	return *(*string)(unsafe.Pointer(&b))
	// pb := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	// ps := (*reflect.StringHeader)(unsafe.Pointer(&s))
	// ps.Data = pb.Data
	// ps.Len = pb.Len
	// return
}

// zero-coy, string类型转为[]byte
// 注意，这种做法下，一旦string变化，程序立马崩溃且不能recover
// 谨慎，黑科技！！除非性能瓶颈，否则请使用[]byte(s)
func String2Bytes(s string) (b []byte) {
	return *(*[]byte)(unsafe.Pointer(&s))
	// pb := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	// ps := (*reflect.StringHeader)(unsafe.Pointer(&s))
	// pb.Data = ps.Data
	// pb.Len = ps.Len
	// pb.Cap = ps.Len
	// return
}
