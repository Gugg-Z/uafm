package service

import (
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
	"time"
)

func TestLimitedOfferPriceExpiresAtBoundary(t *testing.T) {
	now := time.Now()
	expiry := now.Add(time.Minute)
	offer := 6.0
	plan := &dbent.SubscriptionPlan{Price: 10, LimitedOfferPrice: &offer, LimitedOfferExpiresAt: &expiry}
	require.Equal(t, 6.0, EffectivePlanPrice(plan, now))
	require.Equal(t, 10.0, EffectivePlanPrice(plan, expiry))
	require.Equal(t, 10.0, EffectivePlanPrice(plan, expiry.Add(time.Second)))
}

func TestLimitedOfferValidation(t *testing.T) {
	expiry := time.Now().Add(time.Hour)
	for _, price := range []float64{-1, 0, 10, 11, math.NaN(), math.Inf(1)} {
		require.Error(t, validatePlanLimitedOffer(10, &price, &expiry))
	}
	price := 5.0
	require.NoError(t, validatePlanLimitedOffer(10, &price, &expiry))
	require.Error(t, validatePlanLimitedOffer(10, &price, nil))
	require.Error(t, validatePlanLimitedOffer(10, nil, &expiry))
	require.NoError(t, validatePlanLimitedOffer(10, nil, nil))
}

func TestBalanceSubscriptionUsesUSDAndDoesNotAccrueRebate(t *testing.T) {
	order := &dbent.PaymentOrder{PaymentType: payment.TypeBalance, OrderType: payment.OrderTypeSubscription, Amount: 10, PayAmount: 10}
	require.Equal(t, "USD", PaymentOrderCurrency(order))
	_, _, eligible := (&PaymentService{}).affiliateRebateBaseAmountUSD(t.Context(), order)
	require.False(t, eligible)
}

func TestSubscriptionTargetFallsBackToPayerForLegacyOrders(t *testing.T) {
	order := &dbent.PaymentOrder{UserID: 1}
	require.Equal(t, int64(1), paymentOrderSubscriptionTargetUserID(order))
	order.ProviderSnapshot = buildPaymentOrderSubscriptionTargetSnapshot(&User{ID: 2, Email: "target@example.com"})
	require.Equal(t, int64(2), paymentOrderSubscriptionTargetUserID(order))
}
