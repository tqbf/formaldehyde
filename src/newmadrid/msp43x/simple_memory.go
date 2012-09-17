package msp43x

import (
	"bytes"
	"fmt"
)

type SimpleMemory struct {
	raw	[65536]byte
}

func (mem *SimpleMemory) Load6Bytes(address uint16) ([]byte, error) {
	if address & 1 != 0 {
		return nil, newError(E_AddressUnaligned, "insn address unaligned")
	}

	if address > 0xfffa {
		return nil, newError(E_AddressTooHigh, "insn address would wrap")

	}

	return mem.raw[address:address+6], nil
}

func (mem *SimpleMemory) LoadWord(address uint16) (uint16, error) {
	if address & 1 != 0 {
		return 0, newError(E_AddressUnaligned, "load address unaligned")
	}

	if address > 0xfffc {
		return 0, newError(E_AddressTooHigh, "load address would wrap")

	}

	// here marks the spot where golang's integer type system fucked me.
	ret := uint16(int(int(mem.raw[address+1])<<8)|int(mem.raw[address]))
	return ret, nil
}

func (mem *SimpleMemory) StoreWord(address uint16, value uint16) error {
	switch address {
	case 0x5ce:
		fmt.Printf("%c", value)
	
	default:
		mem.raw[address+1] = byte(value>>8&0xff)
		mem.raw[address] = byte(value&0xff)
	}

	return nil
}

func (mem *SimpleMemory) LoadByte(address uint16) (uint8, error) {
	return uint8(mem.raw[address]), nil
}

func (mem *SimpleMemory) StoreByte(address uint16, value uint8) error {
	switch address {
	case 0x5ce:
		fmt.Printf("%c", value)
	default:
		mem.raw[address] = byte(value);
	}
	return nil	
}

func (mem *SimpleMemory) String() string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("mem @ %p/%p\n", mem, &(mem.raw)))

	last_blank := false
	for i := 0; i < 0x10000; i += 16 {
		blank := true

		for q := 0; q < 16; q++ {
			if mem.raw[i + q] != 0 {
				blank = false
				break
			}
		}

		if blank && last_blank {
			continue
		}

		if blank {
			last_blank = true
		}
		
		buf.WriteString(fmt.Sprintf("%0.4x  ", i))

		for j := 0; j < 8; j++ {
			buf.WriteString(fmt.Sprintf("%0.2x ", mem.raw[i + j]))
		}

		buf.WriteString(" ")

		for j := 0; j < 8; j++ {
			buf.WriteString(fmt.Sprintf("%0.2x ", mem.raw[i + 8 + j]))			
		}

		buf.WriteString("\n")
	}

	return buf.String()
}