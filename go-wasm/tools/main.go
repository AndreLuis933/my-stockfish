//go:build ignore

// gen_wasm_types gera wasm-contract.ts a partir dos tipos reais do Go.
//
// Estrategia em duas etapas:
//   1. Le cmd/wasm/main.go via AST puro (sem type-check) para achar
//      engine.Set("add", js.FuncOf(addJS)) e o que addJS chama internamente.
//   2. Le o pacote de logica pura (pkg/engine) com type-check completo
//      para extrair as assinaturas reais e gerar interfaces de structs.
//
// Nao precisa compilar para wasm — le so o AST do main e os tipos do engine.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"golang.org/x/tools/go/packages"
)
//  go build -o bin\gen-types.exe tools/main.go
func main() {
	// ── 1. Parse AST do main.go do WASM ──────────────────────────────────
	fset := token.NewFileSet()
	wasmFile, err := parser.ParseFile(fset, "cmd/wasm/main.go", nil, 0)
	if err != nil {
		panic(fmt.Sprintf("erro ao parsear main.go: %v", err))
	}

	// ── 2. Acha engine.Set("jsName", js.FuncOf(bridgeFunc)) ──────────────
	registered := map[string]string{} // jsName -> nome da bridge

	ast.Inspect(wasmFile, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Set" || len(call.Args) != 2 {
			return true
		}

		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok {
			return true
		}
		jsName := strings.Trim(lit.Value, `"`)

		inner, ok := call.Args[1].(*ast.CallExpr)
		if !ok || len(inner.Args) != 1 {
			return true
		}
		bridgeIdent, ok := inner.Args[0].(*ast.Ident)
		if !ok {
			return true
		}

		registered[jsName] = bridgeIdent.Name
		return true
	})

	// ── 3. Acha qual funcao pura cada bridge chama ────────────────────────
	bridgeToPure := map[string]string{} // bridgeName -> "engine.Add"

	funcDecls := map[string]*ast.FuncDecl{}
	for _, decl := range wasmFile.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			funcDecls[fn.Name.Name] = fn
		}
	}

	for _, bridgeName := range registered {
		bridgeDecl, ok := funcDecls[bridgeName]
		if !ok {
			continue
		}

		ast.Inspect(bridgeDecl.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			pkgIdent, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}

			pkgName := pkgIdent.Name
			if pkgName == "js" || pkgName == "args" || pkgName == "json" {
				return true
			}

			if _, exists := bridgeToPure[bridgeName]; !exists {
				bridgeToPure[bridgeName] = fmt.Sprintf("%s.%s", pkgName, sel.Sel.Name)
			}
			return true
		})
	}

	// ── 4. Type-check do pacote de logica pura + AST para structs ─────────
	pureFset := token.NewFileSet()
	purePkg, err := parser.ParseDir(pureFset, "pkg/engine", nil, parser.ParseComments)
	if err != nil {
		panic(fmt.Sprintf("erro ao parsear pkg/engine: %v", err))
	}

	var pureFiles []*ast.File
	for _, pkg := range purePkg {
		for _, f := range pkg.Files {
			pureFiles = append(pureFiles, f)
		}
	}

	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedFiles | packages.NeedName,
		Dir:  ".",
		Fset: pureFset,
	}

	pkgs, err := packages.Load(cfg, "./pkg/engine")
	if err != nil {
		panic(fmt.Sprintf("erro ao carregar pacote: %v", err))
	}
	if len(pkgs) != 1 {
		panic(fmt.Sprintf("esperado 1 pacote, encontrados %d", len(pkgs)))
	}

	loadedPkg := pkgs[0]
	if len(loadedPkg.Errors) > 0 {
		panic(fmt.Sprintf("erros no pacote: %v", loadedPkg.Errors))
	}

	checkedPkg := loadedPkg.Types
	info := loadedPkg.TypesInfo
	if info == nil {
		panic("TypesInfo nao disponivel")
	}

	// Coleta AST de structs para gerar interfaces depois
	structASTs := map[string]*ast.StructType{}
	for _, f := range loadedPkg.Syntax {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				structASTs[ts.Name.Name] = st
			}
		}
	}

	// ── 5. Extrai assinatura de cada funcao pura ──────────────────────────
	pureSignatures := map[string]*types.Signature{} // "engine.Add" -> Signature

	scope := checkedPkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		fn, ok := obj.(*types.Func)
		if !ok {
			continue
		}
		sig := fn.Type().(*types.Signature)
		key := fmt.Sprintf("engine.%s", name)
		pureSignatures[key] = sig
	}

	// ── 6. Monta resultado final ──────────────────────────────────────────
	contractEntries := map[string]string{} // jsName -> contract entry
	usedStructs := map[string]bool{}       // structs usadas em args/returns

	for jsName, bridgeName := range registered {
		pureKey, ok := bridgeToPure[bridgeName]
		if !ok {
			fmt.Printf("⚠️  %q: bridge %q nao tem chamada pura rastreavel\n", jsName, bridgeName)
			contractEntries[jsName] = "{ args: unknown[]; return: unknown }"
			continue
		}

		sig, ok := pureSignatures[pureKey]
		if !ok {
			fmt.Printf("⚠️  %q: funcao %q nao encontrada no type-check\n", jsName, pureKey)
			contractEntries[jsName] = "{ args: unknown[]; return: unknown }"
			continue
		}

		contractEntries[jsName] = buildContractEntry(sig)
		collectUsedStructs(sig, usedStructs)
		fmt.Printf("✅  %-15q -> %-25s -> %s\n", jsName, pureKey, contractEntries[jsName])
	}

	// ── 7. Gera interfaces de structs ─────────────────────────────────────
	var structInterfaces []string
	for name := range usedStructs {
		st, ok := structASTs[name]
		if !ok {
			continue
		}
		iface := buildStructInterface(name, st, info, pureFset)
		if iface != "" {
			structInterfaces = append(structInterfaces, iface)
		}
	}

	// ── 8. Gera wasm-contract.ts ──────────────────────────────────────────
	var sb strings.Builder
	sb.WriteString("// AUTO-GENERATED by tools/gen_wasm_types/main.go\n")
	sb.WriteString("// Do not edit manually. Run: go run tools/gen_wasm_types/main.go\n\n")

	if len(structInterfaces) > 0 {
		sb.WriteString("// --- Struct Interfaces ---\n")
		for _, iface := range structInterfaces {
			sb.WriteString(iface)
			sb.WriteString("\n\n")
		}
	}

	sb.WriteString("// --- Function Contract ---\n")
	sb.WriteString("export interface WasmContract {\n")
	for jsName, entry := range contractEntries {
		sb.WriteString(fmt.Sprintf("  %s: %s;\n", jsName, entry))
	}
	sb.WriteString("}\n\n")

	sb.WriteString("export type WasmFunctionName = keyof WasmContract;\n\n")

	sb.WriteString("export type WasmEngine = {\n")
	sb.WriteString("  [K in WasmFunctionName]: (\n")
	sb.WriteString("    ...args: WasmContract[K][\"args\"]\n")
	sb.WriteString("  ) => Promise<WasmContract[K][\"return\"]>;\n")
	sb.WriteString("};\n\n")

	sb.WriteString("export type WasmResult<T> =\n")
	sb.WriteString("  | { ok: true; value: T }\n")
	sb.WriteString("  | { ok: false; error: string };\n")

	outPath := filepath.Join("..", "front", "src", "wasm", "generated", "wasm-contract.ts")
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
		panic(err)
	}
	fmt.Println("\n📄 front/src/wasm/generated/wasm-contract.ts gerado!")
}

