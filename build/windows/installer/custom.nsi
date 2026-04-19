; Presto Custom NSIS Script
; Extends Wails-generated installer with shortcuts, registry, and duplicate detection
;
; This script is included by the Wails NSIS build process via !include.
; It adds: installation options page, shortcut creation, registry entries,
; duplicate installation detection, and post-install launch option.

!include MUI2.nsh
!include nsDialogs.nsh
!include LogicLib.nsh

; ========================================
; Variables
; ========================================
Var CreateDesktopShortcut
Var CreateStartMenuShortcut
Var DownloadTemplates
Var LaunchAfterInstall

; ========================================
; Installation Options Page
; ========================================
Page custom OptionsPage OptionsLeave

Function OptionsPage
  !insertmacro MUI_HEADER_TEXT "Installation Options" "Choose additional installation options"

  nsDialogs::Create 1018
  Pop $0

  ${If} $0 == error
    Abort
  ${EndIf}

  ; Desktop Shortcut (default: checked)
  ${NSD_CreateCheckbox} 0 0 100% 12u "Create desktop shortcut"
  Pop $CreateDesktopShortcut
  ${NSD_SetState} $CreateDesktopShortcut ${BST_CHECKED}

  ; Start Menu Shortcut (default: checked)
  ${NSD_CreateCheckbox} 0 20u 100% 12u "Create start menu shortcut"
  Pop $CreateStartMenuShortcut
  ${NSD_SetState} $CreateStartMenuShortcut ${BST_CHECKED}

  ; Download Templates (default: checked)
  ${NSD_CreateCheckbox} 0 40u 100% 12u "Download official templates (recommended)"
  Pop $DownloadTemplates
  ${NSD_SetState} $DownloadTemplates ${BST_CHECKED}

  ; Launch after install (default: checked)
  ${NSD_CreateCheckbox} 0 60u 100% 12u "Launch Presto after installation"
  Pop $LaunchAfterInstall
  ${NSD_SetState} $LaunchAfterInstall ${BST_CHECKED}

  nsDialogs::Show
FunctionEnd

Function OptionsLeave
  ; Save checkbox states
  ${NSD_GetState} $CreateDesktopShortcut $CreateDesktopShortcut
  ${NSD_GetState} $CreateStartMenuShortcut $CreateStartMenuShortcut
  ${NSD_GetState} $DownloadTemplates $DownloadTemplates
  ${NSD_GetState} $LaunchAfterInstall $LaunchAfterInstall
FunctionEnd

; ========================================
; Shortcut Creation Section
; ========================================
Section "Shortcuts" SEC_SHORTCUTS
  ; Create desktop shortcut if selected
  ${If} $CreateDesktopShortcut == ${BST_CHECKED}
    CreateShortCut "$DESKTOP\Presto.lnk" "$INSTDIR\Presto.exe" "" "$INSTDIR\Presto.exe" 0
  ${EndIf}

  ; Create start menu shortcuts if selected
  ${If} $CreateStartMenuShortcut == ${BST_CHECKED}
    CreateDirectory "$SMPROGRAMS\Presto"
    CreateShortCut "$SMPROGRAMS\Presto\Presto.lnk" "$INSTDIR\Presto.exe" "" "$INSTDIR\Presto.exe" 0
    CreateShortCut "$SMPROGRAMS\Presto\Uninstall Presto.lnk" "$INSTDIR\uninstall.exe" "" "$INSTDIR\uninstall.exe" 0
  ${EndIf}
SectionEnd

; ========================================
; Template Download Section
; ========================================
Section "Download Templates" SEC_TEMPLATES
  ${If} $DownloadTemplates == ${BST_CHECKED}
    ; Show progress banner
    Banner::show /set 76 "Downloading Templates" "Please wait while templates are downloaded..."

    ; Execute Presto.exe --download-templates and capture return code
    ExecWait '"$INSTDIR\Presto.exe" --download-templates' $0

    ; Hide banner
    Banner::destroy

    ; Check return code (0 = success, non-zero = failure)
    ${If} $0 != 0
      ; Download failed - ask user whether to continue
      MessageBox MB_YESNO|MB_ICONWARNING \
        "Template download failed. You can download templates later from within the application.$\n$\nContinue installation?" \
        IDYES continue_install
      Abort

      continue_install:
    ${EndIf}
  ${EndIf}
SectionEnd

