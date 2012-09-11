package msp43x

// bits 15:13
const (
	ItSingleOperand family = 0
	ItCondJump             = 1
	ItDoubleOperand        = 2
)

// from As/Ad
const (
	AmRegDirect = iota
	AmIndexed
	AmRegIndirect
	AmIndirectIncr
	AmSymbolic
	AmImmediate
	AmAbsolute
	AmConst4
	AmConst8
	AmConst0
	AmConst1
	AmConst2
	AmConstNeg1
)

const (
	CondJnz = 0
	CondJz  = 1
	CondJnc = 2
	CondJc  = 3
	CondJn  = 4
	CondJge = 5
	CondJl  = 6
	CondJmp = 7
)

const (
	Op2Mov  = 4
	Op2Add  = 5
	Op2Addc = 6
	Op2Subc = 7
	Op2Sub  = 8
	Op2Cmp  = 9
	Op2Dadd = 10
	Op2Bit  = 11
	Op2Bic  = 12
	Op2Bis  = 13
	Op2Xor  = 14
	Op2And  = 15
)

const (
	Op1Rrc  = 0
	Op1Swpb = 1
	Op1Rra  = 2
	Op1Sxt  = 3
	Op1Push = 4
	Op1Call = 5
	Op1Reti = 6
)

// this stuff is very silly but is leftover from debugging. blame emacs macros.

func (c family) String() string {
	switch c {
	case ItSingleOperand:
		return "Single Operand"
	case ItCondJump:
		return "Conditional Jump"
	case ItDoubleOperand:
		return "Double Operand"
	}

	return "Invalid"
}

func (m addrMode) String() string {
	switch m {
	case AmRegDirect:
		return "RegDirect"
	case AmIndexed:
		return "Indexed"
	case AmRegIndirect:
		return "RegIndirect"
	case AmIndirectIncr:
		return "IndirectIncr"
	case AmSymbolic:
		return "Symbolic"
	case AmImmediate:
		return "Immediate"
	case AmAbsolute:
		return "Absolute"
	case AmConst4:
		return "Const4"
	case AmConst8:
		return "Const8"
	case AmConst0:
		return "Const0"
	case AmConst1:
		return "Const1"
	case AmConst2:
		return "Const2"
	case AmConstNeg1:
		return "ConstNeg1"
	}
	return "Invalid"
}

func (c condition) String() string {
	switch c {
	case CondJnz:
		return "JNZ"
	case CondJz:
		return "JZ"
	case CondJnc:
		return "JNC"
	case CondJc:
		return "JC"
	case CondJn:
		return "JN"
	case CondJge:
		return "JGE"
	case CondJl:
		return "JL"
	case CondJmp:
		return "JMP"
	}
	return "Invalid"
}

func opcode_string(o opcode, single bool) string {
	if single == true {
		switch o {
		case Op1Rrc:
			return "RRC"
		case Op1Swpb:
			return "SWPB"
		case Op1Rra:
			return "RRA"
		case Op1Sxt:
			return "SXT"
		case Op1Push:
			return "PUSH"
		case Op1Call:
			return "CALL"
		case Op1Reti:
			return "RETI"
		}

	} else {
		switch o {
		case Op2Mov:
			return "MOV"
		case Op2Add:
			return "ADD"
		case Op2Addc:
			return "ADDC"
		case Op2Subc:
			return "SUBC"
		case Op2Sub:
			return "SUB"
		case Op2Cmp:
			return "CMP"
		case Op2Dadd:
			return "DADD"
		case Op2Bit:
			return "BIT"
		case Op2Bic:
			return "BIC"
		case Op2Bis:
			return "BIS"
		case Op2Xor:
			return "XOR"
		case Op2And:
			return "AND"
		}
	}

	return "INVALID"
}
