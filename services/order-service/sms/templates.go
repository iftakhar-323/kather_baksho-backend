package sms

import "fmt"

// OrderNotificationType defines the lifecycle event for SMS notifications.
type OrderNotificationType string

const (
	EventOrderPlaced     OrderNotificationType = "ORDER_PLACED"
	EventPaymentReceived OrderNotificationType = "PAYMENT_RECEIVED"
	EventOutForDelivery  OrderNotificationType = "OUT_FOR_DELIVERY"
	EventOrderDelivered  OrderNotificationType = "ORDER_DELIVERED"
)

// BuildOrderMessage generates a bilingual SMS Message based on event type.
func BuildOrderMessage(phone, orderID, amount string, eventType OrderNotificationType, lang string) Message {
	var content string

	switch eventType {
	case EventOrderPlaced:
		if lang == "bn" {
			content = fmt.Sprintf("আপনার কাঠের বাক্স অর্ডার #%s নিশ্চিত হয়েছে। মোট পরিমাণ: ৳%s। ধন্যবাদ আমাদের সাথে থাকার জন্য!", orderID, amount)
		} else {
			content = fmt.Sprintf("Your Kather Baksho order #%s is confirmed. Total: BDT %s. Thank you for shopping with us!", orderID, amount)
		}

	case EventPaymentReceived:
		if lang == "bn" {
			content = fmt.Sprintf("আপনার অর্ডার #%s-এর জন্য ৳%s পেমেন্ট সফল হয়েছে। কাঠের বাক্স।", orderID, amount)
		} else {
			content = fmt.Sprintf("Payment of BDT %s for order #%s received successfully. Kather Baksho.", amount, orderID)
		}

	case EventOutForDelivery:
		if lang == "bn" {
			content = fmt.Sprintf("আপনার কাঠের বাক্স অর্ডার #%s ডেলিভারির উদ্দেশ্যে পাঠানো হয়েছে। শীঘ্রই পৌঁছাবে।", orderID)
		} else {
			content = fmt.Sprintf("Your Kather Baksho order #%s is out for delivery. Courier will arrive shortly.", orderID)
		}

	case EventOrderDelivered:
		if lang == "bn" {
			content = fmt.Sprintf("আপনার অর্ডার #%s সফলভাবে ডেলিভারি হয়েছে। আমাদের সাথে থাকার জন্য আন্তরিক ধন্যবাদ!", orderID)
		} else {
			content = fmt.Sprintf("Your order #%s has been successfully delivered. Thank you for choosing Kather Baksho!", orderID)
		}

	default:
		content = fmt.Sprintf("Update on your Kather Baksho order #%s.", orderID)
	}

	return Message{
		To:       phone,
		Content:  content,
		Language: lang,
	}
}
