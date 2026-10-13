; Moly installer for Windows (Inno Setup 6). Per-user: no administrator rights, nothing outside the user's own folders.
; Build with build.ps1 (or build-docker.sh), which passes:  /DAppVersion=0.1.0-beta.1  /DExtensionIds=id1,id2  /DSourceExe=...\moly.exe

#ifndef AppVersion
  #define AppVersion "0.0.0-dev"
#endif
#ifndef ExtensionIds
  #error ExtensionIds is empty: pass /DExtensionIds=<chrome id>[,<edge id>] (see installer\extension-ids.txt)
#endif
#ifndef SourceExe
  #define SourceExe "moly.exe"
#endif
#ifndef LicenseFile
  #define LicenseFile "..\..\LICENSE"
#endif

[Setup]
; Fixed identifier: the same installer upgrades an older installation.
AppId={{6E1D9C34-5B0A-4D52-9D71-3F4D6B0C7A11}
AppName=Moly
AppVersion={#AppVersion}
AppPublisher=Hermes Software
AppPublisherURL=https://github.com/Nireus79/Moly
AppSupportURL=https://github.com/Nireus79/Moly
DefaultDirName={localappdata}\Moly
DisableDirPage=yes
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
OutputDir=..\..\dist
OutputBaseFilename=moly-setup-{#AppVersion}
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
LicenseFile={#LicenseFile}
UninstallDisplayName=Moly
UninstallDisplayIcon={app}\moly.exe
; Signing is added later (SignPath or a certificate): SignTool=...
VersionInfoVersion=0.1.0.0
VersionInfoProductName=Moly

[Files]
Source: "{#SourceExe}"; DestDir: "{app}"; DestName: "moly.exe"; Flags: ignoreversion
Source: "{#LicenseFile}"; DestDir: "{app}"; DestName: "LICENSE.txt"; Flags: ignoreversion

[Run]
; Registers the app for Chrome, Chromium, Edge and Brave (registry keys under HKCU) and records the extension IDs.
Filename: "{app}\moly.exe"; Parameters: "--register-host --extension-ids ""{#ExtensionIds}"" --path ""{app}\moly.exe"""; Flags: runhidden waituntilterminated; StatusMsg: "Connecting Moly to your browsers..."

[UninstallRun]
Filename: "{app}\moly.exe"; Parameters: "--stop"; Flags: runhidden waituntilterminated; RunOnceId: "StopMoly"
Filename: "{app}\moly.exe"; Parameters: "--unregister-host"; Flags: runhidden waituntilterminated; RunOnceId: "UnregisterMoly"

[Code]
// Upgrade: a running backend would keep moly.exe locked, so ask it to stop first.
function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  ResultCode: Integer;
begin
  if FileExists(ExpandConstant('{app}\moly.exe')) then
    Exec(ExpandConstant('{app}\moly.exe'), '--stop', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Result := '';
end;

// Uninstall keeps the user's data unless they say otherwise. A silent uninstall always keeps it.
procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  DataDir, ConfigDir: String;
begin
  if CurUninstallStep = usPostUninstall then
  begin
    DataDir := ExpandConstant('{%USERPROFILE}\.moly');
    ConfigDir := ExpandConstant('{userappdata}\Moly');
    if (DirExists(DataDir) or DirExists(ConfigDir)) and (not UninstallSilent) then
    begin
      if MsgBox('Delete your Moly data as well (conversations, contacts, settings and the encrypted database)?' + #13#10 + #13#10 +
                'Choose No to keep it for a later installation.', mbConfirmation, MB_YESNO or MB_DEFBUTTON2) = IDYES then
      begin
        DelTree(DataDir, True, True, True);
        DelTree(ConfigDir, True, True, True);
      end;
    end;
  end;
end;
