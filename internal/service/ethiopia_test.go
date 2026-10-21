package service

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/SukGebeya/Backend/internal/chapa"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/internal/testutil"
	"github.com/SukGebeya/Backend/pkg/password"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func requireStatus(t *testing.T, err error, status int) {
	t.Helper()
	require.Error(t, err)
	appErr, ok := domainerrors.IsAppError(err)
	require.True(t, ok, "expected an app error, got %v", err)
	require.Equal(t, status, appErr.HTTPStatus, err.Error())
}

func (h *harness) setDelivery(t *testing.T, f fixture, feeCents int64, threshold *int64) {
	t.Helper()
	merchant, err := h.merchants.GetByID(h.ctx, f.merchant.ID)
	require.NoError(t, err)
	merchant.DeliveryFeeCents = feeCents
	merchant.FreeDeliveryThresholdCents = threshold
	_, err = h.merchants.Update(h.ctx, merchant)
	require.NoError(t, err)
}

func (h *harness) shippedOrder(t *testing.T, f fixture, qty int) *domain.Order {
	t.Helper()
	order := h.paidOrder(t, f, qty)
	_, err := h.order.UpdateStatus(h.ctx, order.ID, domain.OrderStatusProcessing)
	require.NoError(t, err)
	shipped, err := h.order.MarkOrderShipped(h.ctx, MarkOrderShippedInput{OrderID: order.ID})
	require.NoError(t, err)
	return shipped
}

func TestCheckout_AddsMerchantDeliveryFee(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 10000, 20)
	threshold := int64(50000)
	h.setDelivery(t, f, 15000, &threshold)

	var escrowAmount int64
	h.blnk.CreateEscrowHoldFunc = func(ctx context.Context, req blnk.EscrowHoldRequest) (string, error) {
		escrowAmount = req.AmountCents
		return "inflight_" + req.OrderID.String(), nil
	}
	var credited blnk.MerchantCreditRequest
	h.blnk.CreditMerchantFunc = func(ctx context.Context, req blnk.MerchantCreditRequest) (string, error) {
		credited = req
		return "txn_credit_" + req.OrderID.String(), nil
	}

	order := h.paidOrder(t, f, 2)
	require.Equal(t, int64(20000), order.SubtotalCents)
	require.Equal(t, int64(15000), order.DeliveryFeeCents)
	require.Equal(t, int64(35000), order.TotalCents)
	require.Equal(t, int64(1000), order.PlatformFeeCents, "platform fee is charged on goods only")
	require.Equal(t, int64(35000), escrowAmount)

	_, err := h.order.UpdateStatus(h.ctx, order.ID, domain.OrderStatusProcessing)
	require.NoError(t, err)
	_, err = h.order.MarkOrderShipped(h.ctx, MarkOrderShippedInput{OrderID: order.ID})
	require.NoError(t, err)
	_, err = h.order.ConfirmDelivery(h.ctx, order.ID)
	require.NoError(t, err)
	require.NoError(t, h.order.SettleOrder(h.ctx, order.ID))

	require.Equal(t, int64(15000), credited.DeliveryFeeCents)
	txn, err := h.walletTxns.GetByOrderID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, int64(20000-1000+15000), txn.AmountCents, "merchant keeps the delivery fee")

	big := h.paidOrder(t, f, 5)
	require.Equal(t, int64(0), big.DeliveryFeeCents, "free delivery above the threshold")
	require.Equal(t, int64(50000), big.TotalCents)
}

func TestCheckout_RequiresVerifiedPhoneAndAddress(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 1000, 10)
	h.addToCart(t, f.buyer.ID, f.variant.ID, 1, f.variant.PriceCents)

	_, err := h.checkout.Checkout(h.ctx, CheckoutInput{UserID: f.buyer.ID})
	requireStatus(t, err, 400)

	unverified := testutil.NewUserBuilder().WithEmail("nophone@test.com").WithPhone("+251911999888").Build()
	unverified, err = h.users.Create(h.ctx, unverified)
	require.NoError(t, err)
	_, err = h.checkout.Checkout(h.ctx, CheckoutInput{UserID: unverified.ID, ShippingAddressID: &f.address.ID})
	requireStatus(t, err, 403)
}

