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

!define INFO_COMPANYNAME "Presto-io"
!define INFO_PRODUCTNAME "Presto"
!define INFO_PRODUCTVERSION "${ARG_VERSION}"
!define INFO_COPYRIGHT "Copyright (c) 2026 Presto-io"
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
!insertmacro MUI_LANGUAGE "SimpChinese"

Function .onInit
  ${IfNot} ${AtLeastWin10}
    MessageBox MB_OK|MB_ICONSTOP "Presto 仅支持 Windows 10（Server 2016）及以上版本。"
    Quit
  ${EndIf}

  !if "${ARG_ARCH}" == "arm64"
    ${IfNot} ${IsNativeARM64}
      MessageBox MB_OK|MB_ICONSTOP "此安装程序仅支持原生 Windows ARM64 设备。"
      Quit
    ${EndIf}
  !else
    ${IfNot} ${IsNativeAMD64}
      MessageBox MB_OK|MB_ICONSTOP "此安装程序仅支持原生 Windows x64 设备。"
      Quit
    ${EndIf}
  !endif

  ReadRegStr $0 HKLM "${UNINST_KEY}" "UninstallString"
  ${If} $0 != ""
    ExecWait '"$0" /S _?=$INSTDIR'
  ${EndIf}
FunctionEnd

Function OptionsPage
  !insertmacro MUI_HEADER_TEXT "安装选项" "选择附加安装选项"

  nsDialogs::Create 1018
  Pop $0

  ${If} $0 == error
    Abort
  ${EndIf}

  ${NSD_CreateCheckbox} 0 0 100% 12u "创建桌面快捷方式"
  Pop $CreateDesktopShortcut
  ${NSD_SetState} $CreateDesktopShortcut ${BST_CHECKED}

  ${NSD_CreateCheckbox} 0 20u 100% 12u "创建开始菜单快捷方式"
  Pop $CreateStartMenuShortcut
  ${NSD_SetState} $CreateStartMenuShortcut ${BST_CHECKED}

  ${NSD_CreateCheckbox} 0 40u 100% 12u "下载官方模板（推荐）"
  Pop $DownloadTemplates
  ${NSD_SetState} $DownloadTemplates ${BST_CHECKED}

  ${NSD_CreateCheckbox} 0 60u 100% 12u "安装完成后启动 Presto"
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

Section "主程序文件" SEC_MAIN
  SetOutPath $INSTDIR
  File "/oname=${PRODUCT_EXECUTABLE}" "${ARG_BINARY}"
  File "/oname=${TYPST_EXECUTABLE}" "${ARG_TYPST_BINARY}"
  WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

Section "快捷方式" SEC_SHORTCUTS
  ${If} $CreateDesktopShortcut == ${BST_CHECKED}
    CreateShortCut "$DESKTOP\Presto.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}" "" "$INSTDIR\${PRODUCT_EXECUTABLE}" 0
  ${EndIf}

  ${If} $CreateStartMenuShortcut == ${BST_CHECKED}
    CreateDirectory "$SMPROGRAMS\Presto"
    CreateShortCut "$SMPROGRAMS\Presto\Presto.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}" "" "$INSTDIR\${PRODUCT_EXECUTABLE}" 0
    CreateShortCut "$SMPROGRAMS\Presto\Uninstall Presto.lnk" "$INSTDIR\uninstall.exe" "" "$INSTDIR\uninstall.exe" 0
  ${EndIf}
SectionEnd

Section "下载模板" SEC_TEMPLATES
  ${If} $DownloadTemplates == ${BST_CHECKED}
    Banner::show /set 76 "正在下载模板" "请稍候，正在下载模板..."
    ExecWait '"$INSTDIR\${PRODUCT_EXECUTABLE}" --download-templates' $0
    Banner::destroy
  ${EndIf}
SectionEnd

Section "注册表" SEC_REGISTRY
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

  WriteRegStr HKLM "Software\Classes\Presto.Markdown" "" "Presto Markdown 文档"
  WriteRegStr HKLM "Software\Classes\Presto.Markdown" "FriendlyTypeName" "Markdown 文档"
  WriteRegStr HKLM "Software\Classes\Presto.Markdown\DefaultIcon" "" "$INSTDIR\${PRODUCT_EXECUTABLE},0"
  WriteRegStr HKLM "Software\Classes\Presto.Markdown\shell" "" "open"
  WriteRegStr HKLM "Software\Classes\Presto.Markdown\shell\open" "" "使用 Presto 打开"
  WriteRegStr HKLM "Software\Classes\Presto.Markdown\shell\open\command" "" "$\"$INSTDIR\${PRODUCT_EXECUTABLE}$\" $\"%1$\""

  WriteRegStr HKLM "Software\Classes\Applications\Presto.exe" "FriendlyAppName" "Presto"
  WriteRegStr HKLM "Software\Classes\Applications\Presto.exe\DefaultIcon" "" "$INSTDIR\${PRODUCT_EXECUTABLE},0"
  WriteRegStr HKLM "Software\Classes\Applications\Presto.exe\shell\open" "" "使用 Presto 打开"
  WriteRegStr HKLM "Software\Classes\Applications\Presto.exe\shell\open\command" "" "$\"$INSTDIR\${PRODUCT_EXECUTABLE}$\" $\"%1$\""
  WriteRegStr HKLM "Software\Classes\Applications\Presto.exe\SupportedTypes" ".md" ""
  WriteRegStr HKLM "Software\Classes\Applications\Presto.exe\SupportedTypes" ".markdown" ""

  WriteRegStr HKLM "Software\Classes\.md\OpenWithProgids" "Presto.Markdown" ""
  WriteRegStr HKLM "Software\Classes\.markdown\OpenWithProgids" "Presto.Markdown" ""

  WriteRegStr HKLM "Software\RegisteredApplications" "Presto" "Software\Clients\Presto\Capabilities"
  WriteRegStr HKLM "Software\Clients\Presto\Capabilities" "ApplicationName" "Presto"
  WriteRegStr HKLM "Software\Clients\Presto\Capabilities" "ApplicationDescription" "Markdown 转 Typst 转 PDF 编辑器"
  WriteRegStr HKLM "Software\Clients\Presto\Capabilities\FileAssociations" ".md" "Presto.Markdown"
  WriteRegStr HKLM "Software\Clients\Presto\Capabilities\FileAssociations" ".markdown" "Presto.Markdown"

  System::Call 'shell32::SHChangeNotify(i 0x08000000, i 0, p 0, p 0)'
SectionEnd

Function .onInstSuccess
  ${If} $LaunchAfterInstall == ${BST_CHECKED}
    Exec '"$INSTDIR\${PRODUCT_EXECUTABLE}"'
  ${EndIf}
FunctionEnd

Section "卸载"
  SetRegView 64

  StrCpy $KeepUserData 0
  IfSilent silent_uninstall prompt_confirm

  prompt_confirm:
    MessageBox MB_YESNO "确定要卸载 Presto 吗？" IDYES ask_keep_data
    Abort

  ask_keep_data:
    MessageBox MB_YESNO "保留用户数据目录？$\n$\n如果您选择'是'，您的文档、模板和设置将被保留。$\n$\n位置：$PROFILE\.presto" IDYES keep_data
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
    MessageBox MB_OK "Presto 已成功卸载。"

  done:
SectionEnd
