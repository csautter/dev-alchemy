$script:SetupScript = Join-Path $PSScriptRoot "dev-alchemy-self-setup.ps1"

BeforeAll {
    $previousImportOnly = $env:DEV_ALCHEMY_SELF_SETUP_IMPORT_ONLY
    try {
        $env:DEV_ALCHEMY_SELF_SETUP_IMPORT_ONLY = "1"
        . $script:SetupScript
    } finally {
        if ($null -eq $previousImportOnly) {
            Remove-Item Env:\DEV_ALCHEMY_SELF_SETUP_IMPORT_ONLY -ErrorAction SilentlyContinue
        } else {
            $env:DEV_ALCHEMY_SELF_SETUP_IMPORT_ONLY = $previousImportOnly
        }
    }

    function Get-TestPathMockTarget {
        param(
            [Parameter(Mandatory = $true)]
            [hashtable]$BoundParameters
        )

        if ($BoundParameters.ContainsKey("LiteralPath")) {
            return $BoundParameters["LiteralPath"]
        }

        return $BoundParameters["Path"]
    }

    function Set-TestPathMock {
        param(
            [string[]]$ExistingPaths = @()
        )

        $script:ExistingPaths = $ExistingPaths
        Mock Test-Path {
            $target = Get-TestPathMockTarget -BoundParameters $PSBoundParameters
            return $script:ExistingPaths -contains $target
        }
    }

    function Initialize-SelfSetupMainMocks {
        $script:EnsuredPackages = @()
        $script:EnsuredPaths = @()
        $script:CygwinEnsureCalls = @()

        Mock Test-IsAdministrator { return $true }
        Mock Invoke-SelfElevated { throw "Self-elevation should not be requested by unit tests." }
        Mock Ensure-ChocolateyInstalled {}
        Mock Ensure-GoInstallRootClean {}
        Mock Assert-GoToolchainLayout {}
        Mock Ensure-ChocolateyPackage {
            $script:EnsuredPackages += [pscustomobject]@{
                PackageName = $PackageName
                Version = $Version
                ExtraArgs = @($ExtraArgs)
            }
        }
        Mock Assert-NativePythonAvailable {}
        Mock Export-CommandDirectoryToGitHubPath {}
        Mock Ensure-CygwinChocolateyPackage {
            $script:CygwinEnsureCalls += [pscustomobject]@{
                Version = $Version
                InstallRoot = $InstallRoot
            }
        }
        Mock Ensure-PathContains { $script:EnsuredPaths += $PathEntry }
        Mock Ensure-CygwinPackage {}
        Mock Ensure-CygwinPipPackage {}
        Mock Show-InstalledToolVersions {}
    }
}

