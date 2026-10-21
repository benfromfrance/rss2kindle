package mailer

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/benoit/rss2kindle/internal/store"
	"github.com/wneessen/go-mail"
)

// Mailer gère l'envoi de mails SMTP avec pièces jointes EPUB compatibles Send-to-Kindle.
type Mailer struct{}

func New() *Mailer {
	return &Mailer{}
}

// SendEpub envoie un fichier EPUB à l'adresse email Kindle configurée.
func (m *Mailer) SendEpub(ctx context.Context, st store.Settings, title, epubPath string) error {
	if !st.MailConfigured() {
		return fmt.Errorf("configuration SMTP ou adresse Kindle incomplète")
	}

	msg := mail.NewMsg()
	if err := msg.FromFormat(st.SenderName, st.SenderEmail); err != nil {
		return fmt.Errorf("adresse expéditeur invalide: %w", err)
	}
	if err := msg.To(st.KindleEmail); err != nil {
		return fmt.Errorf("adresse Kindle destinataire invalide: %w", err)
	}

	subject := title
	if subject == "" {
		subject = "RSS to Kindle Delivery"
	}
	msg.Subject(subject)

	// Amazon Send-to-Kindle attend souvent un corps de texte simple
	msg.SetBodyString(mail.TypeTextPlain, fmt.Sprintf("Veuillez trouver ci-joint votre recueil Kindle: %s\nGénéré le %s par RSS to Kindle.", title, time.Now().Format("02/01/2006 15:04")))

	// Pièce jointe EPUB
	filename := filepath.Base(epubPath)
	if !strings.HasSuffix(strings.ToLower(filename), ".epub") {
		filename += ".epub"
	}

	msg.AttachFile(epubPath, mail.WithFileName(filename))

	// Options client SMTP
	opts := []mail.Option{
		mail.WithPort(st.SMTPPort),
		mail.WithTimeout(30 * time.Second),
	}

	switch st.SMTPSecurity {
	case "tls":
		opts = append(opts, mail.WithSSLPort(true))
	case "none":
		opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
	default: // "starttls"
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory))
	}

	if st.SMTPUser != "" && st.SMTPPassword != "" {
		opts = append(opts,
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(st.SMTPUser),
			mail.WithPassword(st.SMTPPassword),
		)
	}

	client, err := mail.NewClient(st.SMTPHost, opts...)
	if err != nil {
		return fmt.Errorf("création client SMTP: %w", err)
	}

	if err := client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("envoi SMTP vers %s échoué: %w", st.KindleEmail, err)
	}

	return nil
}

// SendTest envoie un simple e-mail de test de configuration.
func (m *Mailer) SendTest(ctx context.Context, st store.Settings, recipient string) error {
	if recipient == "" {
		recipient = st.KindleEmail
	}
	if recipient == "" {
		return fmt.Errorf("adresse destinataire requise pour le test")
	}

	msg := mail.NewMsg()
	if err := msg.FromFormat(st.SenderName, st.SenderEmail); err != nil {
		return fmt.Errorf("adresse expéditeur invalide: %w", err)
	}
	if err := msg.To(recipient); err != nil {
		return fmt.Errorf("adresse destinataire invalide: %w", err)
	}

	msg.Subject("RSS to Kindle - Test SMTP réussi")
	msg.SetBodyString(mail.TypeTextPlain, "Félicitations !\nLa configuration SMTP de votre instance RSS to Kindle fonctionne correctement.\nDate: "+time.Now().Format(time.RFC1123))

	opts := []mail.Option{
		mail.WithPort(st.SMTPPort),
		mail.WithTimeout(20 * time.Second),
	}
	switch st.SMTPSecurity {
	case "tls":
		opts = append(opts, mail.WithSSLPort(true))
	case "none":
		opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
	default:
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory))
	}

	if st.SMTPUser != "" && st.SMTPPassword != "" {
		opts = append(opts,
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(st.SMTPUser),
			mail.WithPassword(st.SMTPPassword),
		)
	}

	client, err := mail.NewClient(st.SMTPHost, opts...)
	if err != nil {
		return fmt.Errorf("création client SMTP: %w", err)
	}

	return client.DialAndSendWithContext(ctx, msg)
}