; ========================================
; Registry Configuration Section
; ========================================
Section "Registry" SEC_REGISTRY
  ; Write Add/Remove Programs registry entries
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Presto" \
                   "DisplayName" "Presto"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Presto" \
                   "DisplayVersion" "1.0.2"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Presto" \
                   "Publisher" "Presto Team"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Presto" \
                   "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Presto" \
                   "InstallLocation" '"$INSTDIR"'
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Presto" \
                   "DisplayIcon" '"$INSTDIR\Presto.exe,0"'
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Presto" \
                     "NoModify" 1
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Presto" \
                     "NoRepair" 1

  ; Register Presto as an available editor for Markdown files
  WriteRegStr HKLM "Software\Classes\Presto.Markdown" "" "Presto Markdown Document"
  WriteRegStr HKLM "Software\Classes\Presto.Markdown" "FriendlyTypeName" "Markdown Document"
  WriteRegStr HKLM "Software\Classes\Presto.Markdown\DefaultIcon" "" "$INSTDIR\Presto.exe,0"
  WriteRegStr HKLM "Software\Classes\Presto.Markdown\shell" "" "open"
  WriteRegStr HKLM "Software\Classes\Presto.Markdown\shell\open" "" "Open with Presto"
  WriteRegStr HKLM "Software\Classes\Presto.Markdown\shell\open\command" "" "$\"$INSTDIR\Presto.exe$\" $\"%1$\""

  WriteRegStr HKLM "Software\Classes\Applications\Presto.exe" "FriendlyAppName" "Presto"
  WriteRegStr HKLM "Software\Classes\Applications\Presto.exe\DefaultIcon" "" "$INSTDIR\Presto.exe,0"
  WriteRegStr HKLM "Software\Classes\Applications\Presto.exe\shell\open" "" "Open with Presto"
  WriteRegStr HKLM "Software\Classes\Applications\Presto.exe\shell\open\command" "" "$\"$INSTDIR\Presto.exe$\" $\"%1$\""
  WriteRegStr HKLM "Software\Classes\Applications\Presto.exe\SupportedTypes" ".md" ""
  WriteRegStr HKLM "Software\Classes\Applications\Presto.exe\SupportedTypes" ".markdown" ""

  WriteRegStr HKLM "Software\Classes\.md\OpenWithProgids" "Presto.Markdown" ""
  WriteRegStr HKLM "Software\Classes\.markdown\OpenWithProgids" "Presto.Markdown" ""

  WriteRegStr HKLM "Software\RegisteredApplications" "Presto" "Software\Clients\Presto\Capabilities"
  WriteRegStr HKLM "Software\Clients\Presto\Capabilities" "ApplicationName" "Presto"
  WriteRegStr HKLM "Software\Clients\Presto\Capabilities" "ApplicationDescription" "Markdown to Typst to PDF editor"
  WriteRegStr HKLM "Software\Clients\Presto\Capabilities\FileAssociations" ".md" "Presto.Markdown"
  WriteRegStr HKLM "Software\Clients\Presto\Capabilities\FileAssociations" ".markdown" "Presto.Markdown"

  System::Call 'shell32::SHChangeNotify(i 0x08000000, i 0, p 0, p 0)'
SectionEnd

; ========================================
; Post-Install: Launch Application
; ========================================
Function .onInstSuccess
  ${If} $LaunchAfterInstall == ${BST_CHECKED}
    Exec '"$INSTDIR\Presto.exe"'
  ${EndIf}
FunctionEnd

; ========================================
; Pre-Install: Duplicate Installation Check
; ========================================
Function .onInit
  ; Check if Presto is already installed via registry
  ReadRegStr $0 HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Presto" "UninstallString"
  ${If} $0 != ""
    MessageBox MB_YESNO|MB_ICONQUESTION \
      "Presto is already installed.$\n$\nDo you want to uninstall the existing version first?" \
      IDYES uninstall_prev
    Abort

    uninstall_prev:
      ; Run existing uninstaller silently, wait for completion
      ExecWait '"$0" /S _?=$INSTDIR'
  ${EndIf}
FunctionEnd

; ========================================
; Uninstall Section
; ========================================
Section "Uninstall"
  ; Ask for confirmation
  MessageBox MB_YESNO "Are you sure you want to uninstall Presto?" IDYES confirm
  Abort

  confirm:

  ; Ask whether to keep user data
  MessageBox MB_YESNO "Keep user data directory?$\n$\nIf you select Yes, your documents, templates, and settings will be preserved.$\n$\nLocation: $PROFILE\.presto" IDYES keep_data

  ; Delete user data
  RMDir /r "$PROFILE\.presto"

  keep_data:

  ; Delete installation files
  Delete "$INSTDIR\Presto.exe"
  Delete "$INSTDIR\typst.exe"
  Delete "$INSTDIR\uninstall.exe"
  Delete "$INSTDIR\*.dll"
  Delete "$INSTDIR\resources\*.*"
  RMDir "$INSTDIR\resources"
  RMDir "$INSTDIR"

  ; Delete desktop shortcut
  Delete "$DESKTOP\Presto.lnk"

  ; Delete start menu shortcuts
  Delete "$SMPROGRAMS\Presto\Presto.lnk"
  Delete "$SMPROGRAMS\Presto\Uninstall Presto.lnk"
  RMDir "$SMPROGRAMS\Presto"

  ; Delete registry key
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Presto"
  DeleteRegValue HKLM "Software\Classes\.md\OpenWithProgids" "Presto.Markdown"
  DeleteRegValue HKLM "Software\Classes\.markdown\OpenWithProgids" "Presto.Markdown"
  DeleteRegKey HKLM "Software\Classes\Applications\Presto.exe"
  DeleteRegKey HKLM "Software\Classes\Presto.Markdown"
  DeleteRegValue HKLM "Software\RegisteredApplications" "Presto"
  DeleteRegKey HKLM "Software\Clients\Presto"
  System::Call 'shell32::SHChangeNotify(i 0x08000000, i 0, p 0, p 0)'

  ; Display completion message
  MessageBox MB_OK "Presto has been successfully uninstalled."
SectionEnd
