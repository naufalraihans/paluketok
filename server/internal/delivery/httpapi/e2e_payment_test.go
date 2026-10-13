// Mock port.Payment untuk E2E: tidak memanggil DOKU sungguhan, mengembalikan
// PaymentResponse sah supaya alur order berlanjut ke webhook.
package httpapi

import (
	"context"

	"paluketok/server/internal/port"
)

type memPayment struct{}

func (memPayment) CreateOrder(_ context.Context, req port.PaymentRequest) (*port.PaymentResponse, error) {
	return &port.PaymentResponse{
		PaymentURL:  "https://sandbox.doku.example/pay/" + req.InvoiceNumber,
		ExpiredDate: "20261020120000",
	}, nil
}

func (memPayment) CheckStatus(_ context.Context, invoice string) (*port.PaymentStatus, error) {
	return &port.PaymentStatus{InvoiceNumber: invoice, Transaction: "SUCCESS", AmountIDR: 25000}, nil
}

var _ port.Payment = memPayment{}
