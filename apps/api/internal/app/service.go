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

type CreateTransactionInput struct {
	Type            string `json:"type"`
	AmountCents     int64  `json:"amountCents"`
	CategoryID      string `json:"categoryId"`
	TransactionDate string `json:"transactionDate"`
	Note            string `json:"note"`
}

type UpdateTransactionInput struct {
	Type            string `json:"type"`
	AmountCents     int64  `json:"amountCents"`
	CategoryID      string `json:"categoryId"`
	TransactionDate string `json:"transactionDate"`
	Note            string `json:"note"`
}

type TransactionDTO struct {
	ID              string      `json:"id"`
	Type            string      `json:"type"`
	AmountCents     int64       `json:"amountCents"`
	Category        CategoryDTO `json:"category"`
	Member          MemberDTO   `json:"member"`
	TransactionDate string      `json:"transactionDate"`
	Note            string      `json:"note"`
	CreatedAt       string      `json:"createdAt"`
	UpdatedAt       string      `json:"updatedAt"`
}

type MonthlyCategoryTotalDTO struct {
	CategoryID   string `json:"categoryId"`
	CategoryName string `json:"categoryName"`
	IconKey      string `json:"iconKey"`
	ColorKey     string `json:"colorKey"`
	ExpenseCents int64  `json:"expenseCents"`
}

type MonthlyOverviewDTO struct {
	Month          string                    `json:"month"`
	IncomeCents    int64                     `json:"incomeCents"`
	ExpenseCents   int64                     `json:"expenseCents"`
	BalanceCents   int64                     `json:"balanceCents"`
	CategoryTotals []MonthlyCategoryTotalDTO `json:"categoryTotals"`
	Recent         []TransactionDTO          `json:"recent"`
}

type MemberAdminDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Active bool   `json:"active"`
}

type CreateMemberInput struct {
	Name string `json:"name"`
	Role string `json:"role"`
	PIN  string `json:"pin"`
}

type ResetMemberPINInput struct {
	PIN string `json:"pin"`
}

type ChangeOwnPINInput struct {
	CurrentPIN string `json:"currentPin"`
	NewPIN     string `json:"newPin"`
}

type CreateCategoryInput struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	IconKey  string `json:"iconKey"`
	ColorKey string `json:"colorKey"`
}