Describe "dev-alchemy self setup Cygwin helpers" -Skip:(-not $IsWindows) {
    BeforeEach {
        $script:PreferredRoot = "C:\tools\cygwin"
        $script:CygwinVersion = "3.6.9"
    }

    Context "Get-CygwinRootDirIfAvailable" {
        It "returns the registry root before filesystem fallbacks" {
            Set-TestPathMock -ExistingPaths @(
                "HKLM:\SOFTWARE\Cygwin\setup",
                (Join-Path $script:PreferredRoot "bin\bash.exe")
            )
            Mock Get-ItemProperty { return [pscustomobject]@{ rootdir = "D:\cygwin" } }

            Get-CygwinRootDirIfAvailable | Should -Be "D:\cygwin"
        }

        It "falls back to an existing Cygwin bash path when registry roots are absent" {
            Set-TestPathMock -ExistingPaths @("C:\cygwin64\bin\bash.exe")
            Mock Get-ItemProperty { throw "Registry should not be queried when keys are absent." }

            Get-CygwinRootDirIfAvailable | Should -Be "C:\cygwin64"
        }

        It "returns null when no registry or fallback Cygwin root exists" {
            Set-TestPathMock
            Mock Get-ItemProperty { throw "Registry should not be queried when keys are absent." }

            Get-CygwinRootDirIfAvailable | Should -BeNullOrEmpty
        }
    }

    Context "Test-SameWindowsPath" {
        It "matches paths case-insensitively and ignores trailing separators" {
            Test-SameWindowsPath -Left "C:\TOOLS\cygwin\" -Right "c:/tools/cygwin" | Should -BeTrue
        }

        It "rejects empty and distinct paths" {
            Test-SameWindowsPath -Left "" -Right "C:\tools\cygwin" | Should -BeFalse
            Test-SameWindowsPath -Left "C:\tools\cygwin" -Right "D:\cygwin" | Should -BeFalse
        }
    }

    Context "Get-CygwinAlternateInstallRoot" {
        It "returns the versioned preferred sibling when it is available" {
            Set-TestPathMock

            Get-CygwinAlternateInstallRoot -PreferredRoot $script:PreferredRoot -Version $script:CygwinVersion |
                Should -Be "C:\tools\cygwin-3.6.9"
        }

        It "increments the versioned sibling when earlier candidates already exist" {
            Set-TestPathMock -ExistingPaths @(
                "C:\tools\cygwin-3.6.9",
                "C:\tools\cygwin-3.6.9-1"
            )

            Get-CygwinAlternateInstallRoot -PreferredRoot $script:PreferredRoot -Version $script:CygwinVersion |
                Should -Be "C:\tools\cygwin-3.6.9-2"
        }
    }

    Context "Get-CleanCygwinInstallRoot" {
        It "uses the preferred root when Cygwin is missing" {
            Set-TestPathMock
            Mock Stop-CygwinProcesses { throw "No Cygwin process should be stopped." }
            Mock Remove-CygwinInstallRoot { throw "No Cygwin root should be removed." }

            Get-CleanCygwinInstallRoot -PreferredRoot $script:PreferredRoot -ExistingRoot $null -Version $script:CygwinVersion |
                Should -Be $script:PreferredRoot
        }

        It "cleans the mismatched existing root before the preferred root" {
            $script:CleanCalls = @()
            Set-TestPathMock -ExistingPaths @("D:\cygwin", $script:PreferredRoot)
            Mock Stop-CygwinProcesses { $script:CleanCalls += ("stop:{0}" -f $RootDir) }
            Mock Remove-CygwinInstallRoot {
                $script:CleanCalls += ("remove:{0}" -f $RootDir)
                return $true
            }

            Get-CleanCygwinInstallRoot -PreferredRoot $script:PreferredRoot -ExistingRoot "D:\cygwin" -Version $script:CygwinVersion |
                Should -Be $script:PreferredRoot
            ($script:CleanCalls -join "|") |
                Should -Be "stop:D:\cygwin|remove:D:\cygwin|stop:C:\tools\cygwin|remove:C:\tools\cygwin"
        }

        It "uses an alternate root when the preferred root is locked" {
            Set-TestPathMock -ExistingPaths @($script:PreferredRoot)
            Mock Stop-CygwinProcesses {}
            Mock Remove-CygwinInstallRoot { return $false }
            Mock Get-CygwinAlternateInstallRoot { return "C:\tools\cygwin-3.6.9" }

            Get-CleanCygwinInstallRoot -PreferredRoot $script:PreferredRoot -ExistingRoot $null -Version $script:CygwinVersion |
                Should -Be "C:\tools\cygwin-3.6.9"
        }
    }

    Context "Stop-CygwinProcesses" {
        It "stops only processes whose executables are under the Cygwin root" {
            $script:StoppedProcessIds = @()
            Set-TestPathMock -ExistingPaths @($script:PreferredRoot)
            Mock Assert-CygwinRootSafeForAutomation {}
            Mock Get-CimInstance {
                return @(
                    [pscustomobject]@{ ProcessId = 101; Name = "bash"; ExecutablePath = "C:\tools\cygwin\bin\bash.exe" },
                    [pscustomobject]@{ ProcessId = 102; Name = "git"; ExecutablePath = "C:\Program Files\Git\bin\git.exe" },
                    [pscustomobject]@{ ProcessId = $PID; Name = "pwsh"; ExecutablePath = "C:\tools\cygwin\bin\pwsh.exe" }
                )
            }
            Mock Stop-Process { $script:StoppedProcessIds += $Id }

            Stop-CygwinProcesses -RootDir $script:PreferredRoot

            ($script:StoppedProcessIds -join ",") | Should -Be "101"
        }
    }

    Context "Ensure-CygwinChocolateyPackage" {
        It "does nothing when the pinned Cygwin version is already installed" {
            Mock Get-ChocolateyInstalledVersion { return $script:CygwinVersion }
            Mock Get-CleanCygwinInstallRoot { throw "Clean install root should not be resolved for an already pinned Cygwin package." }
            Mock Ensure-ChocolateyPackage { throw "Cygwin should not be installed when the pinned version is already present." }

            $null = Ensure-CygwinChocolateyPackage -Version $script:CygwinVersion -InstallRoot $script:PreferredRoot
        }

        It "installs missing Cygwin at the preferred root" {
            $script:InstalledCygwinArgs = @()
            Mock Get-ChocolateyInstalledVersion { return $null }
            Mock Get-CygwinRootDirIfAvailable { return $null }
            Mock Get-CleanCygwinInstallRoot { return $PreferredRoot }
            Mock Ensure-ChocolateyPackage { $script:InstalledCygwinArgs = @($ExtraArgs) }

            $null = Ensure-CygwinChocolateyPackage -Version $script:CygwinVersion -InstallRoot $script:PreferredRoot

            $script:InstalledCygwinArgs[0] | Should -Be "--params"
            $script:InstalledCygwinArgs[1] | Should -Be '"/InstallDir:C:\tools\cygwin /NoStartMenu"'
        }

        It "uninstalls cyg-get before a mismatched Cygwin package reinstall" {
            $script:Order = @()
            Mock Get-ChocolateyInstalledVersion {
                switch ($PackageName) {
                    "cygwin" { return "3.6.8" }
                    "cyg-get" { return "1.2.2" }
                    default { return $null }
                }
            }
            Mock Get-CygwinRootDirIfAvailable { return "C:\cygwin64" }
            Mock Stop-CygwinProcesses { $script:Order += ("stop:{0}" -f $RootDir) }
            Mock Invoke-ChocolateyPackageCommand {
                $script:Order += ("choco:{0}:{1}:{2}" -f $Command, $PackageName, $Version)
            }
            Mock Get-CleanCygwinInstallRoot {
                $script:Order += ("clean:{0}" -f $ExistingRoot)
                return $PreferredRoot
            }
            Mock Ensure-ChocolateyPackage {
                $script:Order += ("ensure:{0}:{1}" -f $PackageName, $Version)
            }

            $null = Ensure-CygwinChocolateyPackage -Version $script:CygwinVersion -InstallRoot $script:PreferredRoot

            ($script:Order -join "|") |
                Should -Be "stop:C:\cygwin64|choco:uninstall:cyg-get:1.2.2|choco:uninstall:cygwin:3.6.8|clean:C:\cygwin64|ensure:cygwin:3.6.9"
        }

        It "passes an alternate root to Chocolatey when cleanup cannot remove the preferred root" {
            $script:InstalledCygwinArgs = @()
            Mock Get-ChocolateyInstalledVersion { return $null }
            Mock Get-CygwinRootDirIfAvailable { return $null }
            Mock Get-CleanCygwinInstallRoot { return "C:\tools\cygwin-3.6.9" }
            Mock Ensure-ChocolateyPackage { $script:InstalledCygwinArgs = @($ExtraArgs) }

            $null = Ensure-CygwinChocolateyPackage -Version $script:CygwinVersion -InstallRoot $script:PreferredRoot

            $script:InstalledCygwinArgs[1] | Should -Be '"/InstallDir:C:\tools\cygwin-3.6.9 /NoStartMenu"'
        }

        It "stops before uninstalling when the registry root is corrupt and unsafe" {
            $script:Order = @()
            Mock Get-ChocolateyInstalledVersion {
                if ($PackageName -eq "cygwin") {
                    return "3.6.8"
                }
                return $null
            }
            Mock Get-CygwinRootDirIfAvailable { return "C:\Users" }
            Mock Stop-CygwinProcesses { throw "Refusing to automatically manage Cygwin root '$RootDir'." }
            Mock Invoke-ChocolateyPackageCommand { $script:Order += "uninstall" }
            Mock Ensure-ChocolateyPackage { $script:Order += "install" }

            { Ensure-CygwinChocolateyPackage -Version $script:CygwinVersion -InstallRoot $script:PreferredRoot } |
                Should -Throw "*Refusing to automatically manage Cygwin root 'C:\Users'*"
            $script:Order.Count | Should -Be 0
        }
    }

    Context "Invoke-DevAlchemySelfSetup" {
        It "uses the default install path without installing VirtualBox" {
            Initialize-SelfSetupMainMocks

            $null = Invoke-DevAlchemySelfSetup

            @($script:EnsuredPackages | Where-Object { $_.PackageName -eq "virtualbox" }).Count | Should -Be 0
            $script:EnsuredPaths | Should -Not -Contain "C:\Program Files\Oracle\VirtualBox"
            $script:CygwinEnsureCalls[0].InstallRoot | Should -Be $script:PreferredRoot
        }

        It "installs VirtualBox and adds its path when -VirtualBox is supplied" {
            Initialize-SelfSetupMainMocks

            $null = Invoke-DevAlchemySelfSetup -VirtualBox

            @($script:EnsuredPackages | Where-Object { $_.PackageName -eq "virtualbox" }).Count | Should -Be 1
            $script:EnsuredPaths | Should -Contain "C:\Program Files\Oracle\VirtualBox"
            $script:CygwinEnsureCalls[0].InstallRoot | Should -Be $script:PreferredRoot
        }
    }
}
