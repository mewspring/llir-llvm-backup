// TODO: rename from translateFoo to astToIRFoo.

// Problems to solve.
//
// phi instructions can reference local variables defined in basic blocks not
// yet visited when translating basic blocks in linear order.
//
// Terminator instructions can reference basic blocks not yet visited when
// translating basic blocks in linear order.
//
// The function parameters, basic blocks and local variables (produced by the
// result of instructions) of a function may be unnamed. They are assigned the
// first unused local ID (e.g. %42) when traversing the body of the function in
// linear order; where function parameters are assigned first, then for each
// basic block, assign an ID to the basic block and then to the result of its
// instructions. Note, instructions that produce void results are ignored.
// Non-value instructions (e.g. store) are always ignored. Notably, the call
// instruction may be ignored if the callee has a void return.

// TODO: make concurrent :)

package asm

import (
	"fmt"

	"github.com/llir/l/ir"
	"github.com/llir/l/ir/types"
	"github.com/llir/l/ir/value"
	"github.com/mewmew/l-tm/asm/ll/ast"
	"github.com/mewmew/l-tm/internal/enc"
	"github.com/pkg/errors"
)

type funcGen struct {
	// Module generator.
	gen *generator

	// LLVM IR function being generated.
	f *ir.Function

	// ls maps from local identifier (without '%' prefix) to corresponding IR
	// value.
	ls map[string]value.Value
}

func newFuncGen(gen *generator, f *ir.Function) *funcGen {
	return &funcGen{
		gen: gen,
		f:   f,
		ls:  make(map[string]value.Value),
	}
}

// resolveLocals resolves the local variables, basic blocks and function
// parameters of the given function body. The returned value maps from local
// identifier (without '%' prefix) to the corresponding IR value.
func (fgen *funcGen) resolveLocals(body ast.FuncBody) (map[string]value.Value, error) {
	// Create instructions (without bodies), in preparation for index.
	f := fgen.f
	bbs := body.Blocks()
	for _, b := range bbs {
		blockName := label(*b.Name())
		block := ir.NewBlock(blockName)
		for _, i := range b.Insts() {
			inst, err := fgen.newIRInst(i)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			block.Insts = append(block.Insts, inst)
		}
		f.Blocks = append(f.Blocks, block)
	}
	// Assign local IDs.
	if err := f.AssignIDs(); err != nil {
		return nil, errors.WithStack(err)
	}
	// Index local identifiers.
	for _, param := range f.Params {
		if prev, ok := fgen.ls[param.ParamName]; ok {
			return nil, errors.Errorf("IR local identifier %q already present; prev `%s`, new `%s`", enc.Local(param.ParamName), prev, param)
		}
		fgen.ls[param.ParamName] = param
	}
	for _, block := range f.Blocks {
		if prev, ok := fgen.ls[block.LocalName]; ok {
			return nil, errors.Errorf("IR local identifier %q already present; prev `%s`, new `%s`", enc.Local(block.LocalName), prev, block)
		}
		// TODO: Rename block.LocalName to block.BlockName?
		fgen.ls[block.LocalName] = block
		for _, inst := range block.Insts {
			if n, ok := inst.(value.Named); ok {
				// Skip call instruction if callee has void return type.
				if n, ok := n.(*ir.InstCall); ok {
					if n.Type().Equal(types.Void) {
						continue
					}
				}
				if prev, ok := fgen.ls[n.Name()]; ok {
					return nil, errors.Errorf("IR local identifier %q already present; prev `%s`, new `%s`", enc.Local(n.Name()), prev, n)
				}
				fgen.ls[n.Name()] = n
			}
		}
	}
	// Translate instructions.
	for i, block := range f.Blocks {
		insts := bbs[i].Insts()
		for j, inst := range block.Insts {
			old := insts[j]
			if _, err := fgen.translateInst(inst, old); err != nil {
				return nil, errors.WithStack(err)
			}
		}
	}
	// Translate terminators.
	for i, block := range f.Blocks {
		old := bbs[i].Term()
		term, err := fgen.translateTerm(old)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		block.Term = term
	}
	return fgen.ls, nil
}

// newIRInst returns a new IR instruction (without body but with type) based on
// the given AST instruction.
func (fgen *funcGen) newIRInst(old ast.Instruction) (ir.Instruction, error) {
	switch old := old.(type) {
	// Value instruction.
	case *ast.LocalDef:
		name := local(old.Name())
		return fgen.newIRValueInst(name, old.Inst())
	case ast.ValueInstruction:
		return fgen.newIRValueInst("", old)
	// Non-value instructions.
	case *ast.StoreInst:
		return &ir.InstStore{}, nil
	case *ast.FenceInst:
		return &ir.InstFence{}, nil
	default:
		panic(fmt.Errorf("support for AST instruction type %T not yet implemented", old))
	}
}

