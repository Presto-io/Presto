; Presto Windows installer
; Standalone NSIS script used by CI and local release builds.

Unicode true

!include MUI2.nsh
!include nsDialogs.nsh
!include LogicLib.nsh
!include x64.nsh
!include WinVer.nsh
!include FileFunc.nsh

!ifndef ARG_VERSION
  !error "ARG_VERSION is required"
!endif

!ifndef ARG_FILE_VERSION
  !define ARG_FILE_VERSION "0.0.0.0"
!endif

!ifndef ARG_ARCH
  !error "ARG_ARCH is required"
!endif

!ifndef ARG_BINARY
  !error "ARG_BINARY is required"
!endif

!ifndef ARG_TYPST_BINARY
  !error "ARG_TYPST_BINARY is required"
!endif

!ifndef ARG_OUTPUT_NAME
  !define ARG_OUTPUT_NAME "Presto-${ARG_VERSION}-windows-${ARG_ARCH}-installer.exe"
!endif

!define INFO_COMPANYNAME "Presto Team"
!define INFO_PRODUCTNAME "Presto"
!define INFO_PRODUCTVERSION "${ARG_VERSION}"
!define INFO_COPYRIGHT "Copyright (c) 2026 Presto Team"
!define INFO_FILEVERSION "${ARG_FILE_VERSION}"
!define PRODUCT_EXECUTABLE "Presto.exe"
!define TYPST_EXECUTABLE "typst.exe"
!define UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\Presto"

Name "${INFO_PRODUCTNAME}"
OutFile "${ARG_OUTPUT_NAME}"
InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
RequestExecutionLevel admin
ShowInstDetails show
ShowUnInstDetails show

VIProductVersion "${INFO_FILEVERSION}"
VIFileVersion "${INFO_FILEVERSION}"
VIAddVersionKey "CompanyName" "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion" "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion" "${INFO_FILEVERSION}"
VIAddVersionKey "LegalCopyright" "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName" "${INFO_PRODUCTNAME}"

ManifestDPIAware true

!define MUI_ICON "resources\icon.ico"
!define MUI_UNICON "resources\icon.ico"
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_ABORTWARNING

Var CreateDesktopShortcut
Var CreateStartMenuShortcut
Var DownloadTemplates
Var LaunchAfterInstall
Var KeepUserData

!insertmacro MUI_PAGE_WELCOME
Page custom OptionsPage OptionsLeave
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Function .onInit
  ${IfNot} ${AtLeastWin10}
    MessageBox MB_OK|MB_ICONSTOP "Presto only supports Windows 10 (Server 2016) and later."
    Quit
  ${EndIf}

  !if "${ARG_ARCH}" == "arm64"
    ${IfNot} ${IsNativeARM64}
      MessageBox MB_OK|MB_ICONSTOP "This installer only supports native Windows ARM64 devices."
      Quit
    ${EndIf}
  !else
    ${IfNot} ${IsNativeAMD64}
      MessageBox MB_OK|MB_ICONSTOP "This installer only supports native Windows x64 devices."
      Quit
    ${EndIf}
  !endif

  ReadRegStr $0 HKLM "${UNINST_KEY}" "UninstallString"
  ${If} $0 != ""
    Goto existing_found
  ${EndIf}
  Goto init_done

  existing_found:
  MessageBox MB_YESNO|MB_ICONQUESTION "Presto is already installed.$\n$\nDo you want to uninstall the existing version first?" IDYES do_uninstall
  Abort

  do_uninstall:
  ExecWait '"$0" /S _?=$INSTDIR'

  init_done:
FunctionEnd