func TestCheckout_SendsChapaLocalPhoneFormat(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 1000, 10)
	var sentPhone, firstName string
	h.chapa.InitializePaymentFunc = func(ctx context.Context, req chapa.InitializePaymentRequest) (*chapa.InitializePaymentResponse, error) {
		sentPhone, firstName = req.Phone, req.FirstName
		return &chapa.InitializePaymentResponse{Status: chapa.StatusSuccess, Data: chapa.InitializePaymentData{CheckoutURL: "https://checkout.chapa.co/x"}}, nil
	}
	h.checkoutOrder(t, f, 1)

	require.Len(t, sentPhone, 10)
	require.True(t, strings.HasPrefix(sentPhone, "09"), "Chapa needs 09xxxxxxxx, got %q", sentPhone)
	require.NotEmpty(t, firstName)
}

func TestProcessPayment_RejectsTestModeInProduction(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 1000, 10)
	h.payment.requireLiveMode = true
	order := h.checkoutOrder(t, f, 1)

	require.Error(t, h.payment.ProcessPayment(h.ctx, *order.ChapaTxRef))
	order, err := h.orders.GetByID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusPendingPayment, order.Status)
}

func TestShipping_SchedulesAutoConfirm(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 1000, 10)
	order := h.shippedOrder(t, f, 1)
	require.Equal(t, 1, h.enqueued("auto_confirm_delivery"))

	require.NoError(t, h.order.AutoConfirmDelivery(h.ctx, order.ID))
	delivered, err := h.orders.GetByID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusDelivered, delivered.Status)
	require.Equal(t, 1, h.enqueued("settle_order"))

	require.NoError(t, h.order.AutoConfirmDelivery(h.ctx, order.ID), "repeat is a no-op")
	require.Equal(t, 1, h.enqueued("settle_order"))
}

func TestDispute_BlocksAutoConfirmAndRefunds(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 5000, 10)
	order := h.shippedOrder(t, f, 2)

	_, err := h.order.OpenDispute(h.ctx, order.ID, "")
	requireStatus(t, err, 400)

	disputed, err := h.order.OpenDispute(h.ctx, order.ID, "Package never arrived")
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusDisputed, disputed.Status)
	require.NotNil(t, disputed.DisputedAt)

	require.NoError(t, h.order.AutoConfirmDelivery(h.ctx, order.ID))
	still, err := h.orders.GetByID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusDisputed, still.Status, "auto-confirm must not release a disputed order")
	requireStatus(t, h.order.SettleOrder(h.ctx, order.ID), 400)
	_, err = h.order.ConfirmDelivery(h.ctx, order.ID)
	require.Error(t, err)

	resolved, err := h.order.ResolveDispute(h.ctx, order.ID, DisputeRefund, false)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusCancelled, resolved.Status)
	require.Equal(t, 1, h.enqueued("refund_order"))

	require.NoError(t, h.payment.RefundOrder(h.ctx, order.ID))
	require.Equal(t, 1, h.chapa.CallCount("Refund"))
	require.Equal(t, 1, h.blnk.CallCount("ChargeProcessingFee"), "Chapa keeps its fee on refunds; it is booked as a cost")

	quantity, _ := h.stock(t, f.variant.ID)
	require.Equal(t, 8, quantity, "no restock when the goods stayed with the buyer")
}

