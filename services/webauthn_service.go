package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
	"github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/api/v2010"
)

// WebAuthnTOTPService handles TOTP operations for the WebAuthn handlers.
// It is distinct from TOTPService (user-flow's tenant TOTP service backed by a DB repository).
type WebAuthnTOTPService struct{}

func NewWebAuthnTOTPService() *WebAuthnTOTPService {
	return &WebAuthnTOTPService{}
}

// GenerateSecret generates a new TOTP secret for a user.
func (s *WebAuthnTOTPService) GenerateSecret(accountName, issuer string) (*otp.Key, error) {
	return totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
		SecretSize:  32,
	})
}

// ValidateCode validates a TOTP code.
func (s *WebAuthnTOTPService) ValidateCode(secret, code string) bool {
	return totp.Validate(code, secret)
}

// ValidateCodeWithWindow validates a TOTP code with a custom time window.
func (s *WebAuthnTOTPService) ValidateCodeWithWindow(secret, code string, window int) bool {
	now := time.Now()
	for i := -window; i <= window; i++ {
		t := now.Add(time.Duration(i) * 30 * time.Second)
		valid, err := totp.ValidateCustom(code, secret, t, totp.ValidateOpts{
			Period:    30,
			Skew:      0,
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		})
		if err == nil && valid {
			return true
		}
	}
	return false
}

// GenerateQRCode generates a QR code PNG for the TOTP key.
func (s *WebAuthnTOTPService) GenerateQRCode(key *otp.Key, size int) ([]byte, error) {
	return qrcode.Encode(key.String(), qrcode.Medium, size)
}

// GenerateBackupCodes generates n random backup codes.
func (s *WebAuthnTOTPService) GenerateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code, err := s.generateRandomCode(8)
		if err != nil {
			return nil, err
		}
		codes[i] = code
	}
	return codes, nil
}

func (s *WebAuthnTOTPService) generateRandomCode(length int) (string, error) {
	b := make([]byte, length/2)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(b)), nil
}

// FormatBackupCode formats a backup code for display (e.g. "ABCD-EFGH").
func (s *WebAuthnTOTPService) FormatBackupCode(code string) string {
	if len(code) != 8 {
		return code
	}
	return fmt.Sprintf("%s-%s", code[:4], code[4:])
}

// SMSService sends SMS messages via Twilio.
type SMSService struct {
	client *twilio.RestClient
}

func NewSMSService() *SMSService {
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")

	if accountSid == "" || authToken == "" {
		log.Println("  Twilio credentials not set, SMS service will use mock mode")
		return &SMSService{client: nil}
	}

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSid,
		Password: authToken,
	})
	return &SMSService{client: client}
}

// GenerateCode generates a random 6-digit SMS verification code.
func (s *SMSService) GenerateCode() (string, error) {
	max := big.NewInt(999999)
	min := big.NewInt(100000)
	n, err := rand.Int(rand.Reader, max.Sub(max, min).Add(max, big.NewInt(1)))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Add(n, min).Int64()), nil
}

// SendCode sends the verification code to phoneNumber via SMS.
func (s *SMSService) SendCode(phoneNumber, code string) error {
	fromNumber := os.Getenv("TWILIO_FROM_NUMBER")
	if fromNumber == "" {
		fromNumber = "+1234567890"
	}

	message := fmt.Sprintf("Your AuthSec verification code is: %s. This code expires in 5 minutes.", code)

	if s.client == nil {
		log.Printf("[MOCK SMS] To: %s, Code: %s", phoneNumber, code)
		return nil
	}

	params := &api.CreateMessageParams{}
	params.SetTo(phoneNumber)
	params.SetFrom(fromNumber)
	params.SetBody(message)

	_, err := s.client.Api.CreateMessage(params)
	if err != nil {
		log.Printf("Failed to send SMS: %v", err)
		return err
	}

	log.Printf("SMS sent to %s", phoneNumber)
	return nil
}

// ValidatePhoneNumber checks basic E.164 format.
func (s *SMSService) ValidatePhoneNumber(phoneNumber string) bool {
	if len(phoneNumber) < 10 || len(phoneNumber) > 15 {
		return false
	}
	if phoneNumber[0] != '+' {
		return false
	}
	for _, c := range phoneNumber[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// FormatPhoneForDisplay masks the middle digits of a phone number.
func (s *SMSService) FormatPhoneForDisplay(phoneNumber string) string {
	if len(phoneNumber) < 8 {
		return phoneNumber
	}
	return phoneNumber[:4] + "***" + phoneNumber[len(phoneNumber)-4:]
}
