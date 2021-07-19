package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// SecureToken creates a new random token
func SecureToken() string {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(err.Error()) // rand should never fail
	}
	return removePadding(base64.URLEncoding.EncodeToString(b))
}

func removePadding(token string) string {
	return strings.TrimRight(token, "=")
}

// GenerateOtp generates a random n digit otp
func GenerateOtp(digits int) (string, error) {
	upper := math.Pow10(digits)
	val, err := rand.Int(rand.Reader, big.NewInt(int64(upper)))
	if err != nil {
		return "", errors.WithMessage(err, "Error generating otp")
	}
	expr := "%0" + strconv.Itoa(digits) + "v"
	otp := fmt.Sprintf(expr, val.String())
	return otp, nil
}