func TestDispute_ReleasePaysMerchant(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 5000, 10)
	order := h.shippedOrder(t, f, 1)
	_, err := h.order.OpenDispute(h.ctx, order.ID, "Wrong size")
	require.NoError(t, err)

	released, err := h.order.ResolveDispute(h.ctx, order.ID, DisputeRelease, false)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusDelivered, released.Status)
	require.Equal(t, 1, h.enqueued("settle_order"))
	require.NoError(t, h.order.SettleOrder(h.ctx, order.ID))
	require.Equal(t, 1, h.blnk.CallCount("CreditMerchant"))

	_, err = h.order.ResolveDispute(h.ctx, order.ID, DisputeRelease, false)
	require.Error(t, err, "a dispute is resolved once")
}

func TestReversal_BeforeSettlementCancelsOrder(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 5000, 10)
	order := h.paidOrder(t, f, 3)

	require.NoError(t, h.payment.ProcessReversal(h.ctx, *order.ChapaTxRef))
	require.NoError(t, h.payment.ProcessReversal(h.ctx, *order.ChapaTxRef))

	reversed, err := h.orders.GetByID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusCancelled, reversed.Status)
	payment, err := h.payments.GetByOrderID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, domain.PaymentStatusReversed, payment.Status)
	require.Equal(t, 1, h.blnk.CallCount("VoidInflight"))
	require.Equal(t, 0, h.blnk.CallCount("DebitMerchantForReversal"))

	quantity, _ := h.stock(t, f.variant.ID)
	require.Equal(t, 10, quantity, "unshipped goods return to stock")
}

func TestReversal_AfterSettlementDebitsMerchant(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 10000, 10)
	order := h.shippedOrder(t, f, 1)
	_, err := h.order.ConfirmDelivery(h.ctx, order.ID)
	require.NoError(t, err)
	require.NoError(t, h.order.SettleOrder(h.ctx, order.ID))

	var debited int64
	h.blnk.DebitReversalFunc = func(ctx context.Context, req blnk.ReversalRequest) (string, error) {
		debited = req.AmountCents
		return "txn_reversal_" + req.OrderID.String(), nil
	}
	feeRefs := map[string]bool{}
	h.blnk.ChargeProcessingFeeFunc = func(ctx context.Context, req blnk.ProcessingFeeRequest) (string, error) {
		feeRefs[blnk.ProcessingFeeReference(req.OrderID)] = true
		return "txn_fee_" + req.OrderID.String(), nil
	}

	require.NoError(t, h.payment.ProcessReversal(h.ctx, *order.ChapaTxRef))
	require.NoError(t, h.payment.ProcessReversal(h.ctx, *order.ChapaTxRef))

	require.Equal(t, int64(10000-500), debited)
	require.Equal(t, 1, h.blnk.CallCount("DebitMerchantForReversal"))
	require.Len(t, feeRefs, 1, "every fee booking for the order uses one Blnk reference, so Blnk records it once")

	net, err := h.walletTxns.NetByMerchant(h.ctx, f.merchant.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), net)
}

func TestPayout_PolicyChecks(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 1000, 1)
	h.wallet.WithPolicy(PayoutPolicy{MinCents: 100, DailyLimitCents: 100000, RequireKYC: true})

	_, err := h.wallet.RequestPayout(h.ctx, f.merchant.ID, domain.RequestPayoutInput{
		AmountCents: 5000, Method: domain.PayoutMethodTelebirr, Destination: "12345", AccountName: "Tigist", BankCode: "855",
	})
	requireStatus(t, err, 400)

	_, err = h.wallet.RequestPayout(h.ctx, f.merchant.ID, domain.RequestPayoutInput{
		AmountCents: 5000, Method: domain.PayoutMethodTelebirr, Destination: "0911234567", AccountName: "Tigist", BankCode: "999",
	})
	requireStatus(t, err, 400)

	payout, err := h.wallet.RequestPayout(h.ctx, f.merchant.ID, domain.RequestPayoutInput{
		AmountCents: 60000, Method: domain.PayoutMethodTelebirr, Destination: "+251 91 123 4567", AccountName: "Tigist", BankCode: "855",
	})
	require.NoError(t, err)
	require.Equal(t, "0911234567", payout.Destination, "mobile money destination stored in Chapa's format")

	_, err = h.wallet.RequestPayout(h.ctx, f.merchant.ID, payoutInput(50000))
	requireStatus(t, err, 400)

	seller, err := h.users.GetByID(h.ctx, f.seller.ID)
	require.NoError(t, err)
	seller.KYCStatus = domain.KYCStatusPending
	_, err = h.users.Update(h.ctx, seller)
	require.NoError(t, err)
	_, err = h.wallet.RequestPayout(h.ctx, f.merchant.ID, payoutInput(1000))
	requireStatus(t, err, 403)
}

