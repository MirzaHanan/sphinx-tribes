package utils

import (
	"errors"
	"strings"
)

type Category string

const (
	WebDevelopment     Category = "Web development"
	BackendDevelopment Category = "Backend development"
	Design             Category = "Design"
	Other              Category = "Other"
)

type TicketReviewRequest struct {
	Value struct {
		FeatureUUID       string    `json:"featureUUID"`
		PhaseUUID         string    `json:"phaseUUID"`
		TicketUUID        string    `json:"ticketUUID" validate:"required"`
		TicketDescription string    `json:"ticketDescription" validate:"required"`
		TicketName        string    `json:"ticketName,omitempty"`
		TicketAmount      *int64    `json:"ticketAmount,omitempty"`
		TicketCategory    *Category `json:"ticketCategory,omitempty"`
	} `json:"value"`
	RequestUUID     string `json:"requestUUID"`
	SourceWebsocket string `json:"sourceWebsocket"`
}

func ValidateTicketReviewRequest(req *TicketReviewRequest) error {
	if req == nil {
		return errors.New("nil request")
	}
	if strings.TrimSpace(req.Value.TicketUUID) == "" {
		return errors.New("ticketUUID is required")
	}
	if strings.TrimSpace(req.Value.TicketDescription) == "" {
		return errors.New("ticketDescription is required")
	}
	return nil
}
