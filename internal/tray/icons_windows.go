package tray

import _ "embed"

// O Shell_NotifyIcon só aceita ICO.

//go:embed icons/tray.ico
var iconNormal []byte

//go:embed icons/tray-dot.ico
var iconDot []byte
