package template

import (
	"bytes"
	"fmt"
	"text/template"
)

// Registered as Go text/template strings and rendered in-process — no files on disk.

const instructionTmpl = `📖 Инструкция по использованию
<a href="https://sub.uuu-uuu.tech/docs/user-guide">PDF документ с руководством</a>

1. Откройте приложение по кнопке под этим сообщением ↓
2. Нажмите "Login" в приложении и запросите доступ
3. Я подтверждаю доступ (<a href="tg://user?id=5477149309">@Augustine_kptz</a>)
4. После получения подтверждения доступа нажать еще раз "Login" и создайте профиль
5. Скачайте Happ для нужной платформы
   <a href="https://play.google.com/store/apps/details?id=com.happproxy">Android</a> | <a href="https://github.com/Happ-proxy/happ-android/releases/latest/download/Happ.apk">Android APK</a>
   <a href="https://apps.apple.com/ru/app/happ-proxy-utility-plus/id6746188973">iOS</a>
   <a href="https://github.com/Happ-proxy/happ-desktop/releases/latest/download/setup-Happ.x64.exe">Windows</a>
   <a href="https://github.com/Happ-proxy/happ-desktop/releases/latest/download/Happ.macOS.universal.dmg">macOS</a>
6. Отсканируйте QR-код в <a href="https://www.happ.su/main/ru">Happ приложении</a> или скопируйте ссылку

Готово! VPN активирован 🎉`

const profileReadyTmpl = `Ваш профиль готов ✅. Отсканируйте QR или импортируйте файл конфигурации`

const approveTmpl = `{{if .Approved}}✅ Доступ к xray подтвержден{{else}}❌ Доступ к xray отклонен{{end}}`

const batteryAlertTmpl = `⚠️ <b>Низкий заряд батареи</b>

Текущий заряд: <b>{{.Capacity}}%</b>

Пожалуйста, подключите зарядное устройство!`

var (
	approveTemplate      = template.Must(template.New("approve").Parse(approveTmpl))
	batteryAlertTemplate = template.Must(template.New("battery_alert").Parse(batteryAlertTmpl))
)

// Instruction returns the static bot-instruction text (mini-app onboarding flow).
func Instruction() string {
	return instructionTmpl
}

// ProfileReadyCaption returns the caption attached to a freshly generated VPN config document.
func ProfileReadyCaption() string {
	return profileReadyTmpl
}

// ApproveData holds the fields the approve/reject template renders.
type ApproveData struct {
	Approved bool
}

// RenderApprove renders the approval/rejection message.
func RenderApprove(data ApproveData) (string, error) {
	return render(approveTemplate, data)
}

// BatteryAlertData holds the fields the battery-alert template renders.
type BatteryAlertData struct {
	Capacity int
}

// RenderBatteryAlert renders the low-battery HTML alert message.
func RenderBatteryAlert(data BatteryAlertData) (string, error) {
	return render(batteryAlertTemplate, data)
}

func render(tmpl *template.Template, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render template %s: %w", tmpl.Name(), err)
	}
	return buf.String(), nil
}
