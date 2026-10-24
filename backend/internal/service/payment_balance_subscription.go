package service

import (
	"context"
	"fmt"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"strings"
	"time"
)

func (s *PaymentService) CreateBalanceSubscriptionOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error) {
	req.OrderType = payment.OrderTypeSubscription
	req.PaymentType = payment.TypeBalance
	cfg, err := s.configService.GetPaymentConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("get payment config: %w", err)
	}
	if !cfg.Enabled {
		return nil, infraerrors.Forbidden("PAYMENT_DISABLED", "payment system is disabled")
	}
	plan, err := s.validateSubOrder(ctx, req)
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user.Status != payment.EntityStatusActive {
		return nil, infraerrors.Forbidden("USER_INACTIVE", "user account is disabled")
	}
	if !user.AllowBalanceSubscriptionPurchase {
		return nil, infraerrors.Forbidden("BALANCE_SUBSCRIPTION_NOT_ALLOWED", "balance subscription purchase is not enabled for this user")
	}
	targetUser, err := s.resolveBalanceSubscriptionTargetUser(ctx, req, user)
	if err != nil {
		return nil, err
	}
	if s.notificationEmailService != nil {
		s.notificationEmailService.RememberRecipientLocale(ctx, req.UserID, user.Email, req.Locale)
	}
	now := time.Now()
	planPrice := EffectivePlanPrice(plan, now)
	balanceCost := planPrice
	if balanceCost <= 0 {
		return nil, infraerrors.BadRequest("INVALID_AMOUNT", "subscription balance cost must be positive")
	}
	order, err := s.createBalanceSubscriptionOrderInTx(ctx, req, user, targetUser, plan, cfg, planPrice, balanceCost, now)
	if err != nil {
		return nil, err
	}
	s.writeAuditLog(ctx, order.ID, "ORDER_CREATED", fmt.Sprintf("user:%d", req.UserID), map[string]any{
		"paymentAmount":     planPrice,
		"balanceCost":       balanceCost,
		"paymentType":       payment.TypeBalance,
		"orderType":         payment.OrderTypeSubscription,
		"paymentSource":     NormalizePaymentSource(req.PaymentSource),
		"balanceMultiplier": normalizeBalanceRechargeMultiplier(cfg.BalanceRechargeMultiplier),
		"targetUserID":      targetUser.ID,
		"targetUserEmail":   targetUser.Email,
	})
	s.invalidateRefundBalanceCache(ctx, req.UserID)
	if err := s.ExecuteSubscriptionFulfillment(ctx, order.ID); err != nil {
		// Funds are already captured; return the paid order so the client never suggests paying again.
		s.writeAuditLog(ctx, order.ID, "BALANCE_SUBSCRIPTION_FULFILLMENT_PENDING", "system", map[string]any{"error": err.Error()})
	}
	if refreshed, err := s.entClient.PaymentOrder.Get(ctx, order.ID); err == nil {
		order = refreshed
	}
	return &CreateOrderResponse{
		OrderID:     order.ID,
		Amount:      order.Amount,
		PayAmount:   order.PayAmount,
		FeeRate:     order.FeeRate,
		Status:      order.Status,
		ResultType:  payment.CreatePaymentResultOrderCreated,
		PaymentType: payment.TypeBalance,
		OutTradeNo:  order.OutTradeNo,
		Currency:    "USD",
		ExpiresAt:   order.ExpiresAt,
	}, nil
}

func (s *PaymentService) GetBalanceSubscriptionPayer(ctx context.Context, userID int64) (*User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

func (s *PaymentService) resolveBalanceSubscriptionTargetUser(ctx context.Context, req CreateOrderRequest, payer *User) (*User, error) {
	targetEmail := strings.TrimSpace(req.TargetUserEmail)
	if req.TargetUserID <= 0 && targetEmail == "" {
		return payer, nil
	}
	if (req.TargetUserID <= 0 || req.TargetUserID == payer.ID) && (targetEmail == "" || strings.EqualFold(targetEmail, payer.Email)) {
		return payer, nil
	}

	var (
		target *User
		err    error
	)
	if req.TargetUserID > 0 {
		target, err = s.userRepo.GetByID(ctx, req.TargetUserID)
	} else {
		target, err = s.userRepo.GetByEmail(ctx, targetEmail)
	}
	if err != nil || target == nil {
		return nil, infraerrors.NotFound("TARGET_USER_NOT_FOUND", "target user not found")
	}
	if targetEmail != "" && !strings.EqualFold(target.Email, targetEmail) {
		return nil, infraerrors.BadRequest("TARGET_USER_MISMATCH", "target user email does not match")
	}
	if target.Status != payment.EntityStatusActive {
		return nil, infraerrors.Forbidden("TARGET_USER_INACTIVE", "target user account is disabled")
	}
	return target, nil
}

func (s *PaymentService) createBalanceSubscriptionOrderInTx(ctx context.Context, req CreateOrderRequest, user *User, targetUser *User, plan *dbent.SubscriptionPlan, cfg *PaymentConfig, planPrice, balanceCost float64, now time.Time) (*dbent.PaymentOrder, error) {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.checkPendingLimit(ctx, tx, req.UserID, cfg.MaxPendingOrders); err != nil {
		return nil, err
	}
	if err := s.checkDailyLimit(ctx, tx, req.UserID, planPrice, cfg.DailyLimit); err != nil {
		return nil, err
	}
	outTradeNo, err := s.allocateOutTradeNo(ctx, tx)
	if err != nil {
		return nil, err
	}
	exp := now.Add(time.Minute)
	builder := tx.PaymentOrder.Create().
		SetUserID(req.UserID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetNillableUserNotes(psNilIfEmpty(user.Notes)).
		SetAmount(planPrice).
		SetPayAmount(balanceCost).
		SetFeeRate(0).
		SetRechargeCode("").
		SetOutTradeNo(outTradeNo).
		SetPaymentType(payment.TypeBalance).
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeSubscription).
		SetPlanID(plan.ID).
		SetSubscriptionGroupID(plan.GroupID).
		SetSubscriptionDays(psComputeValidityDays(plan.ValidityDays, plan.ValidityUnit)).
		SetStatus(OrderStatusPaid).
		SetPaidAt(now).
		SetExpiresAt(exp).
		SetClientIP(req.ClientIP).
		SetSrcHost(req.SrcHost)
	if snapshot := buildPaymentOrderSubscriptionTargetSnapshot(targetUser); snapshot != nil {
		builder.SetProviderSnapshot(snapshot)
	}
	order, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create balance subscription order: %w", err)
	}
	code := fmt.Sprintf("BAL-SUB-%d-%d", order.ID, now.UnixNano()%100000)
	order, err = tx.PaymentOrder.UpdateOneID(order.ID).SetRechargeCode(code).Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("set recharge code: %w", err)
	}
	updated, err := tx.Client().User.Update().
		Where(dbuser.IDEQ(req.UserID), dbuser.DeletedAtIsNil(), dbuser.StatusEQ(payment.EntityStatusActive), dbuser.AllowBalanceSubscriptionPurchaseEQ(true), dbuser.BalanceGTE(balanceCost)).
		AddBalance(-balanceCost).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("deduct balance: %w", err)
	}
	if updated == 0 {
		return nil, infraerrors.BadRequest("INSUFFICIENT_BALANCE", "insufficient balance")
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit balance subscription order: %w", err)
	}
	return order, nil
}
