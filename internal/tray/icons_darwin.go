package tray

import _ "embed"

// Template image: o macOS pinta conforme o tema da barra de menus, então o ícone é só a silhueta.

//go:embed icons/tray-template.png
var iconNormal []byte

//go:embed icons/tray-template-dot.png
var iconDot []byte