func TestReconciliation_DetectsDrift(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 10000, 10)
	order := h.shippedOrder(t, f, 1)
	_, err := h.order.ConfirmDelivery(h.ctx, order.ID)
	require.NoError(t, err)
	require.NoError(t, h.order.SettleOrder(h.ctx, order.ID))

	merchantBalance := int64(9500)
	h.blnk.GetBalanceFunc = func(ctx context.Context, balanceID string) (*blnk.Balance, error) {
		if balanceID == h.blnk.EscrowBalanceID() {
			return &blnk.Balance{}, nil
		}
		return &blnk.Balance{BalanceCents: merchantBalance, AvailableCents: merchantBalance}, nil
	}

	report, err := h.reconcile.Run(h.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, report.MerchantsChecked)
	require.Empty(t, report.Mismatches)

	merchantBalance = 9400
	report, err = h.reconcile.Run(h.ctx)
	require.NoError(t, err)
	require.Len(t, report.Mismatches, 1)
	require.Equal(t, int64(9400), report.Mismatches[0].LedgerCents)
	require.Equal(t, int64(9500), report.Mismatches[0].ExpectedCents)

	h.paidOrder(t, f, 2)
	merchantBalance = 9500
	report, err = h.reconcile.Run(h.ctx)
	require.NoError(t, err)
	require.Len(t, report.Mismatches, 1, "escrow holds nothing in the mock but 20000 is owed")
	require.Equal(t, "escrow", report.Mismatches[0].Account)
}

type capturedSMS struct {
	mu       sync.Mutex
	messages map[string]string
}

func (c *capturedSMS) Send(ctx context.Context, to, message string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.messages == nil {
		c.messages = map[string]string{}
	}
	c.messages[to] = message
	return nil
}

func (c *capturedSMS) codeFor(t *testing.T, to string) string {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	msg := c.messages[to]
	for i := 0; i+6 <= len(msg); i++ {
		chunk := msg[i : i+6]
		if strings.Trim(chunk, "0123456789") == "" {
			return chunk
		}
	}
	t.Fatalf("no code in message %q", msg)
	return ""
}

func newAuthHarness(t *testing.T) (*harness, *AuthService, *capturedSMS) {
	h := newHarness(t)
	sms := &capturedSMS{}
	otp := NewOTPService(repository.NewOTPRepository(h.redis), sms, []byte("test-secret"), "")
	auth := NewAuthService(h.users, repository.NewCircuitBreakerRefreshTokenRepository(h.redis), testutil.NewTestJWTManager(t),
		password.NewHasher(password.Config{Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}), time.Hour).WithOTP(otp)
	return h, auth, sms
}

func TestAuth_EthiopianPhoneNormalization(t *testing.T) {
	_, auth, _ := newAuthHarness(t)
	ctx := context.Background()

	local := "0912345678"
	result, err := auth.Register(ctx, RegisterInput{Phone: &local, Password: "Password123!"})
	require.NoError(t, err)
	require.Equal(t, "+251912345678", *result.User.Phone)

	dup := "+251 91 234 5678"
	_, err = auth.Register(ctx, RegisterInput{Phone: &dup, Password: "Password123!"})
	require.Error(t, err, "the same number in another format is a duplicate")

	for _, identifier := range []string{"0912345678", "912345678", "+251912345678", "251912345678"} {
		_, err := auth.Login(ctx, LoginInput{Identifier: identifier, Password: "Password123!"})
		require.NoError(t, err, identifier)
	}

	foreign := "+14155550100"
	_, err = auth.Register(ctx, RegisterInput{Phone: &foreign, Password: "Password123!"})
	require.Error(t, err)

	email := "  Abebe@Example.com "
	_, err = auth.Register(ctx, RegisterInput{Email: &email, Password: "Password123!"})
	require.NoError(t, err)
	_, err = auth.Login(ctx, LoginInput{Identifier: "abebe@example.COM", Password: "Password123!"})
	require.NoError(t, err)
}