// buildContractEntry converte *types.Signature em entrada do WasmContract.
func buildContractEntry(sig *types.Signature) string {
	argTypes := make([]string, sig.Params().Len())
	for i := range argTypes {
		argTypes[i] = goTypeToTS(sig.Params().At(i).Type())
	}

	ret := "void"
	if sig.Results().Len() == 1 {
		ret = goTypeToTS(sig.Results().At(0).Type())
	} else if sig.Results().Len() > 1 {
		parts := make([]string, sig.Results().Len())
		for i := range parts {
			parts[i] = goTypeToTS(sig.Results().At(i).Type())
		}
		ret = fmt.Sprintf("[%s]", strings.Join(parts, ", "))
	}

	return fmt.Sprintf("{ args: [%s]; return: %s }", strings.Join(argTypes, ", "), ret)
}

// collectUsedStructs encontra todos os tipos struct usados em uma assinatura.
func collectUsedStructs(sig *types.Signature, seen map[string]bool) {
	for i := 0; i < sig.Params().Len(); i++ {
		collectStructTypes(sig.Params().At(i).Type(), seen)
	}
	for i := 0; i < sig.Results().Len(); i++ {
		collectStructTypes(sig.Results().At(i).Type(), seen)
	}
}

func collectStructTypes(t types.Type, seen map[string]bool) {
	switch t := t.(type) {
	case *types.Named:
		if _, ok := t.Underlying().(*types.Struct); ok {
			seen[t.Obj().Name()] = true
		}
		// Tambem verifica argumentos de tipos genericos
		if t.TypeArgs() != nil {
			for i := 0; i < t.TypeArgs().Len(); i++ {
				collectStructTypes(t.TypeArgs().At(i), seen)
			}
		}
	case *types.Pointer:
		collectStructTypes(t.Elem(), seen)
	case *types.Slice:
		collectStructTypes(t.Elem(), seen)
	case *types.Array:
		collectStructTypes(t.Elem(), seen)
	case *types.Map:
		collectStructTypes(t.Key(), seen)
		collectStructTypes(t.Elem(), seen)
	case *types.Struct:
		// Tipo struct anonimo — raro, mas possivel
		for i := 0; i < t.NumFields(); i++ {
			collectStructTypes(t.Field(i).Type(), seen)
		}
	}
}

