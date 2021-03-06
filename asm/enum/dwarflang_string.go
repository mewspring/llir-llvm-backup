// Code generated by "string2enum -linecomment -type DwarfLang /home/u/Desktop/go/src/github.com/llir/llvm/ir/enum"; DO NOT EDIT.

package enum

import "fmt"
import "github.com/llir/llvm/ir/enum"

const (
	_DwarfLang_name_0 = "DW_LANG_C89DW_LANG_CDW_LANG_Ada83DW_LANG_C_plus_plusDW_LANG_Cobol74DW_LANG_Cobol85DW_LANG_Fortran77DW_LANG_Fortran90DW_LANG_Pascal83DW_LANG_Modula2DW_LANG_JavaDW_LANG_C99DW_LANG_Ada95DW_LANG_Fortran95DW_LANG_PLIDW_LANG_ObjCDW_LANG_ObjC_plus_plusDW_LANG_UPCDW_LANG_DDW_LANG_PythonDW_LANG_OpenCLDW_LANG_GoDW_LANG_Modula3DW_LANG_HaskellDW_LANG_C_plus_plus_03DW_LANG_C_plus_plus_11DW_LANG_OCamlDW_LANG_RustDW_LANG_C11DW_LANG_SwiftDW_LANG_JuliaDW_LANG_DylanDW_LANG_C_plus_plus_14DW_LANG_Fortran03DW_LANG_Fortran08DW_LANG_RenderScriptDW_LANG_BLISS"
	_DwarfLang_name_1 = "DW_LANG_Mips_Assembler"
	_DwarfLang_name_2 = "DW_LANG_GOOGLE_RenderScript"
	_DwarfLang_name_3 = "DW_LANG_BORLAND_Delphi"
)

var (
	_DwarfLang_index_0 = [...]uint16{0, 11, 20, 33, 52, 67, 82, 99, 116, 132, 147, 159, 170, 183, 200, 211, 223, 245, 256, 265, 279, 293, 303, 318, 333, 355, 377, 390, 402, 413, 426, 439, 452, 474, 491, 508, 528, 541}
)

func DwarfLangFromString(s string) enum.DwarfLang {
	if len(s) == 0 {
		return 0
	}
	for i := range _DwarfLang_index_0[:len(_DwarfLang_index_0)-1] {
		if s == _DwarfLang_name_0[_DwarfLang_index_0[i]:_DwarfLang_index_0[i+1]] {
			return enum.DwarfLang(i + 1)
		}
	}
	if s == _DwarfLang_name_1 {
		return enum.DwarfLang(32769)
	}
	if s == _DwarfLang_name_2 {
		return enum.DwarfLang(36439)
	}
	if s == _DwarfLang_name_3 {
		return enum.DwarfLang(45056)
	}
	panic(fmt.Errorf("unable to locate DwarfLang enum corresponding to %q", s))
}
