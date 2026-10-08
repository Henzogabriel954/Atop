package main

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed celular.png
var celularIconBytes []byte

// Recurso estático embutido no binário
var celularIconResource fyne.Resource = fyne.NewStaticResource("celular.png", celularIconBytes)
