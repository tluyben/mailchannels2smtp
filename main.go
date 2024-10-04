package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/gomail.v2"
)

type EmailAddress struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Attachment struct {
	Filename string `json:"filename"`
	Type     string `json:"type"`
	Content  string `json:"content"`
}

type ContentItem struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Personalization struct {
	To             []EmailAddress      `json:"to"`
	CC             []EmailAddress      `json:"cc"`
	BCC            []EmailAddress      `json:"bcc"`
	Subject        string              `json:"subject"`
	Headers        map[string]string   `json:"headers"`
	DKIMDomain     string              `json:"dkim_domain"`
	DKIMPrivateKey string              `json:"dkim_private_key"`
	DKIMSelector   string              `json:"dkim_selector"`
	ReplyTo        *EmailAddress       `json:"reply_to"`
	From           EmailAddress        `json:"from"`
}

type MailSendBody struct {
	Headers          map[string]string   `json:"headers"`
	Personalizations []Personalization   `json:"personalizations"`
	Attachments      []Attachment        `json:"attachments"`
	ReplyTo          *EmailAddress       `json:"reply_to"`
	Subject          string              `json:"subject"`
	From             EmailAddress        `json:"from"`
	MailFrom         *EmailAddress       `json:"mailfrom"`
	Content          []ContentItem       `json:"content"`
}

func main() {
	// Load environment variables
	loadEnv()

	http.HandleFunc("/tx/v1/send", handleSendEmail)
	port := getEnv("PORT", "8080")
	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func loadEnv() {
	// Check if required environment variables are already set
	requiredVars := []string{"SMTP_HOST", "SMTP_USER", "SMTP_PASSWORD", "SMTP_PORT", "SMTP_ENCRYPT"}
	missingVars := []string{}

	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			missingVars = append(missingVars, v)
		}
	}

	// If any required variables are missing, try to load from .env file
	if len(missingVars) > 0 {
		if err := godotenv.Load(); err != nil {
			log.Printf("Error loading .env file: %v", err)
		}

		// Check again for missing variables
		stillMissing := []string{}
		for _, v := range missingVars {
			if os.Getenv(v) == "" {
				stillMissing = append(stillMissing, v)
			}
		}

		if len(stillMissing) > 0 {
			log.Fatalf("Missing required environment variables: %s", strings.Join(stillMissing, ", "))
		}
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func handleSendEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var mailBody MailSendBody
	err := json.NewDecoder(r.Body).Decode(&mailBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	dryRun := r.URL.Query().Get("dry-run") == "true"

	if dryRun {
		renderedMessages := make([]string, len(mailBody.Personalizations))
		for i, p := range mailBody.Personalizations {
			renderedMessages[i] = renderMessage(mailBody, p)
		}
		json.NewEncoder(w).Encode(map[string][]string{"data": renderedMessages})
		return
	}

	err = sendEmails(mailBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func sendEmails(mailBody MailSendBody) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	smtpPort, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	smtpEncrypt := os.Getenv("SMTP_ENCRYPT")

	d := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPassword)

	switch smtpEncrypt {
	case "SSL":
		d.SSL = true
	case "TLS":
		d.SSL = false
		d.TLSConfig = nil // Use default TLS config
	case "PLAIN":
		d.SSL = false
		d.TLSConfig = nil
	default:
		return fmt.Errorf("invalid SMTP_ENCRYPT value: %s", smtpEncrypt)
	}

	for _, p := range mailBody.Personalizations {
		m := gomail.NewMessage()
		m.SetHeader("From", m.FormatAddress(mailBody.From.Email, mailBody.From.Name))
		for _, to := range p.To {
			m.SetHeader("To", m.FormatAddress(to.Email, to.Name))
		}
		for _, cc := range p.CC {
			m.SetHeader("Cc", m.FormatAddress(cc.Email, cc.Name))
		}
		for _, bcc := range p.BCC {
			m.SetHeader("Bcc", m.FormatAddress(bcc.Email, bcc.Name))
		}
		m.SetHeader("Subject", p.Subject)

		if p.ReplyTo != nil {
			m.SetHeader("Reply-To", m.FormatAddress(p.ReplyTo.Email, p.ReplyTo.Name))
		} else if mailBody.ReplyTo != nil {
			m.SetHeader("Reply-To", m.FormatAddress(mailBody.ReplyTo.Email, mailBody.ReplyTo.Name))
		}

		for k, v := range p.Headers {
			m.SetHeader(k, v)
		}

		for _, content := range mailBody.Content {
			m.SetBody(content.Type, content.Value)
		}

		for _, attachment := range mailBody.Attachments {
			decodedContent, err := base64.StdEncoding.DecodeString(attachment.Content)
			if err != nil {
				return fmt.Errorf("failed to decode attachment content: %v", err)
			}
			m.Attach(attachment.Filename, gomail.SetCopyFunc(func(w io.Writer) error {
				_, err := w.Write(decodedContent)
				return err
			}))
		}

		if err := d.DialAndSend(m); err != nil {
			return fmt.Errorf("failed to send email: %v", err)
		}
	}

	return nil
}

func renderMessage(mailBody MailSendBody, p Personalization) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("From: %s <%s>\n", mailBody.From.Name, mailBody.From.Email))
	sb.WriteString(fmt.Sprintf("To: %s\n", formatAddressList(p.To)))
	if len(p.CC) > 0 {
		sb.WriteString(fmt.Sprintf("CC: %s\n", formatAddressList(p.CC)))
	}
	if len(p.BCC) > 0 {
		sb.WriteString(fmt.Sprintf("BCC: %s\n", formatAddressList(p.BCC)))
	}
	sb.WriteString(fmt.Sprintf("Subject: %s\n", p.Subject))
	
	if p.ReplyTo != nil {
		sb.WriteString(fmt.Sprintf("Reply-To: %s <%s>\n", p.ReplyTo.Name, p.ReplyTo.Email))
	} else if mailBody.ReplyTo != nil {
		sb.WriteString(fmt.Sprintf("Reply-To: %s <%s>\n", mailBody.ReplyTo.Name, mailBody.ReplyTo.Email))
	}

	for k, v := range p.Headers {
		sb.WriteString(fmt.Sprintf("%s: %s\n", k, v))
	}

	sb.WriteString("\n")

	for _, content := range mailBody.Content {
		sb.WriteString(fmt.Sprintf("Content-Type: %s\n\n", content.Type))
		sb.WriteString(content.Value)
		sb.WriteString("\n\n")
	}

	for _, attachment := range mailBody.Attachments {
		sb.WriteString(fmt.Sprintf("Attachment: %s (Type: %s)\n", attachment.Filename, attachment.Type))
	}

	return sb.String()
}

func formatAddressList(addresses []EmailAddress) string {
	formatted := make([]string, len(addresses))
	for i, addr := range addresses {
		formatted[i] = fmt.Sprintf("%s <%s>", addr.Name, addr.Email)
	}
	return strings.Join(formatted, ", ")
}