Function OptionsPage
  !insertmacro MUI_HEADER_TEXT "Installation Options" "Choose additional installation options"

  nsDialogs::Create 1018
  Pop $0

  ${If} $0 == error
    Abort
  ${EndIf}

  ${NSD_CreateCheckbox} 0 0 100% 12u "Create desktop shortcut"
  Pop $CreateDesktopShortcut
  ${NSD_SetState} $CreateDesktopShortcut ${BST_CHECKED}

  ${NSD_CreateCheckbox} 0 20u 100% 12u "Create start menu shortcut"
  Pop $CreateStartMenuShortcut
  ${NSD_SetState} $CreateStartMenuShortcut ${BST_CHECKED}

  ${NSD_CreateCheckbox} 0 40u 100% 12u "Download official templates (recommended)"
  Pop $DownloadTemplates
  ${NSD_SetState} $DownloadTemplates ${BST_CHECKED}

  ${NSD_CreateCheckbox} 0 60u 100% 12u "Launch Presto after installation"
  Pop $LaunchAfterInstall
  ${NSD_SetState} $LaunchAfterInstall ${BST_CHECKED}

  nsDialogs::Show
FunctionEnd

Function OptionsLeave
  ${NSD_GetState} $CreateDesktopShortcut $CreateDesktopShortcut
  ${NSD_GetState} $CreateStartMenuShortcut $CreateStartMenuShortcut
  ${NSD_GetState} $DownloadTemplates $DownloadTemplates
  ${NSD_GetState} $LaunchAfterInstall $LaunchAfterInstall
FunctionEnd

Section "Main Files" SEC_MAIN
  SetOutPath $INSTDIR
  File "/oname=${PRODUCT_EXECUTABLE}" "${ARG_BINARY}"
  File "/oname=${TYPST_EXECUTABLE}" "${ARG_TYPST_BINARY}"
  WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

Section "Shortcuts" SEC_SHORTCUTS
  ${If} $CreateDesktopShortcut == ${BST_CHECKED}
    CreateShortCut "$DESKTOP\Presto.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}" "" "$INSTDIR\${PRODUCT_EXECUTABLE}" 0
  ${EndIf}

  ${If} $CreateStartMenuShortcut == ${BST_CHECKED}
    CreateDirectory "$SMPROGRAMS\Presto"
    CreateShortCut "$SMPROGRAMS\Presto\Presto.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}" "" "$INSTDIR\${PRODUCT_EXECUTABLE}" 0
    CreateShortCut "$SMPROGRAMS\Presto\Uninstall Presto.lnk" "$INSTDIR\uninstall.exe" "" "$INSTDIR\uninstall.exe" 0
  ${EndIf}
SectionEnd

Section "Download Templates" SEC_TEMPLATES
  ${If} $DownloadTemplates == ${BST_CHECKED}
    Banner::show /set 76 "Downloading Templates" "Please wait while templates are downloaded..."
    ExecWait '"$INSTDIR\${PRODUCT_EXECUTABLE}" --download-templates' $0
    Banner::destroy

    ${If} $0 != 0
      Goto template_download_failed
    ${EndIf}
  ${EndIf}
  Goto template_done

  template_download_failed:
  MessageBox MB_YESNO|MB_ICONWARNING "Template download failed. You can download templates later from within the application.$\n$\nContinue installation?" IDYES skip_abort_templates
  Abort

  skip_abort_templates:
  template_done:
SectionEnd

