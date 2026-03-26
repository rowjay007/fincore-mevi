package commands

import (
	"context"
	"encoding/json"
	"errors"

	"fincore/pkg/ids"
	"fincore/services/payment-service/application/ports"
	"fincore/services/payment-service/domain"
)

func UnmarshalPaymentEvent(typ string, data []byte) (any, error) {
	switch typ {
	case (domain.PaymentInitiated{}).EventType():
		var e domain.PaymentInitiated
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case (domain.PaymentAuthorized{}).EventType():
		var e domain.PaymentAuthorized
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case (domain.PaymentSettled{}).EventType():
		var e domain.PaymentSettled
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case (domain.PaymentFailed{}).EventType():
		var e domain.PaymentFailed
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return e, nil
	default:
		return nil, errors.New("unknown event type")
	}
}

func LoadPayment(ctx context.Context, es ports.PaymentEventStore, paymentID ids.ID) (*domain.Payment, error) {
	evs, err := es.Read(ctx, paymentID.String(), 0, 10000)
	if err != nil {
		return nil, err
	}
	p := &domain.Payment{}
	for _, se := range evs {
		ev, err := UnmarshalPaymentEvent(se.Type, se.Data)
		if err != nil {
			return nil, err
		}
		if err := p.Apply(ev); err != nil {
			return nil, err
		}
	}
	return p, nil
}
