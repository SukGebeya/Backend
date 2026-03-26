package service

import (
	"context"
	"log/slog"

	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/google/uuid"
)

type WalletService struct {
	merchantRepo repository.MerchantRepository
	walletRepo   repository.WalletTransactionRepository
	payoutRepo   repository.PayoutRepository
	blnkClient   blnk.BlnkClient
}

var _ WalletManager = (*WalletService)(nil)

func NewWalletService(
	merchantRepo repository.MerchantRepository,
	walletRepo repository.WalletTransactionRepository,
	payoutRepo repository.PayoutRepository,
	blnkClient blnk.BlnkClient,
) *WalletService {
	return &WalletService{
		merchantRepo: merchantRepo,
		walletRepo:   walletRepo,
		payoutRepo:   payoutRepo,
		blnkClient:   blnkClient,
	}
}

func (s *WalletService) GetBalance(ctx context.Context, merchantID uuid.UUID) (*WalletBalanceResponse, error) {
	merchant, err := s.merchantRepo.GetByID(ctx, merchantID)
	if err != nil {
		return nil, err
	}

	if merchant.BlnkBalanceID == nil || *merchant.BlnkBalanceID == "" {
		slog.Debug("GetBalance: merchant has no Blnk wallet", "merchant_id", merchantID)
		return &WalletBalanceResponse{
			BalanceCents: 0,
			Currency:     "ETB",
		}, nil
	}

	balanceCents, err := s.blnkClient.GetBalance(ctx, *merchant.BlnkBalanceID)
	if err != nil {
		slog.Error("GetBalance: failed to get balance from Blnk",
			"error", err,
			"merchant_id", merchantID,
			"blnk_balance_id", *merchant.BlnkBalanceID,
		)
		return nil, domainerrors.Internal("failed to get wallet balance", err)
	}

	return &WalletBalanceResponse{
		BalanceCents: balanceCents,
		Currency:     "ETB",
	}, nil
}

func (s *WalletService) GetTransactionHistory(ctx context.Context, merchantID uuid.UUID, page, pageSize int) (*domain.PaginatedResult[domain.WalletTransaction], error) {
	transactions, err := s.walletRepo.ListByMerchantID(ctx, merchantID, page, pageSize)
	if err != nil {
		return nil, err
	}

	total, err := s.walletRepo.CountByMerchantID(ctx, merchantID)
	if err != nil {
		return nil, err
	}

	return &domain.PaginatedResult[domain.WalletTransaction]{
		Items:      transactions,
		Total:      total,
		Page:       page,
		PerPage:    pageSize,
		TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
	}, nil
}

func (s *WalletService) RequestPayout(ctx context.Context, merchantID uuid.UUID, input domain.RequestPayoutInput) (*domain.Payout, error) {
	merchant, err := s.merchantRepo.GetByID(ctx, merchantID)
	if err != nil {
		return nil, err
	}

	if merchant.BlnkBalanceID == nil || *merchant.BlnkBalanceID == "" {
		return nil, domainerrors.BadRequest("merchant does not have a wallet")
	}

	balanceCents, err := s.blnkClient.GetBalance(ctx, *merchant.BlnkBalanceID)
	if err != nil {
		return nil, domainerrors.Internal("failed to verify balance", err)
	}

	if balanceCents < input.AmountCents {
		return nil, domainerrors.BadRequest("insufficient balance for payout")
	}

	if input.AmountCents < 100 {
		return nil, domainerrors.BadRequest("minimum payout amount is 1 ETB (100 cents)")
	}

	payout := &domain.Payout{
		ID:          uuid.New(),
		MerchantID:  merchantID,
		AmountCents: input.AmountCents,
		Currency:    "ETB",
		Method:      input.Method,
		Destination: input.Destination,
		Status:      domain.PayoutStatusPending,
	}

	created, err := s.payoutRepo.Create(ctx, payout)
	if err != nil {
		return nil, err
	}

	slog.Info("RequestPayout: payout created",
		"payout_id", created.ID,
		"merchant_id", merchantID,
		"amount_cents", input.AmountCents,
		"method", input.Method,
	)

	return created, nil
}

func (s *WalletService) ListPayoutsByMerchant(ctx context.Context, merchantID uuid.UUID, page, pageSize int) (*domain.PaginatedResult[domain.Payout], error) {
	payouts, err := s.payoutRepo.ListByMerchantID(ctx, merchantID, page, pageSize)
	if err != nil {
		return nil, err
	}

	total, err := s.payoutRepo.CountByMerchantID(ctx, merchantID)
	if err != nil {
		return nil, err
	}

	return &domain.PaginatedResult[domain.Payout]{
		Items:      payouts,
		Total:      total,
		Page:       page,
		PerPage:    pageSize,
		TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
	}, nil
}
