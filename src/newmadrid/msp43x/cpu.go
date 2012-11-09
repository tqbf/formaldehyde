// Package msp43x implements an emulator for a simplified and abstracted MSP430-series
// MCU; it can run C code generated by mspgcc.
package msp43x

import (
	"fmt"
	"bytes"
	"errors"
)

// What the CPU boots from and writes/reads from. A trivial memory is
// provided in SimpleMemory
type Memory interface {
	Load6Bytes(address uint16) ([]byte, error)
	LoadWord(address uint16) (uint16, error)
	LoadWordDirect(address uint16) (uint16, error)
	StoreWord(address uint16, value uint16) error
	StoreWordDirect(address uint16, value uint16) error
	LoadByte(address uint16) (uint8, error)
	StoreByte(address uint16, value uint8) error
    Read(address uint16, len uint16) ([]byte, error)
}

// The CPU (no user serviceable parts inside). Just instantiate directly.
type CPU struct {
	regs   [16]uint16
	memory Memory
}

// Set the initial values for registers. The code mspgcc produces expects
// PC (R0) to be 4400h; all other registers can be zero.
func (cpu *CPU) SetRegs(regs [16]uint16) {
	for i, v := range(regs) {
		cpu.regs[i] = v
	}
}

// For debugger purposes: get the whole register file
func (cpu *CPU) GetRegs()(regs [16]uint16) {
	return cpu.regs
}

// Set the image the CPU is running from. You can swap memories at runtime
// (as long as your compiled code can handle that)
func (cpu *CPU) SetMemory(memory Memory) {
	cpu.memory = memory
}

// The program counter (R0)
func (cpu *CPU) Pc() uint16 {
	return cpu.regs[0]
}

// The stack pointer (R1)
func (cpu *CPU) Sp() uint16 {
	return cpu.regs[1]
}

// The status register (R2), includes flag bits, CPUOFF
func (cpu *CPU) Sr() uint16 {
	return cpu.regs[2]
}

// The constant generator (R3) --- you'll rarely care about this as a lib client
func (cpu *CPU) Cg2() uint16 {
	return cpu.regs[3]
}

type family int
type addrMode int
type opcode int8
type condition int8
type int2 int8
type int4 int8
type int10 uint16

// Exports only one variable, "Width", to allow debuggers to step instructions;
// inside this struct is an ugly, inefficient, denormalized disassembly of an
// instruction read from memory.
type Insn struct {
	family    family
	mode      addrMode
	dstMode   addrMode
	opcode    opcode
	condition condition

	bw          int
	as          int2
	offset      int16
	ad          int
	source      int4
	destination int4

	srcx int16
	dstx int16

	Width int

	raw	[]byte
}

// BUG(tqbf) Byte shouldn't be public

// BUG(tqbf) Better yet, Byte should just be part of disasm

// Is this instruction operating on bytes or words
func (i *Insn) Byte() bool {
	switch i.family {
	case ItSingleOperand:
		switch i.opcode {
		case Op1Rrc:	
			fallthrough
		case Op1Rra:
			fallthrough
		case Op1Push:
			if i.bw != 0 {
				return true
			}
		}
	case ItDoubleOperand:
		if i.bw != 0 {
			return true
		}
	}
	return false
}

// Read the carry flag
func (cpu *CPU) S_C() int {
	return int(cpu.Sr() & 1)
}

// Set the carry flag
func (cpu *CPU) set_C(i uint16) {
	cpu.regs[2] = (cpu.regs[2] & 0xfffe) | (i & 1)
}

// Read the zero flag
func (cpu *CPU) S_Z() int {
	return int((cpu.Sr() >> 1) & 1)
}

// Set the zero flag
func (cpu *CPU) set_Z(i uint16) {
	cpu.regs[2] = (cpu.regs[2] & 0xfd) | ((i & 1) << 1)
}

// Read the negative flag
func (cpu *CPU) S_N() int {
	return int((cpu.Sr() >> 2) & 1)
}

