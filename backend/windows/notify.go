//go:build windows

package windows

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"fyne.io/fyne/v2"
)

// The general approach here is copied from Fyne.
// While it seems very hacky (create a temporary Powershell script and execute it),
// shockingly it may be the best approach, at least in the non-installed case.
// The proper Windows APIs for this require WinRT (ie C++/ a DLL), and also require
// the app to be installed with a unique ID in the start menu, and to pass this ID
// when sending the notification. This could be a future exploration for the installer.

const notificationTemplate = `$title = "%s"
$content = "%s"
$iconPath = "file:///%s"
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null
$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastImageAndText02)
$toastXml = [xml] $template.GetXml()
$toastXml.GetElementsByTagName("text")[0].AppendChild($toastXml.CreateTextNode($title)) > $null
$toastXml.GetElementsByTagName("text")[1].AppendChild($toastXml.CreateTextNode($content)) > $null
$toastXml.GetElementsByTagName("image")[0].SetAttribute("src", $iconPath) > $null
$audio = $toastXml.CreateElement("audio")
$audio.SetAttribute("silent", "true") > $null
$toastXml.DocumentElement.AppendChild($audio) > $null
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($toastXml.OuterXml)
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("%s").Show($toast);`

func SendNotification(n *fyne.Notification, iconFilePath string) {
	title := escapeNotificationString(n.Title)
	content := escapeNotificationString(n.Content)

	script := fmt.Sprintf(notificationTemplate, title, content, iconFilePath, "supersonic")
	go runScript("notify", script)
}

func escapeNotificationString(in string) string {
	noSlash := strings.ReplaceAll(in, "`", "``")
	return strings.ReplaceAll(noSlash, "\"", "`\"")
}

var scriptNum = 0

func runScript(name, script string) {
	scriptNum++
	appID := fyne.CurrentApp().UniqueID()
	fileName := fmt.Sprintf("supersonic-%s-%s-%d.ps1", appID, name, scriptNum)

	tmpFilePath := filepath.Join(os.TempDir(), fileName)
	err := os.WriteFile(tmpFilePath, []byte(script), 0o600)
	if err != nil {
		fyne.LogError("Could not write script to show notification", err)
		return
	}
	defer os.Remove(tmpFilePath)

	launch := "(Get-Content -Encoding UTF8 -Path " + tmpFilePath + " -Raw) | Invoke-Expression"
	cmd := exec.Command("PowerShell", "-ExecutionPolicy", "Bypass", launch)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err = cmd.Run()
	if err != nil {
		fyne.LogError("Failed to launch windows notify script", err)
	}
}
