package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/SukGebeya/Backend/internal/chapa"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/jobs"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/pkg/phone"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const MinPayoutCents = 100

// PayoutPolicy holds the risk limits applied to merchant withdrawals.
type PayoutPolicy struct {
	MinCents        int64
	DailyLimitCents int64
	Location        *time.Location
	RequireKYC      bool
}

func DefaultPayoutPolicy() PayoutPolicy {
	loc, err := time.LoadLocation("Africa/Addis_Ababa")
	if err != nil {
		loc = time.FixedZone("EAT", 3*60*60)
	}
	return PayoutPolicy{MinCents: 10000, DailyLimitCents: 10000000, Location: loc, RequireKYC: true}
}

type WalletService struct {
	merchantRepo repository.MerchantRepository
	userRepo     repository.UserRepository
	walletRepo   repository.WalletTransactionRepository
	payoutRepo   repository.PayoutRepository
	blnkClient   blnk.BlnkClient
	chapaClient  chapa.ChapaClient
	txManager    repository.TransactionManager
	enqueuer     jobs.Enqueuer
	policy       PayoutPolicy

	banksMu      sync.Mutex
	banks        []chapa.Bank
	banksFetched time.Time
}

func (s *WalletService) WithPolicy(policy PayoutPolicy) *WalletService {
	if policy.Location == nil {
		policy.Location = DefaultPayoutPolicy().Location
	}
	s.policy = policy
	return s
}

var _ WalletManager = (*WalletService)(nil)
var _ jobs.PayoutProcessor = (*WalletService)(nil)

func NewWalletService(
	merchantRepo repository.MerchantRepository,
	userRepo repository.UserRepository,
	walletRepo repository.WalletTransactionRepository,
	payoutRepo repository.PayoutRepository,
	blnkClient blnk.BlnkClient,
	chapaClient chapa.ChapaClient,
	txManager repository.TransactionManager,
	enqueuer jobs.Enqueuer,
) *WalletService {
	return &WalletService{
		merchantRepo: merchantRepo,
		userRepo:     userRepo,
		walletRepo:   walletRepo,
		payoutRepo:   payoutRepo,
		blnkClient:   blnkClient,
		chapaClient:  chapaClient,
		txManager:    txManager,
		enqueuer:     enqueuer,
		policy:       DefaultPayoutPolicy(),
	}
}

