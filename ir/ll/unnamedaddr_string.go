// Code generated by "stringer -linecomment -type UnnamedAddr"; DO NOT EDIT.

package ll

import "strconv"

const _UnnamedAddr_name = "nonelocal_unnamed_addrunnamed_addr"

var _UnnamedAddr_index = [...]uint8{0, 4, 22, 34}

func (i UnnamedAddr) String() string {
	if i >= UnnamedAddr(len(_UnnamedAddr_index)-1) {
		return "UnnamedAddr(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _UnnamedAddr_name[_UnnamedAddr_index[i]:_UnnamedAddr_index[i+1]]
}
