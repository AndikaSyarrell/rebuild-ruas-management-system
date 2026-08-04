package service

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/smtp"
)

type MailConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
	FromName string
}

type MailService struct {
	cfg    MailConfig
	logger *slog.Logger
}

func NewMailService(cfg MailConfig) *MailService {
	return &MailService{cfg: cfg, logger: slog.Default()}
}

// send mengirim email HTML. Jika SMTPHost belum dikonfigurasi, panggilan ini
// no-op (aman dipakai di lingkungan development tanpa SMTP server).
func (m *MailService) send(to, subject, htmlBody string) error {
	if m.cfg.Host == "" {
		return nil
	}

	headers := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n",
		m.cfg.FromName, m.cfg.From, to, subject)

	msg := []byte(headers + htmlBody)

	var err error
	if m.cfg.Port == "465" {
		// Port 465 = implicit TLS (server mengharapkan TLS handshake sejak
		// koneksi pertama dibuka) - net/smtp.SendMail TIDAK support ini,
		// harus buka koneksi TLS manual dulu baru serahkan ke smtp.Client.
		err = m.sendImplicitTLS(to, msg)
	} else {
		// Port 587 (atau lainnya) = STARTTLS, sudah didukung smtp.SendMail bawaan.
		err = m.sendStartTLS(to, msg)
	}

	if err != nil {
		m.logger.Error("gagal mengirim email",
			slog.String("to", to),
			slog.String("subject", subject),
			slog.String("smtp_host", m.cfg.Host),
			slog.String("smtp_port", m.cfg.Port),
			slog.String("error", err.Error()),
		)
	}
	return err
}

func (m *MailService) sendStartTLS(to string, msg []byte) error {
	addr := fmt.Sprintf("%s:%s", m.cfg.Host, m.cfg.Port)
	auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)
	return smtp.SendMail(addr, auth, m.cfg.From, []string{to}, msg)
}

func (m *MailService) sendImplicitTLS(to string, msg []byte) error {
	addr := fmt.Sprintf("%s:%s", m.cfg.Host, m.cfg.Port)

	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.cfg.Host})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := client.Mail(m.cfg.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}
	return client.Quit()
}

func wrapTemplate(body string) string {
	return `<!DOCTYPE html><html><body>
	<div style="width:500px;margin:0 auto;font-family:sans-serif;">` + body + `
	<br><br><em>*Do not reply to this e-mail.<br/><br/>Thank you!</em>
	</div></body></html>`
}

func (m *MailService) SendActivationEmail(toEmail, userName, activationURL string) error {
	body := fmt.Sprintf(`<div>Dear %s,<br><br>Ruas Management System.<br><br>
	The administrator has invited you to join.<br><br>
	<a href="%s" style="padding:6px 25px;background-color:#f26723;display:inline-block;color:#ffffff;text-decoration:none !important;">Activate Account</a>
	</div>`, userName, activationURL)
	return m.send(toEmail, "Ruas Management System Account Register", wrapTemplate(body))
}

func (m *MailService) SendNewAdminRegisteredNotice(toEmails []string, adminName, adminEmail string) error {
	if len(toEmails) == 0 {
		return nil
	}
	body := fmt.Sprintf(`<div>Dear Admin Warehouse,<br><br>
	There is a new admin registered in our Ruas Management System<br><br>
	With Name %s and Email %s.</div>`, adminName, adminEmail)
	subject := "New Admin Register - Ruas Management System"
	var lastErr error
	for _, to := range toEmails {
		if err := m.send(to, subject, wrapTemplate(body)); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (m *MailService) SendForgetPasswordEmail(toEmail, resetURL string) error {
	body := fmt.Sprintf(`<div>Dear Admin RMS,<br><br>
	This is your reset password link.<br><br>
	<a href="%s" style="padding:6px 25px;background-color:#f26723;display:inline-block;color:#ffffff;text-decoration:none !important;">Reset Password</a>
	</div>`, resetURL)
	return m.send(toEmail, "Ruas Management System - Reset Password", wrapTemplate(body))
}
