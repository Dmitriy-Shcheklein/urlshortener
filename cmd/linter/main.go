package main

import (
	"github.com/Dmitriy-Shcheklein/urlshortener/analyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() { singlechecker.Main(analyzer.Analyzer) }
