package dto

import "testing"

// --- LoginRequest ---

func TestLoginRequest_Normalize(t *testing.T) {
	req := LoginRequest{Email: "  Admin@RMS.Local  "}
	req.Normalize()
	if req.Email != "admin@rms.local" {
		t.Errorf("expected normalized email, got %q", req.Email)
	}
}

func TestLoginRequest_Valid(t *testing.T) {
	req := LoginRequest{Email: "admin@rms.local", Password: "Admin123!"}
	if err := req.Validate(); err != nil {
		t.Errorf("expected valid request to pass, got error: %v", err)
	}
}

func TestLoginRequest_MissingEmail(t *testing.T) {
	req := LoginRequest{Password: "Admin123!"}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing email")
	}
}

func TestLoginRequest_MissingPassword(t *testing.T) {
	req := LoginRequest{Email: "admin@rms.local"}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing password")
	}
}

// --- RefreshRequest ---

func TestRefreshRequest_Valid(t *testing.T) {
	req := RefreshRequest{RefreshToken: "some-token"}
	if err := req.Validate(); err != nil {
		t.Errorf("expected valid request to pass, got error: %v", err)
	}
}

func TestRefreshRequest_Empty(t *testing.T) {
	req := RefreshRequest{}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing refresh_token")
	}
}

// --- ResetPasswordRequest ---

func TestResetPasswordRequest_Valid(t *testing.T) {
	req := ResetPasswordRequest{OldPassword: "Old123!", NewPassword: "New123!", ConfirmPassword: "New123!"}
	if err := req.Validate(); err != nil {
		t.Errorf("expected valid request to pass, got error: %v", err)
	}
}

func TestResetPasswordRequest_MismatchedConfirmation(t *testing.T) {
	req := ResetPasswordRequest{OldPassword: "Old123!", NewPassword: "New123!", ConfirmPassword: "Different123!"}
	if err := req.Validate(); err == nil {
		t.Error("expected error when new_password and confirm_password mismatch")
	}
}

func TestResetPasswordRequest_MissingField(t *testing.T) {
	req := ResetPasswordRequest{OldPassword: "Old123!", NewPassword: "New123!"}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing confirm_password")
	}
}

// --- ForgotPasswordRequest ---

func TestForgotPasswordRequest_Normalize(t *testing.T) {
	req := ForgotPasswordRequest{Email: "  Admin@RMS.Local  "}
	req.Normalize()
	if req.Email != "admin@rms.local" {
		t.Errorf("expected normalized email, got %q", req.Email)
	}
}

func TestForgotPasswordRequest_Empty(t *testing.T) {
	req := ForgotPasswordRequest{}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing email")
	}
}

// --- ResetWithCodeRequest ---

func TestResetWithCodeRequest_Valid(t *testing.T) {
	req := ResetWithCodeRequest{ResetCode: "CODE", NewPassword: "New123!"}
	if err := req.Validate(); err != nil {
		t.Errorf("expected valid request to pass, got error: %v", err)
	}
}

func TestResetWithCodeRequest_MissingCode(t *testing.T) {
	req := ResetWithCodeRequest{NewPassword: "New123!"}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing reset_code")
	}
}

// --- ActivateRequest ---

func TestActivateRequest_Normalize(t *testing.T) {
	req := ActivateRequest{Email: "  NewAdmin@RMS.Local  "}
	req.Normalize()
	if req.Email != "newadmin@rms.local" {
		t.Errorf("expected normalized email, got %q", req.Email)
	}
}

func TestActivateRequest_Valid(t *testing.T) {
	req := ActivateRequest{Email: "a@b.com", Token: "tok", Password: "New123!"}
	if err := req.Validate(); err != nil {
		t.Errorf("expected valid request to pass, got error: %v", err)
	}
}

func TestActivateRequest_MissingToken(t *testing.T) {
	req := ActivateRequest{Email: "a@b.com", Password: "New123!"}
	if err := req.Validate(); err == nil {
		t.Error("expected error for missing token")
	}
}
