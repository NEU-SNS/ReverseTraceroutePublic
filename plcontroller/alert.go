package plcontroller

import (
	"fmt"
	"net/smtp"
)

func SendEmail(to []string, subject string, body string) error {

  // from Sender data.
  user := "reverse.traceroute@gmail.com"
  from := user
  // to Receiver email address.


  // smtp server configuration.
  smtpHost := "smtp.gmail.com"
  smtpPort := "587"

  // Message.
  msg := "From: " + from + "\n" +
		"To: " + to[0] + "\n" +
		"Subject: " + subject + "\n\n" + body

// msg := "From: idriss.aberkane@hyperdoctor.edu\n" +
// "To: " + to[0] + "\n" +
// "Subject: " + subject + "\n\n" + body
  
  // Authentication.
  auth := smtp.PlainAuth("", user, appPassword, smtpHost)
  
  // Sending email.
  err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, []byte(msg))
  if err != nil {
    fmt.Println(err)
    return err
  }
  return nil 
}
