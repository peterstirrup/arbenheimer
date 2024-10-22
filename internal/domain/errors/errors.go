package arberrors

import (
	"errors"
)

var (
	ErrMarketNotFound         = errors.New("market not found")
	ErrInvalidMarketTimestamp = errors.New("market timestamp invalid")
)
