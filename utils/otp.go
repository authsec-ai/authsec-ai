package utils

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/smtp"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
)

// GenerateOTPFunc is the function variable for generating OTPs, allowing for mocking in tests
var GenerateOTPFunc = func() (string, error) {
	max := big.NewInt(999999)
	min := big.NewInt(100000)

	n, err := rand.Int(rand.Reader, max.Sub(max, min))
	if err != nil {
		return "", err
	}

	otp := n.Add(n, min).String()
	return otp, nil
}

// GenerateOTP generates a 6-digit OTP using the swappable function variable
func GenerateOTP() (string, error) {
	return GenerateOTPFunc()
}

// SendOTPEmailFunc is the function variable for sending OTP emails, allowing for mocking in tests
var SendOTPEmailFunc = func(email, otp string) error {
	log.Printf("SendOTPEmail: preparing OTP email for %s", email)
	// Email configuration from environment variables
	smtpHost := config.AppConfig.SMTPHost
	smtpPort := config.AppConfig.SMTPPort
	smtpUser := config.AppConfig.SMTPUser
	smtpPass := config.AppConfig.SMTPPassword

	if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPass == "" {
		log.Printf("SendOTPEmail: incomplete SMTP configuration (host=%q port=%q user=%q)", smtpHost, smtpPort, smtpUser)
		return fmt.Errorf("SMTP configuration is incomplete")
	}

	log.Printf("SendOTPEmail: using SMTP host=%s port=%s user=%s", smtpHost, smtpPort, smtpUser)

	// Email content
	subject := "Email Verification - Your OTP Code"
	body := fmt.Sprintf(`
Dear User,

Your email verification OTP is: %s

This OTP will expire in 10 minutes. Please do not share this code with anyone.

If you didn't request this verification, please ignore this email.

Best regards,
Your App Team
    `, otp)

	message := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", email, subject, body)

	// SMTP authentication
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	// Send email
	log.Printf("SendOTPEmail: attempting to send OTP email to %s", email)
	err := smtp.SendMail(
		fmt.Sprintf("%s:%s", smtpHost, smtpPort),
		auth,
		smtpUser,
		[]string{email},
		[]byte(message),
	)

	if err != nil {
		log.Printf("SendOTPEmail: failed to send OTP email to %s: %v", email, err)
	} else {
		log.Printf("SendOTPEmail: successfully sent OTP email to %s", email)
	}

	return err
}

// SendOTPEmail sends OTP via email using the swappable function variable
func SendOTPEmail(email, otp string) error {
	return SendOTPEmailFunc(email, otp)
}

// Add this function to your utils/otp.go file alongside your existing SendOTPEmail function

// SendPasswordResetOTPEmailFunc is the function variable for sending password reset OTP emails
var SendPasswordResetOTPEmailFunc = func(email, otp string) error {
	log.Printf("SendPasswordResetOTPEmail: preparing password reset OTP email for %s", email)
	// Email configuration from environment variables (same as existing function)
	smtpHost := config.AppConfig.SMTPHost
	smtpPort := config.AppConfig.SMTPPort
	smtpUser := config.AppConfig.SMTPUser
	smtpPass := config.AppConfig.SMTPPassword

	if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPass == "" {
		log.Printf("SendPasswordResetOTPEmail: incomplete SMTP configuration (host=%q port=%q user=%q)", smtpHost, smtpPort, smtpUser)
		return fmt.Errorf("SMTP configuration is incomplete")
	}

	log.Printf("SendPasswordResetOTPEmail: using SMTP host=%s port=%s user=%s", smtpHost, smtpPort, smtpUser)

	// Password reset email content
	subject := "Password Reset - Your OTP Code"
	body := fmt.Sprintf(`
Dear User,

You have requested to reset your password. Your password reset OTP is: %s

This OTP will expire in 10 minutes. Please do not share this code with anyone.

If you didn't request a password reset, please ignore this email or contact support if you have concerns.

Best regards,
Your App Team
    `, otp)

	message := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", email, subject, body)

	// SMTP authentication (same as existing function)
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	// Send email
	log.Printf("SendPasswordResetOTPEmail: attempting to send password reset OTP email to %s", email)
	err := smtp.SendMail(
		fmt.Sprintf("%s:%s", smtpHost, smtpPort),
		auth,
		smtpUser,
		[]string{email},
		[]byte(message),
	)

	if err != nil {
		log.Printf("SendPasswordResetOTPEmail: failed to send password reset OTP email to %s: %v", email, err)
	} else {
		log.Printf("SendPasswordResetOTPEmail: successfully sent password reset OTP email to %s", email)
	}

	return err
}