// Set the negative flag
func (cpu *CPU) set_N(i uint16) {
	cpu.regs[2] = (cpu.regs[2] & 0xfb) | ((i & 1) << 2)
}

// Read the overflow flag
func (cpu *CPU) S_V() int {
	return int((cpu.Sr() >> 8) & 1)
}

// Set the overflow flag
func (cpu *CPU) set_V(i uint16) {
	cpu.regs[2] = (cpu.regs[2] & 0xff) | ((i & 1) << 8)
}

// Decode the meaning of the CG2 register for a given insn
func (i *Insn) cg2v() int16 {
	switch i.as {
	case 0:
		return 4
	case 1:
		return 8
	case 2:
		return 0
	case 3:
		return -1
	}
	panic("bad cg2")
	return -10

}

// Turn a word into a byte only for byte-based insns
func (cpu *CPU) bw_eval(i *Insn, val int32) int32 {
	if i.Byte() { 
		switch i.opcode {
		case Op1Swpb, Op1Sxt, Op1Call, Op1Reti:
			// don't honor B/W
		default:
			val = val & 0xff
		}
	}

	return val
}

// Store a byte from a word only for byte-based insns
func (cpu *CPU) bw_store(i *Insn, reg int4, val uint16) {
	if i.Byte() { 
		switch i.opcode {
		case Op1Swpb, Op1Sxt, Op1Call, Op1Reti:
			cpu.regs[reg] = val
		default:
			cpu.regs[reg] = (val & 0xff)
		}
	} else {
		cpu.regs[reg] = val
	}

	return
}

// Abstract value-of-a-register to return CG2 constants as well
func (cpu *CPU) v(i *Insn, v int4) (val int32) {
	if v == 3 {
		val = int32(i.cg2v())
	} else {
		val = int32(cpu.regs[v])
	}

	return
}

// Resolve the source operand of an instruction to its value, reading from
// memory if needed
func (cpu *CPU) src_operand(i *Insn) (uint16, error) {
	switch {
	case i.mode == AmAbsolute:	
		x, err := cpu.memory.LoadWord(uint16(i.srcx))
		if err != nil {
			return 0, err
		}
		return uint16(cpu.bw_eval(i, int32(x))), nil
	case i.mode == AmImmediate:
		return uint16(cpu.bw_eval(i, int32(i.srcx))), nil
	case i.mode == AmRegDirect:
		return uint16(cpu.bw_eval(i, cpu.v(i, i.source))), nil
	case i.mode == AmIndexed:
		base := cpu.v(i, i.source)
		base += int32(i.srcx)

		var (
			v uint16
			err error
			x uint8
		)

		if i.Byte() { 
			x, err = cpu.memory.LoadByte(uint16(base))
			v = uint16(x)
		} else {
			v, err = cpu.memory.LoadWord(uint16(base))
		}

		return uint16(cpu.bw_eval(i, int32(v))), err
	case i.mode == AmRegIndirect:
		addr := cpu.v(i, i.source)
		var v uint16
		var err error
		if i.Byte() {
			var x uint8
			x, err = cpu.memory.LoadByte(uint16(addr))
			v = uint16(x)
		} else {
			v, err = cpu.memory.LoadWord(uint16(addr))
		}
		return uint16(cpu.bw_eval(i, int32(v))), err
	case i.mode == AmIndirectIncr:
		addr := cpu.v(i, i.source)
		var (
			v uint16
			x uint8
			err error
		)
		if i.Byte() { 
			x, err = cpu.memory.LoadByte(uint16(addr))
			v = uint16(x)
			cpu.regs[i.source] += 1
		} else {
			v, err = cpu.memory.LoadWord(uint16(addr))
			cpu.regs[i.source] += 2
		}
		return uint16(cpu.bw_eval(i, int32(v))), err
	case i.mode == AmConst4:
		return uint16(4), nil
	case i.mode == AmConst8:
		return uint16(8), nil
	case i.mode == AmConst0:
		return uint16(0), nil
	case i.mode == AmConst1:
		return uint16(1), nil
	case i.mode == AmConst2:
		return uint16(2), nil
	case i.mode == AmConstNeg1:
		return uint16(0xffff), nil
	default:
		return 0, errors.New("unknown/invalid source addressing mode")
	}

	// unreached
	return 0, nil
}

