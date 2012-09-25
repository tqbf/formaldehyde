package msp43x

import (
	"fmt"
	"bytes"
)

func interpAs(as int2) (mode addrMode) {
	switch {
	case as == 0:
		mode = AmRegDirect
	case as == 1:
		mode = AmIndexed
	case as == 2:
		mode = AmRegIndirect
	case as == 3:
		mode = AmIndirectIncr
	}
	return
}

// Disassemble an array of (at least 6, the maximum size of an MSP430 insn) bytes
// into an Insn struct.
func Disassemble(raw []byte) (i Insn, err error) {
	i = Insn{
		Width: 2,
	}
	err = nil

	isConstant := func(i *Insn) bool { 
		if i.source != 2 && i.source != 3 {
			return false
		}

		if i.source == 2 && (i.as != 2 && i.as != 3) {
			return false
		}

		return true
	}

	constantMode := func(i *Insn) addrMode {
		switch {
		case i.as == 0 && i.source == 3:
			return AmConst0
		case i.as == 1 && i.source == 3:
			return AmConst1
		case i.as == 2 && i.source == 3:
			return AmConst2
		case i.as == 3 && i.source == 3:
			return AmConstNeg1
		case i.as == 2 && i.source == 2:
			return AmConst4
		case i.as == 3 && i.source == 2:
			return AmConst8
		}
	
		panic("bad constant mode logic")
		return 0
	}

	if cap(raw) < 2 {
		err = newError(E_TooShort, "less than 2 bytes")
		return
	}

	iword := uint16(int(raw[1])<<8 | int(raw[0]))

	switch (iword >> 13) & 7 {
	case 0:
		i.family = ItSingleOperand
	case 1:
		i.family = ItCondJump
	case 2, 3, 4, 5, 6, 7:
		i.family = ItDoubleOperand
	}

	switch i.family {
	case ItSingleOperand:
		i.opcode = opcode(iword >> 7 & 7)
		if (iword >> 6 & 1) == 0 {
			i.bw = 0
		} else {
			i.bw = 1
		}
		i.as = int2(iword >> 4 & 3)
		i.source = int4(iword & 15)

		i.mode = interpAs(i.as)

		if isConstant(&i) {
			i.mode = constantMode(&i)
		} else {
	 		if i.as == 1 {
	 			if cap(raw) < 4 {
	 				err = newError(E_TooShort, "missing index word")
	 				return
	 			}
	 
	 			i.srcx = int16(int(raw[3])<<8 | int(raw[2]))
	 			i.Width = 4
	 
	 			if i.source == 2 {
	 				i.mode = AmAbsolute
	 			}
	 		} else if i.as == 3 && i.source == 0 {
	 			if cap(raw) < 4 {
	 				err = newError(E_TooShort, "missing immediate word")
	 				return
	 			}
	 
	 			i.Width = 4
	 			i.mode = AmImmediate
	 			i.srcx = int16(int(raw[3])<<8 | int(raw[2]))
	 		} 
		}
	case ItCondJump:
		i.condition = condition(iword >> 10 & 7)
		off := int(iword) & 1023
		i.offset = int16(off * 2)
		if off&0x200 != 0 {
			off = (0x3f << 10) | off
			i.offset = int16(off) * 2
		} else {
			i.offset = int16(off * 2)
		}
	case ItDoubleOperand:
		i.opcode = opcode(iword >> 12 & 15)
		i.source = int4(iword >> 8 & 15)
		if (iword >> 7 & 1) == 0 {
			i.ad = 0
		} else {
			i.ad = 1
		}

		if (iword >> 6 & 1) == 0 {
			i.bw = 0
		} else {
			i.bw = 1
		}

		i.as = int2(iword >> 4 & 3)
		i.destination = int4(iword & 15)

		i.mode = interpAs(i.as)
		if i.as == 1 && i.source == 2 {
			i.dstMode = AmAbsolute
		}
	
		if i.ad == 1 {
			if i.destination == 2 {
				i.dstMode = AmAbsolute
			} else {
				i.dstMode = AmIndexed
			}
		} else {
			i.dstMode = AmRegDirect
		}

		switch {
		case isConstant(&i) == false && i.as == 1 && i.ad == 1:
			if cap(raw) < 6 {
				err = newError(E_TooShort, "missing src and dst index words")
				return
			}

			i.srcx = int16(int(raw[3])<<8 | int(raw[2]))
			i.dstx = int16(int(raw[5])<<8 | int(raw[4]))
			i.Width = 6

		case isConstant(&i) == false && i.as == 1:
			if cap(raw) < 4 {
				err = newError(E_TooShort, "missing src index word")
				return
			}

			i.srcx = int16(int(raw[3])<<8 | int(raw[2]))
			i.Width = 4

		case i.as == 3 && i.source == 0 && i.ad == 1:
			if cap(raw) < 6 {
				err = newError(E_TooShort, "missing src immed and dst index words")
				return
			}

			i.srcx = int16(int(raw[3])<<8 | int(raw[2]))
			i.dstx = int16(int(raw[5])<<8 | int(raw[4]))
			i.Width = 6
			i.mode = AmImmediate

		case i.as == 3 && i.source == 0 && i.ad == 0:
			if cap(raw) < 4 {
				err = newError(E_TooShort, "missing src immed word")
				return
			}

			i.srcx = int16(int(raw[3])<<8 | int(raw[2]))
			i.Width = 4
			i.mode = AmImmediate

		case isConstant(&i) && i.ad == 1:
			i.mode = constantMode(&i)
			fallthrough

		case i.ad == 1:
			if cap(raw) < 4 {
				err = newError(E_TooShort, "missing dst index word")
				return
			}

			i.dstx = int16(int(raw[3])<<8 | int(raw[2]))
			i.Width = 4

		}

		if i.mode == AmIndexed && i.source == 2 {
			i.mode = AmAbsolute
		}

		if i.dstMode == AmIndexed && i.destination == 2 {
			i.dstMode = AmAbsolute
		}

		if isConstant(&i) {
			i.mode = constantMode(&i)
		}
	}

	i.raw = raw[0:i.Width]

	return
}


