package golang

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/gopls/internal/protocol/semtok"
)

func isDefaultLibraryBaseType(t types.Type) bool {
	t = types.Unalias(t)

	switch tt := t.(type) {
	case *types.Basic:
		return tt.Kind() != types.Invalid

	case *types.Named:
		if obj := tt.Obj(); obj != nil && obj.Pkg() == nil {
			return true
		}
		return false

	case *types.Alias:
		if obj := tt.Obj(); obj != nil && obj.Pkg() == nil {
			return true
		}
		return false
	}

	return false
}

func derefPointers(t types.Type) types.Type {
	for {
		u := types.Unalias(t).Underlying()
		p, ok := u.(*types.Pointer)
		if !ok {
			return t
		}
		t = p.Elem()
	}
}

func stripTypeMods(mods []semtok.Modifier) []semtok.Modifier {
	out := mods[:0]
	for _, m := range mods {
		switch m {
		case semtok.ModInterface, semtok.ModStruct, semtok.ModSignature,
			semtok.ModPointer, semtok.ModArray, semtok.ModMap, semtok.ModSlice, semtok.ModChan,
			semtok.ModString, semtok.ModBool, semtok.ModNumber:
		default:
			out = append(out, m)
		}
	}
	return out
}

// inEmbeddedStructFieldDecl reports whether the current Ident is part of the
// type expression of an embedded field in a struct type declaration:
//
//	type S struct { pkg.T; *U; V }
//
// It is intentionally AST-contextual: we want the "embedded" modifier only
// at the field declaration site, not at uses like s.T or in composite keys.
func (tv *tokenVisitor) inEmbeddedStructFieldDecl() bool {
	// stack is [..., Ident]; check nearest enclosing *ast.Field
	for i := len(tv.stack) - 2; i >= 0; i-- { // parent of Ident and above
		f, ok := tv.stack[i].(*ast.Field)
		if !ok {
			continue
		}

		// Embedded field has no explicit names: `T` or `*T` or `pkg.T`
		if f.Names != nil {
			return false
		}

		// Ensure we're in a struct type field list (not interface methods,
		// func params, etc.). Typical stack: StructType -> FieldList -> Field -> ...
		if i >= 2 {
			if fl, ok := tv.stack[i-1].(*ast.FieldList); ok {
				if st, ok := tv.stack[i-2].(*ast.StructType); ok && st.Fields == fl {
					return true
				}
			}
		}
		return false
	}
	return false
}