Section "Registry" SEC_REGISTRY
  SetRegView 64

  WriteRegStr HKLM "${UNINST_KEY}" "Publisher" "${INFO_COMPANYNAME}"
  WriteRegStr HKLM "${UNINST_KEY}" "DisplayName" "${INFO_PRODUCTNAME}"
  WriteRegStr HKLM "${UNINST_KEY}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
  WriteRegStr HKLM "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\${PRODUCT_EXECUTABLE},0"
  WriteRegStr HKLM "${UNINST_KEY}" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegStr HKLM "${UNINST_KEY}" "QuietUninstallString" '"$INSTDIR\uninstall.exe" /S'
  WriteRegStr HKLM "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegDWORD HKLM "${UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKLM "${UNINST_KEY}" "NoRepair" 1

  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD HKLM "${UNINST_KEY}" "EstimatedSize" "$0"

  WriteRegStr HKLM "Software\Classes\Presto.Markdown" "" "Presto Markdown Document"
  WriteRegStr HKLM "Software\Classes\Presto.Markdown" "FriendlyTypeName" "Markdown Document"
  WriteRegStr HKLM "Software\Classes\Presto.Markdown\DefaultIcon" "" "$INSTDIR\${PRODUCT_EXECUTABLE},0"
  WriteRegStr HKLM "Software\Classes\Presto.Markdown\shell" "" "open"
  WriteRegStr HKLM "Software\Classes\Presto.Markdown\shell\open" "" "Open with Presto"
  WriteRegStr HKLM "Software\Classes\Presto.Markdown\shell\open\command" "" "$\"$INSTDIR\${PRODUCT_EXECUTABLE}$\" $\"%1$\""

  WriteRegStr HKLM "Software\Classes\Applications\Presto.exe" "FriendlyAppName" "Presto"
  WriteRegStr HKLM "Software\Classes\Applications\Presto.exe\DefaultIcon" "" "$INSTDIR\${PRODUCT_EXECUTABLE},0"
  WriteRegStr HKLM "Software\Classes\Applications\Presto.exe\shell\open" "" "Open with Presto"
  WriteRegStr HKLM "Software\Classes\Applications\Presto.exe\shell\open\command" "" "$\"$INSTDIR\${PRODUCT_EXECUTABLE}$\" $\"%1$\""
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

Function .onInstSuccess
  ${If} $LaunchAfterInstall == ${BST_CHECKED}
    Exec '"$INSTDIR\${PRODUCT_EXECUTABLE}"'
  ${EndIf}
FunctionEnd

Section "Uninstall"
  SetRegView 64

  StrCpy $KeepUserData 0
  IfSilent silent_uninstall prompt_confirm

  prompt_confirm:
    MessageBox MB_YESNO "Are you sure you want to uninstall Presto?" IDYES ask_keep_data
    Abort

  ask_keep_data:
    MessageBox MB_YESNO "Keep user data directory?$\n$\nIf you select Yes, your documents, templates, and settings will be preserved.$\n$\nLocation: $PROFILE\.presto" IDYES keep_data
    Goto continue_uninstall

  keep_data:
    StrCpy $KeepUserData 1
    Goto continue_uninstall

  silent_uninstall:
    StrCpy $KeepUserData 1

  continue_uninstall:
    ${If} $KeepUserData != 1
      RMDir /r "$PROFILE\.presto"
    ${EndIf}

    Delete "$INSTDIR\${PRODUCT_EXECUTABLE}"
    Delete "$INSTDIR\${TYPST_EXECUTABLE}"
    Delete "$INSTDIR\uninstall.exe"
    Delete "$INSTDIR\*.dll"
    Delete "$INSTDIR\resources\*.*"
    RMDir "$INSTDIR\resources"
    RMDir "$INSTDIR"

    Delete "$DESKTOP\Presto.lnk"
    Delete "$SMPROGRAMS\Presto\Presto.lnk"
    Delete "$SMPROGRAMS\Presto\Uninstall Presto.lnk"
    RMDir "$SMPROGRAMS\Presto"

    DeleteRegKey HKLM "${UNINST_KEY}"
    DeleteRegValue HKLM "Software\Classes\.md\OpenWithProgids" "Presto.Markdown"
    DeleteRegValue HKLM "Software\Classes\.markdown\OpenWithProgids" "Presto.Markdown"
    DeleteRegKey HKLM "Software\Classes\Applications\Presto.exe"
    DeleteRegKey HKLM "Software\Classes\Presto.Markdown"
    DeleteRegValue HKLM "Software\RegisteredApplications" "Presto"
    DeleteRegKey HKLM "Software\Clients\Presto"
    System::Call 'shell32::SHChangeNotify(i 0x08000000, i 0, p 0, p 0)'

    IfSilent done
    MessageBox MB_OK "Presto has been successfully uninstalled."

  done:
SectionEnd