func mode_string(mode addrMode, reg int4, ext int16) (src string) {
	src = "???"

	switch mode {
	case AmImmediate:
		return fmt.Sprintf("#%d", ext)
	case AmConst8:
		return fmt.Sprintf("#8")
	case AmConst4:
		return fmt.Sprintf("#4")
	case AmConst2:
		return fmt.Sprintf("#2")
	case AmConst1:
		return fmt.Sprintf("#1")
	case AmConst0:
		return fmt.Sprintf("#0")
	case AmConstNeg1:
		return fmt.Sprintf("#-1")
	}

	if reg != 3 {
		switch mode {
		case AmRegDirect:
			src = fmt.Sprintf("R%d", reg)
		case AmIndexed:
			if reg == 2 {
				src = fmt.Sprintf("&%0.2x", ext)
			} else {
				src = fmt.Sprintf("%d(R%d)", ext, reg)
			}
		case AmRegIndirect:
			src = fmt.Sprintf("@R%d", reg)
		case AmIndirectIncr:
			src = fmt.Sprintf("@R%d+", reg)
		}
	} else {
		switch mode {
		case AmRegDirect:
			src = fmt.Sprintf("4")
		case AmIndexed:
			src = fmt.Sprintf("8")
		case AmRegIndirect:
			src = fmt.Sprintf("0")
		case AmIndirectIncr:
			src = fmt.Sprintf("-1")
		}
	}

	return
}

// Print a rough translation of our Insn struct as an MSP430 assembly directive
func (i Insn) String() string {
	var buf bytes.Buffer

	for k := 0; k < 6; k++ {
		if k < len(i.raw) {
			buf.WriteString(fmt.Sprintf("%0.2x ", i.raw[k]))
		} else {
			buf.WriteString("   ")
		}
	}

	buf.WriteString("\t")

	bytestr := func(i *Insn) string {
		if i.Byte() {
			return ".B"
		}
		return "";
	}

	switch i.family {
	case ItSingleOperand:
		buf.WriteString(fmt.Sprintf("%s%s %s",
			opcode_string(i.opcode, true), bytestr(&i),
			mode_string(i.mode, i.source, i.srcx)))
	case ItCondJump:
		buf.WriteString(fmt.Sprintf("%v $%d", i.condition, i.offset+2))
	case ItDoubleOperand:
		buf.WriteString(fmt.Sprintf("%s%s %s, %s",
			opcode_string(i.opcode, false), bytestr(&i),
			mode_string(i.mode, i.source, i.srcx),
			mode_string(i.dstMode, i.destination, i.dstx)))
	}

	return buf.String()
}
