package ir

import (
	"bytes"
	"fmt"

	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

// A BasicBlock represents an LLVM IR basic block, which consists of a sequence
// of non-branching instructions, terminated by a control flow instruction (e.g.
// br or ret).
//
// Basic blocks may be referenced from terminators (e.g. br), and are thus
// considered LLVM IR values of label type.
type BasicBlock struct {
	// Parent function of the basic block.
	parent *Function
	// Label name of the basic block; or empty if anonymous basic block.
	name string
	// Non-branching instructions of the basic block.
	insts []Instruction
	// Terminator of the basic block.
	term Terminator
}

// NewBlock returns a new basic block based on the given label name. An empty
// label name indicates an anonymous basic block.
func NewBlock(name string) *BasicBlock {
	return &BasicBlock{name: name}
}

// Type returns the type of the basic block.
func (b *BasicBlock) Type() types.Type {
	return types.Label
}

// Ident returns the identifier associated with the basic block.
func (b *BasicBlock) Ident() string {
	// TODO: Encode name if containing special characters.
	return "%" + b.name
}

// LLVMString returns the LLVM syntax representation of the basic block.
func (b *BasicBlock) LLVMString() string {
	buf := &bytes.Buffer{}
	// TODO: Encode name if containing special characters.
	fmt.Fprintf(buf, "%s:\n", b.name)
	for _, i := range b.Insts() {
		fmt.Fprintf(buf, "\t%s\n", i.LLVMString())
	}
	fmt.Fprintf(buf, "\t%s", b.Term().LLVMString())
	return buf.String()
}

// Insts returns the non-branching instructions of the basic block.
func (b *BasicBlock) Insts() []Instruction {
	return b.insts
}

// Term returns the terminator of the basic block.
func (b *BasicBlock) Term() Terminator {
	return b.term
}

// SetTerm sets the terminator of the basic block.
func (b *BasicBlock) SetTerm(t Terminator) {
	if t, ok := t.(parentSetter); ok {
		t.SetParent(b)
	}
	b.term = t
}

// Parent returns the parent function of the basic block.
func (b *BasicBlock) Parent() *Function {
	return b.parent
}

// SetParent sets the parent function of the basic block.
func (b *BasicBlock) SetParent(parent *Function) {
	b.parent = parent
}

// AppendInst appends the given instruction to the basic block.
func (b *BasicBlock) AppendInst(i Instruction) {
	if i, ok := i.(parentSetter); ok {
		i.SetParent(b)
	}
	b.insts = append(b.insts, i)
}

// --- [ Binary instructions ] -------------------------------------------------

// NewAdd appends a new add instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewAdd(x, y value.Value) *InstAdd {
	i := NewAdd(x, y)
	b.AppendInst(i)
	return i
}

// NewFAdd appends a new fadd instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewFAdd(x, y value.Value) *InstFAdd {
	i := NewFAdd(x, y)
	b.AppendInst(i)
	return i
}

// NewSub appends a new sub instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewSub(x, y value.Value) *InstSub {
	i := NewSub(x, y)
	b.AppendInst(i)
	return i
}

// NewFSub appends a new fsub instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewFSub(x, y value.Value) *InstFSub {
	i := NewFSub(x, y)
	b.AppendInst(i)
	return i
}

// NewMul appends a new mul instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewMul(x, y value.Value) *InstMul {
	i := NewMul(x, y)
	b.AppendInst(i)
	return i
}

// NewFMul appends a new fmul instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewFMul(x, y value.Value) *InstFMul {
	i := NewFMul(x, y)
	b.AppendInst(i)
	return i
}

// NewUDiv appends a new udiv instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewUDiv(x, y value.Value) *InstUDiv {
	i := NewUDiv(x, y)
	b.AppendInst(i)
	return i
}

// NewSDiv appends a new sdiv instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewSDiv(x, y value.Value) *InstSDiv {
	i := NewSDiv(x, y)
	b.AppendInst(i)
	return i
}

// NewFDiv appends a new fdiv instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewFDiv(x, y value.Value) *InstFDiv {
	i := NewFDiv(x, y)
	b.AppendInst(i)
	return i
}

// NewURem appends a new urem instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewURem(x, y value.Value) *InstURem {
	i := NewURem(x, y)
	b.AppendInst(i)
	return i
}

// NewSRem appends a new srem instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewSRem(x, y value.Value) *InstSRem {
	i := NewSRem(x, y)
	b.AppendInst(i)
	return i
}

// NewFRem appends a new frem instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewFRem(x, y value.Value) *InstFRem {
	i := NewFRem(x, y)
	b.AppendInst(i)
	return i
}

// --- [ Bitwise instructions ] ------------------------------------------------

// NewShL appends a new shl instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewShL(x, y value.Value) *InstShL {
	i := NewShL(x, y)
	b.AppendInst(i)
	return i
}

// NewLShR appends a new lshr instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewLShR(x, y value.Value) *InstLShR {
	i := NewLShR(x, y)
	b.AppendInst(i)
	return i
}

// NewAShR appends a new ashr instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewAShR(x, y value.Value) *InstAShR {
	i := NewAShR(x, y)
	b.AppendInst(i)
	return i
}

