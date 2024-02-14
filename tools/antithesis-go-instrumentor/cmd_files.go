package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

type CommandFiles struct {
	OutputDirectory     string
	InputDirectory      string
	CustomerDirectory   string
	SymbolsDirectory    string
	CatalogPath         string
	exclusions          map[string]bool
	exclude_file        string
	wants_instrumentor  bool
	module_name         string
	symtable_prefix     string
	symbolTableFilename string
}

func (cfx *CommandFiles) ParseExclusionsFile() (err error) {
	if cfx.exclude_file == "" {
		return
	}
	exclusions := map[string]bool{}
	err, exclusions = ParseExclusionsFile(cfx.exclude_file, cfx.InputDirectory)
	if err != nil {
		cfx.exclusions = exclusions
	}
	return
}

// FindSourceCode scans an input directory recursively for .go files,
// skipping any files or directories specified in exclusions.
func (cfx *CommandFiles) FindSourceCode() (err error, paths []string) {
	paths = []string{}
	logger.Printf("Scanning %s recursively for .go source", cfx.InputDirectory)
	// Files are read in lexical order, i.e. we can later deterministically
	// hash their content: https://pkg.go.dev/path/filepath#WalkDir
	err = filepath.WalkDir(cfx.InputDirectory,
		func(path string, info fs.DirEntry, erx error) error {
			if erx != nil {
				logger.Printf("Error %v in directory %s; skipping", erx, path)
				return erx
			}

			if b := filepath.Base(path); strings.HasPrefix(b, ".") {
				if verbose_level(2) {
					logger.Printf("Skipping %s", path)
				}
				if info.IsDir() {
					return fs.SkipDir
				}
				return nil
			}

			if cfx.exclusions[path] {
				if info.IsDir() {
					logger.Printf("Skipping excluded directory %s and its children", path)
					return fs.SkipDir
				}
				logger.Printf("Skipping excluded file %s", path)
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			// This is the mandatory format of unit test file names.
			if strings.HasSuffix(path, "_test.go") {
				if verbose_level(2) {
					logger.Printf("Skipping test file %s", path)
				}
				return nil
			} else if strings.HasSuffix(path, ".pb.go") {
				if verbose_level(1) {
					logger.Printf("Skipping generated file %s", path)
				}
				return nil
			}

			paths = append(paths, path)

			return nil
		})

	if err != nil {
		err = fmt.Errorf("Error walking input directory %s: %v", cfx.InputDirectory, err)
	}
	return
}

func (cfx *CommandFiles) UsingSymbols() string {
	usingSymbols := ""
	if cfx.wants_instrumentor {
		usingSymbols = cfx.symbolTableFilename
	}
	return usingSymbols
}

func (cfx *CommandFiles) CreateSymbolTableWriter(files_hash string) (symWriter *SymbolTable) {
	symWriter = nil

	if cfx.wants_instrumentor {
		symbolTableFileBasename := fmt.Sprintf("%sgo-%s", cfx.symtable_prefix, files_hash)
		cfx.symbolTableFilename = symbolTableFileBasename + ".sym.tsv"
		symbolsPath := filepath.Join(cfx.SymbolsDirectory, cfx.symbolTableFilename)
		symWriter = CreateSymbolTableFile(symbolsPath, symbolTableFileBasename)
	}
	return
}
