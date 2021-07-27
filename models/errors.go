package models

// IsNotFoundError returns whether an error represents a "not found" error.
func IsNotFoundError(err error) bool {
	switch err.(type) {
	case UserNotFoundError:
		return true
	case ConfirmationTokenNotFoundError:
		return true
	case RefreshTokenNotFoundError:
		return true
	case InstanceNotFoundError:
		return true
	case TotpSecretNotFoundError:
		return true
	}
	return false
}

// UserNotFoundError represents when a user is not found.
type UserNotFoundError struct{}

func (e UserNotFoundError) Error() string {
	return "User not found"
}

// ConfirmationTokenNotFoundError represents when a confirmation token is not found.
type ConfirmationTokenNotFoundError struct{}

func (e ConfirmationTokenNotFoundError) Error() string {
	return "Confirmation Token not found"
}

// RefreshTokenNotFoundError represents when a refresh token is not found.
type RefreshTokenNotFoundError struct{}

func (e RefreshTokenNotFoundError) Error() string {
	return "Refresh Token not found"
}

// InstanceNotFoundError represents when an instance is not found.
type InstanceNotFoundError struct{}

func (e InstanceNotFoundError) Error() string {
	return "Instance not found"
}

type TotpSecretNotFoundError struct{}

func (e TotpSecretNotFoundError) Error() string {
	return "Totp Secret not found"
}
