# Installer Resources

## icon.ico

Generated from `frontend/static/icon-512x512.png` with transparent rounded
corners for Windows app surfaces.

The ICO contains common Windows sizes so Explorer, the taskbar, title bar, and
shortcuts can select an exact bitmap instead of scaling a nearby size:

- 16x16
- 20x20
- 24x24
- 30x30
- 32x32
- 36x36
- 40x40
- 48x48
- 60x60
- 64x64
- 72x72
- 80x80
- 96x96
- 128x128
- 256x256

`cmd/presto-desktop/winres.json` must define the application icon as resource
`#3`. Wails v2 loads the Windows window icon from that resource ID at runtime.
go-winres reads the matching PNG size set from `winres/` to build the exe
resource; Inno Setup uses `icon.ico` for the installer, desktop shortcut, and
start menu shortcut.
