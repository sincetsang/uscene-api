package common

const ()

const (
	VerificationCodeStatusUnused = "unused"
	VerificationCodeStatusUsed   = "used"
)

const (
	VerificationCodeOrderStatusUnpaid = "unpaid"
	VerificationCodeOrderStatusPaid   = "paid"
	VerificationCodeOrderStatusCancel = "cancel"
)

const (
	ProductOrderStatusPendingPayment    = "pending_payment"
	ProductOrderStatusPaymentProcessing = "payment_processing"
	ProductOrderStatusPendingShipment   = "pending_shipment"
	ProductOrderStatusPendingReceipt    = "pending_receipt"
	ProductOrderStatusCompleted         = "completed"
	ProductOrderStatusCancelled         = "cancelled"
	ProductOrderStatusClosed            = "closed"
	// ProductOrderStatusRefundInProgress = "refund_in_progress"
	// ProductOrderStatusRefunded         = "refunded"
)

const (
	// ProductRefundStatusPendingApprove = "pending_approve"
	ProductRefundStatusNoRefund       = "no_refund"
	ProductRefundStatusRefunding      = "refunding"
	ProductRefundStatusRefunded       = "refunded"
	ProductRefundStatusRefundRejected = "refund_rejected"
)

const (
	UserImageStatusPending  = 0
	UserImageStatusApproved = 1
	UserImageStatusRejected = 2
)

const (
	SpotAuditStatusPending  = 0
	SpotAuditStatusApproved = 1
	SpotAuditStatusRejected = 2
)

const (
	OrderStatusUnpaid = "pending"
	OrderStatusPaid   = "paid"
	OrderStatusCancel = "expired"
)

const (
	PaymentBizTypeProduct = "product"
	PaymentBizTypeCode    = "code"
	PaymentBizTypeMember  = "member"
)

const (
	CheckInCommentStatusNormal  = 0
	CheckInCommentStatusHidden  = 1
	CheckInCommentStatusDeleted = 2
)

const (
	CheckInStatusPending  = 0
	CheckInStatusApproved = 1
	CheckInStatusRejected = 2
	CheckInStatusHidden   = 3
)

const (
	CheckInMessageTypeLike        = 1
	CheckInMessageTypeComment     = 2
	CheckInMessageTypeCommentLike = 3
)

const (
	MemberOrderStatusUnpaid            = "unpaid"
	MemberOrderStatusPaymentProcessing = "payment_processing"
	MemberOrderStatusPaid              = "paid"
	MemberOrderStatusCancel            = "cancel"
)
