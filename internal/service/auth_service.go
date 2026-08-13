package service

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"log"
	"os"

	"github.com/redis/go-redis/v9"

	"rms-backend/internal/models"
	"rms-backend/internal/repository"
	"rms-backend/internal/utils"
)

// authDebugEnabled mengontrol apakah log [LOGIN DEBUG] dicetak. Log ini
// sebelumnya selalu aktif (termasuk di production), yang berisiko membocorkan
// detail internal (email, admin id, status akun) ke log server. Sekarang
// hanya aktif saat APP_ENV=development.
var authDebugEnabled = os.Getenv("APP_ENV") == "development"

func authDebugf(format string, args ...interface{}) {
	if authDebugEnabled {
		log.Printf(format, args...)
	}
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type AuthService struct {
	admins *repository.AdminRepo
	jwt    *utils.JWTManager
	rdb    *redis.Client
}

func NewAuthService(admins *repository.AdminRepo, jwtManager *utils.JWTManager, rdb *redis.Client) *AuthService {
	return &AuthService{admins: admins, jwt: jwtManager, rdb: rdb}
}

func refreshKey(jti string) string        { return "auth:refresh:" + jti }
func refreshSetKey(adminID string) string { return "auth:refresh:set:" + adminID }
func blacklistKey(jti string) string      { return "auth:blacklist:" + jti }

// Login memverifikasi kredensial terhadap T_Admin (bcrypt) dan menerbitkan
// pasangan access/refresh token. Refresh token disimpan di Redis (bukan hanya
// signature JWT) agar bisa di-revoke sewaktu-waktu (logout / ganti password).
func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, *models.Admin, error) {
	admin, err := s.admins.GetByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		authDebugf("[LOGIN DEBUG] user not found for email=%q", email)
		return nil, nil, ErrInvalidCredentials
	}
	if err != nil {
		authDebugf("[LOGIN DEBUG] GetByEmail query error: %v", err)
		return nil, nil, err
	}

	// DEBUG: confirm exactly what was fetched from the DB
	authDebugf("[LOGIN DEBUG] fetched admin id=%v email=%v active=%q password_is_nil=%v",
		admin.ID, admin.Email, admin.Active, admin.Password == nil)

	if admin.Active != "active" {
		authDebugf("[LOGIN DEBUG] rejected: active column = %q (expected exact string \"active\")", admin.Active)
		return nil, nil, ErrAccountInactive
	}

	if admin.Password == nil {
		authDebugf("[LOGIN DEBUG] rejected: admin.Password is nil")
		return nil, nil, ErrInvalidCredentials
	}

	// DEBUG: log lengths, not raw values, so you're not putting real
	// passwords/hashes in plaintext logs
	match := utils.CheckPassword(*admin.Password, password)
	authDebugf("[LOGIN DEBUG] CheckPassword result=%v hash_len=%d input_password_len=%d",
		match, len(*admin.Password), len(password))

	if !match {
		return nil, nil, ErrInvalidCredentials
	}

	pair, err := s.issueTokenPair(ctx, admin)
	if err != nil {
		authDebugf("[LOGIN DEBUG] issueTokenPair failed: %v", err)
		return nil, nil, err
	}
	return pair, admin, nil
}

// RefreshTTL mengekspos umur refresh token agar handler bisa mengatur
// Max-Age cookie tanpa perlu tahu detail JWTManager.
func (s *AuthService) RefreshTTL() time.Duration {
	return s.jwt.RefreshTTL()
}

func (s *AuthService) issueTokenPair(ctx context.Context, admin *models.Admin) (*TokenPair, error) {
	access, ajti, err := s.jwt.GenerateAccessToken(admin.ID, admin.Email, admin.RefRole)
	if err != nil {
		return nil, err
	}
	refresh, rjti, err := s.jwt.GenerateRefreshToken(admin.ID)
	if err != nil {
		return nil, err
	}

	if err := s.rdb.Set(ctx, refreshKey(rjti), admin.ID, s.jwt.RefreshTTL()).Err(); err != nil {
		return nil, err
	}
	if err := s.rdb.SAdd(ctx, refreshSetKey(admin.ID), rjti).Err(); err != nil {
		return nil, err
	}
	// jaga agar set tidak hidup abadi jika admin tidak pernah logout secara eksplisit.
	s.rdb.Expire(ctx, refreshSetKey(admin.ID), s.jwt.RefreshTTL())

	_ = ajti // jti access token tidak perlu disimpan kecuali saat blacklist (lihat Logout)

	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(s.jwt.AccessTTL().Seconds()),
	}, nil
}

