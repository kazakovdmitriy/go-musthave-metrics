package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	fmt.Println("=== Запуск генератора Reset() методов ===")

	rootDir := "."
	if len(os.Args) > 1 {
		rootDir = os.Args[1]
	}

	absPath, _ := filepath.Abs(rootDir)
	fmt.Printf("Сканируем директорию: %s\n", absPath)

	packages := make(map[string]*packageInfo)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			name := info.Name()
			// Пропускаем только скрытые директории (кроме . и ..) и vendor
			if (name != "." && name != ".." && strings.HasPrefix(name, ".")) || name == "vendor" {
				fmt.Printf("Пропускаем директорию: %s\n", path)
				return filepath.SkipDir
			}
			return nil
		}

		// Обрабатываем только .go файлы (не сгенерированные)
		if strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), ".gen.go") {
			fmt.Printf("Анализируем файл: %s\n", path)
			processFile(path, packages)
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Ошибка при сканировании: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== Результаты сканирования ===\n")
	fmt.Printf("Найдено пакетов: %d\n", len(packages))

	// Генерируем файлы
	generatedCount := 0
	for pkgPath, pkgInfo := range packages {
		if len(pkgInfo.structs) > 0 {
			fmt.Printf("\nПакет: %s\n", pkgPath)
			for _, s := range pkgInfo.structs {
				fmt.Printf("  - %s\n", s.name)
			}

			err := generateResetFile(pkgPath, pkgInfo)
			if err != nil {
				fmt.Printf("✗ Ошибка генерации: %v\n", err)
			} else {
				fmt.Printf("✓ Файл reset.gen.go создан\n")
				generatedCount++
			}
		} else {
			fmt.Printf("\nПакет: %s (без структур для генерации)\n", pkgPath)
		}
	}

	fmt.Printf("\n=== Генерация завершена ===\n")
	fmt.Printf("Сгенерировано файлов: %d\n", generatedCount)
}

// Структура для хранения информации о пакете
type packageInfo struct {
	name    string
	structs []*structInfo
	imports []importInfo
}

// Информация об импорте
type importInfo struct {
	name string
	path string
}

// Структура для хранения информации о структуре
type structInfo struct {
	name   string
	fields []fieldInfo
}

// Информация о поле структуры
type fieldInfo struct {
	name string
	typ  string
	tag  string
}

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
	if strings.HasPrefix(pkgDir, "./") {
		pkgDir = pkgDir[2:]
	}

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
	outputPath := filepath.Join(pkgPath, "reset.gen.go")

	var code strings.Builder

	// Заголовок файла
	code.WriteString("// Code generated by reset generator. DO NOT EDIT.\n")
	code.WriteString("//go:generate go run ./cmd/reset/main.go\n\n")
	code.WriteString("package " + pkgInfo.name + "\n\n")

	// Собираем необходимые импорты на основе используемых типов
	neededImports := collectNeededImports(pkgInfo)

	// Добавляем импорты, если они нужны
	if len(neededImports) > 0 {
		code.WriteString("import (\n")
		for _, imp := range neededImports {
			if imp.name != "" {
				code.WriteString(fmt.Sprintf("\t%s \"%s\"\n", imp.name, imp.path))
			} else {
				code.WriteString(fmt.Sprintf("\t\"%s\"\n", imp.path))
			}
		}
		code.WriteString(")\n\n")
	}

	// Генерируем методы Reset() для каждой структуры
	for _, structInfo := range pkgInfo.structs {
		code.WriteString(fmt.Sprintf("func (x *%s) Reset() {\n", structInfo.name))

		for _, field := range structInfo.fields {
			if field.name == "" {
				continue
			}

			resetCode := generateFieldReset(field, pkgInfo.structs)
			if resetCode != "" {
				// Добавляем отступы для многострочных выражений
				lines := strings.Split(resetCode, "\n")
				for i, line := range lines {
					if i == 0 {
						code.WriteString("\t" + line + "\n")
					} else {
						code.WriteString("\t" + line + "\n")
					}
				}
			}
		}

		code.WriteString("}\n\n")
	}

	formatted, err := format.Source([]byte(code.String()))
	if err != nil {
		return fmt.Errorf("ошибка форматирования кода: %v\nИсходный код:\n%s", err, code.String())
	}

	err = os.WriteFile(outputPath, formatted, 0644)
	if err != nil {
		return fmt.Errorf("ошибка записи файла: %v", err)
	}

	return nil
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
				if strings.HasPrefix(baseType, "*") {
					baseType = baseType[1:]
				}
				if strings.HasPrefix(baseType, "[]") {
					baseType = baseType[2:]
				}
				if strings.HasPrefix(baseType, "map[") {
					// Для map[K]V, ищем закрывающую скобку
					idx := strings.Index(baseType, "]")
					if idx > 0 {
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
