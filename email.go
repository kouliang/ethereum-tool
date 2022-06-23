package ethereumtool

import (
	"net/smtp"

	"github.com/jordan-wright/email"
)

func SenEmail(subject, content string, to []string) error {
	auth := smtp.PlainAuth("", "coderkl@qq.com", "rqglgzkwaqksgffb", "smtp.qq.com")

	e := email.NewEmail()
	e.From = "coderkl@qq.com"
	e.To = to
	e.Subject = subject
	e.Text = []byte(content)

	return e.Send("smtp.qq.com:25", auth)
}
