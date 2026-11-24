package domain

import "strings"

var BankAccounts = ItemSection{"bankAccounts", "bank", []string{"bankName", "branchName", "accountHolder", "accountNumber", "currency", "bankPrimary"}, []string{"bankName", "accountHolder", "accountNumber", "currency"}}

// BankNumberKey folds only ASCII letters and presentation spaces, preserving leading zeros.
func BankNumberKey(number string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == ' ':
			return -1
		case r >= 'a' && r <= 'z':
			return r - 'a' + 'A'
		default:
			return r
		}
	}, number)
}