// Refresh melakukan rotasi refresh token: token lama langsung dicabut begitu
// dipakai, token baru diterbitkan. Jika refresh token sudah tidak ada di Redis
// (sudah dipakai / di-revoke / expired), permintaan ditolak.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.jwt.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	storedAdminID, err := s.rdb.Get(ctx, refreshKey(claims.ID)).Result()
	if errors.Is(err, redis.Nil) {
		// jti tidak ada di store aktif: token ini sudah pernah dirotasi
		// (dipakai sebelumnya) atau memang tidak pernah valid. Kita tidak
		// bisa membedakan keduanya dari Redis saja, jadi sebagai tindakan
		// defensif terhadap refresh-token-reuse (token dicuri lalu dipakai
		// setelah pemilik asli sudah rotasi), kita revoke SEMUA sesi refresh
		// milik admin tersebut supaya token curian langsung tidak berguna.
		s.revokeAllRefreshSessions(ctx, claims.AdminID)
		return nil, ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, err
	}
	if storedAdminID != claims.AdminID {
		// Ketidakcocokan admin id vs jti: sinyal kuat token dipalsukan/disalahgunakan.
		s.revokeAllRefreshSessions(ctx, claims.AdminID)
		return nil, ErrInvalidRefreshToken
	}

	admin, err := s.admins.GetByID(ctx, claims.AdminID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, err
	}
	if admin.Active != "active" {
		return nil, ErrAccountInactive
	}

	// cabut refresh token lama sebelum menerbitkan yang baru (rotation).
	s.rdb.Del(ctx, refreshKey(claims.ID))
	s.rdb.SRem(ctx, refreshSetKey(admin.ID), claims.ID)

	return s.issueTokenPair(ctx, admin)
}

// Logout memblokir access token yang sedang dipakai (sampai masa berlaku aslinya
// habis) dan mencabut refresh token terkait bila disertakan.
func (s *AuthService) Logout(ctx context.Context, claims *utils.Claims, refreshToken string) error {
	if claims.ExpiresAt != nil {
		remaining := time.Until(claims.ExpiresAt.Time)
		if remaining > 0 {
			if err := s.rdb.Set(ctx, blacklistKey(claims.ID), "1", remaining).Err(); err != nil {
				return err
			}
		}
	}

	if refreshToken != "" {
		if rc, err := s.jwt.ParseRefreshToken(refreshToken); err == nil {
			s.rdb.Del(ctx, refreshKey(rc.ID))
			s.rdb.SRem(ctx, refreshSetKey(claims.AdminID), rc.ID)
		}
	}

	return nil
}

// IsAccessTokenBlacklisted dipakai middleware.Auth pada setiap request.
func (s *AuthService) IsAccessTokenBlacklisted(ctx context.Context, jti string) bool {
	n, err := s.rdb.Exists(ctx, blacklistKey(jti)).Result()
	if err != nil {
		// fail-closed akan mengunci semua user jika Redis down; di sini kita fail-open
		// karena signature+expiry JWT tetap tervalidasi terlebih dahulu oleh middleware.Auth.
		return false
	}
	return n > 0
}

// ChangePassword mewajibkan password lama benar, melarang password baru identik
// dengan yang lama, lalu me-revoke SEMUA sesi (refresh token) admin tersebut -
// termasuk access token yang sedang dipakai untuk memanggil endpoint ini -
// sehingga admin wajib login ulang di semua perangkat dengan password baru.
func (s *AuthService) ChangePassword(ctx context.Context, claims *utils.Claims, oldPassword, newPassword string) error {
	admin, err := s.admins.GetByID(ctx, claims.AdminID)
	if err != nil {
		return err
	}
	if admin.Active != "active" {
		return ErrAccountInactive
	}
	if admin.Password == nil || !utils.CheckPassword(*admin.Password, oldPassword) {
		return ErrWrongOldPassword
	}
	if utils.CheckPassword(*admin.Password, newPassword) {
		return ErrSamePassword
	}

	hash, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.admins.UpdatePassword(ctx, admin.ID, hash); err != nil {
		return err
	}

	s.revokeAllSessions(ctx, claims)
	return nil
}

// revokeAllRefreshSessions mencabut seluruh refresh token milik satu admin
// (dipakai baik untuk ChangePassword maupun untuk deteksi reuse token saat Refresh).
func (s *AuthService) revokeAllRefreshSessions(ctx context.Context, adminID string) {
	jtis, err := s.rdb.SMembers(ctx, refreshSetKey(adminID)).Result()
	if err == nil {
		for _, jti := range jtis {
			s.rdb.Del(ctx, refreshKey(jti))
		}
		s.rdb.Del(ctx, refreshSetKey(adminID))
	}
}

func (s *AuthService) revokeAllSessions(ctx context.Context, claims *utils.Claims) {
	s.revokeAllRefreshSessions(ctx, claims.AdminID)

	if claims.ExpiresAt != nil {
		remaining := time.Until(claims.ExpiresAt.Time)
		if remaining > 0 {
			s.rdb.Set(ctx, blacklistKey(claims.ID), "1", remaining)
		}
	}
}

// ActivateAccount memvalidasi token undangan, menetapkan password pertama, dan
// mengaktifkan akun (menggantikan Admin::activate_acc di PHP).
func (s *AuthService) ActivateAccount(ctx context.Context, email, token, newPassword string) error {
	if err := utils.ValidatePasswordStrength(newPassword); err != nil {
		return err
	}
	hash, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.admins.ActivateAccount(ctx, email, token, hash)
}

// RequestPasswordReset membuat kode reset acak & aman secara kriptografis
// (menggantikan sha1(mt_rand()) di PHP) dan mengembalikannya untuk dikirim via email.
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	code := utils.RandomHex(24)
	if err := s.admins.SetResetCode(ctx, email, code); err != nil {
		return "", err
	}
	return code, nil
}

func (s *AuthService) ResetPasswordWithCode(ctx context.Context, resetCode, newPassword string) error {
	admin, err := s.admins.GetByResetCode(ctx, resetCode)
	if err != nil {
		return err
	}
	if err := utils.ValidatePasswordStrength(newPassword); err != nil {
		return err
	}
	hash, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.admins.SetNewPassword(ctx, admin.Email, hash)
}