// newIRValueInst returns a new IR value instruction (without body but with
// type) based on the given AST value instruction.
func (fgen *funcGen) newIRValueInst(name string, old ast.ValueInstruction) (ir.Instruction, error) {
	switch old := old.(type) {
	// Binary instructions
	case *ast.AddInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstAdd{LocalName: name, Typ: typ}, nil
	case *ast.FAddInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFAdd{LocalName: name, Typ: typ}, nil
	case *ast.SubInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstSub{LocalName: name, Typ: typ}, nil
	case *ast.FSubInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFSub{LocalName: name, Typ: typ}, nil
	case *ast.MulInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstMul{LocalName: name, Typ: typ}, nil
	case *ast.FMulInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFMul{LocalName: name, Typ: typ}, nil
	case *ast.UDivInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstUDiv{LocalName: name, Typ: typ}, nil
	case *ast.SDivInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstSDiv{LocalName: name, Typ: typ}, nil
	case *ast.FDivInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFDiv{LocalName: name, Typ: typ}, nil
	case *ast.URemInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstURem{LocalName: name, Typ: typ}, nil
	case *ast.SRemInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstSRem{LocalName: name, Typ: typ}, nil
	case *ast.FRemInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFRem{LocalName: name, Typ: typ}, nil
	// Bitwise instructions
	case *ast.ShlInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstShl{LocalName: name, Typ: typ}, nil
	case *ast.LShrInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstLShr{LocalName: name, Typ: typ}, nil
	case *ast.AShrInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstAShr{LocalName: name, Typ: typ}, nil
	case *ast.AndInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstAnd{LocalName: name, Typ: typ}, nil
	case *ast.OrInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstOr{LocalName: name, Typ: typ}, nil
	case *ast.XorInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstXor{LocalName: name, Typ: typ}, nil
	// Vector instructions
	case *ast.ExtractElementInst:
		xType, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		t, ok := xType.(*types.VectorType)
		if !ok {
			panic(fmt.Errorf("invalid vector type; expected *types.VectorType, got %T", xType))
		}
		return &ir.InstExtractElement{LocalName: name, Typ: t.ElemType}, nil
	case *ast.InsertElementInst:
		xType, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		t, ok := xType.(*types.VectorType)
		if !ok {
			panic(fmt.Errorf("invalid vector type; expected *types.VectorType, got %T", xType))
		}
		return &ir.InstInsertElement{LocalName: name, Typ: t}, nil
	case *ast.ShuffleVectorInst:
		xType, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		xt, ok := xType.(*types.VectorType)
		if !ok {
			panic(fmt.Errorf("invalid vector type; expected *types.VectorType, got %T", xType))
		}
		maskType, err := fgen.gen.irType(old.Mask().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		mt, ok := maskType.(*types.VectorType)
		if !ok {
			panic(fmt.Errorf("invalid vector type; expected *types.VectorType, got %T", maskType))
		}
		typ := types.NewVector(mt.Len, xt.ElemType)
		return &ir.InstShuffleVector{LocalName: name, Typ: typ}, nil
	// Aggregate instructions
	case *ast.ExtractValueInst:
		xType, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		indices := uintSlice(old.Indices())
		typ := aggregateElemType(xType, indices)
		return &ir.InstExtractValue{LocalName: name, Typ: typ}, nil
	case *ast.InsertValueInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstInsertValue{LocalName: name, Typ: typ}, nil
	// Memory instructions
	case *ast.AllocaInst:
		elemType, err := fgen.gen.irType(old.ElemType())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstAlloca{LocalName: name, ElemType: elemType}, nil
	case *ast.LoadInst:
		elemType, err := fgen.gen.irType(old.ElemType())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstLoad{LocalName: name, Typ: elemType}, nil
	case *ast.CmpXchgInst:
		oldType, err := fgen.gen.irType(old.New().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		typ := types.NewStruct(oldType, types.I8)
		return &ir.InstCmpXchg{LocalName: name, Typ: typ}, nil
	case *ast.AtomicRMWInst:
		return &ir.InstAtomicRMW{LocalName: name}, nil
	case *ast.GetElementPtrInst:
		return &ir.InstGetElementPtr{LocalName: name}, nil
	// Conversion instructions
	case *ast.TruncInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstTrunc{LocalName: name, To: to}, nil
	case *ast.ZExtInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstZExt{LocalName: name, To: to}, nil
	case *ast.SExtInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstSExt{LocalName: name, To: to}, nil
	case *ast.FPTruncInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFPTrunc{LocalName: name, To: to}, nil
	case *ast.FPExtInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFPExt{LocalName: name, To: to}, nil
	case *ast.FPToUIInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFPToUI{LocalName: name, To: to}, nil
	case *ast.FPToSIInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstFPToSI{LocalName: name, To: to}, nil
	case *ast.UIToFPInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstUIToFP{LocalName: name, To: to}, nil
	case *ast.SIToFPInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstSIToFP{LocalName: name, To: to}, nil
	case *ast.PtrToIntInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstPtrToInt{LocalName: name, To: to}, nil
	case *ast.IntToPtrInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstIntToPtr{LocalName: name, To: to}, nil
	case *ast.BitCastInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstBitCast{LocalName: name, To: to}, nil
	case *ast.AddrSpaceCastInst:
		to, err := fgen.gen.irType(old.To())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstAddrSpaceCast{LocalName: name, To: to}, nil
	// Other instructions
	case *ast.ICmpInst:
		return &ir.InstICmp{LocalName: name}, nil
	case *ast.FCmpInst:
		return &ir.InstFCmp{LocalName: name}, nil
	case *ast.PhiInst:
		typ, err := fgen.gen.irType(old.Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstPhi{LocalName: name, Typ: typ}, nil
	case *ast.SelectInst:
		typ, err := fgen.gen.irType(old.X().Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstSelect{LocalName: name, Typ: typ}, nil
	case *ast.CallInst:
		// NOTE: We need to store the type of call instructions before invoking
		// f.AssignIDs, since call instructions may be value instructions or
		// non-value instructions based on return type.
		typ, err := fgen.gen.irType(old.Typ())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &ir.InstCall{LocalName: name, Typ: typ}, nil
	case *ast.VAArgInst:
		return &ir.InstVAArg{LocalName: name}, nil
	case *ast.LandingPadInst:
		return &ir.InstLandingPad{LocalName: name}, nil
	case *ast.CatchPadInst:
		return &ir.InstCatchPad{LocalName: name}, nil
	case *ast.CleanupPadInst:
		return &ir.InstCleanupPad{LocalName: name}, nil
	default:
		panic(fmt.Errorf("support for AST value instruction type %T not yet implemented", old))
	}
}

// === [ Instructions ] ========================================================

// translateInst translates the AST instruction into an equivalent IR
// instruction.
func (fgen *funcGen) translateInst(inst ir.Instruction, old ast.Instruction) (ir.Instruction, error) {
	switch old := old.(type) {
	// Value instruction.
	case *ast.LocalDef:
		name := local(old.Name())
		v, ok := fgen.ls[name]
		if !ok {
			return nil, errors.Errorf("unable to locate local variable %q", name)
		}
		i, ok := v.(ir.Instruction)
		if !ok {
			return nil, errors.Errorf("invalid instruction type of %q; expected ir.Instruction, got %T", name, v)
		}
		return fgen.translateValueInst(i, old.Inst())
	case ast.ValueInstruction:
		return fgen.translateValueInst(inst, old)
	// Non-value instructions.
	case *ast.StoreInst:
		return fgen.translateStoreInst(inst, old)
	case *ast.FenceInst:
		return fgen.translateFenceInst(inst, old)
	default:
		panic(fmt.Errorf("support for instruction type %T not yet implemented", old))
	}
}

// translateValueInst translates the AST value instruction into an equivalent IR
// value instruction.
func (fgen *funcGen) translateValueInst(inst ir.Instruction, old ast.ValueInstruction) (ir.Instruction, error) {
	switch old := old.(type) {
	// Binary instructions
	case *ast.AddInst:
		return fgen.translateAddInst(inst, old)
	case *ast.FAddInst:
		return fgen.translateFAddInst(inst, old)
	case *ast.SubInst:
		return fgen.translateSubInst(inst, old)
	case *ast.FSubInst:
		return fgen.translateFSubInst(inst, old)
	case *ast.MulInst:
		return fgen.translateMulInst(inst, old)
	case *ast.FMulInst:
		return fgen.translateFMulInst(inst, old)
	case *ast.UDivInst:
		return fgen.translateUDivInst(inst, old)
	case *ast.SDivInst:
		return fgen.translateSDivInst(inst, old)
	case *ast.FDivInst:
		return fgen.translateFDivInst(inst, old)
	case *ast.URemInst:
		return fgen.translateURemInst(inst, old)
	case *ast.SRemInst:
		return fgen.translateSRemInst(inst, old)
	case *ast.FRemInst:
		return fgen.translateFRemInst(inst, old)
	// Bitwise instructions
	case *ast.ShlInst:
		return fgen.translateShlInst(inst, old)
	case *ast.LShrInst:
		return fgen.translateLShrInst(inst, old)
	case *ast.AShrInst:
		return fgen.translateAShrInst(inst, old)
	case *ast.AndInst:
		return fgen.translateAndInst(inst, old)
	case *ast.OrInst:
		return fgen.translateOrInst(inst, old)
	case *ast.XorInst:
		return fgen.translateXorInst(inst, old)
	// Vector instructions
	case *ast.ExtractElementInst:
		return fgen.translateExtractElementInst(inst, old)
	case *ast.InsertElementInst:
		return fgen.translateInsertElementInst(inst, old)
	case *ast.ShuffleVectorInst:
		return fgen.translateShuffleVectorInst(inst, old)
	// Aggregate instructions
	case *ast.ExtractValueInst:
		return fgen.translateExtractValueInst(inst, old)
	case *ast.InsertValueInst:
		return fgen.translateInsertValueInst(inst, old)
	// Memory instructions
	case *ast.AllocaInst:
		return fgen.translateAllocaInst(inst, old)
	case *ast.LoadInst:
		return fgen.translateLoadInst(inst, old)
	case *ast.CmpXchgInst:
		return fgen.translateCmpXchgInst(inst, old)
	case *ast.AtomicRMWInst:
		return fgen.translateAtomicRMWInst(inst, old)
	case *ast.GetElementPtrInst:
		return fgen.translateGetElementPtrInst(inst, old)
	// Conversion instructions
	case *ast.TruncInst:
		return fgen.translateTruncInst(inst, old)
	case *ast.ZExtInst:
		return fgen.translateZExtInst(inst, old)
	case *ast.SExtInst:
		return fgen.translateSExtInst(inst, old)
	case *ast.FPTruncInst:
		return fgen.translateFPTruncInst(inst, old)
	case *ast.FPExtInst:
		return fgen.translateFPExtInst(inst, old)
	case *ast.FPToUIInst:
		return fgen.translateFPToUIInst(inst, old)
	case *ast.FPToSIInst:
		return fgen.translateFPToSIInst(inst, old)
	case *ast.UIToFPInst:
		return fgen.translateUIToFPInst(inst, old)
	case *ast.SIToFPInst:
		return fgen.translateSIToFPInst(inst, old)
	case *ast.PtrToIntInst:
		return fgen.translatePtrToIntInst(inst, old)
	case *ast.IntToPtrInst:
		return fgen.translateIntToPtrInst(inst, old)
	case *ast.BitCastInst:
		return fgen.translateBitCastInst(inst, old)
	case *ast.AddrSpaceCastInst:
		return fgen.translateAddrSpaceCastInst(inst, old)
	// Other instructions
	case *ast.ICmpInst:
		return fgen.translateICmpInst(inst, old)
	case *ast.FCmpInst:
		return fgen.translateFCmpInst(inst, old)
	case *ast.PhiInst:
		return fgen.translatePhiInst(inst, old)
	case *ast.SelectInst:
		return fgen.translateSelectInst(inst, old)
	case *ast.CallInst:
		return fgen.translateCallInst(inst, old)
	case *ast.VAArgInst:
		return fgen.translateVAArgInst(inst, old)
	case *ast.LandingPadInst:
		return fgen.translateLandingPadInst(inst, old)
	case *ast.CatchPadInst:
		return fgen.translateCatchPadInst(inst, old)
	case *ast.CleanupPadInst:
		return fgen.translateCleanupPadInst(inst, old)
	default:
		panic(fmt.Errorf("support for value instruction type %T not yet implemented", old))
	}
}

// --- [ Binary instructions ] -------------------------------------------------

// ~~~ [ add ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateAddInst(inst ir.Instruction, old *ast.AddInst) (*ir.InstAdd, error) {
	i, ok := inst.(*ir.InstAdd)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstAdd, got %T", inst))
	}
	// Overflow flags.
	i.OverflowFlags = irOverflowFlags(old.OverflowFlags())
	// X operand.
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(x.Type(), old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// ~~~ [ fadd ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateFAddInst(inst ir.Instruction, old *ast.FAddInst) (*ir.InstFAdd, error) {
	i, ok := inst.(*ir.InstFAdd)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstFAdd, got %T", inst))
	}
	// Fast math flags.
	i.FastMathFlags = irFastMathFlags(old.FastMathFlags())
	// X operand.
	// TODO: remove xType in favour of x.Type().
	xType, err := fgen.gen.irType(old.X().Typ())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(xType, old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// ~~~ [ sub ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateSubInst(inst ir.Instruction, old *ast.SubInst) (*ir.InstSub, error) {
	i, ok := inst.(*ir.InstSub)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstSub, got %T", inst))
	}
	// Overflow flags.
	i.OverflowFlags = irOverflowFlags(old.OverflowFlags())
	// X operand.
	xType, err := fgen.gen.irType(old.X().Typ())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(xType, old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// ~~~ [ fsub ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateFSubInst(inst ir.Instruction, old *ast.FSubInst) (*ir.InstFSub, error) {
	i, ok := inst.(*ir.InstFSub)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstFSub, got %T", inst))
	}
	// Fast math flags.
	i.FastMathFlags = irFastMathFlags(old.FastMathFlags())
	// X operand.
	xType, err := fgen.gen.irType(old.X().Typ())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(xType, old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// ~~~ [ mul ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateMulInst(inst ir.Instruction, old *ast.MulInst) (*ir.InstMul, error) {
	i, ok := inst.(*ir.InstMul)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstMul, got %T", inst))
	}
	// Overflow flags.
	i.OverflowFlags = irOverflowFlags(old.OverflowFlags())
	// X operand.
	xType, err := fgen.gen.irType(old.X().Typ())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(xType, old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// ~~~ [ fmul ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateFMulInst(inst ir.Instruction, old *ast.FMulInst) (*ir.InstFMul, error) {
	i, ok := inst.(*ir.InstFMul)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstFMul, got %T", inst))
	}
	// Fast math flags.
	i.FastMathFlags = irFastMathFlags(old.FastMathFlags())
	// X operand.
	xType, err := fgen.gen.irType(old.X().Typ())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(xType, old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// ~~~ [ udiv ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateUDivInst(inst ir.Instruction, old *ast.UDivInst) (*ir.InstUDiv, error) {
	i, ok := inst.(*ir.InstUDiv)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstUDiv, got %T", inst))
	}
	// X operand.
	xType, err := fgen.gen.irType(old.X().Typ())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(xType, old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// ~~~ [ sdiv ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateSDivInst(inst ir.Instruction, old *ast.SDivInst) (*ir.InstSDiv, error) {
	i, ok := inst.(*ir.InstSDiv)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstSDiv, got %T", inst))
	}
	// X operand.
	xType, err := fgen.gen.irType(old.X().Typ())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(xType, old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// ~~~ [ fdiv ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateFDivInst(inst ir.Instruction, old *ast.FDivInst) (*ir.InstFDiv, error) {
	i, ok := inst.(*ir.InstFDiv)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstFDiv, got %T", inst))
	}
	// Fast math flags.
	i.FastMathFlags = irFastMathFlags(old.FastMathFlags())
	// X operand.
	xType, err := fgen.gen.irType(old.X().Typ())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(xType, old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// ~~~ [ urem ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateURemInst(inst ir.Instruction, old *ast.URemInst) (*ir.InstURem, error) {
	i, ok := inst.(*ir.InstURem)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstURem, got %T", inst))
	}
	// X operand.
	xType, err := fgen.gen.irType(old.X().Typ())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(xType, old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// ~~~ [ srem ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateSRemInst(inst ir.Instruction, old *ast.SRemInst) (*ir.InstSRem, error) {
	i, ok := inst.(*ir.InstSRem)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstSRem, got %T", inst))
	}
	// X operand.
	xType, err := fgen.gen.irType(old.X().Typ())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(xType, old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// ~~~ [ frem ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateFRemInst(inst ir.Instruction, old *ast.FRemInst) (*ir.InstFRem, error) {
	i, ok := inst.(*ir.InstFRem)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstFRem, got %T", inst))
	}
	// Fast math flags.
	i.FastMathFlags = irFastMathFlags(old.FastMathFlags())
	// X operand.
	xType, err := fgen.gen.irType(old.X().Typ())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(xType, old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// --- [ Bitwise instructions ] ------------------------------------------------

// ~~~ [ shl ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateShlInst(inst ir.Instruction, old *ast.ShlInst) (*ir.InstShl, error) {
	i, ok := inst.(*ir.InstShl)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstShl, got %T", inst))
	}
	// Overflow flags.
	i.OverflowFlags = irOverflowFlags(old.OverflowFlags())
	// X operand.
	xType, err := fgen.gen.irType(old.X().Typ())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(xType, old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// ~~~ [ lshr ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateLShrInst(inst ir.Instruction, old *ast.LShrInst) (*ir.InstLShr, error) {
	i, ok := inst.(*ir.InstLShr)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstLShr, got %T", inst))
	}
	// X operand.
	xType, err := fgen.gen.irType(old.X().Typ())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(xType, old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// ~~~ [ ashr ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateAShrInst(inst ir.Instruction, old *ast.AShrInst) (*ir.InstAShr, error) {
	i, ok := inst.(*ir.InstAShr)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstAShr, got %T", inst))
	}
	// X operand.
	xType, err := fgen.gen.irType(old.X().Typ())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(xType, old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// ~~~ [ and ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateAndInst(inst ir.Instruction, old *ast.AndInst) (*ir.InstAnd, error) {
	i, ok := inst.(*ir.InstAnd)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstAnd, got %T", inst))
	}
	// X operand.
	xType, err := fgen.gen.irType(old.X().Typ())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(xType, old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// ~~~ [ or ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateOrInst(inst ir.Instruction, old *ast.OrInst) (*ir.InstOr, error) {
	i, ok := inst.(*ir.InstOr)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstOr, got %T", inst))
	}
	// X operand.
	xType, err := fgen.gen.irType(old.X().Typ())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(xType, old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// ~~~ [ xor ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateXorInst(inst ir.Instruction, old *ast.XorInst) (*ir.InstXor, error) {
	i, ok := inst.(*ir.InstXor)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstXor, got %T", inst))
	}
	// X operand.
	xType, err := fgen.gen.irType(old.X().Typ())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	x, err := fgen.astToIRTypeValue(old.X())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.X = x
	// Y operand.
	y, err := fgen.astToIRValue(xType, old.Y())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	i.Y = y
	return i, nil
}

// --- [ Vector instructions ] -------------------------------------------------

// ~~~ [ extractelement ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateExtractElementInst(inst ir.Instruction, old *ast.ExtractElementInst) (*ir.InstExtractElement, error) {
	i, ok := inst.(*ir.InstExtractElement)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstExtractElement, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ insertelement ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateInsertElementInst(inst ir.Instruction, old *ast.InsertElementInst) (*ir.InstInsertElement, error) {
	i, ok := inst.(*ir.InstInsertElement)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstInsertElement, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ shufflevector ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateShuffleVectorInst(inst ir.Instruction, old *ast.ShuffleVectorInst) (*ir.InstShuffleVector, error) {
	i, ok := inst.(*ir.InstShuffleVector)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstShuffleVector, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// --- [ Aggregate instructions ] ----------------------------------------------

// ~~~ [ extractvalue ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateExtractValueInst(inst ir.Instruction, old *ast.ExtractValueInst) (*ir.InstExtractValue, error) {
	i, ok := inst.(*ir.InstExtractValue)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstExtractValue, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ insertvalue ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateInsertValueInst(inst ir.Instruction, old *ast.InsertValueInst) (*ir.InstInsertValue, error) {
	i, ok := inst.(*ir.InstInsertValue)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstInsertValue, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// --- [ Memory instructions ] -------------------------------------------------

// ~~~ [ alloca ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateAllocaInst(inst ir.Instruction, old *ast.AllocaInst) (*ir.InstAlloca, error) {
	i, ok := inst.(*ir.InstAlloca)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstAlloca, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ load ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateLoadInst(inst ir.Instruction, old *ast.LoadInst) (*ir.InstLoad, error) {
	i, ok := inst.(*ir.InstLoad)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstLoad, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ store ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateStoreInst(inst ir.Instruction, old *ast.StoreInst) (*ir.InstStore, error) {
	i, ok := inst.(*ir.InstStore)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstStore, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ fence ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateFenceInst(inst ir.Instruction, old *ast.FenceInst) (*ir.InstFence, error) {
	i, ok := inst.(*ir.InstFence)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstFence, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ cmpxchg ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateCmpXchgInst(inst ir.Instruction, old *ast.CmpXchgInst) (*ir.InstCmpXchg, error) {
	i, ok := inst.(*ir.InstCmpXchg)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstCmpXchg, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ atomicrmw ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateAtomicRMWInst(inst ir.Instruction, old *ast.AtomicRMWInst) (*ir.InstAtomicRMW, error) {
	i, ok := inst.(*ir.InstAtomicRMW)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstAtomicRMW, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ getelementptr ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateGetElementPtrInst(inst ir.Instruction, old *ast.GetElementPtrInst) (*ir.InstGetElementPtr, error) {
	i, ok := inst.(*ir.InstGetElementPtr)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstGetElementPtr, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// --- [ Conversion instructions ] ---------------------------------------------

// ~~~ [ trunc ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateTruncInst(inst ir.Instruction, old *ast.TruncInst) (*ir.InstTrunc, error) {
	i, ok := inst.(*ir.InstTrunc)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstTrunc, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ zext ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateZExtInst(inst ir.Instruction, old *ast.ZExtInst) (*ir.InstZExt, error) {
	i, ok := inst.(*ir.InstZExt)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstZExt, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ sext ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateSExtInst(inst ir.Instruction, old *ast.SExtInst) (*ir.InstSExt, error) {
	i, ok := inst.(*ir.InstSExt)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstSExt, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ fptrunc ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateFPTruncInst(inst ir.Instruction, old *ast.FPTruncInst) (*ir.InstFPTrunc, error) {
	i, ok := inst.(*ir.InstFPTrunc)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstFPTrunc, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ fpext ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateFPExtInst(inst ir.Instruction, old *ast.FPExtInst) (*ir.InstFPExt, error) {
	i, ok := inst.(*ir.InstFPExt)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstFPExt, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ fptoui ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateFPToUIInst(inst ir.Instruction, old *ast.FPToUIInst) (*ir.InstFPToUI, error) {
	i, ok := inst.(*ir.InstFPToUI)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstFPToUI, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ fptosi ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateFPToSIInst(inst ir.Instruction, old *ast.FPToSIInst) (*ir.InstFPToSI, error) {
	i, ok := inst.(*ir.InstFPToSI)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstFPToSI, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ uitofp ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateUIToFPInst(inst ir.Instruction, old *ast.UIToFPInst) (*ir.InstUIToFP, error) {
	i, ok := inst.(*ir.InstUIToFP)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstUIToFP, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ sitofp ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateSIToFPInst(inst ir.Instruction, old *ast.SIToFPInst) (*ir.InstSIToFP, error) {
	i, ok := inst.(*ir.InstSIToFP)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstSIToFP, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ ptrtoint ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translatePtrToIntInst(inst ir.Instruction, old *ast.PtrToIntInst) (*ir.InstPtrToInt, error) {
	i, ok := inst.(*ir.InstPtrToInt)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstPtrToInt, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ inttoptr ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateIntToPtrInst(inst ir.Instruction, old *ast.IntToPtrInst) (*ir.InstIntToPtr, error) {
	i, ok := inst.(*ir.InstIntToPtr)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstIntToPtr, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ bitcast ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateBitCastInst(inst ir.Instruction, old *ast.BitCastInst) (*ir.InstBitCast, error) {
	i, ok := inst.(*ir.InstBitCast)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstBitCast, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ addrspacecast ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateAddrSpaceCastInst(inst ir.Instruction, old *ast.AddrSpaceCastInst) (*ir.InstAddrSpaceCast, error) {
	i, ok := inst.(*ir.InstAddrSpaceCast)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstAddrSpaceCast, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// --- [ Other instructions ] --------------------------------------------------

// ~~~ [ icmp ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateICmpInst(inst ir.Instruction, old *ast.ICmpInst) (*ir.InstICmp, error) {
	i, ok := inst.(*ir.InstICmp)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstICmp, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ fcmp ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateFCmpInst(inst ir.Instruction, old *ast.FCmpInst) (*ir.InstFCmp, error) {
	i, ok := inst.(*ir.InstFCmp)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstFCmp, got %T", inst))
	}
	// Fast math flags.
	i.FastMathFlags = irFastMathFlags(old.FastMathFlags())
	// TODO: implement
	return i, nil
}

// ~~~ [ phi ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translatePhiInst(inst ir.Instruction, old *ast.PhiInst) (*ir.InstPhi, error) {
	i, ok := inst.(*ir.InstPhi)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstPhi, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ select ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateSelectInst(inst ir.Instruction, old *ast.SelectInst) (*ir.InstSelect, error) {
	i, ok := inst.(*ir.InstSelect)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstSelect, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ call ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateCallInst(inst ir.Instruction, old *ast.CallInst) (*ir.InstCall, error) {
	i, ok := inst.(*ir.InstCall)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstCall, got %T", inst))
	}
	// Fast math flags.
	i.FastMathFlags = irFastMathFlags(old.FastMathFlags())
	// TODO: implement
	return i, nil
}

// ~~~ [ va_arg ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateVAArgInst(inst ir.Instruction, old *ast.VAArgInst) (*ir.InstVAArg, error) {
	i, ok := inst.(*ir.InstVAArg)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstVAArg, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ landingpad ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateLandingPadInst(inst ir.Instruction, old *ast.LandingPadInst) (*ir.InstLandingPad, error) {
	i, ok := inst.(*ir.InstLandingPad)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstLandingPad, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ catchpad ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateCatchPadInst(inst ir.Instruction, old *ast.CatchPadInst) (*ir.InstCatchPad, error) {
	i, ok := inst.(*ir.InstCatchPad)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstCatchPad, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// ~~~ [ cleanuppad ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (fgen *funcGen) translateCleanupPadInst(inst ir.Instruction, old *ast.CleanupPadInst) (*ir.InstCleanupPad, error) {
	i, ok := inst.(*ir.InstCleanupPad)
	if !ok {
		// NOTE: panic since this would indicate a bug in the implementation.
		panic(fmt.Errorf("invalid IR instruction for AST instruction; expected *ir.InstCleanupPad, got %T", inst))
	}
	// TODO: implement
	return i, nil
}

// === [ Terminators ] =========================================================

// translateTerm translates the AST terminator into an equivalent IR terminator.
func (fgen *funcGen) translateTerm(old ast.Terminator) (ir.Terminator, error) {
	switch old := old.(type) {
	case *ast.RetTerm:
		return fgen.translateRetTerm(old)
	case *ast.BrTerm:
		return fgen.translateBrTerm(old)
	case *ast.CondBrTerm:
		return fgen.translateCondBrTerm(old)
	case *ast.SwitchTerm:
		return fgen.translateSwitchTerm(old)
	case *ast.IndirectBrTerm:
		return fgen.translateIndirectBrTerm(old)
	case *ast.InvokeTerm:
		return fgen.translateInvokeTerm(old)
	case *ast.ResumeTerm:
		return fgen.translateResumeTerm(old)
	case *ast.CatchSwitchTerm:
		return fgen.translateCatchSwitchTerm(old)
	case *ast.CatchRetTerm:
		return fgen.translateCatchRetTerm(old)
	case *ast.CleanupRetTerm:
		return fgen.translateCleanupRetTerm(old)
	case *ast.UnreachableTerm:
		return fgen.translateUnreachableTerm(old)
	default:
		panic(fmt.Errorf("support for AST terminator type %T not yet implemented", old))
	}
}

// --- [ ret ] -----------------------------------------------------------------

func (fgen *funcGen) translateRetTerm(old *ast.RetTerm) (*ir.TermRet, error) {
	term := &ir.TermRet{}
	// TODO: implement
	return term, nil
}

// --- [ br ] ------------------------------------------------------------------

func (fgen *funcGen) translateBrTerm(old *ast.BrTerm) (*ir.TermBr, error) {
	term := &ir.TermBr{}
	// TODO: implement
	return term, nil
}

func (fgen *funcGen) translateCondBrTerm(old *ast.CondBrTerm) (*ir.TermCondBr, error) {
	term := &ir.TermCondBr{}
	// TODO: implement
	return term, nil
}

// --- [ switch ] --------------------------------------------------------------

func (fgen *funcGen) translateSwitchTerm(old *ast.SwitchTerm) (*ir.TermSwitch, error) {
	term := &ir.TermSwitch{}
	// TODO: implement
	return term, nil
}

// --- [ indirectbr ] ----------------------------------------------------------

func (fgen *funcGen) translateIndirectBrTerm(old *ast.IndirectBrTerm) (*ir.TermIndirectBr, error) {
	term := &ir.TermIndirectBr{}
	// TODO: implement
	return term, nil
}

// --- [ invoke ] --------------------------------------------------------------

func (fgen *funcGen) translateInvokeTerm(old *ast.InvokeTerm) (*ir.TermInvoke, error) {
	term := &ir.TermInvoke{}
	// TODO: implement
	return term, nil
}

// --- [ resume ] --------------------------------------------------------------

func (fgen *funcGen) translateResumeTerm(old *ast.ResumeTerm) (*ir.TermResume, error) {
	term := &ir.TermResume{}
	// TODO: implement
	return term, nil
}

// --- [ catchswitch ] ---------------------------------------------------------

func (fgen *funcGen) translateCatchSwitchTerm(old *ast.CatchSwitchTerm) (*ir.TermCatchSwitch, error) {
	term := &ir.TermCatchSwitch{}
	// TODO: implement
	return term, nil
}

// --- [ catchret ] ------------------------------------------------------------

func (fgen *funcGen) translateCatchRetTerm(old *ast.CatchRetTerm) (*ir.TermCatchRet, error) {
	term := &ir.TermCatchRet{}
	// TODO: implement
	return term, nil
}

// --- [ cleanupret ] ----------------------------------------------------------

func (fgen *funcGen) translateCleanupRetTerm(old *ast.CleanupRetTerm) (*ir.TermCleanupRet, error) {
	term := &ir.TermCleanupRet{}
	// TODO: implement
	return term, nil
}

// --- [ unreachable ] ---------------------------------------------------------

func (fgen *funcGen) translateUnreachableTerm(old *ast.UnreachableTerm) (*ir.TermUnreachable, error) {
	term := &ir.TermUnreachable{}
	// TODO: implement
	return term, nil
}

// ### [ Helper functions ] ####################################################

// NOTE: aggregateElemType is copied from llir/l/ir/inst_aggregate.go and the
// type of indicies is updated from []int64 to []uint64

// aggregateElemType returns the element type at the position in the aggregate
// type specified by the given indices.
func aggregateElemType(t types.Type, indices []uint64) types.Type {
	// Base case.
	if len(indices) == 0 {
		return t
	}
	switch t := t.(type) {
	case *types.ArrayType:
		return aggregateElemType(t.ElemType, indices[1:])
	case *types.StructType:
		return aggregateElemType(t.Fields[indices[0]], indices[1:])
	default:
		panic(fmt.Errorf("support for aggregate type %T not yet implemented", t))
	}
}
