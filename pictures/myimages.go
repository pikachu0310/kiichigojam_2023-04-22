package myimages

import (
	_ "embed"
)

var (
	//go:embed slime.png
	SlimePng []byte
	//go:embed weapon.png
	WeaponPng []byte
	//go:embed armor.png
	ArmorPng []byte
	//go:embed item.png
	ItemPng []byte

	//go:embed hoshizora.png
	HoshizoraPng []byte
)