// SendPasswordResetOTPEmail sends password reset OTP via email using the swappable function variable
func SendPasswordResetOTPEmail(email, otp string) error {
	return SendPasswordResetOTPEmailFunc(email, otp)
}

func GenerateTemporaryPassword() (string, error) {
	const (
		length = 20 // Consistent with admin invite password length
		chars  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	)

	password := make([]byte, length)
	for i := range password {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random character: %w", err)
		}
		password[i] = chars[n.Int64()]
	}

	return string(password), nil
}

// SendTemporaryPasswordEmail sends temporary password via email
func SendTemporaryPasswordEmail(email, tempPassword string) error {
	// Email configuration from environment variables (same as existing function)
	smtpHost := config.AppConfig.SMTPHost
	smtpPort := config.AppConfig.SMTPPort
	smtpUser := config.AppConfig.SMTPUser
	smtpPass := config.AppConfig.SMTPPassword

	if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPass == "" {
		return fmt.Errorf("SMTP configuration is incomplete")
	}

	// Temporary password email content
	subject := "Temporary Password - Account Access"
	body := fmt.Sprintf(`
Dear User,

Your account password has been reset by an administrator. Please use the following temporary password to log in:

Temporary Password: %s

IMPORTANT:
- This is a temporary password generated by an administrator
- Please change this password immediately after logging in
- For security reasons, do not share this password with anyone

If you did not request this password reset, please contact your administrator immediately.

Best regards,
Your Security Team
    `, tempPassword)

	message := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", email, subject, body)

	// SMTP authentication (same as existing function)
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	// Send email
	err := smtp.SendMail(
		fmt.Sprintf("%s:%s", smtpHost, smtpPort),
		auth,
		smtpUser,
		[]string{email},
		[]byte(message),
	)

	return err
}

// SendAdminInviteEmail sends a tailored invite email for admin users with login URL, username, and temp password.
func SendAdminInviteEmail(email, username, tenantDomain, tempPassword string) error {
	log.Printf("SendAdminInviteEmail: preparing admin invite email for %s", email)
	// Email configuration from environment variables
	smtpHost := config.AppConfig.SMTPHost
	smtpPort := config.AppConfig.SMTPPort
	smtpUser := config.AppConfig.SMTPUser
	smtpPass := config.AppConfig.SMTPPassword

	if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPass == "" {
		log.Printf("SendAdminInviteEmail: incomplete SMTP configuration (host=%q port=%q user=%q)", smtpHost, smtpPort, smtpUser)
		return fmt.Errorf("SMTP configuration is incomplete")
	}

	// Build login URL from tenant domain; fall back to base URL if tenant domain missing.
	loginURL := strings.TrimSpace(tenantDomain)
	if loginURL == "" {
		loginURL = strings.TrimSpace(config.AppConfig.BaseURL)
	}
	if loginURL != "" && !strings.HasPrefix(strings.ToLower(loginURL), "http") {
		loginURL = "https://" + loginURL
	}

	log.Printf("SendAdminInviteEmail: using SMTP host=%s port=%s user=%s", smtpHost, smtpPort, smtpUser)

	subject := "You have been invited to AuthSec"
	body := fmt.Sprintf(`
Hello,

You have been invited to AuthSec. Use the details below to sign in:

- Username: %s
- Temporary Password: %s
- Login URL: %s

This temporary password is valid for your first login. Please sign in and change it immediately.

If you did not expect this invite, contact your administrator.

Regards,
AuthSec Team
`, username, tempPassword, loginURL)

	message := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", email, subject, body)

	// SMTP authentication
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	// Send email
	log.Printf("SendAdminInviteEmail: attempting to send admin invite email to %s", email)
	err := smtp.SendMail(
		fmt.Sprintf("%s:%s", smtpHost, smtpPort),
		auth,
		smtpUser,
		[]string{email},
		[]byte(message),
	)

	if err != nil {
		log.Printf("SendAdminInviteEmail: failed to send admin invite email to %s: %v", email, err)
	} else {
		log.Printf("SendAdminInviteEmail: successfully sent admin invite email to %s", email)
	}

	return err
}

