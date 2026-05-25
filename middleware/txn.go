package middleware

import "github.com/julienschmidt/httprouter"

func WithTxn(next httprouter.Handle) httprouter.Handle {
	return next
}
