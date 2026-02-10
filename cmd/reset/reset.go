package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Обработка одного файла
func processFile(filePath string, packages map[string]*packageInfo) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		fmt.Printf("  Ошибка парсинга %s: %v\n", filePath, err)
		return
	}

	pkgDir := filepath.Dir(filePath)

	// Нормализуем путь (убираем ./ если есть)
	pkgDir = strings.TrimPrefix(pkgDir, "./")

	pkgInfo, exists := packages[pkgDir]
	if !exists {
		pkgInfo = &packageInfo{
			name:    node.Name.Name,
			structs: []*structInfo{},
			imports: []importInfo{},
		}
		packages[pkgDir] = pkgInfo
	}

	// Собираем импорты из файла
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		var importName string

		if imp.Name != nil {
			importName = imp.Name.Name
		}

		pkgInfo.imports = append(pkgInfo.imports, importInfo{
			name: importName,
			path: importPath,
		})
	}

	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		hasResetComment := false
		if genDecl.Doc != nil {
			for _, comment := range genDecl.Doc.List {
				if strings.Contains(comment.Text, "generate:reset") {
					hasResetComment = true
					break
				}
			}
		}

		if !hasResetComment {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			structInfo := &structInfo{
				name:   typeSpec.Name.Name,
				fields: []fieldInfo{},
			}

			if structType.Fields != nil {
				for _, field := range structType.Fields.List {
					if len(field.Names) == 0 {
						continue
					}

					fieldName := field.Names[0].Name
					fieldType := exprToString(field.Type)

					fieldTag := ""
					if field.Tag != nil {
						fieldTag = field.Tag.Value
					}

					structInfo.fields = append(structInfo.fields, fieldInfo{
						name: fieldName,
						typ:  fieldType,
						tag:  fieldTag,
					})
				}
			}

			pkgInfo.structs = append(pkgInfo.structs, structInfo)
		}
	}
}

// Преобразование ast.Expr в строку
func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + exprToString(e.X)
	case *ast.ArrayType:
		if e.Len == nil {
			return "[]" + exprToString(e.Elt)
		}
		return "[...]" + exprToString(e.Elt)
	case *ast.MapType:
		return "map[" + exprToString(e.Key) + "]" + exprToString(e.Value)
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	}
	return ""
}

