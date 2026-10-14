package biz

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/app/user/service/internal/data"
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"
	"github.com/hellokitty980808-ctrl/im-core/pkg/storage"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/mojocn/base64Captcha"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var accountPattern = regexp.MustCompile(`^[a-z0-9_]{3,32}$`)

// AccountUseCase 处理账户和好友；身份仅取自 Gateway 确认的内部 RPC 元数据。
type AccountUseCase struct {
	repo            *data.Repository
	captchaDisabled bool
}

func NewAccountUseCase(repo *data.Repository, r *platform.Runtime) *AccountUseCase {
	return &AccountUseCase{
		repo:            repo,
		captchaDisabled: r.Config.CaptchaDisabled,
	}
}
func profile(row storage.Account) *pb.Profile {
	return &pb.Profile{
		UserId:    row.ID,
		Account:   row.Account,
		Nickname:  row.Nickname,
		AvatarUrl: row.AvatarURL,
		Bio:       row.Bio,
	}
}
func bad(message string) error           { return kerrors.BadRequest("INVALID_ARGUMENT", message) }
func missing() error                     { return kerrors.NotFound("NOT_FOUND", "记录不存在") }
func forbidden() error                   { return kerrors.Forbidden("PERMISSION_DENIED", "无权执行此操作") }
func validPassword(password string) bool { return len(password) >= 8 && len(password) <= 72 }
func (u *AccountUseCase) account(ctx context.Context, id uint64) (*storage.Account, error) {
	row := &storage.Account{}
	err := u.repo.DB.WithContext(ctx).First(row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, missing()
	}
	if err != nil {
		return nil, err
	}
	if row.Disabled {
		return nil, forbidden()
	}
	return row, nil
}

// captchaStore 使用 GETDEL 一次性消费，避免并发重用验证码。
type captchaStore struct{ repo *data.Repository }

