package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/SukGebeya/Backend/internal/chapa"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/jobs"
	"github.com/SukGebeya/Backend/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCheckout_TransfersCartReservationWithoutDoubleCounting(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 125050, 10)

	cart := h.addToCart(t, f.buyer.ID, f.variant.ID, 3, f.variant.PriceCents)
	_, reserved := h.stock(t, f.variant.ID)
	require.Equal(t, 3, reserved)

	result, err := h.checkout.Checkout(h.ctx, CheckoutInput{UserID: f.buyer.ID, ShippingAddressID: &f.address.ID})
	require.NoError(t, err)
	require.Equal(t, int64(3*125050), result.TotalCents)
	require.True(t, strings.HasPrefix(result.ChapaTxRef, "GB-"+strings.ReplaceAll(result.OrderID.String(), "-", "")+"-"))
	require.LessOrEqual(t, len(result.ChapaTxRef), 50)

	quantity, reserved := h.stock(t, f.variant.ID)
	require.Equal(t, 10, quantity)
	require.Equal(t, 3, reserved, "checkout must not reserve the same stock a second time")

	items, err := h.cartItems.ListByCart(h.ctx, cart.ID)
	require.NoError(t, err)
	require.Empty(t, items)

	order, err := h.orders.GetByID(h.ctx, result.OrderID)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusPendingPayment, order.Status)
	require.Equal(t, int64(3*125050*5/100), order.PlatformFeeCents)

	orderItems, err := h.orders.ListItemsByOrderID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Len(t, orderItems, 1)
	require.Equal(t, "Habesha Kemis", orderItems[0].ProductName)
	require.Equal(t, f.variant.Title, orderItems[0].VariantTitle)

	require.Equal(t, 1, h.enqueued("expire_unpaid_order"))
	require.Equal(t, int64(3*125050), h.chapaPayments[result.ChapaTxRef])
}

func TestCheckout_ChapaFailureKeepsCart(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 1000, 10)
	cart := h.addToCart(t, f.buyer.ID, f.variant.ID, 2, f.variant.PriceCents)

	h.chapa.InitializePaymentFunc = func(ctx context.Context, req chapa.InitializePaymentRequest) (*chapa.InitializePaymentResponse, error) {
		return nil, errors.New("chapa down")
	}

	_, err := h.checkout.Checkout(h.ctx, CheckoutInput{UserID: f.buyer.ID, ShippingAddressID: &f.address.ID})
	require.Error(t, err)

	items, err := h.cartItems.ListByCart(h.ctx, cart.ID)
	require.NoError(t, err)
	require.Len(t, items, 1, "cart must survive a failed payment start")

	_, reserved := h.stock(t, f.variant.ID)
	require.Equal(t, 2, reserved)

	orders, err := h.orders.ListByBuyerID(h.ctx, f.buyer.ID, 1, 10)
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, domain.OrderStatusCancelled, orders[0].Status)
}

func TestCheckout_RejectsAnotherUsersAddress(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 1000, 10)
	h.addToCart(t, f.buyer.ID, f.variant.ID, 1, f.variant.PriceCents)

	stranger := testutil.NewUserBuilder().WithEmail("stranger@test.com").WithPhone("+251900000001").Build()
	stranger, err := h.users.Create(h.ctx, stranger)
	require.NoError(t, err)
	address, err := h.addresses.Create(h.ctx, testutil.NewAddressBuilder().WithUserID(stranger.ID).Build())
	require.NoError(t, err)

	_, err = h.checkout.Checkout(h.ctx, CheckoutInput{UserID: f.buyer.ID, ShippingAddressID: &address.ID})
	require.Error(t, err)
	appErr, ok := domainerrors.IsAppError(err)
	require.True(t, ok)
	require.Equal(t, 400, appErr.HTTPStatus)
}

func TestProcessPayment_SuccessIsIdempotent(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 2000, 10)
	order := h.checkoutOrder(t, f, 4)

	require.NoError(t, h.payment.ProcessPayment(h.ctx, *order.ChapaTxRef))
	require.NoError(t, h.payment.ProcessPayment(h.ctx, *order.ChapaTxRef))

	order, err := h.orders.GetByID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusPaid, order.Status)
	require.NotNil(t, order.BlnkTransactionID)

	payment, err := h.payments.GetByOrderID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, domain.PaymentStatusSucceeded, payment.Status)

	quantity, reserved := h.stock(t, f.variant.ID)
	require.Equal(t, 6, quantity)
	require.Equal(t, 0, reserved)

	require.Equal(t, 1, h.blnk.CallCount("CreateEscrowHold"))
	require.Equal(t, 1, h.enqueued("send_order_notification"))
}

