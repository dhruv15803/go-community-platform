package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"

	"gopkg.in/gomail.v2"
)

type Mailer struct {
	host     string
	port     int
	username string
	password string
}

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

	log.Println("client url: ", os.Getenv("CLIENT_URL"))

	if err := tmpl.Execute(&body, VerificationMailData{
		Subject:         subject,
		Email:           toEmail,
		VerificationUrl: fmt.Sprintf("%s/activate?token=%s", os.Getenv("CLIENT_URL"), token),
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
