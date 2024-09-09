package mail

import (
	"net/smtp"
	"os"
)

const (
	smtpHost = "smtp.gmail.com"
	smtpPort = "587"
)

var (
	from string
	auth smtp.Auth
)

func SendMail(subject string, message string) error {
	to := []string{os.Getenv("GMAIL_EMAIL")}

	mail := []byte("Subject: " + subject + "\n" + message)

	if err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, mail); err != nil {
		return err
	}

	return nil
}

func Init() {
	password := os.Getenv("GMAIL_PASSWORD")

	from = os.Getenv("GMAIL_EMAIL")
	auth = smtp.PlainAuth("", from, password, smtpHost)
}