// Resolve the target of a store for an instruction, writing to memory 
// if needed; this is simpler than src_operand because we don't need to
// handle writes to as many places as we can read from
func (cpu *CPU) src_operand_store(i *Insn, v uint16) (err error) {
	err = nil
	switch {
	case i.mode == AmAbsolute:
		if i.Byte() { 
			return cpu.memory.StoreByte(uint16(i.srcx), uint8(v))
		} else {
			return cpu.memory.StoreWord(uint16(i.srcx), uint16(v))
		}
	case i.mode == AmRegDirect:
		if i.source == 3 {
			return newError(E_BadOperand, "can't store to CG2")
		}

		cpu.bw_store(i, i.source, v)
	case i.mode == AmIndexed:
		base := cpu.v(i, i.source)
		base += int32(i.srcx)
		if i.Byte() {
			err = cpu.memory.StoreByte(uint16(base), uint8(v&0xff))
		} else {
			err = cpu.memory.StoreWord(uint16(base), v)
		}
	case i.mode == AmRegIndirect:
		addr := cpu.v(i, i.source)
		if i.Byte() { 
			err = cpu.memory.StoreByte(uint16(addr), uint8(v&0xff))
		} else {
			err = cpu.memory.StoreWord(uint16(addr), v)
		}
	case i.mode == AmIndirectIncr:
		addr := cpu.v(i, i.source)
		if i.Byte() {
			err = cpu.memory.StoreByte(uint16(addr), uint8(v&0xff))
			cpu.regs[i.source] += 1
		} else {
			err = cpu.memory.StoreWord(uint16(addr), v)
			cpu.regs[i.source] += 2
		}
	default:
		return errors.New("unknown/invalid source addressing mode")
	}

	return err
}

// Resolve the destination operand of an instruction to its value, reading from
// memory if needed; simpler than source, because there are fewer modes.
func (cpu *CPU) dst_operand(i *Insn) (retv uint16, err error) {
	err = nil
	var (
		v uint16
		x uint8
	)

	switch {
	case i.dstMode == AmAbsolute:
		if i.Byte() { 
			x, err = cpu.memory.LoadByte(uint16(i.dstx))
			v = uint16(x)
		} else {
			v, err = cpu.memory.LoadWord(uint16(i.dstx))
		}

		if err != nil {
			return 0, err
		}

		return uint16(cpu.bw_eval(i, int32(x))), nil
	case i.dstMode == AmRegDirect:
		if i.destination == 3 {
			retv = uint16(i.cg2v())
		} else {
			retv = cpu.regs[i.destination]
		}
		
		retv = uint16(cpu.bw_eval(i, int32(retv)))
		return 
	case i.dstMode == AmIndexed:
		base := cpu.v(i, i.destination)
		base += int32(i.dstx)
	
		if i.Byte() {
			x, err = cpu.memory.LoadByte(uint16(base))
			v = uint16(x)
		} else {
			v, err = cpu.memory.LoadWord(uint16(base))
		}

		retv = uint16(cpu.bw_eval(i, int32(v)))
		return retv, err
	default:
		return 0, errors.New("unknown addressing mode")
	}

	// notreached
	return 0, nil
}

