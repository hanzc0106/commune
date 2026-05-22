package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hanzc0106/commune/apps/api/internal/auth"
	"github.com/hanzc0106/commune/apps/api/internal/db/queries"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool    *pgxpool.Pool
	queries *queries.Queries
}

type MemberDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type CategoryDTO struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	IconKey       string `json:"iconKey"`
	ColorKey      string `json:"colorKey"`
	SortOrder     int32  `json:"sortOrder"`
	SystemDefault bool   `json:"systemDefault"`
}

type InitializeInput struct {
	HouseholdName string `json:"householdName"`
	AdminName     string `json:"adminName"`
	PIN           string `json:"pin"`
}

type InitializeResult struct {
	Member MemberDTO `json:"member"`
}

type BootstrapResult struct {
	Initialized   bool        `json:"initialized"`
	HouseholdName string      `json:"householdName"`
	Session       *SessionDTO `json:"session"`
}

type SessionDTO struct {
	Member MemberDTO `json:"member"`
}

type LoginMemberDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type LoginInput struct {
	MemberID string
	PIN      string
}

type LoginResult struct {
	Member MemberDTO `json:"member"`
}

var defaultCategories = []struct {
	name      string
	kind      string
	iconKey   string
	colorKey  string
	sortOrder int32
}{
	{"餐饮", "expense", "utensils", "emerald", 10},
	{"日用", "expense", "shopping-bag", "sky", 20},
	{"交通", "expense", "bus", "amber", 30},
	{"住房", "expense", "home", "slate", 40},
	{"医疗", "expense", "heart-pulse", "rose", 50},
	{"娱乐", "expense", "gamepad-2", "violet", 60},
	{"孩子", "expense", "baby", "pink", 70},
	{"其他支出", "expense", "circle", "zinc", 990},
	{"工资", "income", "wallet", "emerald", 10},
	{"其他收入", "income", "plus-circle", "teal", 990},
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		pool:    pool,
		queries: queries.New(pool),
	}
}

func (s *Service) Initialize(ctx context.Context, input InitializeInput) (InitializeResult, string, error) {
	householdName := strings.TrimSpace(input.HouseholdName)
	adminName := strings.TrimSpace(input.AdminName)
	if householdName == "" {
		return InitializeResult{}, "", errors.New("household name is required")
	}
	if adminName == "" {
		return InitializeResult{}, "", errors.New("admin name is required")
	}

	exists, err := s.queries.AppSettingsExist(ctx)
	if err != nil {
		return InitializeResult{}, "", err
	}
	if exists {
		return InitializeResult{}, "", errors.New("application is already initialized")
	}

	pinHash, err := auth.HashPIN(input.PIN)
	if err != nil {
		return InitializeResult{}, "", err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return InitializeResult{}, "", err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)
	if _, err := qtx.CreateAppSettings(ctx, householdName); err != nil {
		return InitializeResult{}, "", err
	}
	member, err := qtx.CreateMember(ctx, queries.CreateMemberParams{
		Name:    adminName,
		PinHash: pinHash,
		Role:    "admin",
	})
	if err != nil {
		return InitializeResult{}, "", err
	}

	for _, category := range defaultCategories {
		if _, err := qtx.CreateCategory(ctx, queries.CreateCategoryParams{
			Name:          category.name,
			Type:          category.kind,
			IconKey:       category.iconKey,
			ColorKey:      category.colorKey,
			SortOrder:     category.sortOrder,
			SystemDefault: true,
		}); err != nil {
			return InitializeResult{}, "", err
		}
	}

	token, err := auth.NewSessionToken()
	if err != nil {
		return InitializeResult{}, "", err
	}
	if _, err := qtx.CreateSession(ctx, queries.CreateSessionParams{
		MemberID:  member.ID,
		TokenHash: auth.HashSessionToken(token),
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(30 * 24 * time.Hour),
			Valid: true,
		},
	}); err != nil {
		return InitializeResult{}, "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return InitializeResult{}, "", err
	}

	return InitializeResult{Member: memberDTO(member.ID, member.Name, member.Role)}, token, nil
}

