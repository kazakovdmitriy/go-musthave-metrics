package main

import (
	"fmt"
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
