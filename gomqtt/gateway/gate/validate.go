package gate

/* 登录验证模块 */

import "im.zgl/talents"

var validated = make(map[string]string)

func validate(u []byte, p []byte) bool {
	pw, ok := validated[string(u)]
	if !ok {
		// 验证

		// 验证通过
		validated[string(u)] = talents.Bytes2String(p)
		return true
	}

	if pw == talents.Bytes2String(p) {
		// 之前已经登陆过
		return true
	}

	// 缓存不通过，重新验证

	//if 验证通过
	validated[string(u)] = talents.Bytes2String(p)
	return true

	// else 验证失败

	// 连续验证失败N次，恶意攻击
}
