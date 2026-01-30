package main

// Структуры данных для шаблона
type tmplImport struct {
	Name string
	Path string
}

type tmplStruct struct {
	Name string
	Body string
}

type tmplData struct {
	PackageName string
	Imports     []tmplImport
	Structs     []tmplStruct
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