func (s *WalletService) GetBalance(ctx context.Context, merchantID uuid.UUID) (*WalletBalanceResponse, error) {
	merchant, err := s.merchantRepo.GetByID(ctx, merchantID)
	if err != nil {
		return nil, err
	}

	if merchant.BlnkBalanceID == nil || *merchant.BlnkBalanceID == "" {
		return &WalletBalanceResponse{Currency: "ETB"}, nil
	}

	balance, err := s.blnkClient.GetBalance(ctx, *merchant.BlnkBalanceID)
	if err != nil {
		return nil, domainerrors.Internal("failed to get wallet balance", err)
	}

	return &WalletBalanceResponse{
		BalanceCents:       balance.BalanceCents,
		PendingPayoutCents: balance.InflightDebitCents,
		AvailableCents:     balance.AvailableCents,
		Currency:           "ETB",
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

	return paginate(transactions, total, page, pageSize), nil
}

// RequestPayout places an inflight hold on the merchant's Blnk balance before the
// payout row is written, so concurrent requests cannot overdraw the wallet.
func (s *WalletService) RequestPayout(ctx context.Context, merchantID uuid.UUID, input domain.RequestPayoutInput) (*domain.Payout, error) {
	minCents := max(s.policy.MinCents, MinPayoutCents)
	if input.AmountCents < minCents {
		return nil, domainerrors.BadRequest(fmt.Sprintf("minimum payout amount is %s ETB", chapa.FormatAmount(minCents)))
	}

	merchant, err := s.merchantRepo.GetByID(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	if merchant.BlnkBalanceID == nil || *merchant.BlnkBalanceID == "" {
		return nil, domainerrors.BadRequest("merchant does not have a wallet")
	}

	if s.policy.RequireKYC {
		if !merchant.IsVerified {
			return nil, domainerrors.Forbidden("your shop must be verified before you can withdraw")
		}
		owner, err := s.userRepo.GetByID(ctx, merchant.UserID)
		if err != nil {
			return nil, err
		}
		if owner.KYCStatus != domain.KYCStatusVerified {
			return nil, domainerrors.Forbidden("complete identity verification (KYC) before you can withdraw")
		}
	}

	destination, err := normalizePayoutDestination(input.Method, input.Destination)
	if err != nil {
		return nil, err
	}
	input.Destination = destination

	if err := s.checkBankCode(ctx, input.BankCode); err != nil {
		return nil, err
	}

	if s.policy.DailyLimitCents > 0 {
		now := time.Now().In(s.policy.Location)
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.policy.Location)
		usedToday, err := s.payoutRepo.SumSince(ctx, merchantID, startOfDay)
		if err != nil {
			return nil, err
		}
		if usedToday+input.AmountCents > s.policy.DailyLimitCents {
			return nil, domainerrors.BadRequest(fmt.Sprintf("daily withdrawal limit of %s ETB reached; %s ETB remaining today",
				chapa.FormatAmount(s.policy.DailyLimitCents), chapa.FormatAmount(max(s.policy.DailyLimitCents-usedToday, 0))))
		}
	}

	balance, err := s.blnkClient.GetBalance(ctx, *merchant.BlnkBalanceID)
	if err != nil {
		return nil, domainerrors.Internal("failed to verify balance", err)
	}
	if balance.AvailableCents < input.AmountCents {
		return nil, domainerrors.BadRequest("insufficient balance for payout")
	}

	payoutID := uuid.New()
	holdTxnID, err := s.blnkClient.HoldPayout(ctx, blnk.PayoutRequest{
		PayoutID:       payoutID,
		MerchantID:     merchantID,
		AmountCents:    input.AmountCents,
		Method:         string(input.Method),
		Destination:    input.Destination,
		MerchantBlnkID: *merchant.BlnkBalanceID,
	})
	if err != nil {
		if blnk.IsInsufficientFunds(err) {
			return nil, domainerrors.BadRequest("insufficient balance for payout")
		}
		return nil, domainerrors.Internal("failed to hold payout funds", err)
	}

	accountName := strings.TrimSpace(input.AccountName)
	bankCode := strings.TrimSpace(input.BankCode)
	created, err := s.payoutRepo.Create(ctx, &domain.Payout{
		ID:                payoutID,
		MerchantID:        merchantID,
		AmountCents:       input.AmountCents,
		Currency:          "ETB",
		Method:            input.Method,
		Destination:       strings.TrimSpace(input.Destination),
		AccountName:       &accountName,
		BankCode:          &bankCode,
		Status:            domain.PayoutStatusPending,
		BlnkTransactionID: &holdTxnID,
	})
	if err != nil {
		if voidErr := s.blnkClient.VoidInflight(ctx, holdTxnID); voidErr != nil {
			slog.Error("CRITICAL: payout hold left open after failed insert", "payout_id", payoutID, "hold_txn_id", holdTxnID, "error", voidErr)
		}
		return nil, err
	}

	slog.Info("payout requested", "payout_id", created.ID, "merchant_id", merchantID, "amount_cents", input.AmountCents)
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

	return paginate(payouts, total, page, pageSize), nil
}

func (s *WalletService) GetPayout(ctx context.Context, merchantID, payoutID uuid.UUID) (*domain.Payout, error) {
	payout, err := s.payoutRepo.GetByID(ctx, payoutID)
	if err != nil {
		return nil, err
	}
	if payout.MerchantID != merchantID {
		return nil, domainerrors.NotFound("payout not found")
	}
	return payout, nil
}

// ListBanks returns Chapa's transfer destinations, cached for an hour.
func (s *WalletService) ListBanks(ctx context.Context) ([]chapa.Bank, error) {
	s.banksMu.Lock()
	defer s.banksMu.Unlock()
	if s.banks != nil && time.Since(s.banksFetched) < time.Hour {
		return s.banks, nil
	}
	banks, err := s.chapaClient.ListBanks(ctx)
	if err != nil {
		if s.banks != nil {
			return s.banks, nil
		}
		return nil, domainerrors.Internal("failed to list banks", err)
	}
	s.banks, s.banksFetched = banks, time.Now()
	return banks, nil
}

func (s *WalletService) checkBankCode(ctx context.Context, code string) error {
	banks, err := s.ListBanks(ctx)
	if err != nil {
		return err
	}
	for _, b := range banks {
		if b.ID.String() == strings.TrimSpace(code) {
			return nil
		}
	}
	return domainerrors.BadRequest("unknown bank_code; use one from GET /merchant/wallet/banks")
}

// normalizePayoutDestination validates the account for the chosen method. Mobile
// money accounts are phone numbers, which Chapa expects in 09…/07… form.
func normalizePayoutDestination(method domain.PayoutMethod, destination string) (string, error) {
	destination = strings.TrimSpace(destination)
	switch method {
	case domain.PayoutMethodTelebirr, domain.PayoutMethodCBEBirr:
		local, err := phone.Local(destination)
		if err != nil {
			return "", domainerrors.BadRequest("destination must be the Ethiopian mobile number registered with " + string(method))
		}
		return local, nil
	case domain.PayoutMethodBankTransfer:
		digits := strings.ReplaceAll(strings.ReplaceAll(destination, " ", ""), "-", "")
		if len(digits) < 8 || len(digits) > 16 {
			return "", domainerrors.BadRequest("bank account number must be 8 to 16 digits")
		}
		for _, r := range digits {
			if r < '0' || r > '9' {
				return "", domainerrors.BadRequest("bank account number must contain only digits")
			}
		}
		return digits, nil
	default:
		return "", domainerrors.BadRequest("unsupported payout method")
	}
}

func (s *WalletService) ListPayouts(ctx context.Context, status *domain.PayoutStatus, page, pageSize int) (*domain.PaginatedResult[domain.Payout], error) {
	if status != nil {
		payouts, err := s.payoutRepo.ListByStatus(ctx, *status, page, pageSize)
		if err != nil {
			return nil, err
		}
		total, err := s.payoutRepo.CountByStatus(ctx, *status)
		if err != nil {
			return nil, err
		}
		return paginate(payouts, total, page, pageSize), nil
	}

	payouts, err := s.payoutRepo.ListAll(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &domain.PaginatedResult[domain.Payout]{Items: payouts, Page: page, PerPage: pageSize}, nil
}

// ApprovePayout is the admin action that starts the transfer. Claiming the payout
// and enqueueing the job share a transaction, so a payout is processed at most once.
func (s *WalletService) ApprovePayout(ctx context.Context, payoutID uuid.UUID) (*domain.Payout, error) {
	var payout *domain.Payout
	err := s.txManager.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		payout, err = s.payoutRepo.WithTx(tx).ClaimForProcessing(ctx, payoutID)
		if err != nil {
			return err
		}
		return s.enqueuer.EnqueueTx(ctx, tx, jobs.ProcessPayoutArgs{PayoutID: payoutID}, nil)
	})
	if err != nil {
		return nil, err
	}
	return payout, nil
}

