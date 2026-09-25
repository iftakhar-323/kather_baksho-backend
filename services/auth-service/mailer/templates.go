package mailer

import "fmt"

// VerifyEmail is sent right after Register and whenever the user clicks
// "Resend verification". The 6-digit code is good for 15 minutes.
func VerifyEmail(to, code string) Email {
	return Email{
		To:      to,
		Subject: "Verify your KatherBox email",
		HTML: fmt.Sprintf(`<div style="font-family:system-ui,sans-serif;max-width:520px;margin:auto">
			<h2 style="color:#2f7d4f">Welcome to KatherBox 🌱</h2>
			<p>Your verification code is:</p>
			<p style="font-size:32px;font-weight:bold;letter-spacing:6px;color:#2f7d4f">%s</p>
			<p>This code expires in 15 minutes. If you didn't sign up, you can safely ignore this email.</p>
		</div>`, code),
		Text: "Your KatherBox verification code: " + code + " (expires in 15 min)",
	}
}

// OrderPlaced is sent from the order controller after a successful Place.
func OrderPlaced(to, orderID string, total float64) Email {
	return Email{
		To:      to,
		Subject: fmt.Sprintf("Order #%s confirmed", orderID),
		HTML: fmt.Sprintf(`<div style="font-family:system-ui,sans-serif;max-width:520px;margin:auto">
			<h2 style="color:#2f7d4f">Thanks for your order! 🌿</h2>
			<p>Order <b>#%s</b> for <b>BDT %.2f</b> is being packed.</p>
			<p>You'll get another email when it's out for delivery.</p>
		</div>`, orderID, total),
		Text: fmt.Sprintf("Order #%s for BDT %.2f confirmed.", orderID, total),
	}
}

// PasswordReset is sent from the ForgotPassword endpoint.
func PasswordReset(to, link string) Email {
	return Email{
		To:      to,
		Subject: "Reset your Kather Baksho password",
		HTML: fmt.Sprintf(`<div style="font-family:system-ui,sans-serif;max-width:520px;margin:auto">
			<h2 style="color:#2f7d4f">Reset your password</h2>
			<p>Click the link below to set a new password. It expires in 30 minutes.</p>
			<p><a href="%s" style="background:#2f7d4f;color:#fff;padding:10px 18px;border-radius:6px;text-decoration:none">Reset password</a></p>
			<p style="font-size:12px;color:#888">If the button doesn't work, paste this link: %s</p>
		</div>`, link, link),
		Text: "Reset your KatherBox password (valid 30 min): " + link,
	}
}
