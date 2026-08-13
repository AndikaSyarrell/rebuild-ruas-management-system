package dto

// TokenPairResponse adalah representasi pasangan access/refresh token yang
// dikirim ke client setelah login/refresh. Dipisah dari service.TokenPair
// supaya field response API terkontrol eksplisit di sini, terlepas dari
// bagaimana struct internal service berevolusi.
type TokenPairResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// LoginResponse membungkus token pair + data admin yang login.
type LoginResponse struct {
	TokenPairResponse
	Admin AdminResponse `json:"admin"`
}

// MeResponse adalah representasi identitas admin dari JWT claims (endpoint
// GET /auth/me) - sengaja tipis, cuma field yang memang ada di claims,
// bukan hasil query ulang ke DB.
type MeResponse struct {
	AdminID string `json:"admin_id"`
	Email   string `json:"email"`
	RoleID  *int   `json:"role_id"`
}