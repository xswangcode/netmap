//go:build !windows
// +build !windows

package singleinstance

// Instance 非 Windows 平台的单实例占位对象。
type Instance struct{}

// Acquire 非 Windows 平台暂不执行单实例检查。
func Acquire(name string) (*Instance, bool, error) {
	return &Instance{}, false, nil
}

// Release 与 Windows 实现保持接口一致。
func (i *Instance) Release() {}