// Store to the target of a two-operand instruction like MOV or XOR.
func (cpu *CPU) dst_operand_store(i *Insn, v uint16) (err error) {
	err = nil

	switch {
	case i.dstMode == AmAbsolute:
		if i.Byte() { 
			return cpu.memory.StoreByte(uint16(i.dstx), uint8(v))
		} else {
			return cpu.memory.StoreWord(uint16(i.dstx), uint16(v))
		}
	case i.dstMode == AmRegDirect:
		if i.destination == 3 {
			err = newError(E_BadOperand, "can't store to CG2 or SR")
		} else {
			cpu.bw_store(i, i.destination, v)
		}
	case i.dstMode == AmIndexed:
		addr := cpu.v(i, i.destination) + int32(i.dstx)

		if i.Byte() { 
			err = cpu.memory.StoreByte(uint16(addr), uint8(v))
		} else { 
			err = cpu.memory.StoreWord(uint16(addr), v) // .b
		}
	default:
		return errors.New("unknown addressing mode")
	}
	return
}

// Execute a parsed instruction on the target CPU, reading and writing to memory
// as appropriate. Expects to be called with PC pointing to the NEXT (not current)
// instruction; jumps, calls, and rets update PC, but nothing else does.
func (cpu *CPU) Execute(i *Insn) (err error) {
	if i.family == ItCondJump {
		jmp := false

		switch i.condition {
		case CondJnz:
			if cpu.S_Z() == 0 {
				jmp = true
			}								
		case CondJz:
			if cpu.S_Z() != 0 {
				jmp = true
			}
		case CondJnc:
			if cpu.S_C() == 0 {
				jmp = true
			}
		case CondJc:
			if cpu.S_C() != 0 {
				jmp = true
			}
		case CondJn:
			if cpu.S_N() != 0 {
				jmp = true
			}
		case CondJge:
			if cpu.S_N() ^ cpu.S_V() == 0 {
				jmp = true
			}
		case CondJl:
			if cpu.S_N() ^ cpu.S_V() != 0 {
				jmp = true
			}
		case CondJmp:
			jmp = true
		}

		if jmp == true {
			cpu.regs[0] += uint16(i.offset)
		}
	} else {
		var src, dst, tmp uint16

		switch i.family {
		case ItSingleOperand:
			// S1: LOAD

			if src, err = cpu.src_operand(i); err != nil {
				return
			}

			// S2: EVAL
			switch i.opcode {
			case Op1Rrc:
				if i.Byte() { 
					x := src & 1
					tmp  = uint16(uint8(cpu.S_C()) << 7 | uint8(src >> 1))
					cpu.set_C(x)
				} else {
					x := src & 1
					tmp  = uint16(cpu.S_C()) << 15
					tmp |= src >> 1
					cpu.set_C(x)
				}
			case Op1Swpb:
				tmp = (src >> 8) | ((src & 0xff) << 8)

			case Op1Rra:
				if i.Byte() { 
					x := src & 1
					y := (src >> 7) & 1
					tmp = src >> 1
					tmp = uint16(uint8((tmp & 0x40) | (y << 7)))
					cpu.set_C(x)
				} else {
					x := src & 1
					y := (src >> 15) & 1
					tmp = src >> 1
					tmp = uint16((tmp & 0x4000) | (y << 15))
					cpu.set_C(x)
				}
			case Op1Sxt:
				if src & 0x80 != 0 {
					tmp = (0xff<<8) | src
				} else {
					tmp = src & 0xff
				}

			case Op1Push: 
				cpu.regs[1] -= 2
				if i.Byte() {
					cpu.memory.StoreByte(cpu.regs[1], uint8(src))
				} else {
					cpu.memory.StoreWord(cpu.regs[1], src)
				}

			case Op1Call:
				cpu.regs[1] -= 2
				cpu.memory.StoreWord(cpu.regs[1], cpu.regs[0])
				cpu.regs[0] = src

			case Op1Reti:
				panic("Not implemented RETI")

			}

			// S3 FLAGS
			
			// C
			if i.opcode == Op1Sxt {
				if tmp == 0 { 
					cpu.set_C(0)
				} else {
					cpu.set_C(1)
				}
			}

			// V
			
			switch i.opcode {
			case Op1Rrc:
				fallthrough
			case Op1Rra:
				fallthrough
			case Op1Sxt:
				cpu.set_V(0)
			}

			// Z

			switch i.opcode {
			case Op1Rrc:
				fallthrough
			case Op1Rra:
				fallthrough
			case Op1Sxt:
				if tmp == 0 {
					cpu.set_Z(1)
				} else {
					cpu.set_Z(0)
				}
			}			

			// N
			
			switch i.opcode {
			case Op1Rrc:
				fallthrough
			case Op1Rra:
				fallthrough
			case Op1Sxt:
				if i.Byte() { 
					if int8(tmp) < 0  {
						cpu.set_N(1)
					} else {
						cpu.set_Z(0)
					}
				} else {
					if int16(tmp) < 0  {
						cpu.set_N(1)
					} else {
						cpu.set_Z(0)
					}
				}
			}			
		
			// S4 STORE

			switch i.opcode {
			case Op1Push:
			case Op1Call:
			case Op1Reti:
				// these don't do anything
			default:
				cpu.src_operand_store(i, tmp)
			}

		case ItDoubleOperand:
			// S1 LOAD

			if src, err = cpu.src_operand(i); err != nil {
				return
			}

			if dst, err = cpu.dst_operand(i); err != nil {
				return
			}

			// S2 EVAL

			switch i.opcode {
			case Op2Mov:
				if i.Byte() {
					tmp = src & 0xff
				} else {
					tmp = src
				}

			case Op2Add:
				if i.Byte() {
					x := uint8(src) + uint8(dst)
					tmp = uint16(x)
				} else {
					tmp = src + dst
				}
			case Op2Addc:
				if i.Byte() {
					x := uint8(src) + uint8(dst)
					if(cpu.S_C() != 0) {
						x += 1
					}
					tmp = uint16(x)				
				} else {
					tmp = src + dst
					if(cpu.S_C() != 0) {
						tmp += 1
					}
				}
			case Op2Cmp:
				fallthrough
			case Op2Sub:
				if i.Byte() {
					x := uint8(dst) + uint8((^src)+1)
					tmp = uint16(x)
				} else {
					tmp = dst + ((^src) + 1)
				}
			case Op2Subc:
				if i.Byte() {
					x := uint8(dst) + uint8((^src)+1) + uint8(cpu.S_C())
					tmp = uint16(x)
				} else {
					tmp = dst + ((^src) + 1) + uint16(cpu.S_C())
				}
			case Op2Dadd:
				// NOT IMPLEMENTED

			case Op2Bis:
				if i.Byte() {
					tmp = uint16(uint8(src) | uint8(dst))
				} else {
					tmp = src | dst
				}
			case Op2Xor:
				if i.Byte() {
					tmp = uint16(uint8(src) ^ uint8(dst))
				} else {
					tmp = src ^ dst
				}
			case Op2Bic:
				src = ^src
			case Op2Bit:
				fallthrough
			case Op2And:
				if i.Byte() {
					tmp = uint16(uint8(src) & uint8(dst))
				} else {
					tmp = src & dst
				}
			}

			// S3 FLAGS
			
			// C
			switch i.opcode {
			case Op2Add:
				fallthrough
			case Op2Addc:
				if i.Byte() { 
					if int32(src) + int32(dst) > 0xff {	
						cpu.set_C(1)
					} else {
						cpu.set_C(0)
					}
				} else {
					if int32(src) + int32(dst) > 0xffff {	
						cpu.set_C(1)
					} else { 
						cpu.set_C(0)
					}
				}		
			case Op2Sub:
				fallthrough
			case Op2Subc:
				fallthrough
			case Op2Cmp:
				if dst >= src {
					cpu.set_C(1)
				} else {
					cpu.set_C(0)
				}
			case Op2Bit:
				if tmp == 0 {
					cpu.set_C(0)
				} else {
					cpu.set_C(1)
				}
			case Op2Xor:
				fallthrough
			case Op2And:
				if tmp != 0 {
					cpu.set_C(1)
				} else {
					cpu.set_C(0)
				}
			}			

			// V
			switch i.opcode {
			case Op2Add:
				fallthrough
			case Op2Addc:
				if (int16(src) > 0 && int16(dst) > 0 && int16(tmp) < 0) ||
					(int16(src) < 0 && int16(dst) < 0 && int16(tmp) > 0) {
					cpu.set_V(1)
				} else {
					cpu.set_V(0)
				}
			case Op2Sub:
				fallthrough
			case Op2Subc:
				fallthrough
			case Op2Cmp:
				if (int16(src) > 0 && int16(dst) < 0 && int16(tmp) < 0) ||
					(int16(src) < 0 && int16(dst) > 0 && int16(tmp) > 0) {
					cpu.set_V(1)
				} else {
					cpu.set_V(0)
				}				
			case Op2Bit:
				fallthrough
			case Op2And:
				cpu.set_V(0)
			case Op2Xor:
				if int16(dst) < 0 && int16(src) < 0 { 
					cpu.set_V(1)
				} else {
					cpu.set_V(0)
				}
			}

			// N
			switch i.opcode {
			case Op2Add:
				fallthrough
			case Op2Addc:
				fallthrough
			case Op2Sub:
				fallthrough
			case Op2Subc:
				fallthrough
			case Op2Cmp:
				if int16(tmp) < 0 {
					cpu.set_N(1)
 				} else {
					cpu.set_N(0)
				}
			case Op2Xor:
				fallthrough
			case Op2Bit:
				fallthrough
			case Op2And:
				if i.Byte() {
					if tmp & 0x80 != 0 {
						cpu.set_N(1)
					} else {
						cpu.set_N(0)
					}
				} else {
					if tmp & 0x8000 != 0 {
						cpu.set_N(1)
					} else {
						cpu.set_N(0)
					}
				}
			}	

			// Z
			switch i.opcode {
			case Op2Add:
				fallthrough
			case Op2Addc:
				fallthrough
			case Op2Sub:
				fallthrough
			case Op2Subc:
				fallthrough
			case Op2Cmp:
				fallthrough
			case Op2Bit:
				fallthrough
			case Op2Xor:
				fallthrough
			case Op2And:
				if tmp == 0 {
					cpu.set_Z(1)
				} else {
					cpu.set_Z(0)
				}
			}

			// S4 STORE

			if(i.opcode != Op2Bit && i.opcode != Op2Cmp) {
				cpu.dst_operand_store(i, tmp)
			}
		}
	}

	if cpu.Sr() & 16 != 0 {
		return newError(E_Halted, "CPUOFF set")
	}

	return nil
}

