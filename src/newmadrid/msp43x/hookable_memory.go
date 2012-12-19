package msp43x
//import "fmt"

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

func NewHookableMemory(mem *SimpleMemory) *HookableMemory { 
	return &HookableMemory{
		mem: *mem,
        read_hooks: make(map[uint16]ReadHook),
        write_hooks: make(map[uint16]WriteHook),
	}
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

func (self *HookableMemory) Load6Bytes(address uint16) ([]byte, error) {
	if _, ok := self.read_hooks[address]; ok {
		return nil, newError(E_BadAddressFault, "can't load instruction from addr")
	}

	return self.mem.Load6Bytes(address)
}

// bypasses hooks
func (self *HookableMemory) LoadWordDirect(address uint16) (uint16, error) {
	return self.mem.LoadWord(address)
}

func (self *HookableMemory) LoadWord(address uint16) (uint16, error) {
	if hook, ok := self.read_hooks[address]; ok {
		val, err :=  hook.ReadMemory(address, self)
        // only return the value from the hook if there was no error.
        if err == nil {
                return val, err
        }

	}

	return self.mem.LoadWord(address)
}

func (self *HookableMemory) LoadByte(address uint16) (uint8, error) {
	if hook, ok := self.read_hooks[address]; ok {
		i, e := hook.ReadMemory(address, self)
		return uint8(i & 0xff), e
	}

	return self.mem.LoadByte(address)
}

// bypasses hooks
func (self *HookableMemory) StoreWordDirect(address uint16, value uint16) error {
	return self.mem.StoreWord(address, value)
}


func (self *HookableMemory) StoreWord(address uint16, value uint16) error {
	if hook, ok := self.write_hooks[address]; ok {
		return hook.WriteMemory(address, value, self)
	}

	return self.mem.StoreWord(address, value)
}

func (self *HookableMemory) StoreByte(address uint16, value uint8) error {
	if hook, ok := self.write_hooks[address]; ok {
		return hook.WriteMemory(address, uint16(value), self)
	}

	return self.mem.StoreByte(address, value)
}

func (self *HookableMemory) String() string {
	return self.mem.String()
}

func (self *HookableMemory) Clear() {
	self.mem.Clear()
}

func (self *HookableMemory) Read(address uint16, len uint16) ([]byte, error) {
	return self.mem.Read(address, len)
}