func (s captchaStore) Set(id, value string) error {
	return s.repo.Redis.Set(context.Background(), "imcore:captcha:"+id, value, 5*time.Minute).Err()
}
func (s captchaStore) Get(id string, clear bool) string {
	if clear {
		return s.repo.Redis.GetDel(context.Background(), "imcore:captcha:"+id).Val()
	}
	return s.repo.Redis.Get(context.Background(), "imcore:captcha:"+id).Val()
}
func (s captchaStore) Verify(id, answer string, clear bool) bool {
	actual := s.Get(id, clear)
	return actual != "" && strings.EqualFold(actual, answer)
}
func (u *AccountUseCase) verifyCaptcha(id, code string) error {
	if u.captchaDisabled {
		return nil
	}
	if !(captchaStore{repo: u.repo}).Verify(id, code, true) {
		return bad("验证码错误或已过期")
	}
	return nil
}
func (u *AccountUseCase) CreateCaptcha(ctx context.Context, _ *emptypb.Empty) (*pb.Captcha, error) {
	driver := base64Captcha.NewDriverDigit(80, 240, 6, 0.7, 80)
	id, image, _, err := base64Captcha.NewCaptcha(driver, captchaStore{repo: u.repo}).Generate()
	if err != nil {
		return nil, err
	}
	return &pb.Captcha{
		CaptchaId: id,
		Image:     image,
	}, nil
}
func (u *AccountUseCase) Register(ctx context.Context, in *pb.Registration) (*pb.Session, error) {
	account := strings.ToLower(in.Account)
	nicknameLength := utf8.RuneCountInString(strings.TrimSpace(in.Nickname))
	if !accountPattern.MatchString(account) || !validPassword(in.Password) ||
		nicknameLength < 2 || nicknameLength > 32 {
		return nil, bad("账号、密码或昵称格式错误")
	}
	if err := u.verifyCaptcha(in.CaptchaId, in.CaptchaCode); err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	row := storage.Account{
		Account:      account,
		PasswordHash: string(hash),
		Nickname:     strings.TrimSpace(in.Nickname),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err = u.repo.DB.WithContext(ctx).Create(&row).Error; err != nil {
		var count int64
		if e := u.repo.DB.WithContext(ctx).Model(&storage.Account{}).Where("account = ?", account).Count(&count).Error; e != nil {
			return nil, e
		}
		if count > 0 {
			return nil, kerrors.Conflict("ACCOUNT_EXISTS", "账号已存在")
		}
		return nil, err
	}
	return u.issue(ctx, row, "")
}
func (u *AccountUseCase) issue(ctx context.Context, row storage.Account, expected string) (*pb.Session, error) {
	token, expires, err := u.repo.Sessions.Issue(ctx, row.ID, expected)
	if err != nil {
		return nil, err
	}
	return &pb.Session{
		Token:     token,
		ExpiresAt: expires,
		User:      profile(row),
	}, nil
}
func (u *AccountUseCase) Login(ctx context.Context, in *pb.Credentials) (*pb.Session, error) {
	if err := u.verifyCaptcha(in.CaptchaId, in.CaptchaCode); err != nil {
		return nil, err
	}
	var result *pb.Session
	// 与密码修改共用账户行锁，防止旧密码登录跨越密码变更。
	err := u.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row storage.Account
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("account = ?", strings.ToLower(in.Account)).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return kerrors.Unauthorized("INVALID_CREDENTIALS", "账号或密码错误")
		}
		if err != nil {
			return err
		}
		if row.Disabled || bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(in.Password)) != nil {
			return kerrors.Unauthorized("INVALID_CREDENTIALS", "账号或密码错误")
		}
		result, err = u.issue(ctx, row, "")
		return err
	})
	return result, err
}
func (u *AccountUseCase) Authenticate(ctx context.Context, in *pb.SessionToken) (*pb.Identity, error) {
	id, sid, err := u.repo.Sessions.Verify(ctx, in.Token)
	if err != nil {
		return nil, err
	}
	if _, err = u.account(ctx, id); err != nil {
		return nil, err
	}
	return &pb.Identity{
		UserId:    id,
		SessionId: sid,
	}, nil
}
func (u *AccountUseCase) Refresh(ctx context.Context, _ *emptypb.Empty) (*pb.Session, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	row, err := u.account(ctx, id)
	if err != nil {
		return nil, err
	}
	return u.issue(ctx, *row, platform.SessionID(ctx))
}
func (u *AccountUseCase) Logout(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, u.repo.Sessions.Revoke(ctx, id, platform.SessionID(ctx))
}
func (u *AccountUseCase) GetMe(ctx context.Context, _ *emptypb.Empty) (*pb.Profile, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	return u.GetProfile(ctx, &pb.UserID{UserId: id})
}
func (u *AccountUseCase) GetProfile(ctx context.Context, in *pb.UserID) (*pb.Profile, error) {
	if in.UserId == 0 {
		return nil, bad("用户 ID 不能为空")
	}
	row, err := u.account(ctx, in.UserId)
	if err != nil {
		return nil, err
	}
	return profile(*row), nil
}
func (u *AccountUseCase) FindUser(ctx context.Context, in *pb.AccountQuery) (*pb.Profile, error) {
	var row storage.Account
	err := u.repo.DB.WithContext(ctx).Where("account = ? AND disabled = false", strings.ToLower(in.Account)).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, missing()
	}
	if err != nil {
		return nil, err
	}
	return profile(row), nil
}
func (u *AccountUseCase) UpdateMe(ctx context.Context, in *pb.ProfileUpdate) (*pb.Profile, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	changes := map[string]any{"updated_at": time.Now().UnixMilli()}
	if in.Nickname != nil {
		if n := utf8.RuneCountInString(strings.TrimSpace(*in.Nickname)); n < 2 || n > 32 {
			return nil, bad("昵称长度为 2–32 个字符")
		}
		changes["nickname"] = strings.TrimSpace(*in.Nickname)
	}
	if in.AvatarUrl != nil {
		if len(*in.AvatarUrl) > 512 {
			return nil, bad("头像地址过长")
		}
		changes["avatar_url"] = *in.AvatarUrl
	}
	if in.Bio != nil {
		if utf8.RuneCountInString(*in.Bio) > 200 {
			return nil, bad("简介过长")
		}
		changes["bio"] = *in.Bio
	}
	if err = u.repo.DB.WithContext(ctx).Model(&storage.Account{}).Where("id = ?", id).Updates(changes).Error; err != nil {
		return nil, err
	}
	return u.GetProfile(ctx, &pb.UserID{UserId: id})
}
func (u *AccountUseCase) ChangePassword(ctx context.Context, in *pb.PasswordChange) (*emptypb.Empty, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	if !validPassword(in.NewPassword) {
		return nil, bad("密码长度为 8–72 字节")
	}
	err = u.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row storage.Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error; err != nil {
			return err
		}
		if bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(in.CurrentPassword)) != nil {
			return bad("当前密码错误")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if err = tx.Model(&row).Updates(map[string]any{
			"password_hash": string(hash),
			"updated_at":    time.Now().UnixMilli(),
		}).Error; err != nil {
			return err
		}
		return u.repo.Sessions.Revoke(ctx, id, "")
	})
	return &emptypb.Empty{}, err
}