// Trivial implementation of CPU step function, including advancing PC; you can
// loop this function to "run" the CPU. 
func (cpu *CPU) Step() (err error) {
	err = nil

//    fmt.Printf("PC: %x\n", cpu.Pc())
//    fmt.Printf("Mem: %v\n", cpu.memory)
	bytes, err := cpu.memory.Load6Bytes(cpu.Pc())
	if err != nil {
		return
	}

	i, err := Disassemble(bytes)
	if err != nil {
		return 
	}

	cpu.regs[0] += uint16(i.Width)

    //fmt.Printf("Op: %v\n", i)


	err = (*cpu).Execute(&i)

	return err
}

func (cpu CPU) String() string {
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("msp43x(%p) pc:%0.4X sr:%0.16b sp:%0.4X\n", &cpu, cpu.Pc(), cpu.Sr(), cpu.Sp()))
	buffer.WriteString(fmt.Sprintf("\tr4 %0.4X %0.4X %0.4X %0.4X\n", cpu.regs[4], cpu.regs[5], cpu.regs[6], cpu.regs[7]))
	buffer.WriteString(fmt.Sprintf("\tr8 %0.4X %0.4X %0.4X %0.4X\n", cpu.regs[8], cpu.regs[9], cpu.regs[10], cpu.regs[11]))
	buffer.WriteString(fmt.Sprintf("\trC %0.4X %0.4X %0.4X %0.4X\n", cpu.regs[12], cpu.regs[13], cpu.regs[14], cpu.regs[15]))
	
	return buffer.String()
}