func (s *Service) ListCategories(ctx context.Context) ([]CategoryDTO, error) {
	rows, err := s.queries.ListActiveCategories(ctx)
	if err != nil {
		return nil, err
	}
	categories := make([]CategoryDTO, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, categoryDTO(row))
	}
	return categories, nil
}

func (s *Service) Bootstrap(ctx context.Context, rawToken string) (BootstrapResult, error) {
	exists, err := s.queries.AppSettingsExist(ctx)
	if err != nil {
		return BootstrapResult{}, err
	}
	if !exists {
		return BootstrapResult{Initialized: false, HouseholdName: "", Session: nil}, nil
	}
	settings, err := s.queries.GetAppSettings(ctx)
	if err != nil {
		return BootstrapResult{}, err
	}
	result := BootstrapResult{
		Initialized:   true,
		HouseholdName: settings.HouseholdName,
		Session:       nil,
	}
	if rawToken == "" {
		return result, nil
	}
	session, err := s.SessionFromToken(ctx, rawToken)
	if err != nil {
		return result, nil
	}
	result.Session = &session
	return result, nil
}

func (s *Service) ListLoginMembers(ctx context.Context) ([]LoginMemberDTO, error) {
	rows, err := s.queries.ListActiveLoginMembers(ctx)
	if err != nil {
		return nil, err
	}
	members := make([]LoginMemberDTO, 0, len(rows))
	for _, row := range rows {
		members = append(members, LoginMemberDTO{
			ID:   row.ID.String(),
			Name: row.Name,
		})
	}
	return members, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (LoginResult, string, error) {
	memberID, err := uuidFromString(input.MemberID)
	if err != nil {
		return LoginResult{}, "", errors.New("invalid member ID")
	}
	member, err := s.queries.GetMemberByID(ctx, memberID)
	if err != nil {
		return LoginResult{}, "", errors.New("invalid member or PIN")
	}
	if !member.Active || !auth.VerifyPIN(member.PinHash, input.PIN) {
		return LoginResult{}, "", errors.New("invalid member or PIN")
	}
	token, err := auth.NewSessionToken()
	if err != nil {
		return LoginResult{}, "", err
	}
	if _, err := s.queries.CreateSession(ctx, queries.CreateSessionParams{
		MemberID:  member.ID,
		TokenHash: auth.HashSessionToken(token),
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(30 * 24 * time.Hour),
			Valid: true,
		},
	}); err != nil {
		return LoginResult{}, "", err
	}
	return LoginResult{Member: memberDTO(member.ID, member.Name, member.Role)}, token, nil
}

func (s *Service) SessionFromToken(ctx context.Context, rawToken string) (SessionDTO, error) {
	session, err := s.queries.GetSessionByTokenHash(ctx, auth.HashSessionToken(rawToken))
	if err != nil {
		return SessionDTO{}, err
	}
	if !session.MemberActive || session.ExpiresAt.Time.Before(time.Now()) {
		return SessionDTO{}, errors.New("session expired")
	}
	return SessionDTO{
		Member: memberDTO(session.MemberID, session.MemberName, session.MemberRole),
	}, nil
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	return s.queries.DeleteSessionByTokenHash(ctx, auth.HashSessionToken(rawToken))
}

func memberDTO(id pgtype.UUID, name string, role string) MemberDTO {
	return MemberDTO{
		ID:   id.String(),
		Name: name,
		Role: role,
	}
}

func categoryDTO(category queries.Category) CategoryDTO {
	return CategoryDTO{
		ID:            category.ID.String(),
		Name:          category.Name,
		Type:          category.Type,
		IconKey:       category.IconKey,
		ColorKey:      category.ColorKey,
		SortOrder:     category.SortOrder,
		SystemDefault: category.SystemDefault,
	}
}

func uuidFromString(value string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(value); err != nil {
		return pgtype.UUID{}, err
	}
	return id, nil
}
