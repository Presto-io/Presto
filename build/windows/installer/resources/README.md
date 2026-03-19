# Installer Resources

## icon.ico

Copied from `frontend/static/favicon.ico`. Contains 16x16 and 32x32 sizes.

TODO: Before release, regenerate icon.ico with additional sizes (48x48, 64x64, 128x128, 256x256)
from the source PNG files in `frontend/static/` (icon-192x192.png, icon-512x512.png) for better
display quality in Windows Explorer, taskbar, and desktop shortcuts.

Use ImageMagick or similar tool:
```bash
magick frontend/static/icon-512x512.png -define icon:auto-resize=256,128,64,48,32,16 build/windows/installer/resources/icon.ico
```
