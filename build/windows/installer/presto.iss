; Presto Windows installer
; Built with Inno Setup for CI and local release builds.

#ifndef ARG_VERSION
  #error ARG_VERSION is required
#endif

#ifndef ARG_FILE_VERSION
  #define ARG_FILE_VERSION "0.0.0.0"
#endif

#ifndef ARG_ARCH
  #error ARG_ARCH is required
#endif

#ifndef ARG_BINARY
  #error ARG_BINARY is required
#endif

#ifndef ARG_TYPST_BINARY
  #error ARG_TYPST_BINARY is required
#endif

#ifndef ARG_VC_REDIST
  #error ARG_VC_REDIST is required
#endif

#ifndef ARG_TEMPLATE_DIR
  #error ARG_TEMPLATE_DIR is required
#endif

#ifndef ARG_OUTPUT_DIR
  #define ARG_OUTPUT_DIR "."
#endif

#ifndef ARG_OUTPUT_BASENAME
  #define ARG_OUTPUT_BASENAME "Presto-" + ARG_VERSION + "-windows-" + ARG_ARCH + "-installer"
#endif

#define INFO_COMPANYNAME "Presto-io"
#define INFO_PRODUCTNAME "Presto"
#define INFO_PRODUCTVERSION ARG_VERSION
#define INFO_COPYRIGHT "Copyright (c) 2026 Presto-io"
#define INFO_FILEVERSION ARG_FILE_VERSION
#define PRODUCT_EXECUTABLE "Presto.exe"
#define TYPST_EXECUTABLE "typst.exe"
#define PRODUCT_URL "https://presto.mre.red"

#if ARG_ARCH == "arm64"
  #define ARCHITECTURES_ALLOWED "arm64"
  #define ARCHITECTURES_64BIT "arm64"
#else
  #define ARCHITECTURES_ALLOWED "x64compatible and not arm64"
  #define ARCHITECTURES_64BIT "x64compatible and not arm64"
#endif