func TestAuth_DeactivatedUserCannotLogIn(t *testing.T) {
	h, auth, _ := newAuthHarness(t)
	ctx := context.Background()
	p := "0911111222"
	result, err := auth.Register(ctx, RegisterInput{Phone: &p, Password: "Password123!"})
	require.NoError(t, err)

	user, err := h.users.GetByID(ctx, result.User.ID)
	require.NoError(t, err)
	user.IsActive = false
	_, err = h.users.Update(ctx, user)
	require.NoError(t, err)

	_, err = auth.Login(ctx, LoginInput{Identifier: p, Password: "Password123!"})
	requireStatus(t, err, 403)
	_, err = auth.Refresh(ctx, result.Tokens.RefreshToken)
	require.Error(t, err)
}

func TestAuth_PhoneVerificationAndReset(t *testing.T) {
	h, auth, sms := newAuthHarness(t)
	ctx := context.Background()
	p := "0911333444"
	result, err := auth.Register(ctx, RegisterInput{Phone: &p, Password: "Password123!"})
	require.NoError(t, err)

	require.NoError(t, auth.SendPhoneVerification(ctx, result.User.ID))
	requireStatus(t, auth.SendPhoneVerification(ctx, result.User.ID), 429)

	_, err = auth.VerifyPhone(ctx, result.User.ID, "000000")
	requireStatus(t, err, 400)

	verified, err := auth.VerifyPhone(ctx, result.User.ID, sms.codeFor(t, "+251911333444"))
	require.NoError(t, err)
	require.NotNil(t, verified.PhoneVerifiedAt)

	require.NoError(t, h.redis.FlushAll(ctx).Err())
	require.NoError(t, auth.RequestPasswordReset(ctx, "+251911333444"))
	require.NoError(t, auth.RequestPasswordReset(ctx, "0911000999"), "unknown numbers look identical")

	code := sms.codeFor(t, "+251911333444")
	require.NoError(t, auth.ResetPassword(ctx, "911333444", code, "NewPassword456!"))
	requireStatus(t, auth.ResetPassword(ctx, "911333444", code, "Again789!"), 400)

	_, err = auth.Login(ctx, LoginInput{Identifier: p, Password: "Password123!"})
	require.Error(t, err)
	_, err = auth.Login(ctx, LoginInput{Identifier: p, Password: "NewPassword456!"})
	require.NoError(t, err)
	_, err = auth.Refresh(ctx, result.Tokens.RefreshToken)
	require.Error(t, err, "a password reset signs out every session")
}

func TestOTP_LocksAfterTooManyAttempts(t *testing.T) {
	h := newHarness(t)
	sms := &capturedSMS{}
	otp := NewOTPService(repository.NewOTPRepository(h.redis), sms, []byte("s"), "")
	ctx := context.Background()

	require.NoError(t, otp.Send(ctx, "0911555666", OTPPurposeVerifyPhone))
	code := sms.codeFor(t, "+251911555666")
	for i := 0; i < otpMaxAttempts; i++ {
		require.Error(t, otp.Verify(ctx, "0911555666", OTPPurposeVerifyPhone, "999999"))
	}
	require.Error(t, otp.Verify(ctx, "0911555666", OTPPurposeVerifyPhone, code), "the correct code is useless once locked")
	require.Error(t, otp.Verify(ctx, "0911555666", OTPPurposeResetPassword, code), "codes are bound to their purpose")
}

