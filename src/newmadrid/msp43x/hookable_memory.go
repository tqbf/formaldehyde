package msp43x

// A read hook can be a function or an object with a "ReadMemory" function
type ReadHook interface {
	ReadMemory(addr uint16, mem Memory)(uint16, error)
}
type ReadHookFunc func(addr uint16, mem Memory)(uint16, error)
func (f ReadHookFunc) ReadMemory(addr uint16, mem Memory)(uint16, error) {
	return f(addr, mem)
}

// A write hook can be a function or an object with a "WriteMemory" function
type WriteHook interface {
	WriteMemory(addr, value uint16, mem Memory)(error)
}
type WriteHookFunc func(addr, value uint16, mem Memory)(error)
func (f WriteHookFunc) WriteMemory(addr, value uint16, mem Memory)(error) {
	return f(addr, value, mem)
}

// A SimpleMemory with hook functions assigned to specific regions of memory;
// instantiate directly.
type HookableMemory struct {
	mem		SimpleMemory
	read_hooks	map[uint16]ReadHook
	write_hooks	map[uint16]WriteHook
}

// Assign a hook to a specific word in memory; note that this hook will be 
// called for both bytes and words (which means you can't have one hook handle
// both kinds)
func (mem *HookableMemory) ReadHook(addr uint16, h ReadHook) {
	mem.read_hooks[addr] = h
}

// Assign a hook to a specific word in memory; note that this hook will be 
// called for both bytes and words (which means you can't have one hook handle
// both kinds)
func (mem *HookableMemory) WriteHook(addr uint16, h WriteHook) {
	mem.write_hooks[addr] = h
}

func (mem *HookableMemory) Load6Bytes(address uint16) ([]byte, error) {
	if _, ok := mem.read_hooks[address]; ok {
		return nil, newError(E_BadAddressFault, "can't load instruction from addr")
	}

	return mem.Load6Bytes(address)
}

func (mem *HookableMemory) LoadWord(address uint16) (uint16, error) {
	if hook, ok := mem.read_hooks[address]; ok {
		return hook.ReadMemory(address, mem)
	}

	return mem.LoadWord(address)
}

func (mem *HookableMemory) LoadByte(address uint16) (uint8, error) {
	if hook, ok := mem.read_hooks[address]; ok {
		i, e := hook.ReadMemory(address, mem)
		return uint8(i & 0xff), e
	}

	return mem.LoadByte(address)
}

func (mem *HookableMemory) StoreWord(address uint16, value uint16) error {
	if hook, ok := mem.write_hooks[address]; ok {
		return hook.WriteMemory(address, value, mem)
	}

	return mem.StoreWord(address, value)
}

func (mem *HookableMemory) StoreByte(address uint16, value uint8) error {
	if hook, ok := mem.write_hooks[address]; ok {
		return hook.WriteMemory(address, uint16(value), mem)
	}

	return mem.StoreByte(address, value)
}

func (mem *HookableMemory) String() string {
	return mem.String()
}