func TestProcessPayment_AmountMismatchIsRejected(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 2000, 10)
	order := h.checkoutOrder(t, f, 1)
	h.setChapaVerify(chapa.StatusSuccess, 1)

	err := h.payment.ProcessPayment(h.ctx, *order.ChapaTxRef)
	require.Error(t, err)

	order, err = h.orders.GetByID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusPendingPayment, order.Status)
	require.Equal(t, 0, h.blnk.CallCount("CreateEscrowHold"))
}

func TestProcessPayment_PendingIsNotReady(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 2000, 10)
	order := h.checkoutOrder(t, f, 1)
	h.setChapaVerify(chapa.StatusPending, -1)

	err := h.payment.ProcessPayment(h.ctx, *order.ChapaTxRef)
	require.ErrorIs(t, err, jobs.ErrNotReady)
}

func TestProcessPayment_FailureReleasesStock(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 2000, 10)
	order := h.checkoutOrder(t, f, 3)
	h.setChapaVerify(chapa.StatusFailed, -1)

	require.NoError(t, h.payment.ProcessPayment(h.ctx, *order.ChapaTxRef))

	order, err := h.orders.GetByID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusCancelled, order.Status)

	quantity, reserved := h.stock(t, f.variant.ID)
	require.Equal(t, 10, quantity)
	require.Equal(t, 0, reserved)
}

func TestExpireUnpaidOrder_CancelsWhenChapaHasNoPayment(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 2000, 10)
	order := h.checkoutOrder(t, f, 2)
	h.chapa.VerifyPaymentFunc = func(ctx context.Context, txRef string) (*chapa.VerifyPaymentResponse, error) {
		return nil, &chapa.APIError{StatusCode: 404, Body: "not found"}
	}

	require.NoError(t, h.payment.ExpireUnpaidOrder(h.ctx, order.ID))

	order, err := h.orders.GetByID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusCancelled, order.Status)
	_, reserved := h.stock(t, f.variant.ID)
	require.Equal(t, 0, reserved)
}

func TestExpireUnpaidOrder_HonoursLatePayment(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 2000, 10)
	order := h.checkoutOrder(t, f, 2)

	require.NoError(t, h.payment.ExpireUnpaidOrder(h.ctx, order.ID))

	order, err := h.orders.GetByID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusPaid, order.Status)
}

func TestFulfilmentAndSettlement(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 10000, 10)
	order := h.paidOrder(t, f, 2)

	_, err := h.order.UpdateStatus(h.ctx, order.ID, domain.OrderStatusDelivered)
	require.Error(t, err, "delivered must not be settable through the status endpoint")

	_, err = h.order.ConfirmDelivery(h.ctx, order.ID)
	require.Error(t, err, "cannot confirm delivery before shipping")

	_, err = h.order.UpdateStatus(h.ctx, order.ID, domain.OrderStatusProcessing)
	require.NoError(t, err)
	carrier := "Ethio Post"
	_, err = h.order.MarkOrderShipped(h.ctx, MarkOrderShippedInput{OrderID: order.ID, Carrier: &carrier})
	require.NoError(t, err)

	delivered, err := h.order.ConfirmDelivery(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusDelivered, delivered.Status)
	require.Equal(t, 1, h.enqueued("settle_order"))

	require.NoError(t, h.order.SettleOrder(h.ctx, order.ID))
	require.NoError(t, h.order.SettleOrder(h.ctx, order.ID))

	require.Equal(t, 1, h.blnk.CallCount("CommitInflight"))
	require.Equal(t, 1, h.blnk.CallCount("CreditMerchant"))
	require.Equal(t, 1, h.blnk.CallCount("CreditPlatformFee"))

	txn, err := h.walletTxns.GetByOrderID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, domain.WalletTransactionTypeSaleCredit, txn.Type)
	require.Equal(t, int64(20000-1000), txn.AmountCents)
	require.Equal(t, 1, h.blnk.CallCount("ChargeProcessingFee"), "Chapa's charge is booked at settlement")

	merchant, err := h.merchants.GetByID(h.ctx, f.merchant.ID)
	require.NoError(t, err)
	require.Equal(t, int64(20000), merchant.TotalSalesCents)
}