func (s *WalletService) RejectPayout(ctx context.Context, payoutID uuid.UUID, reason string) (*domain.Payout, error) {
	payout, err := s.payoutRepo.GetByID(ctx, payoutID)
	if err != nil {
		return nil, err
	}
	if payout.Status != domain.PayoutStatusPending {
		return nil, domainerrors.BadRequest("only pending payouts can be rejected")
	}
	if reason == "" {
		reason = "Rejected by admin"
	}
	return s.failPayout(ctx, payout, reason)
}

// ProcessPayout drives a claimed payout to completion. The transfer reference is
// stored before Chapa is called, so a retry verifies the existing transfer rather
// than sending money twice.
func (s *WalletService) ProcessPayout(ctx context.Context, payoutID uuid.UUID) error {
	payout, err := s.payoutRepo.GetByID(ctx, payoutID)
	if err != nil {
		return err
	}
	if payout.Status != domain.PayoutStatusProcessing {
		return nil
	}
	if payout.BlnkTransactionID == nil || *payout.BlnkTransactionID == "" {
		_, err := s.failPayout(ctx, payout, "payout has no ledger hold")
		return err
	}

	if payout.TransferReference == nil {
		reference := "GEBEYA-PO-" + payout.ID.String()
		if payout, err = s.payoutRepo.UpdateTransfer(ctx, payout.ID, reference, nil); err != nil {
			return err
		}
		return s.initiateTransfer(ctx, payout)
	}

	verify, err := s.chapaClient.VerifyTransfer(ctx, *payout.TransferReference)
	if err != nil {
		var apiErr *chapa.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return s.initiateTransfer(ctx, payout)
		}
		return err
	}

	raw, _ := json.Marshal(verify)
	switch strings.ToLower(verify.Data.Status) {
	case chapa.StatusSuccess:
		return s.completePayout(ctx, payout, raw)
	case chapa.StatusFailed, "cancelled", "reverted":
		_, err := s.failPayout(ctx, payout, "transfer failed at provider")
		return err
	default:
		return jobs.ErrNotReady
	}
}