type UpdateCategoryInput struct {
	Name      string `json:"name"`
	IconKey   string `json:"iconKey"`
	ColorKey  string `json:"colorKey"`
	SortOrder int32  `json:"sortOrder"`
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

func (s *Service) CreateTransaction(ctx context.Context, actor MemberDTO, input CreateTransactionInput) (TransactionDTO, error) {
	parsed, err := s.validateTransactionInput(ctx, input.Type, input.AmountCents, input.CategoryID, input.TransactionDate)
	if err != nil {
		return TransactionDTO{}, err
	}
	memberID, err := uuidFromString(actor.ID)
	if err != nil {
		return TransactionDTO{}, errors.New("invalid member ID")
	}
	transaction, err := s.queries.CreateTransaction(ctx, queries.CreateTransactionParams{
		Type:            parsed.category.Type,
		AmountCents:     input.AmountCents,
		CategoryID:      parsed.category.ID,
		MemberID:        memberID,
		TransactionDate: parsed.transactionDate,
		Note:            strings.TrimSpace(input.Note),
	})
	if err != nil {
		return TransactionDTO{}, err
	}
	return s.transactionDTOFromTransaction(ctx, transaction)
}

func (s *Service) ListTransactions(ctx context.Context, month string) ([]TransactionDTO, error) {
	monthRange, err := parseMonthRange(month)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListTransactionsByMonth(ctx, queries.ListTransactionsByMonthParams{
		StartDate: dateValue(monthRange.start),
		EndDate:   dateValue(monthRange.end),
	})
	if err != nil {
		return nil, err
	}
	transactions := make([]TransactionDTO, 0, len(rows))
	for _, row := range rows {
		transactions = append(transactions, transactionDTOFromRow(row))
	}
	return transactions, nil
}

func (s *Service) UpdateTransaction(ctx context.Context, actor MemberDTO, id string, input UpdateTransactionInput) (TransactionDTO, error) {
	transactionID, err := uuidFromString(id)
	if err != nil {
		return TransactionDTO{}, errors.New("invalid transaction ID")
	}
	existing, err := s.queries.GetTransactionByID(ctx, transactionID)
	if err != nil {
		return TransactionDTO{}, err
	}
	if !canManageTransaction(actor, existing.MemberID) {
		return TransactionDTO{}, errors.New("not allowed to edit transaction")
	}
	parsed, err := s.validateTransactionInput(ctx, input.Type, input.AmountCents, input.CategoryID, input.TransactionDate)
	if err != nil {
		return TransactionDTO{}, err
	}
	updated, err := s.queries.UpdateTransaction(ctx, queries.UpdateTransactionParams{
		ID:              transactionID,
		Type:            parsed.category.Type,
		AmountCents:     input.AmountCents,
		CategoryID:      parsed.category.ID,
		TransactionDate: parsed.transactionDate,
		Note:            strings.TrimSpace(input.Note),
	})
	if err != nil {
		return TransactionDTO{}, err
	}
	return s.transactionDTOFromTransaction(ctx, updated)
}

func (s *Service) DeleteTransaction(ctx context.Context, actor MemberDTO, id string) error {
	transactionID, err := uuidFromString(id)
	if err != nil {
		return errors.New("invalid transaction ID")
	}
	existing, err := s.queries.GetTransactionByID(ctx, transactionID)
	if err != nil {
		return err
	}
	if !canManageTransaction(actor, existing.MemberID) {
		return errors.New("not allowed to delete transaction")
	}
	return s.queries.DeleteTransaction(ctx, transactionID)
}

func (s *Service) MonthlyOverview(ctx context.Context, month string) (MonthlyOverviewDTO, error) {
	monthRange, err := parseMonthRange(month)
	if err != nil {
		return MonthlyOverviewDTO{}, err
	}
	totals, err := s.queries.GetMonthlyTotals(ctx, queries.GetMonthlyTotalsParams{
		StartDate: dateValue(monthRange.start),
		EndDate:   dateValue(monthRange.end),
	})
	if err != nil {
		return MonthlyOverviewDTO{}, err
	}
	categoryRows, err := s.queries.ListMonthlyExpenseCategoryTotals(ctx, queries.ListMonthlyExpenseCategoryTotalsParams{
		StartDate: dateValue(monthRange.start),
		EndDate:   dateValue(monthRange.end),
	})
	if err != nil {
		return MonthlyOverviewDTO{}, err
	}
	transactions, err := s.ListTransactions(ctx, monthRange.month)
	if err != nil {
		return MonthlyOverviewDTO{}, err
	}
	categoryTotals := make([]MonthlyCategoryTotalDTO, 0, len(categoryRows))
	for _, row := range categoryRows {
		categoryTotals = append(categoryTotals, MonthlyCategoryTotalDTO{
			CategoryID:   row.CategoryID.String(),
			CategoryName: row.CategoryName,
			IconKey:      row.CategoryIconKey,
			ColorKey:     row.CategoryColorKey,
			ExpenseCents: row.ExpenseCents,
		})
	}
	return MonthlyOverviewDTO{
		Month:          monthRange.month,
		IncomeCents:    totals.IncomeCents,
		ExpenseCents:   totals.ExpenseCents,
		BalanceCents:   totals.IncomeCents - totals.ExpenseCents,
		CategoryTotals: categoryTotals,
		Recent:         transactions,
	}, nil
}

func (s *Service) ListMembers(ctx context.Context, actor MemberDTO) ([]MemberAdminDTO, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	rows, err := s.queries.ListMembers(ctx)
	if err != nil {
		return nil, err
	}
	members := make([]MemberAdminDTO, 0, len(rows))
	for _, row := range rows {
		members = append(members, MemberAdminDTO{
			ID:     row.ID.String(),
			Name:   row.Name,
			Role:   row.Role,
			Active: row.Active,
		})
	}
	return members, nil
}

func (s *Service) CreateMember(ctx context.Context, actor MemberDTO, input CreateMemberInput) (MemberAdminDTO, error) {
	if err := requireAdmin(actor); err != nil {
		return MemberAdminDTO{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return MemberAdminDTO{}, errors.New("member name is required")
	}
	role := strings.TrimSpace(input.Role)
	if role != "admin" && role != "member" {
		return MemberAdminDTO{}, errors.New("member role must be admin or member")
	}
	pinHash, err := auth.HashPIN(input.PIN)
	if err != nil {
		return MemberAdminDTO{}, err
	}
	member, err := s.queries.CreateMember(ctx, queries.CreateMemberParams{
		Name:    name,
		PinHash: pinHash,
		Role:    role,
	})
	if err != nil {
		return MemberAdminDTO{}, err
	}
	return memberAdminDTO(member), nil
}

func (s *Service) DisableMember(ctx context.Context, actor MemberDTO, id string) (MemberAdminDTO, error) {
	if err := requireAdmin(actor); err != nil {
		return MemberAdminDTO{}, err
	}
	memberID, err := uuidFromString(id)
	if err != nil {
		return MemberAdminDTO{}, errors.New("invalid member ID")
	}
	member, err := s.queries.GetMemberByID(ctx, memberID)
	if err != nil {
		return MemberAdminDTO{}, err
	}
	if member.Role == "admin" && member.Active {
		adminCount, err := s.queries.CountActiveAdmins(ctx)
		if err != nil {
			return MemberAdminDTO{}, err
		}
		if adminCount <= 1 {
			return MemberAdminDTO{}, errors.New("cannot disable the last active admin")
		}
	}
	disabled, err := s.queries.DisableMember(ctx, memberID)
	if err != nil {
		return MemberAdminDTO{}, err
	}
	return memberAdminDTO(disabled), nil
}

func (s *Service) ResetMemberPIN(ctx context.Context, actor MemberDTO, id string, input ResetMemberPINInput) error {
	if err := requireAdmin(actor); err != nil {
		return err
	}
	memberID, err := uuidFromString(id)
	if err != nil {
		return errors.New("invalid member ID")
	}
	pinHash, err := auth.HashPIN(input.PIN)
	if err != nil {
		return err
	}
	_, err = s.queries.UpdateMemberPIN(ctx, queries.UpdateMemberPINParams{
		ID:      memberID,
		PinHash: pinHash,
	})
	return err
}

func (s *Service) ChangeOwnPIN(ctx context.Context, actor MemberDTO, input ChangeOwnPINInput) error {
	memberID, err := uuidFromString(actor.ID)
	if err != nil {
		return errors.New("invalid member ID")
	}
	member, err := s.queries.GetMemberByID(ctx, memberID)
	if err != nil {
		return err
	}
	if !member.Active || !auth.VerifyPIN(member.PinHash, input.CurrentPIN) {
		return errors.New("invalid current PIN")
	}
	pinHash, err := auth.HashPIN(input.NewPIN)
	if err != nil {
		return err
	}
	_, err = s.queries.UpdateMemberPIN(ctx, queries.UpdateMemberPINParams{
		ID:      memberID,
		PinHash: pinHash,
	})
	return err
}

func (s *Service) ListAllCategories(ctx context.Context, actor MemberDTO) ([]CategoryDTO, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	rows, err := s.queries.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	categories := make([]CategoryDTO, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, categoryDTO(row))
	}
	return categories, nil
}

func (s *Service) CreateCategory(ctx context.Context, actor MemberDTO, input CreateCategoryInput) (CategoryDTO, error) {
	if err := requireAdmin(actor); err != nil {
		return CategoryDTO{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return CategoryDTO{}, errors.New("category name is required")
	}
	categoryType := strings.TrimSpace(input.Type)
	if categoryType != "expense" && categoryType != "income" {
		return CategoryDTO{}, errors.New("category type must be expense or income")
	}
	iconKey := strings.TrimSpace(input.IconKey)
	if iconKey == "" {
		iconKey = "circle"
	}
	colorKey := strings.TrimSpace(input.ColorKey)
	if colorKey == "" {
		colorKey = "slate"
	}
	category, err := s.queries.CreateCategory(ctx, queries.CreateCategoryParams{
		Name:          name,
		Type:          categoryType,
		IconKey:       iconKey,
		ColorKey:      colorKey,
		SortOrder:     500,
		SystemDefault: false,
	})
	if err != nil {
		return CategoryDTO{}, err
	}
	return categoryDTO(category), nil
}

func (s *Service) UpdateCategory(ctx context.Context, actor MemberDTO, id string, input UpdateCategoryInput) (CategoryDTO, error) {
	if err := requireAdmin(actor); err != nil {
		return CategoryDTO{}, err
	}
	categoryID, err := uuidFromString(id)
	if err != nil {
		return CategoryDTO{}, errors.New("invalid category ID")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return CategoryDTO{}, errors.New("category name is required")
	}
	iconKey := strings.TrimSpace(input.IconKey)
	if iconKey == "" {
		iconKey = "circle"
	}
	colorKey := strings.TrimSpace(input.ColorKey)
	if colorKey == "" {
		colorKey = "slate"
	}
	category, err := s.queries.UpdateCategory(ctx, queries.UpdateCategoryParams{
		ID:        categoryID,
		Name:      name,
		IconKey:   iconKey,
		ColorKey:  colorKey,
		SortOrder: input.SortOrder,
	})
	if err != nil {
		return CategoryDTO{}, err
	}
	return categoryDTO(category), nil
}

func (s *Service) DisableCategory(ctx context.Context, actor MemberDTO, id string) (CategoryDTO, error) {
	if err := requireAdmin(actor); err != nil {
		return CategoryDTO{}, err
	}
	categoryID, err := uuidFromString(id)
	if err != nil {
		return CategoryDTO{}, errors.New("invalid category ID")
	}
	category, err := s.queries.DisableCategory(ctx, categoryID)
	if err != nil {
		return CategoryDTO{}, err
	}
	return categoryDTO(category), nil
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

func memberAdminDTO(member queries.Member) MemberAdminDTO {
	return MemberAdminDTO{
		ID:     member.ID.String(),
		Name:   member.Name,
		Role:   member.Role,
		Active: member.Active,
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

func requireAdmin(actor MemberDTO) error {
	if actor.Role != "admin" {
		return errors.New("admin permission required")
	}
	return nil
}

type parsedTransactionInput struct {
	category        queries.Category
	transactionDate pgtype.Date
}

type monthRange struct {
	month string
	start time.Time
	end   time.Time
}

func (s *Service) validateTransactionInput(ctx context.Context, transactionType string, amountCents int64, categoryID string, transactionDate string) (parsedTransactionInput, error) {
	transactionType = strings.TrimSpace(transactionType)
	if transactionType != "expense" && transactionType != "income" {
		return parsedTransactionInput{}, errors.New("transaction type must be expense or income")
	}
	if amountCents <= 0 {
		return parsedTransactionInput{}, errors.New("amount must be greater than zero")
	}
	categoryUUID, err := uuidFromString(categoryID)
	if err != nil {
		return parsedTransactionInput{}, errors.New("invalid category ID")
	}
	category, err := s.queries.GetCategoryByID(ctx, categoryUUID)
	if err != nil {
		return parsedTransactionInput{}, err
	}
	if !category.Active {
		return parsedTransactionInput{}, errors.New("category is inactive")
	}
	if category.Type != transactionType {
		return parsedTransactionInput{}, errors.New("transaction type does not match category type")
	}
	date, err := parseDate(transactionDate)
	if err != nil {
		return parsedTransactionInput{}, err
	}
	return parsedTransactionInput{
		category:        category,
		transactionDate: dateValue(date),
	}, nil
}

func (s *Service) transactionDTOFromTransaction(ctx context.Context, transaction queries.Transaction) (TransactionDTO, error) {
	category, err := s.queries.GetCategoryByID(ctx, transaction.CategoryID)
	if err != nil {
		return TransactionDTO{}, err
	}
	member, err := s.queries.GetMemberByID(ctx, transaction.MemberID)
	if err != nil {
		return TransactionDTO{}, err
	}
	return TransactionDTO{
		ID:              transaction.ID.String(),
		Type:            transaction.Type,
		AmountCents:     transaction.AmountCents,
		Category:        categoryDTO(category),
		Member:          memberDTO(member.ID, member.Name, member.Role),
		TransactionDate: formatDate(transaction.TransactionDate),
		Note:            transaction.Note,
		CreatedAt:       formatTimestamp(transaction.CreatedAt),
		UpdatedAt:       formatTimestamp(transaction.UpdatedAt),
	}, nil
}

func transactionDTOFromRow(row queries.ListTransactionsByMonthRow) TransactionDTO {
	return TransactionDTO{
		ID:          row.ID.String(),
		Type:        row.Type,
		AmountCents: row.AmountCents,
		Category: CategoryDTO{
			ID:            row.CategoryID.String(),
			Name:          row.CategoryName,
			Type:          row.CategoryType,
			IconKey:       row.CategoryIconKey,
			ColorKey:      row.CategoryColorKey,
			SortOrder:     row.CategorySortOrder,
			SystemDefault: row.CategorySystemDefault,
		},
		Member:          memberDTO(row.MemberID, row.MemberName, row.MemberRole),
		TransactionDate: formatDate(row.TransactionDate),
		Note:            row.Note,
		CreatedAt:       formatTimestamp(row.CreatedAt),
		UpdatedAt:       formatTimestamp(row.UpdatedAt),
	}
}

func canManageTransaction(actor MemberDTO, ownerID pgtype.UUID) bool {
	if actor.Role == "admin" {
		return true
	}
	actorID, err := uuidFromString(actor.ID)
	if err != nil {
		return false
	}
	return actorID == ownerID
}

func parseDate(value string) (time.Time, error) {
	date, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, errors.New("transaction date must use YYYY-MM-DD")
	}
	return date, nil
}

func parseMonthRange(value string) (monthRange, error) {
	month := strings.TrimSpace(value)
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return monthRange{}, errors.New("month must use YYYY-MM")
	}
	return monthRange{
		month: month,
		start: start,
		end:   start.AddDate(0, 1, 0),
	}, nil
}

func dateValue(value time.Time) pgtype.Date {
	return pgtype.Date{Time: value, Valid: true}
}

func formatDate(value pgtype.Date) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format("2006-01-02")
}

func formatTimestamp(value pgtype.Timestamptz) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format(time.RFC3339)
}

func uuidFromString(value string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(value); err != nil {
		return pgtype.UUID{}, err
	}
	return id, nil
}
