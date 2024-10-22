//go:generate sh -c "test store.go -nt $GOFILE && exit 0; mockgen -destination=./store.go -package=mocks github.com/peterstirrup/arbenheimer/internal/domain/usecases Store"
package mocks
