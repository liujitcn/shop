package recommend

import "errors"

// ErrNotImplemented 表示当前仍处于设计骨架阶段，具体算法实现暂未落地。
var ErrNotImplemented = errors.New("recommend: not implemented")