func (s *WalletService) initiateTransfer(ctx context.Context, payout *domain.Payout) error {
	accountName, bankCode := "", ""
	if payout.AccountName != nil {
		accountName = *payout.AccountName
	}
	if payout.BankCode != nil {
		bankCode = *payout.BankCode
	}
	if accountName == "" || bankCode == "" {
		_, err := s.failPayout(ctx, payout, "payout is missing account name or bank code")
		return err
	}

	resp, err := s.chapaClient.InitiateTransfer(ctx, chapa.TransferRequest{
		AccountName:   accountName,
		AccountNumber: payout.Destination,
		AmountCents:   payout.AmountCents,
		Currency:      payout.Currency,
		Reference:     *payout.TransferReference,
		BankCode:      bankCode,
	})
	if err != nil {
		var apiErr *chapa.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
			_, failErr := s.failPayout(ctx, payout, fmt.Sprintf("transfer rejected: %s", apiErr.Body))
			return failErr
		}
		return err
	}

	raw, _ := json.Marshal(resp)
	if _, err := s.payoutRepo.UpdateTransfer(ctx, payout.ID, *payout.TransferReference, raw); err != nil {
		return err
	}

	slog.Info("payout transfer initiated", "payout_id", payout.ID, "reference", *payout.TransferReference)
	return jobs.ErrNotReady
}

func (s *WalletService) completePayout(ctx context.Context, payout *domain.Payout, raw []byte) error {
	if err := s.blnkClient.CommitInflight(ctx, *payout.BlnkTransactionID); err != nil {
		return err
	}

	if _, err := s.payoutRepo.UpdateTransfer(ctx, payout.ID, *payout.TransferReference, raw); err != nil {
		return err
	}

	return s.txManager.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := s.payoutRepo.WithTx(tx).UpdateStatus(ctx, payout.ID, domain.PayoutStatusCompleted, nil, nil); err != nil {
			return err
		}
		description := fmt.Sprintf("Payout to %s via %s", payout.Destination, payout.Method)
		_, err := s.walletRepo.WithTx(tx).Create(ctx, &domain.WalletTransaction{
			ID:                uuid.New(),
			MerchantID:        payout.MerchantID,
			Type:              domain.WalletTransactionTypePayoutDebit,
			AmountCents:       payout.AmountCents,
			Currency:          payout.Currency,
			BlnkTransactionID: *payout.BlnkTransactionID,
			Status:            domain.WalletTransactionStatusApplied,
			Description:       &description,
		})
		if err == nil {
			slog.Info("payout completed", "payout_id", payout.ID, "amount_cents", payout.AmountCents)
		}
		return err
	})
}

func (s *WalletService) failPayout(ctx context.Context, payout *domain.Payout, reason string) (*domain.Payout, error) {
	if payout.BlnkTransactionID != nil && *payout.BlnkTransactionID != "" {
		if err := s.blnkClient.VoidInflight(ctx, *payout.BlnkTransactionID); err != nil {
			return nil, err
		}
	}
	updated, err := s.payoutRepo.UpdateStatus(ctx, payout.ID, domain.PayoutStatusFailed, nil, &reason)
	if err != nil {
		return nil, err
	}
	slog.Warn("payout failed", "payout_id", payout.ID, "reason", reason)
	return updated, nil
}