// Генерация файла reset.gen.go для пакета
func generateResetFile(pkgPath string, pkgInfo *packageInfo) error {
	data := tmplData{
		PackageName: pkgInfo.name,
	}

	// Собираем необходимые импорты
	neededImports := collectNeededImports(pkgInfo)
	for _, imp := range neededImports {
		data.Imports = append(data.Imports, tmplImport{
			Name: imp.name,
			Path: imp.path,
		})
	}

	// Генерируем тело методов для каждой структуры
	for _, s := range pkgInfo.structs {
		body := generateStructResetBody(s, pkgInfo.structs)
		data.Structs = append(data.Structs, tmplStruct{
			Name: s.name,
			Body: body,
		})
	}

	// Выполняем шаблон
	tmpl, err := template.New("reset").Parse(resetTemplate)
	if err != nil {
		return fmt.Errorf("ошибка парсинга шаблона: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("ошибка выполнения шаблона: %w", err)
	}

	source := buf.Bytes()

	// Форматируем сгенерированный код
	formatted, err := format.Source(source)
	if err != nil {
		return fmt.Errorf("ошибка форматирования кода: %w\nИсходный код:\n%s", err, source)
	}

	// Записываем файл
	outputPath := filepath.Join(pkgPath, "reset.gen.go")
	if err := os.WriteFile(outputPath, formatted, 0644); err != nil {
		return fmt.Errorf("ошибка записи файла: %w", err)
	}

	return nil
}

// Генерация тела метода Reset() для структуры
func generateStructResetBody(structInfo *structInfo, allStructs []*structInfo) string {
	var lines []string
	for _, field := range structInfo.fields {
		if field.name == "" {
			continue
		}

		code := generateFieldReset(field, allStructs)
		if code != "" {
			// Добавляем отступы ко всем строкам кода
			indented := indentCode(code, "\t")
			lines = append(lines, indented)
		}
	}
	return strings.Join(lines, "\n")
}

// Добавление отступа ко всем строкам кода
func indentCode(code, indent string) string {
	if code == "" {
		return ""
	}
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}

// Собираем необходимые импорты на основе используемых типов
func collectNeededImports(pkgInfo *packageInfo) []importInfo {
	neededImports := make(map[string]importInfo)

	allTypes := make(map[string]bool)
	for _, structInfo := range pkgInfo.structs {
		for _, field := range structInfo.fields {
			allTypes[field.typ] = true
		}
	}

	// Проверяем каждый тип на наличие импортов
	for typ := range allTypes {
		// Ищем селектор (тип с точкой, например time.Time)
		if strings.Contains(typ, ".") && !strings.HasPrefix(typ, ".") {
			parts := strings.Split(typ, ".")
			if len(parts) > 1 {
				// Убираем возможные префиксы (*, [], map[])
				baseType := parts[0]
				baseType = strings.TrimPrefix(baseType, "*")
				baseType = strings.TrimPrefix(baseType, "[]")
				if strings.HasPrefix(baseType, "map[") {
					// Для map[K]V, ищем закрывающую скобку
					if idx := strings.Index(baseType, "]"); idx > 0 {
						baseType = baseType[idx+1:]
					}
				}

				// Ищем этот пакет в импортах исходного файла
				for _, imp := range pkgInfo.imports {
					importPkgName := getPackageName(imp.path)
					if importPkgName == baseType || (imp.name != "" && imp.name == baseType) {
						neededImports[imp.path] = imp
					}
				}
			}
		}
	}

	result := make([]importInfo, 0, len(neededImports))
	for _, imp := range neededImports {
		result = append(result, imp)
	}

	return result
}

// Получаем имя пакета из пути импорта
func getPackageName(importPath string) string {
	parts := strings.Split(importPath, "/")
	return parts[len(parts)-1]
}

// Генерация кода сброса для одного поля
func generateFieldReset(field fieldInfo, allStructs []*structInfo) string {
	fieldAccess := "x." + field.name

	typ := field.typ

	if isBuiltinType(typ) {
		return generateBuiltinReset(fieldAccess, typ)
	}

	if shouldCallReset(typ, allStructs) {
		if strings.HasPrefix(typ, "*") {
			baseType := typ[1:]
			if isStructType(baseType, allStructs) {
				return fmt.Sprintf("if %s != nil {\n\t%s.Reset()\n}", fieldAccess, fieldAccess)
			}
		} else {
			return fmt.Sprintf("%s.Reset()", fieldAccess)
		}
	}

	switch {
	// Слайсы
	case strings.HasPrefix(typ, "[]"):
		return fmt.Sprintf("%s = %s[:0]", fieldAccess, fieldAccess)

	// Мапы
	case strings.HasPrefix(typ, "map"):
		return fmt.Sprintf("clear(%s)", fieldAccess)

	// Указатели на примитивы
	case strings.HasPrefix(typ, "*"):
		baseType := typ[1:]
		if isPrimitiveType(baseType) {
			return fmt.Sprintf("if %s != nil {\n\t*%s = %s\n}",
				fieldAccess, fieldAccess, getZeroValue(baseType))
		}
		// Для указателей на слайсы
		if strings.HasPrefix(baseType, "[]") {
			return fmt.Sprintf("if %s != nil {\n\t*%s = (*%s)[:0]\n}",
				fieldAccess, fieldAccess, fieldAccess)
		}
		// Для указателей на мапы
		if strings.HasPrefix(baseType, "map") {
			return fmt.Sprintf("if %s != nil {\n\tclear(*%s)\n}",
				fieldAccess, fieldAccess)
		}

	// Примитивные типы
	default:
		if isPrimitiveType(typ) {
			return fmt.Sprintf("%s = %s", fieldAccess, getZeroValue(typ))
		}
	}

	return ""
}

// Генерация сброса для встроенных типов
func generateBuiltinReset(fieldAccess, typ string) string {
	switch {
	case typ == "time.Time":
		return fmt.Sprintf("%s = time.Time{}", fieldAccess)
	case strings.HasPrefix(typ, "*") && strings.Contains(typ, "time.Time"):
		return fmt.Sprintf("if %s != nil {\n\t*%s = time.Time{}\n}", fieldAccess, fieldAccess)
	default:
		return ""
	}
}

func isBuiltinType(typ string) bool {
	if strings.Contains(typ, "time.Time") {
		return true
	}

	// Сюда можно добавить другие встроенные типы при необходимости
	return false
}

// Проверяем, нужно ли вызывать Reset() для этого типа
func shouldCallReset(typ string, allStructs []*structInfo) bool {
	baseTyp := strings.TrimPrefix(typ, "*")

	for _, s := range allStructs {
		if s.name == baseTyp {
			return true
		}
	}

	return false
}

// Проверяем, является ли тип структурой в текущем пакете
func isStructType(typ string, allStructs []*structInfo) bool {
	for _, s := range allStructs {
		if s.name == typ {
			return true
		}
	}
	return false
}

// Проверка, является ли тип примитивным
func isPrimitiveType(typ string) bool {
	primitiveTypes := []string{
		"bool", "string",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "complex64", "complex128",
		"byte", "rune",
	}

	for _, p := range primitiveTypes {
		if typ == p {
			return true
		}
	}

	return false
}

// Получение нулевого значения для примитивного типа
func getZeroValue(typ string) string {
	switch typ {
	case "bool":
		return "false"
	case "string":
		return `""`
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "complex64", "complex128",
		"byte", "rune":
		return "0"
	default:
		return "nil"
	}
}