// SendAccountDeactivationEmail sends notification when user account is deactivated
func SendAccountDeactivationEmail(email string) error {
	log.Printf("SendAccountDeactivationEmail: preparing deactivation email for %s", email)
	// Email configuration from environment variables
	smtpHost := config.AppConfig.SMTPHost
	smtpPort := config.AppConfig.SMTPPort
	smtpUser := config.AppConfig.SMTPUser
	smtpPass := config.AppConfig.SMTPPassword

	if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPass == "" {
		log.Printf("SendAccountDeactivationEmail: incomplete SMTP configuration (host=%q port=%q user=%q)", smtpHost, smtpPort, smtpUser)
		return fmt.Errorf("SMTP configuration is incomplete")
	}

	log.Printf("SendAccountDeactivationEmail: using SMTP host=%s port=%s user=%s", smtpHost, smtpPort, smtpUser)

	// Account deactivation email content
	subject := "Account Deactivation Notice"
	body := `
Dear User,

Your account has been deactivated by an administrator.

You will no longer be able to access the system with this account. If you believe this was done in error or if you need to regain access, please contact your system administrator.

If you have any questions or concerns, please reach out to our support team.

Best regards,
Your Security Team
    `

	message := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", email, subject, body)

	// SMTP authentication
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	// Send email
	log.Printf("SendAccountDeactivationEmail: attempting to send deactivation email to %s", email)
	err := smtp.SendMail(
		fmt.Sprintf("%s:%s", smtpHost, smtpPort),
		auth,
		smtpUser,
		[]string{email},
		[]byte(message),
	)

	if err != nil {
		log.Printf("SendAccountDeactivationEmail: failed to send deactivation email to %s: %v", email, err)
	} else {
		log.Printf("SendAccountDeactivationEmail: successfully sent deactivation email to %s", email)
	}

	return err
}

// SendNewUserRegistrationNotificationEmail sends a notification to the tenant owner when a new user registers.
func SendNewUserRegistrationNotificationEmail(ownerEmail, userName, userEmail, tenantDomain string) error {
	log.Printf("SendNewUserRegistrationNotificationEmail: preparing notification email for owner %s", ownerEmail)

	smtpHost := config.AppConfig.SMTPHost
	smtpPort := config.AppConfig.SMTPPort
	smtpUser := config.AppConfig.SMTPUser
	smtpPass := config.AppConfig.SMTPPassword

	if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPass == "" {
		log.Printf("SendNewUserRegistrationNotificationEmail: incomplete SMTP configuration (host=%q port=%q user=%q)", smtpHost, smtpPort, smtpUser)
		return fmt.Errorf("SMTP configuration is incomplete")
	}

	log.Printf("SendNewUserRegistrationNotificationEmail: using SMTP host=%s port=%s user=%s", smtpHost, smtpPort, smtpUser)

	subject := "New User Registration Notification"
	body := fmt.Sprintf(`
Hello,

A new user has registered under your tenant.

- Name: %s
- Email: %s
- Tenant Domain: %s
- Registration Time: %s

If you did not expect this registration, please review your tenant settings.

Regards,
AuthSec Team
`, userName, userEmail, tenantDomain, time.Now().UTC().Format(time.RFC1123))

	message := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", ownerEmail, subject, body)

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	log.Printf("SendNewUserRegistrationNotificationEmail: attempting to send notification email to %s", ownerEmail)
	err := smtp.SendMail(
		fmt.Sprintf("%s:%s", smtpHost, smtpPort),
		auth,
		smtpUser,
		[]string{ownerEmail},
		[]byte(message),
	)

	if err != nil {
		log.Printf("SendNewUserRegistrationNotificationEmail: failed to send notification email to %s: %v", ownerEmail, err)
	} else {
		log.Printf("SendNewUserRegistrationNotificationEmail: successfully sent notification email to %s", ownerEmail)
	}

	return err
}
