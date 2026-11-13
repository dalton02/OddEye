#!/usr/bin/env bash
# Monta o AppImage a partir do binário em build/bin. O WebKitGTK não vai dentro: o AppImage usa o
# libwebkit2gtk-4.1 do sistema (presente nas distros atuais), como o .deb.
set -euo pipefail
VERSION="${VERSION:?defina VERSION}"
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
APPDIR="$ROOT/build/AppDir"
rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin" "$APPDIR/usr/share/icons/hicolor/512x512/apps"
cp "$ROOT/build/bin/oddeye" "$APPDIR/usr/bin/"
cp "$ROOT/build/linux/oddeye.desktop" "$APPDIR/"
cp "$ROOT/build/appicon.png" "$APPDIR/oddeye.png"
cp "$ROOT/build/appicon.png" "$APPDIR/usr/share/icons/hicolor/512x512/apps/oddeye.png"
cat > "$APPDIR/AppRun" <<'RUN'
#!/bin/sh
HERE="$(dirname "$(readlink -f "$0")")"
exec "$HERE/usr/bin/oddeye" "$@"
RUN
chmod +x "$APPDIR/AppRun"
TOOL="$ROOT/build/appimagetool"
if [ ! -x "$TOOL" ]; then
  curl -fsSL -o "$TOOL" https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
  chmod +x "$TOOL"
fi
ARCH=x86_64 "$TOOL" --appimage-extract-and-run "$APPDIR" "$ROOT/build/bin/Oddeye-${VERSION}-x86_64.AppImage"