[Setup]
AppId={{A0C80F67-6C99-4B86-BB94-3F661D8CF9D2}
AppName={#INFO_PRODUCTNAME}
AppVersion={#INFO_PRODUCTVERSION}
AppVerName={#INFO_PRODUCTNAME} {#INFO_PRODUCTVERSION}
AppPublisher={#INFO_COMPANYNAME}
AppPublisherURL={#PRODUCT_URL}
AppSupportURL={#PRODUCT_URL}
AppUpdatesURL={#PRODUCT_URL}
AppCopyright={#INFO_COPYRIGHT}
AppMutex=com.mrered.presto
DefaultDirName={autopf}\{#INFO_COMPANYNAME}\{#INFO_PRODUCTNAME}
DefaultGroupName={#INFO_PRODUCTNAME}
DisableProgramGroupPage=yes
OutputDir={#ARG_OUTPUT_DIR}
OutputBaseFilename={#ARG_OUTPUT_BASENAME}
SetupIconFile={#SourcePath}\resources\icon.ico
UninstallDisplayIcon={app}\{#PRODUCT_EXECUTABLE}
LicenseFile={#SourcePath}\license.txt
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=admin
MinVersion=10.0
ArchitecturesAllowed={#ARCHITECTURES_ALLOWED}
ArchitecturesInstallIn64BitMode={#ARCHITECTURES_64BIT}
ChangesAssociations=yes
CloseApplications=yes
RestartApplications=no
ShowLanguageDialog=no
VersionInfoVersion={#INFO_FILEVERSION}
VersionInfoCompany={#INFO_COMPANYNAME}
VersionInfoDescription={#INFO_PRODUCTNAME} Installer
VersionInfoProductName={#INFO_PRODUCTNAME}
VersionInfoProductVersion={#INFO_FILEVERSION}
VersionInfoCopyright={#INFO_COPYRIGHT}

[Languages]
Name: "zh"; MessagesFile: "{#SourcePath}\languages\ChineseSimplified.isl"

[CustomMessages]
zh.TaskGroup=附加选项：
zh.CreateDesktopShortcut=创建桌面快捷方式
zh.CreateStartMenuShortcut=创建开始菜单快捷方式
zh.InstallingVCRuntime=正在安装 Microsoft Visual C++ 运行库...
zh.LaunchPresto=安装完成后启动 Presto

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopShortcut}"; GroupDescription: "{cm:TaskGroup}"; Flags: checkedonce
Name: "startmenuicon"; Description: "{cm:CreateStartMenuShortcut}"; GroupDescription: "{cm:TaskGroup}"; Flags: checkedonce

[Files]
Source: "{#ARG_BINARY}"; DestDir: "{app}"; DestName: "{#PRODUCT_EXECUTABLE}"; Flags: ignoreversion
Source: "{#ARG_TYPST_BINARY}"; DestDir: "{app}"; DestName: "{#TYPST_EXECUTABLE}"; Flags: ignoreversion
Source: "{#ARG_VC_REDIST}"; DestDir: "{tmp}"; DestName: "vc_redist.exe"; Flags: deleteafterinstall
Source: "{#ARG_TEMPLATE_DIR}\gongwen\presto-template-gongwen.exe"; DestDir: "{code:UserPrestoDir}\templates\gongwen"; Flags: ignoreversion
Source: "{#ARG_TEMPLATE_DIR}\gongwen\manifest.json"; DestDir: "{code:UserPrestoDir}\templates\gongwen"; Flags: ignoreversion
Source: "{#ARG_TEMPLATE_DIR}\jiaoan-shicao\presto-template-jiaoan-shicao.exe"; DestDir: "{code:UserPrestoDir}\templates\jiaoan-shicao"; Flags: ignoreversion
Source: "{#ARG_TEMPLATE_DIR}\jiaoan-shicao\manifest.json"; DestDir: "{code:UserPrestoDir}\templates\jiaoan-shicao"; Flags: ignoreversion

[Icons]
Name: "{autodesktop}\Presto"; Filename: "{app}\{#PRODUCT_EXECUTABLE}"; WorkingDir: "{app}"; IconFilename: "{app}\{#PRODUCT_EXECUTABLE}"; Tasks: desktopicon
Name: "{group}\Presto"; Filename: "{app}\{#PRODUCT_EXECUTABLE}"; WorkingDir: "{app}"; IconFilename: "{app}\{#PRODUCT_EXECUTABLE}"; Tasks: startmenuicon
Name: "{group}\卸载 Presto"; Filename: "{uninstallexe}"; Tasks: startmenuicon

[Run]
Filename: "{tmp}\vc_redist.exe"; Parameters: "/install /quiet /norestart"; StatusMsg: "{cm:InstallingVCRuntime}"; Flags: runhidden waituntilterminated
Filename: "{app}\{#PRODUCT_EXECUTABLE}"; Description: "{cm:LaunchPresto}"; Flags: nowait postinstall skipifsilent

[Registry]
Root: HKA; Subkey: "Software\Classes\Presto.Markdown"; ValueType: string; ValueName: ""; ValueData: "Presto Markdown 文档"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\Presto.Markdown"; ValueType: string; ValueName: "FriendlyTypeName"; ValueData: "Markdown 文档"
Root: HKA; Subkey: "Software\Classes\Presto.Markdown\DefaultIcon"; ValueType: string; ValueName: ""; ValueData: "{app}\{#PRODUCT_EXECUTABLE},0"
Root: HKA; Subkey: "Software\Classes\Presto.Markdown\shell"; ValueType: string; ValueName: ""; ValueData: "open"
Root: HKA; Subkey: "Software\Classes\Presto.Markdown\shell\open"; ValueType: string; ValueName: ""; ValueData: "使用 Presto 打开"
Root: HKA; Subkey: "Software\Classes\Presto.Markdown\shell\open\command"; ValueType: string; ValueName: ""; ValueData: """{app}\{#PRODUCT_EXECUTABLE}"" ""%1"""

Root: HKA; Subkey: "Software\Classes\.md\OpenWithProgids"; ValueType: none; ValueName: "Presto.Markdown"; Flags: uninsdeletevalue
Root: HKA; Subkey: "Software\Classes\.markdown\OpenWithProgids"; ValueType: none; ValueName: "Presto.Markdown"; Flags: uninsdeletevalue

Root: HKA; Subkey: "Software\Classes\Applications\Presto.exe"; ValueType: string; ValueName: "FriendlyAppName"; ValueData: "Presto"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Classes\Applications\Presto.exe\DefaultIcon"; ValueType: string; ValueName: ""; ValueData: "{app}\{#PRODUCT_EXECUTABLE},0"
Root: HKA; Subkey: "Software\Classes\Applications\Presto.exe\shell\open"; ValueType: string; ValueName: ""; ValueData: "使用 Presto 打开"
Root: HKA; Subkey: "Software\Classes\Applications\Presto.exe\shell\open\command"; ValueType: string; ValueName: ""; ValueData: """{app}\{#PRODUCT_EXECUTABLE}"" ""%1"""
Root: HKA; Subkey: "Software\Classes\Applications\Presto.exe\SupportedTypes"; ValueType: none; ValueName: ".md"; Flags: uninsdeletevalue
Root: HKA; Subkey: "Software\Classes\Applications\Presto.exe\SupportedTypes"; ValueType: none; ValueName: ".markdown"; Flags: uninsdeletevalue

Root: HKA; Subkey: "Software\RegisteredApplications"; ValueType: string; ValueName: "Presto"; ValueData: "Software\Clients\Presto\Capabilities"; Flags: uninsdeletevalue
Root: HKA; Subkey: "Software\Clients\Presto\Capabilities"; ValueType: string; ValueName: "ApplicationName"; ValueData: "Presto"; Flags: uninsdeletekey
Root: HKA; Subkey: "Software\Clients\Presto\Capabilities"; ValueType: string; ValueName: "ApplicationDescription"; ValueData: "Markdown 转 Typst 转 PDF 编辑器"
Root: HKA; Subkey: "Software\Clients\Presto\Capabilities\FileAssociations"; ValueType: string; ValueName: ".md"; ValueData: "Presto.Markdown"
Root: HKA; Subkey: "Software\Clients\Presto\Capabilities\FileAssociations"; ValueType: string; ValueName: ".markdown"; ValueData: "Presto.Markdown"

[UninstallDelete]
Type: filesandordirs; Name: "{code:UserPrestoDir}"; Check: ShouldDeleteUserData

[Code]
var
  KeepUserData: Boolean;

function UserPrestoDir(Param: string): string;
begin
  Result := AddBackslash(GetEnv('USERPROFILE')) + '.presto';
end;

function StripOuterQuotes(Value: string): string;
begin
  Result := Value;
  if (Length(Result) >= 2) and (Copy(Result, 1, 1) = '"') and (Copy(Result, Length(Result), 1) = '"') then
    Result := Copy(Result, 2, Length(Result) - 2);
end;

procedure RunLegacyInstallerUninstaller();
var
  UninstallString: string;
  InstallLocation: string;
  UninstallerPath: string;
  Params: string;
  ResultCode: Integer;
begin
  if not RegQueryStringValue(HKLM64, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\Presto', 'UninstallString', UninstallString) then
    Exit;

  RegQueryStringValue(HKLM64, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\Presto', 'InstallLocation', InstallLocation);
  UninstallerPath := StripOuterQuotes(UninstallString);
  Params := '/S';
  if InstallLocation <> '' then
    Params := Params + ' _?=' + InstallLocation;

  if not Exec(UninstallerPath, Params, '', SW_HIDE, ewWaitUntilTerminated, ResultCode) then begin
    MsgBox('检测到旧版 Presto，但无法启动旧版卸载程序。安装将继续进行。', mbInformation, MB_OK);
    Exit;
  end;

  if ResultCode <> 0 then
    MsgBox('检测到旧版 Presto，但自动卸载失败（错误码: ' + IntToStr(ResultCode) + '）。安装将继续进行。', mbInformation, MB_OK);
end;

function InitializeSetup(): Boolean;
begin
  RunLegacyInstallerUninstaller();
  Result := True;
end;

function InitializeUninstall(): Boolean;
begin
  KeepUserData := True;

  if UninstallSilent then begin
    Result := True;
    Exit;
  end;

  if MsgBox('确定要卸载 Presto 吗？', mbConfirmation, MB_YESNO) = IDNO then begin
    Result := False;
    Exit;
  end;

  KeepUserData :=
    MsgBox(
      '保留用户数据目录？' + #13#10 + #13#10 +
      '如果您选择“是”，您的文档、模板和设置将被保留。' + #13#10 + #13#10 +
      '位置：' + UserPrestoDir(''),
      mbConfirmation,
      MB_YESNO
    ) = IDYES;

  Result := True;
end;

function ShouldDeleteUserData(): Boolean;
begin
  Result := not KeepUserData;
end;
