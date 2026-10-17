# TarsEnterprise PowerShell Adapter — Environment Detection & Tool Listing
$info = @{
    lang = "powershell"
    version = $PSVersionTable.PSVersion.ToString()
    modules = @()
    tools = @()
}

# List installed modules
$info.modules = (Get-Module -ListAvailable | Select-Object -ExpandProperty Name | Sort-Object -Unique)

# Check common tools
$tools = @("dotnet", "docker", "kubectl", "helm", "az", "aws")
foreach ($tool in $tools) {
    if (Get-Command $tool -ErrorAction SilentlyContinue) {
        $info.tools += $tool
    }
}

$info | ConvertTo-Json -Depth 3