func TestMerchantOnboarding_RequiresVerifiedPhoneAndPromotesRole(t *testing.T) {
	h := newHarness(t)
	svc := NewMerchantService(h.merchants, h.users, h.blnk, true)

	user, err := h.users.Create(h.ctx, testutil.NewUserBuilder().WithEmail("shop@test.com").WithPhone("+251911777888").Build())
	require.NoError(t, err)

	input := OnboardInput{UserID: user.ID, ShopName: "ጣና ሱቅ", ContactPhone: "0911777888", City: "Bahir Dar"}
	_, err = svc.Onboard(h.ctx, input)
	requireStatus(t, err, 403)

	require.NoError(t, h.users.MarkPhoneVerified(h.ctx, user.ID))
	merchant, err := svc.Onboard(h.ctx, input)
	require.NoError(t, err)
	require.Equal(t, "+251911777888", merchant.ContactPhone)
	require.NotEmpty(t, merchant.ShopSlug, "Amharic shop names still get a slug")

	promoted, err := h.users.GetByID(h.ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, domain.RoleMerchant, promoted.Role)

	merchant.BlnkBalanceID = nil
	again, err := svc.ProvisionWallet(h.ctx, merchant.ID)
	require.NoError(t, err)
	require.NotNil(t, again.BlnkBalanceID)
	require.Equal(t, 1, h.blnk.CallCount("CreateMerchantWallet"), "wallet already existed, so provisioning is a no-op")
}

func TestBrowseProducts_SearchSummaries(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 25000, 4)
	coffee, err := h.products.Create(h.ctx, testutil.NewProductBuilder().WithMerchantID(f.merchant.ID).WithName("ይርጋጨፌ ቡና 50%").Build())
	require.NoError(t, err)
	v, err := h.variants.Create(h.ctx, testutil.NewVariantBuilder().WithProductID(coffee.ID).WithSKU("SKU-"+uuid.NewString()[:8]).WithPrice(9000).Build())
	require.NoError(t, err)
	_, err = h.inventory.Create(h.ctx, &domain.Inventory{ID: uuid.New(), VariantID: v.ID, Quantity: 0})
	require.NoError(t, err)

	all, err := h.products.Browse(h.ctx, domain.ProductFilter{}, domain.ListOptions{Page: 1, PerPage: 1})
	require.NoError(t, err)
	require.Equal(t, int64(2), all.Total, "total counts every match, not just the page")
	require.Len(t, all.Items, 1)
	require.Equal(t, 2, all.TotalPages)

	found, err := h.products.Browse(h.ctx, domain.ProductFilter{Search: "ቡና"}, domain.ListOptions{Page: 1, PerPage: 20})
	require.NoError(t, err)
	require.Len(t, found.Items, 1)
	require.Equal(t, int64(9000), found.Items[0].MinPriceCents)
	require.Equal(t, int64(0), found.Items[0].AvailableQuantity)
	require.True(t, found.Items[0].MerchantVerified)

	literal, err := h.products.Browse(h.ctx, domain.ProductFilter{Search: "50%"}, domain.ListOptions{Page: 1, PerPage: 20})
	require.NoError(t, err)
	require.Len(t, literal.Items, 1)
	none, err := h.products.Browse(h.ctx, domain.ProductFilter{Search: "%"}, domain.ListOptions{Page: 1, PerPage: 20})
	require.NoError(t, err)
	require.Len(t, none.Items, 1, "a bare % is matched literally, not as a wildcard")

	kemis, err := h.products.Browse(h.ctx, domain.ProductFilter{Search: "kemis", MerchantID: &f.merchant.ID}, domain.ListOptions{Page: 1, PerPage: 20})
	require.NoError(t, err)
	require.Len(t, kemis.Items, 1)
	require.Equal(t, int64(4), kemis.Items[0].AvailableQuantity)
}