// NewAnd appends a new and instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewAnd(x, y value.Value) *InstAnd {
	i := NewAnd(x, y)
	b.AppendInst(i)
	return i
}

// NewOr appends a new or instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewOr(x, y value.Value) *InstOr {
	i := NewOr(x, y)
	b.AppendInst(i)
	return i
}

// NewXor appends a new xor instruction to the basic block based on the given
// operands.
func (b *BasicBlock) NewXor(x, y value.Value) *InstXor {
	i := NewXor(x, y)
	b.AppendInst(i)
	return i
}

// --- [ Vector instructions ] -------------------------------------------------

// --- [ Aggregate instructions ] ----------------------------------------------

// --- [ Memory instructions ] -------------------------------------------------

// NewLoad appends a new load instruction to the basic block based on the given
// source address.
func (b *BasicBlock) NewLoad(src value.Value) *InstLoad {
	i := NewLoad(src)
	b.AppendInst(i)
	return i
}

// --- [ Conversion instructions ] ---------------------------------------------

// NewTrunc appends a new trunc instruction to the basic block based on the
// given source value and target type.
func (b *BasicBlock) NewTrunc(from value.Value, to types.Type) *InstTrunc {
	i := NewTrunc(from, to)
	b.AppendInst(i)
	return i
}

// NewZExt appends a new zext instruction to the basic block based on the given
// source value and target type.
func (b *BasicBlock) NewZExt(from value.Value, to types.Type) *InstZExt {
	i := NewZExt(from, to)
	b.AppendInst(i)
	return i
}

// NewSExt appends a new sext instruction to the basic block based on the given
// source value and target type.
func (b *BasicBlock) NewSExt(from value.Value, to types.Type) *InstSExt {
	i := NewSExt(from, to)
	b.AppendInst(i)
	return i
}

// NewFPTrunc appends a new fptrunc instruction to the basic block based on the
// given source value and target type.
func (b *BasicBlock) NewFPTrunc(from value.Value, to types.Type) *InstFPTrunc {
	i := NewFPTrunc(from, to)
	b.AppendInst(i)
	return i
}

// NewFPExt appends a new fpext instruction to the basic block based on the
// given source value and target type.
func (b *BasicBlock) NewFPExt(from value.Value, to types.Type) *InstFPExt {
	i := NewFPExt(from, to)
	b.AppendInst(i)
	return i
}

// NewFPToUI appends a new fptoui instruction to the basic block based on the
// given source value and target type.
func (b *BasicBlock) NewFPToUI(from value.Value, to types.Type) *InstFPToUI {
	i := NewFPToUI(from, to)
	b.AppendInst(i)
	return i
}

// NewFPToSI appends a new fptosi instruction to the basic block based on the
// given source value and target type.
func (b *BasicBlock) NewFPToSI(from value.Value, to types.Type) *InstFPToSI {
	i := NewFPToSI(from, to)
	b.AppendInst(i)
	return i
}

// NewUIToFP appends a new uitofp instruction to the basic block based on the
// given source value and target type.
func (b *BasicBlock) NewUIToFP(from value.Value, to types.Type) *InstUIToFP {
	i := NewUIToFP(from, to)
	b.AppendInst(i)
	return i
}

// NewSIToFP appends a new sitofp instruction to the basic block based on the
// given source value and target type.
func (b *BasicBlock) NewSIToFP(from value.Value, to types.Type) *InstSIToFP {
	i := NewSIToFP(from, to)
	b.AppendInst(i)
	return i
}

// NewPtrToInt appends a new ptrtoint instruction to the basic block based on
// the given source value and target type.
func (b *BasicBlock) NewPtrToInt(from value.Value, to types.Type) *InstPtrToInt {
	i := NewPtrToInt(from, to)
	b.AppendInst(i)
	return i
}

// NewIntToPtr appends a new inttoptr instruction to the basic block based on
// the given source value and target type.
func (b *BasicBlock) NewIntToPtr(from value.Value, to types.Type) *InstIntToPtr {
	i := NewIntToPtr(from, to)
	b.AppendInst(i)
	return i
}

// NewBitCast appends a new bitcast instruction to the basic block based on the
// given source value and target type.
func (b *BasicBlock) NewBitCast(from value.Value, to types.Type) *InstBitCast {
	i := NewBitCast(from, to)
	b.AppendInst(i)
	return i
}

// NewAddrSpaceCast appends a new addrspacecast instruction to the basic block
// based on the given source value and target type.
func (b *BasicBlock) NewAddrSpaceCast(from value.Value, to types.Type) *InstAddrSpaceCast {
	i := NewAddrSpaceCast(from, to)
	b.AppendInst(i)
	return i
}

// --- [ Other instructions ] --------------------------------------------------

// NewCall appends a new call instruction to the basic block based on the given
// callee and function arguments.
func (b *BasicBlock) NewCall(callee *Function, args ...value.Value) *InstCall {
	i := NewCall(callee, args...)
	b.AppendInst(i)
	return i
}

// --- [ Terminators ] ---------------------------------------------------------

// NewRet sets the terminator of the basic block to a new ret terminator based
// on the given return value. A nil return value indicates a "void" return.
func (b *BasicBlock) NewRet(x value.Value) *TermRet {
	t := NewRet(x)
	b.SetTerm(t)
	return t
}