func TestSettleOrder_RetriesAfterPartialFailure(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 10000, 10)
	order := h.paidOrder(t, f, 1)
	_, err := h.order.UpdateStatus(h.ctx, order.ID, domain.OrderStatusProcessing)
	require.NoError(t, err)
	_, err = h.order.MarkOrderShipped(h.ctx, MarkOrderShippedInput{OrderID: order.ID})
	require.NoError(t, err)
	_, err = h.order.ConfirmDelivery(h.ctx, order.ID)
	require.NoError(t, err)

	fail := true
	h.blnk.CreditMerchantFunc = func(ctx context.Context, req blnk.MerchantCreditRequest) (string, error) {
		if fail {
			return "", errors.New("blnk timeout")
		}
		return "txn_sale_credit", nil
	}

	require.Error(t, h.order.SettleOrder(h.ctx, order.ID))
	_, err = h.walletTxns.GetByOrderID(h.ctx, order.ID)
	require.Error(t, err, "no wallet entry may be written before the merchant is credited")

	fail = false
	require.NoError(t, h.order.SettleOrder(h.ctx, order.ID))
	txn, err := h.walletTxns.GetByOrderID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, "txn_sale_credit", txn.BlnkTransactionID)
}

func TestCancelPaidOrder_RestocksAndRefunds(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 5000, 10)
	order := h.paidOrder(t, f, 3)

	quantity, _ := h.stock(t, f.variant.ID)
	require.Equal(t, 7, quantity)

	cancelled, err := h.order.CancelOrder(h.ctx, order.ID, CancelledByBuyer)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusCancelled, cancelled.Status)

	quantity, reserved := h.stock(t, f.variant.ID)
	require.Equal(t, 10, quantity)
	require.Equal(t, 0, reserved)
	require.Equal(t, 1, h.enqueued("refund_order"))

	require.NoError(t, h.payment.RefundOrder(h.ctx, order.ID))
	require.NoError(t, h.payment.RefundOrder(h.ctx, order.ID))

	require.Equal(t, 1, h.blnk.CallCount("VoidInflight"))
	require.Equal(t, 1, h.chapa.CallCount("Refund"))

	refunded, err := h.orders.GetByID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusRefunded, refunded.Status)
	payment, err := h.payments.GetByOrderID(h.ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, domain.PaymentStatusRefunded, payment.Status)
}

func TestCancelUnpaidOrder_ReleasesReservation(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 5000, 10)
	order := h.checkoutOrder(t, f, 4)

	_, err := h.order.CancelOrder(h.ctx, order.ID, CancelledByBuyer)
	require.NoError(t, err)

	quantity, reserved := h.stock(t, f.variant.ID)
	require.Equal(t, 10, quantity)
	require.Equal(t, 0, reserved)
	require.Equal(t, 0, h.enqueued("refund_order"))
}

func TestBuyerCannotCancelProcessingOrder(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 5000, 10)
	order := h.paidOrder(t, f, 1)
	_, err := h.order.UpdateStatus(h.ctx, order.ID, domain.OrderStatusProcessing)
	require.NoError(t, err)

	_, err = h.order.CancelOrder(h.ctx, order.ID, CancelledByBuyer)
	require.Error(t, err)

	_, err = h.order.CancelOrder(h.ctx, order.ID, CancelledByMerchant)
	require.NoError(t, err)
}

func payoutInput(amount int64) domain.RequestPayoutInput {
	return domain.RequestPayoutInput{
		AmountCents: amount,
		Method:      domain.PayoutMethodTelebirr,
		Destination: "0911234567",
		AccountName: "Tigist Bekele",
		BankCode:    "855",
	}
}

