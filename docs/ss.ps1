function ss {
    # 1. 获取 ~/.ssh/config 路径
    $configPath = "$HOME\.ssh\config"
    if (-not (Test-Path $configPath)) {
        Write-Host "找不到 SSH 配置文件: $configPath" -ForegroundColor Red
        return
    }

    # 2. 提取 Host 列表并用 fzf 筛选
    $hostName = Select-String -Path $configPath -Pattern "^Host\s+(.+)" |
                ForEach-Object { $_.Matches.Groups[1].Value.Trim() } |
                Where-Object { $_ -notlike '*`*' } |
                fzf --height 40% --prompt="Select SSH Host> "

    # 3. 连接选中的主机
    if ($hostName) {
        Write-Host "正在连接到 $hostName..." -ForegroundColor Green
        & "ssh" $hostName
    }
}