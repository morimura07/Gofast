package login

import (
	"fmt"

	"github.com/twilio/twilio-go"
	verify "github.com/twilio/twilio-go/rest/verify/v2"
)

// TwilioClient defines the interface for Twilio's Verify v2 service.
// It's used to abstract the Twilio client for testing purposes.
type TwilioClient interface {
	CreateVerification(serviceSid string, params *verify.CreateVerificationParams) (*verify.VerifyV2Verification, error)
	CreateVerificationCheck(serviceSid string, params *verify.CreateVerificationCheckParams) (*verify.VerifyV2VerificationCheck, error)
}

// Twilio implements the TwilioClient interface using the Twilio SDK.
type Twilio struct {
	Client *verify.ApiService
}

// CreateTwilioClient creates the underlying Twilio client for use with Twilio struct.
func CreateTwilioClient() *verify.ApiService {
	return twilio.NewRestClient().VerifyV2
}

func (t Twilio) CreateVerification(serviceSid string, params *verify.CreateVerificationParams) (*verify.VerifyV2Verification, error) {
	v, err := t.Client.CreateVerification(serviceSid, params)
	if err != nil {
		return nil, fmt.Errorf("twilio create verification: %w", err)
	}
	return v, nil
}

func (t Twilio) CreateVerificationCheck(serviceSid string, params *verify.CreateVerificationCheckParams) (*verify.VerifyV2VerificationCheck, error) {
	v, err := t.Client.CreateVerificationCheck(serviceSid, params)
	if err != nil {
		return nil, fmt.Errorf("twilio create verification check: %w", err)
	}
	return v, nil
}