func TestPayout_FullLifecycle(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 1000, 1)
	h.blnk.GetBalanceFunc = func(ctx context.Context, balanceID string) (*blnk.Balance, error) {
		return &blnk.Balance{BalanceCents: 50000, AvailableCents: 50000}, nil
	}
	h.blnk.HoldPayoutFunc = func(ctx context.Context, req blnk.PayoutRequest) (string, error) {
		return "hold_" + req.PayoutID.String(), nil
	}

	_, err := h.wallet.RequestPayout(h.ctx, f.merchant.ID, payoutInput(60000))
	require.Error(t, err, "payout above available balance must be refused")

	payout, err := h.wallet.RequestPayout(h.ctx, f.merchant.ID, payoutInput(30000))
	require.NoError(t, err)
	require.Equal(t, domain.PayoutStatusPending, payout.Status)
	require.Equal(t, "hold_"+payout.ID.String(), *payout.BlnkTransactionID)

	approved, err := h.wallet.ApprovePayout(h.ctx, payout.ID)
	require.NoError(t, err)
	require.Equal(t, domain.PayoutStatusProcessing, approved.Status)
	require.Equal(t, 1, h.enqueued("process_payout"))

	_, err = h.wallet.ApprovePayout(h.ctx, payout.ID)
	require.Error(t, err, "a payout can only be approved once")

	transferNotFound := true
	h.chapa.VerifyTransferFunc = func(ctx context.Context, reference string) (*chapa.VerifyTransferResponse, error) {
		if transferNotFound {
			return nil, &chapa.APIError{StatusCode: 404, Body: "not found"}
		}
		return &chapa.VerifyTransferResponse{Status: chapa.StatusSuccess, Data: chapa.VerifyTransferData{Reference: reference, Status: chapa.StatusSuccess}}, nil
	}

	require.ErrorIs(t, h.wallet.ProcessPayout(h.ctx, payout.ID), jobs.ErrNotReady)
	require.Equal(t, 1, h.chapa.CallCount("InitiateTransfer"))

	transferNotFound = false
	require.NoError(t, h.wallet.ProcessPayout(h.ctx, payout.ID))
	require.Equal(t, 1, h.chapa.CallCount("InitiateTransfer"), "transfer must not be sent twice")
	require.Equal(t, 1, h.blnk.CallCount("CommitInflight"))

	done, err := h.payouts.GetByID(h.ctx, payout.ID)
	require.NoError(t, err)
	require.Equal(t, domain.PayoutStatusCompleted, done.Status)
	require.NotNil(t, done.ProcessedAt)

	require.NoError(t, h.wallet.ProcessPayout(h.ctx, payout.ID))
	require.Equal(t, 1, h.blnk.CallCount("CommitInflight"))
}

func TestPayout_RejectReleasesHold(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 1000, 1)

	payout, err := h.wallet.RequestPayout(h.ctx, f.merchant.ID, payoutInput(5000))
	require.NoError(t, err)

	rejected, err := h.wallet.RejectPayout(h.ctx, payout.ID, "")
	require.NoError(t, err)
	require.Equal(t, domain.PayoutStatusFailed, rejected.Status)
	require.Equal(t, 1, h.blnk.CallCount("VoidInflight"))

	_, err = h.wallet.ApprovePayout(h.ctx, payout.ID)
	require.Error(t, err)
}

func TestPayout_FailedTransferVoidsHold(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 1000, 1)

	payout, err := h.wallet.RequestPayout(h.ctx, f.merchant.ID, payoutInput(5000))
	require.NoError(t, err)
	_, err = h.wallet.ApprovePayout(h.ctx, payout.ID)
	require.NoError(t, err)

	h.chapa.VerifyTransferFunc = func(ctx context.Context, reference string) (*chapa.VerifyTransferResponse, error) {
		return nil, &chapa.APIError{StatusCode: 404}
	}
	require.ErrorIs(t, h.wallet.ProcessPayout(h.ctx, payout.ID), jobs.ErrNotReady)

	h.chapa.VerifyTransferFunc = func(ctx context.Context, reference string) (*chapa.VerifyTransferResponse, error) {
		return &chapa.VerifyTransferResponse{Data: chapa.VerifyTransferData{Status: chapa.StatusFailed}}, nil
	}
	require.NoError(t, h.wallet.ProcessPayout(h.ctx, payout.ID))

	failed, err := h.payouts.GetByID(h.ctx, payout.ID)
	require.NoError(t, err)
	require.Equal(t, domain.PayoutStatusFailed, failed.Status)
	require.Equal(t, 1, h.blnk.CallCount("VoidInflight"))
	require.Equal(t, 0, h.blnk.CallCount("CommitInflight"))
}