// buildStructInterface gera interface TypeScript a partir do AST Go.
func buildStructInterface(name string, st *ast.StructType, info *types.Info, fset *token.FileSet) string {
	var fields []string
	for _, field := range st.Fields.List {
		for _, ident := range field.Names {
			fieldName := ident.Name
			if field.Tag != nil {
				tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
				if j := tag.Get("json"); j != "" && j != "-" {
					parts := strings.Split(j, ",")
					fieldName = parts[0]
				}
			}

			// Usa o type-checker para obter o tipo real da expressao
			var tsType string
			if tv, ok := info.Types[field.Type]; ok {
				tsType = goTypeToTS(tv.Type)
			} else {
				tsType = astTypeToTS(field.Type)
			}

			fields = append(fields, fmt.Sprintf("  %s: %s;", fieldName, tsType))
		}
	}

	if len(fields) == 0 {
		return fmt.Sprintf("export interface %s {}", name)
	}
	return fmt.Sprintf("export interface %s {\n%s\n}", name, strings.Join(fields, "\n"))
}

// astTypeToTS e um fallback que mapeia AST expression -> TS string.
func astTypeToTS(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return goBasicToTS(t.Name)
	case *ast.ArrayType:
		return astTypeToTS(t.Elt) + "[]"
	case *ast.StarExpr:
		return astTypeToTS(t.X) + " | null"
	case *ast.MapType:
		return fmt.Sprintf("Record<%s, %s>", astTypeToTS(t.Key), astTypeToTS(t.Value))
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return fmt.Sprintf("%s.%s", ident.Name, t.Sel.Name)
		}
	}
	return "unknown"
}

func goBasicToTS(name string) string {
	switch name {
	case "float32", "float64", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "byte", "rune":
		return "number"
	case "string":
		return "string"
	case "bool":
		return "boolean"
	}
	return name // struct nomeado ou outro tipo
}

// goTypeToTS mapeia tipos Go -> TypeScript recursivamente.
func goTypeToTS(t types.Type) string {
	switch t := t.(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.Float32, types.Float64,
			types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
			types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
			return "number"
		case types.String:
			return "string"
		case types.Bool:
			return "boolean"
		case types.UnsafePointer:
			return "unknown"
		}
	case *types.Slice:
		return goTypeToTS(t.Elem()) + "[]"
	case *types.Array:
		return goTypeToTS(t.Elem()) + "[]"
	case *types.Pointer:
		return goTypeToTS(t.Elem()) + " | null"
	case *types.Map:
		return fmt.Sprintf("Record<%s, %s>", goTypeToTS(t.Key()), goTypeToTS(t.Elem()))
	case *types.Named:
		return t.Obj().Name()
	case *types.Interface:
		return "unknown"
	case *types.Struct:
		return "Record<string, unknown>"
	case *types.Tuple:
		parts := make([]string, t.Len())
		for i := range parts {
			parts[i] = goTypeToTS(t.At(i).Type())
		}
		return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
	}
	return "unknown"
}
