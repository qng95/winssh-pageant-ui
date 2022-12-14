#go-winres make
go build -v -ldflags="-H=windowsgui"
Remove-Item ./WinSSH-PageantUI -Force -Recurse -Confirm:$false
new-item ./WinSSH-PageantUI -itemtype directory -Force
copy-item "./winssh-pageant-ui.exe" -Destination "./WinSSH-PageantUI"
copy-item -Path "./img" -Destination "./WinSSH-PageantUI/img" -Recurse