func TestSetStock_CannotDropBelowReserved(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 1000, 10)
	h.addToCart(t, f.buyer.ID, f.variant.ID, 6, f.variant.PriceCents)

	_, err := h.inventorySvc.SetStock(h.ctx, f.variant.ID, 5)
	require.Error(t, err)

	inv, err := h.inventorySvc.SetStock(h.ctx, f.variant.ID, 6)
	require.NoError(t, err)
	require.Equal(t, 6, inv.Quantity)

	err = h.cart.reserve(h.ctx, f.variant.ID, 1)
	require.Error(t, err, "no stock left to reserve")
}

func TestCart_AddItemFailsWhenStockRunsOut(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 1000, 2)
	cart, err := h.cart.GetOrCreateCart(h.ctx, &f.buyer.ID, nil)
	require.NoError(t, err)

	_, err = h.cart.AddItem(h.ctx, cart.ID, f.variant.ID, 3, f.variant.PriceCents)
	require.Error(t, err)
	_, reserved := h.stock(t, f.variant.ID)
	require.Equal(t, 0, reserved)
}

func TestCart_MergeReleasesSurplusAndHandlesManyItems(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 1000, 20)

	other, err := h.variants.Create(h.ctx, testutil.NewVariantBuilder().WithProductID(f.product.ID).WithSKU("SKU-"+uuid.NewString()[:8]).Build())
	require.NoError(t, err)
	_, err = h.inventory.Create(h.ctx, &domain.Inventory{ID: uuid.New(), VariantID: other.ID, Quantity: 20})
	require.NoError(t, err)
	third, err := h.variants.Create(h.ctx, testutil.NewVariantBuilder().WithProductID(f.product.ID).WithSKU("SKU-"+uuid.NewString()[:8]).Build())
	require.NoError(t, err)
	_, err = h.inventory.Create(h.ctx, &domain.Inventory{ID: uuid.New(), VariantID: third.ID, Quantity: 20})
	require.NoError(t, err)

	h.addToCart(t, f.buyer.ID, f.variant.ID, 5, 1000)

	sessionID := "session-" + uuid.NewString()
	guest, err := h.cart.GetOrCreateCart(h.ctx, nil, &sessionID)
	require.NoError(t, err)
	_, err = h.cart.AddItem(h.ctx, guest.ID, f.variant.ID, 2, 1000)
	require.NoError(t, err)
	_, err = h.cart.AddItem(h.ctx, guest.ID, other.ID, 1, 1000)
	require.NoError(t, err)
	_, err = h.cart.AddItem(h.ctx, guest.ID, third.ID, 4, 1000)
	require.NoError(t, err)

	merged, err := h.cart.MergeCart(h.ctx, f.buyer.ID, sessionID)
	require.NoError(t, err)
	require.Len(t, merged.Items, 3)

	_, reserved := h.stock(t, f.variant.ID)
	require.Equal(t, 5, reserved, "overlapping line keeps the larger quantity and releases the rest")
	_, reserved = h.stock(t, other.ID)
	require.Equal(t, 1, reserved)
	_, reserved = h.stock(t, third.ID)
	require.Equal(t, 4, reserved)
}

func TestCart_ExpiryReleasesReservations(t *testing.T) {
	h := newHarness(t)
	f := h.seed(t, 1000, 10)

	sessionID := "session-" + uuid.NewString()
	guest, err := h.cart.GetOrCreateCart(h.ctx, nil, &sessionID)
	require.NoError(t, err)
	_, err = h.cart.AddItem(h.ctx, guest.ID, f.variant.ID, 4, 1000)
	require.NoError(t, err)

	require.NoError(t, h.cart.expire(h.ctx, guest))

	_, reserved := h.stock(t, f.variant.ID)
	require.Equal(t, 0, reserved)

	fresh, err := h.cart.GetOrCreateCart(h.ctx, nil, &sessionID)
	require.NoError(t, err, "a guest must get a new cart after the old one expired")
	require.NotEqual(t, guest.ID, fresh.ID)
}
