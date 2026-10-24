//go:build unit

package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestBalanceSubscriptionDeductionCommitsPaidOrderAndRollsBackOnInsufficientFunds(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	payer, err := client.User.Create().SetEmail("payer@example.com").SetPasswordHash("hash").SetBalance(15).SetAllowBalanceSubscriptionPurchase(true).Save(ctx)
	require.NoError(t, err)
	plan, err := client.SubscriptionPlan.Create().SetGroupID(7).SetName("Pro").SetPrice(10).Save(ctx)
	require.NoError(t, err)
	svc := &PaymentService{entClient: client}
	req := CreateOrderRequest{UserID: payer.ID, PlanID: plan.ID, PaymentType: payment.TypeBalance}
	user := &User{ID: payer.ID, Email: payer.Email}
	target := &User{ID: 99, Email: "recipient@example.com"}
	order, err := svc.createBalanceSubscriptionOrderInTx(ctx, req, user, target, plan, &PaymentConfig{}, 10, 10, time.Now())
	require.NoError(t, err)
	require.Equal(t, OrderStatusPaid, order.Status)
	require.NotNil(t, order.PaidAt)
	require.Equal(t, int64(99), paymentOrderSubscriptionTargetUserID(order))
	updated, err := client.User.Get(ctx, payer.ID)
	require.NoError(t, err)
	require.Equal(t, 5.0, updated.Balance)
	_, err = svc.createBalanceSubscriptionOrderInTx(ctx, req, user, target, plan, &PaymentConfig{}, 10, 10, time.Now())
	require.Error(t, err)
	count, err := client.PaymentOrder.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	updated, err = client.User.Get(ctx, payer.ID)
	require.NoError(t, err)
	require.Equal(t, 5.0, updated.Balance)
}
