package mailer

import (
	"bytes"
	"fmt"
	"gopkg.in/gomail.v2"
	"html/template"
	"log"
	"os"
)

type Mailer struct {
	host     string
	port     int
	username string
	password string
}

var (
	clientUrl = os.Getenv("CLIENT_URL")
)

func NewMailer(host string, port int, username string, password string) *Mailer {
	return &Mailer{
		host:     host,
		port:     port,
		username: username,
		password: password,
	}
}

func (m *Mailer) SendVerificationMail(fromEmail string, toEmail string, subject string, token string, emailTemplatePath string) error {

	tmpl := template.Must(template.ParseFiles(emailTemplatePath))

	type VerificationMailData struct {
		Subject         string
		Email           string
		VerificationUrl string
	}

	var body bytes.Buffer

	log.Println(clientUrl)

	if err := tmpl.Execute(&body, VerificationMailData{
		Subject:         subject,
		Email:           toEmail,
		VerificationUrl: fmt.Sprintf("%s/activate?token=%s", clientUrl, token),
	}); err != nil {
		return err
	}

	message := gomail.NewMessage()

	message.SetHeader("From", fromEmail)
	message.SetHeader("To", toEmail)
	message.SetHeader("Subject", subject)
	message.SetBody("text/html", body.String())

	d := gomail.NewDialer(m.host, m.port, m.username, m.password)

	return d.DialAndSend(message)

}
