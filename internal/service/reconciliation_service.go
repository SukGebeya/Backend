package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/SukGebeya/Backend/internal/jobs"
	"github.com/SukGebeya/Backend/internal/repository"
)

var _ jobs.Reconciler = (*ReconciliationService)(nil)

type ReconciliationMismatch struct {
	Account       string `json:"account"`
	BalanceID     string `json:"balance_id"`
	LedgerCents   int64  `json:"ledger_cents"`
	ExpectedCents int64  `json:"expected_cents"`
}

type ReconciliationReport struct {
	CheckedAt        time.Time                `json:"checked_at"`
	MerchantsChecked int                      `json:"merchants_checked"`
	Mismatches       []ReconciliationMismatch `json:"mismatches"`
}

// ReconciliationService cross-checks Blnk, the financial source of truth, against
// the order and wallet records in Postgres so drift is caught the same day.
type ReconciliationService struct {
	merchantRepo repository.MerchantRepository
	walletRepo   repository.WalletTransactionRepository
	orderRepo    repository.OrderRepository
	blnkClient   blnk.BlnkClient
}

func NewReconciliationService(merchantRepo repository.MerchantRepository, walletRepo repository.WalletTransactionRepository, orderRepo repository.OrderRepository, blnkClient blnk.BlnkClient) *ReconciliationService {
	return &ReconciliationService{merchantRepo: merchantRepo, walletRepo: walletRepo, orderRepo: orderRepo, blnkClient: blnkClient}
}

func (s *ReconciliationService) Reconcile(ctx context.Context) error {
	_, err := s.Run(ctx)
	return err
}

// Run compares every merchant wallet with the sum of its recorded wallet
// transactions, and the escrow balance (settled plus held) with the value of paid
// orders that have not been settled. Brief mismatches can appear while a refund or
// settlement job is mid-flight; persistent ones need investigation.
func (s *ReconciliationService) Run(ctx context.Context) (*ReconciliationReport, error) {
	report := &ReconciliationReport{CheckedAt: time.Now(), Mismatches: []ReconciliationMismatch{}}

	wallets, err := s.merchantRepo.ListWallets(ctx)
	if err != nil {
		return nil, err
	}

	for merchantID, balanceID := range wallets {
		balance, err := s.blnkClient.GetBalance(ctx, balanceID)
		if err != nil {
			return nil, err
		}
		expected, err := s.walletRepo.NetByMerchant(ctx, merchantID)
		if err != nil {
			return nil, err
		}
		report.MerchantsChecked++
		if balance.BalanceCents != expected {
			report.Mismatches = append(report.Mismatches, ReconciliationMismatch{
				Account:       "merchant:" + merchantID.String(),
				BalanceID:     balanceID,
				LedgerCents:   balance.BalanceCents,
				ExpectedCents: expected,
			})
		}
	}

	if escrowID := s.blnkClient.EscrowBalanceID(); escrowID != "" {
		escrow, err := s.blnkClient.GetBalance(ctx, escrowID)
		if err != nil {
			return nil, err
		}
		expected, err := s.orderRepo.SumUnsettledEscrow(ctx)
		if err != nil {
			return nil, err
		}
		held := escrow.BalanceCents + escrow.InflightCreditCents
		if held != expected {
			report.Mismatches = append(report.Mismatches, ReconciliationMismatch{
				Account:       "escrow",
				BalanceID:     escrowID,
				LedgerCents:   held,
				ExpectedCents: expected,
			})
		}
	}

	for _, m := range report.Mismatches {
		slog.Error("RECONCILIATION MISMATCH", "account", m.Account, "balance_id", m.BalanceID, "ledger_cents", m.LedgerCents, "expected_cents", m.ExpectedCents)
	}
	slog.Info("ledger reconciliation finished", "merchants", report.MerchantsChecked, "mismatches", len(report.Mismatches))
	return report, nil
}
